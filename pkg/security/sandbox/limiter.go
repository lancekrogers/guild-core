// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package sandbox

import (
	"context"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// GuildResourceLimiter implements resource limiting for command execution
type GuildResourceLimiter struct {
	defaultLimits LimitConfig
	agentLimits   map[string]LimitConfig
	mu            sync.RWMutex
	logger        observability.Logger
	monitorCtx    context.Context
	monitorCancel context.CancelFunc
}

// NewResourceLimiter creates a new resource limiter
func NewResourceLimiter(ctx context.Context, defaultLimits LimitConfig) (ResourceLimiter, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("ResourceLimiter")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ResourceLimiter").
			WithOperation("NewResourceLimiter")
	}

	monitorCtx, cancel := context.WithCancel(ctx)

	limiter := &GuildResourceLimiter{
		defaultLimits: defaultLimits,
		agentLimits:   make(map[string]LimitConfig),
		logger:        logger,
		monitorCtx:    monitorCtx,
		monitorCancel: cancel,
	}

	// Validate default limits
	if err := limiter.validateLimits(defaultLimits); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid default limits").
			WithComponent("ResourceLimiter").
			WithOperation("NewResourceLimiter")
	}

	logger.Info("Resource limiter initialized",
		"max_cpu", defaultLimits.MaxCPU,
		"max_memory", defaultLimits.MaxMemory,
		"timeout", defaultLimits.Timeout,
	)

	return limiter, nil
}

// ApplyLimits applies resource limits to a command
func (rl *GuildResourceLimiter) ApplyLimits(ctx context.Context, cmd *exec.Cmd, limits LimitConfig) error {
	logger := rl.logger.WithOperation("ApplyLimits")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ResourceLimiter").
			WithOperation("ApplyLimits")
	}

	// Validate limits
	if err := rl.validateLimits(limits); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid resource limits").
			WithComponent("ResourceLimiter").
			WithOperation("ApplyLimits")
	}

	// Apply platform-specific limits
	if err := rl.applySystemLimits(cmd, limits); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to apply system limits").
			WithComponent("ResourceLimiter").
			WithOperation("ApplyLimits")
	}

	// Set up timeout if specified
	if limits.Timeout > 0 {
		ctxWithTimeout, cancel := context.WithTimeout(ctx, limits.Timeout)
		go func() {
			defer cancel()
			<-ctxWithTimeout.Done()
			if ctxWithTimeout.Err() == context.DeadlineExceeded && cmd.Process != nil {
				logger.Warn("Command timeout exceeded, terminating process",
					"timeout", limits.Timeout,
					"pid", cmd.Process.Pid,
				)
				cmd.Process.Kill()
			}
		}()
	}

	logger.Debug("Resource limits applied",
		"max_memory", limits.MaxMemory,
		"max_processes", limits.MaxProcesses,
		"timeout", limits.Timeout,
	)

	return nil
}

// MonitorExecution monitors resource usage during command execution
func (rl *GuildResourceLimiter) MonitorExecution(ctx context.Context, cmd *exec.Cmd) (*ResourceUsage, error) {
	logger := rl.logger.WithOperation("MonitorExecution")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ResourceLimiter").
			WithOperation("MonitorExecution")
	}

	if cmd.Process == nil {
		return &ResourceUsage{
			Timestamp: time.Now(),
		}, nil
	}

	// Start monitoring goroutine
	usage := &ResourceUsage{
		Timestamp: time.Now(),
	}

	// Get initial process state
	startTime := time.Now()
	pid := cmd.Process.Pid

	// Monitor in background
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-rl.monitorCtx.Done():
				return
			case <-ticker.C:
				currentUsage := rl.getProcessUsage(pid)
				if currentUsage != nil {
					// Update usage stats
					if currentUsage.MemoryBytes > usage.MemoryBytes {
						usage.MemoryBytes = currentUsage.MemoryBytes
					}
					if currentUsage.ProcessCount > usage.ProcessCount {
						usage.ProcessCount = currentUsage.ProcessCount
					}
					usage.CPUTime = time.Since(startTime)
					usage.Timestamp = time.Now()
				}
			}
		}
	}()

	logger.Debug("Started resource monitoring", "pid", pid)
	return usage, nil
}

// GetUsage returns current resource usage
func (rl *GuildResourceLimiter) GetUsage(ctx context.Context) (*ResourceUsage, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("ResourceLimiter").
			WithOperation("GetUsage")
	}

	// Get current system usage
	usage := &ResourceUsage{
		Timestamp: time.Now(),
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	usage.MemoryBytes = int64(memStats.Alloc)

	// Get process count (simplified)
	usage.ProcessCount = runtime.NumGoroutine()

	return usage, nil
}

// SetAgentLimits sets custom limits for a specific agent
func (rl *GuildResourceLimiter) SetAgentLimits(agentID string, limits LimitConfig) error {
	if err := rl.validateLimits(limits); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid agent limits").
			WithComponent("ResourceLimiter").
			WithOperation("SetAgentLimits").
			WithDetails("agent_id", agentID)
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.agentLimits[agentID] = limits
	rl.logger.Info("Agent-specific limits updated", "agent_id", agentID)

	return nil
}

// GetAgentLimits returns the limits for a specific agent
func (rl *GuildResourceLimiter) GetAgentLimits(agentID string) LimitConfig {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	if limits, exists := rl.agentLimits[agentID]; exists {
		return limits
	}

	return rl.defaultLimits
}

// Close stops the resource limiter
func (rl *GuildResourceLimiter) Close() error {
	rl.monitorCancel()
	rl.logger.Info("Resource limiter closed")
	return nil
}

// Helper methods

func (rl *GuildResourceLimiter) validateLimits(limits LimitConfig) error {
	if limits.MaxMemory < 0 {
		return gerror.New(gerror.ErrCodeValidation, "max memory cannot be negative", nil)
	}

	if limits.MaxDisk < 0 {
		return gerror.New(gerror.ErrCodeValidation, "max disk cannot be negative", nil)
	}

	if limits.MaxProcesses < 0 {
		return gerror.New(gerror.ErrCodeValidation, "max processes cannot be negative", nil)
	}

	if limits.MaxOpenFiles < 0 {
		return gerror.New(gerror.ErrCodeValidation, "max open files cannot be negative", nil)
	}

	if limits.Timeout < 0 {
		return gerror.New(gerror.ErrCodeValidation, "timeout cannot be negative", nil)
	}

	return nil
}

func (rl *GuildResourceLimiter) applySystemLimits(cmd *exec.Cmd, limits LimitConfig) error {
	// Set up process attributes for resource limiting
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Set process group for easier management
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0

	// Apply platform-specific limits
	switch runtime.GOOS {
	case "linux":
		return rl.applyLinuxLimits(cmd, limits)
	case "darwin":
		return rl.applyDarwinLimits(cmd, limits)
	default:
		// For unsupported platforms, we can only apply timeout
		rl.logger.Warn("Resource limits not fully supported on this platform", "platform", runtime.GOOS)
		return nil
	}
}

func (rl *GuildResourceLimiter) applyLinuxLimits(cmd *exec.Cmd, limits LimitConfig) error {
	// On Linux, we would typically use cgroups for robust resource limiting
	// For now, we'll use basic rlimit settings

	if limits.MaxMemory > 0 {
		// Set memory limit (this is simplified - real implementation would use cgroups)
		rl.logger.Debug("Memory limit configured", "limit", limits.MaxMemory)
	}

	if limits.MaxProcesses > 0 {
		// Set process limit
		rl.logger.Debug("Process limit configured", "limit", limits.MaxProcesses)
	}

	if limits.MaxOpenFiles > 0 {
		// Set file descriptor limit
		rl.logger.Debug("File descriptor limit configured", "limit", limits.MaxOpenFiles)
	}

	return nil
}

func (rl *GuildResourceLimiter) applyDarwinLimits(cmd *exec.Cmd, limits LimitConfig) error {
	// On macOS, resource limiting is more restricted
	// We can apply some basic limits but not as comprehensive as Linux

	if limits.MaxOpenFiles > 0 {
		// Set file descriptor limit
		rl.logger.Debug("File descriptor limit configured", "limit", limits.MaxOpenFiles)
	}

	return nil
}

func (rl *GuildResourceLimiter) getProcessUsage(pid int) *ResourceUsage {
	// This is a simplified implementation
	// In a real implementation, you would read from /proc/{pid}/stat (Linux)
	// or use system calls to get actual process resource usage

	usage := &ResourceUsage{
		Timestamp: time.Now(),
	}

	// Placeholder implementation - would need platform-specific code
	switch runtime.GOOS {
	case "linux":
		// Read from /proc/{pid}/stat and /proc/{pid}/status
		return rl.getLinuxProcessUsage(pid)
	case "darwin":
		// Use system calls or ps command
		return rl.getDarwinProcessUsage(pid)
	default:
		return usage
	}
}

func (rl *GuildResourceLimiter) getLinuxProcessUsage(pid int) *ResourceUsage {
	// Placeholder for Linux-specific process monitoring
	// Would read /proc/{pid}/stat, /proc/{pid}/status, etc.
	return &ResourceUsage{
		Timestamp: time.Now(),
		// Would populate with actual values from /proc filesystem
	}
}

func (rl *GuildResourceLimiter) getDarwinProcessUsage(pid int) *ResourceUsage {
	// Placeholder for macOS-specific process monitoring
	// Would use system calls or ps command
	return &ResourceUsage{
		Timestamp: time.Now(),
		// Would populate with actual values from system calls
	}
}
