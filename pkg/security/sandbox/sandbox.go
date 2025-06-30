// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
)

// GuildSandbox implements the main sandboxing functionality
type GuildSandbox struct {
	config        SandboxConfig
	isolator      Isolator
	limiter       ResourceLimiter
	monitor       SecurityMonitor
	networkFilter NetworkFilter
	logger        observability.Logger
	stats         *SandboxStats
	statsMu       sync.RWMutex
	tempDirs      map[string]string
	tempDirsMu    sync.RWMutex
}

// NewGuildSandbox creates a new sandbox instance
func NewGuildSandbox(ctx context.Context, config SandboxConfig) (*GuildSandbox, error) {
	logger := observability.GetLogger(ctx).
		WithComponent("GuildSandbox")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildSandbox").
			WithOperation("NewGuildSandbox")
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeValidation, "invalid sandbox configuration").
			WithComponent("GuildSandbox").
			WithOperation("NewGuildSandbox")
	}

	sandbox := &GuildSandbox{
		config:   config,
		logger:   logger,
		tempDirs: make(map[string]string),
		stats: &SandboxStats{
			LastActivity: time.Now(),
		},
	}

	// Initialize components
	var err error

	sandbox.isolator, err = NewFilesystemIsolator(ctx, config)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create filesystem isolator").
			WithComponent("GuildSandbox").
			WithOperation("NewGuildSandbox")
	}

	sandbox.limiter, err = NewResourceLimiter(ctx, config.ResourceLimits)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create resource limiter").
			WithComponent("GuildSandbox").
			WithOperation("NewGuildSandbox")
	}

	if config.EnableNetworking {
		sandbox.networkFilter, err = NewNetworkFilter(ctx, config.AllowedHosts)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create network filter").
				WithComponent("GuildSandbox").
				WithOperation("NewGuildSandbox")
		}
	}

	sandbox.monitor, err = NewSecurityMonitor(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create security monitor").
			WithComponent("GuildSandbox").
			WithOperation("NewGuildSandbox")
	}

	// Start monitoring
	if err := sandbox.monitor.StartMonitoring(ctx); err != nil {
		logger.WithError(err).Warn("Failed to start security monitoring")
	}

	logger.Info("Guild sandbox initialized successfully")
	return sandbox, nil
}

// Execute runs a command in the sandboxed environment
func (gs *GuildSandbox) Execute(ctx context.Context, cmd Command) (*ExecutionResult, error) {
	logger := gs.logger.WithOperation("Execute")

	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	startTime := time.Now()

	// Update stats
	gs.updateStats(func(stats *SandboxStats) {
		stats.TotalExecutions++
		stats.LastActivity = time.Now()
	})

	// Validate command
	if err := gs.ValidateCommand(ctx, cmd); err != nil {
		gs.updateStats(func(stats *SandboxStats) {
			stats.BlockedCommands++
		})
		return nil, gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "command validation failed").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	// Monitor command for security
	if err := gs.monitor.MonitorCommand(ctx, cmd); err != nil {
		logger.WithError(err).Warn("Security monitoring failed")
	}

	// Isolate command
	isolatedCmd, err := gs.isolator.Isolate(ctx, cmd)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "command isolation failed").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	// Prepare exec.Cmd
	execCmd := exec.CommandContext(ctx, isolatedCmd.Original.Name, isolatedCmd.Original.Args...)

	// Set working directory
	if isolatedCmd.WorkingDir != "" {
		execCmd.Dir = isolatedCmd.WorkingDir
	} else if cmd.Dir != "" {
		// Validate and use original directory if safe
		if err := gs.isolator.ValidatePath(cmd.Dir, PathOperationExecute); err == nil {
			execCmd.Dir = cmd.Dir
		}
	}

	// Set environment
	execCmd.Env = gs.buildEnvironment(isolatedCmd)

	// Apply resource limits
	if err := gs.limiter.ApplyLimits(ctx, execCmd, isolatedCmd.ResourceLimits); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeResourceLimit, "failed to apply resource limits").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	// Execute command with monitoring
	result := &ExecutionResult{}

	// Capture output
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stdout pipe").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create stderr pipe").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start command").
			WithComponent("GuildSandbox").
			WithOperation("Execute")
	}

	// Monitor resource usage
	go func() {
		usage, err := gs.limiter.MonitorExecution(ctx, execCmd)
		if err != nil {
			logger.WithError(err).Warn("Resource monitoring failed")
		} else {
			result.ResourceUsage = usage
		}
	}()

	// Read output
	stdoutBytes, err := readPipe(stdout)
	if err != nil {
		logger.WithError(err).Warn("Failed to read stdout")
	}
	result.Stdout = string(stdoutBytes)

	stderrBytes, err := readPipe(stderr)
	if err != nil {
		logger.WithError(err).Warn("Failed to read stderr")
	}
	result.Stderr = string(stderrBytes)

	// Wait for completion
	err = execCmd.Wait()
	result.Duration = time.Since(startTime)

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.Error = err
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	// Update average execution time
	gs.updateStats(func(stats *SandboxStats) {
		if stats.TotalExecutions > 0 {
			stats.AverageExecution = time.Duration(
				(int64(stats.AverageExecution)*stats.TotalExecutions + int64(result.Duration)) /
					(stats.TotalExecutions + 1),
			)
		} else {
			stats.AverageExecution = result.Duration
		}
	})

	logger.Debug("Command executed successfully",
		"command", cmd.String(),
		"exit_code", result.ExitCode,
		"duration", result.Duration,
	)

	return result, nil
}

// ValidateCommand checks if a command is safe to execute
func (gs *GuildSandbox) ValidateCommand(ctx context.Context, cmd Command) error {
	logger := gs.logger.WithOperation("ValidateCommand")

	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildSandbox").
			WithOperation("ValidateCommand")
	}

	// Check if command is empty
	if cmd.Name == "" {
		return gerror.New(gerror.ErrCodeValidation, "command name cannot be empty", nil).
			WithComponent("GuildSandbox").
			WithOperation("ValidateCommand")
	}

	// Check against dangerous commands
	if gs.isDangerousCommand(cmd) {
		logger.Warn("Dangerous command blocked", "command", cmd.String())
		return gerror.New(gerror.ErrCodeSecurityViolation, "dangerous command not allowed", nil).
			WithComponent("GuildSandbox").
			WithOperation("ValidateCommand").
			WithDetails("command", cmd.String())
	}

	// Validate working directory
	if cmd.Dir != "" {
		if err := gs.isolator.ValidatePath(cmd.Dir, PathOperationExecute); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "invalid working directory").
				WithComponent("GuildSandbox").
				WithOperation("ValidateCommand")
		}
	}

	// Validate file arguments (paths in command args)
	for _, arg := range cmd.Args {
		if gs.looksLikePath(arg) {
			if err := gs.validateArgPath(arg); err != nil {
				return gerror.Wrap(err, gerror.ErrCodeSecurityViolation, "invalid path in command arguments").
					WithComponent("GuildSandbox").
					WithOperation("ValidateCommand")
			}
		}
	}

	return nil
}

// GetConfig returns the current sandbox configuration
func (gs *GuildSandbox) GetConfig() SandboxConfig {
	return gs.config
}

// UpdateConfig updates the sandbox configuration
func (gs *GuildSandbox) UpdateConfig(config SandboxConfig) error {
	if err := validateConfig(config); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid sandbox configuration").
			WithComponent("GuildSandbox").
			WithOperation("UpdateConfig")
	}

	gs.config = config
	gs.logger.Info("Sandbox configuration updated")
	return nil
}

// GetStats returns sandbox usage statistics
func (gs *GuildSandbox) GetStats(ctx context.Context) (*SandboxStats, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("GuildSandbox").
			WithOperation("GetStats")
	}

	gs.statsMu.RLock()
	defer gs.statsMu.RUnlock()

	// Get current resource usage
	usage, err := gs.limiter.GetUsage(ctx)
	if err != nil {
		gs.logger.WithError(err).Warn("Failed to get current resource usage")
		usage = &ResourceUsage{}
	}

	stats := *gs.stats
	stats.ResourceUsage = *usage
	return &stats, nil
}

// Close releases sandbox resources
func (gs *GuildSandbox) Close() error {
	gs.logger.Info("Closing sandbox")

	var errors []error

	// Stop monitoring
	if err := gs.monitor.StopMonitoring(); err != nil {
		errors = append(errors, err)
	}

	// Close isolator
	if err := gs.isolator.Close(); err != nil {
		errors = append(errors, err)
	}

	// Clean up temp directories
	gs.tempDirsMu.Lock()
	for agentID, tempDir := range gs.tempDirs {
		if err := os.RemoveAll(tempDir); err != nil {
			gs.logger.WithError(err).Warn("Failed to clean up temp directory", "agent_id", agentID, "path", tempDir)
		}
	}
	gs.tempDirs = make(map[string]string)
	gs.tempDirsMu.Unlock()

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to close sandbox components", errors[0]).
			WithComponent("GuildSandbox").
			WithOperation("Close")
	}

	return nil
}

// Helper methods

func (gs *GuildSandbox) updateStats(fn func(*SandboxStats)) {
	gs.statsMu.Lock()
	defer gs.statsMu.Unlock()
	fn(gs.stats)
}

func (gs *GuildSandbox) isDangerousCommand(cmd Command) bool {
	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		"format",
		"fdisk",
		"mkfs",
		"dd if=",
		"sudo",
		"su -",
		"passwd",
		"useradd",
		"userdel",
		"shutdown",
		"reboot",
		"halt",
		"init 0",
		"init 6",
		"poweroff",
		"curl | sh",
		"wget | sh",
		"curl | bash",
		"wget | bash",
		"> /dev/sd",
		"> /dev/hd",
		"chmod 777 /",
		"chown root:",
	}

	// Check both the full command string and individual args
	cmdStr := cmd.String()
	cmdLower := strings.ToLower(cmdStr)

	// Also check args joined with spaces to catch pipe operations
	if len(cmd.Args) > 0 {
		fullCmd := strings.ToLower(cmd.Name + " " + strings.Join(cmd.Args, " "))
		for _, d := range dangerous {
			if strings.Contains(fullCmd, d) {
				return true
			}
		}
	}

	// Special check for pipe operations (curl/wget followed by | sh/bash)
	if (cmd.Name == "curl" || cmd.Name == "wget") && len(cmd.Args) > 0 {
		argsStr := strings.Join(cmd.Args, " ")
		if strings.Contains(argsStr, "|") && (strings.Contains(argsStr, "sh") || strings.Contains(argsStr, "bash")) {
			return true
		}
	}

	for _, d := range dangerous {
		if strings.Contains(cmdLower, d) {
			return true
		}
	}

	return false
}

func (gs *GuildSandbox) looksLikePath(arg string) bool {
	// Simple heuristic: contains / or . or ~
	return strings.Contains(arg, "/") || strings.HasPrefix(arg, ".") || strings.HasPrefix(arg, "~")
}

func (gs *GuildSandbox) validateArgPath(path string) error {
	// Expand relative paths
	if !filepath.IsAbs(path) {
		path = filepath.Join(gs.config.ProjectRoot, path)
	}

	// Check if path is within allowed boundaries
	return gs.isolator.ValidatePath(path, PathOperationRead)
}

func (gs *GuildSandbox) buildEnvironment(isolated IsolatedCommand) []string {
	env := make([]string, 0)

	// Start with essential environment variables
	essential := []string{
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOME=" + gs.isolator.GetTempDir("default"),
		"USER=guild-agent",
		"SHELL=/bin/bash",
		"TERM=xterm",
	}
	env = append(env, essential...)

	// Add sandbox-specific environment
	for key, value := range isolated.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add config environment
	for key, value := range gs.config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}

func validateConfig(config SandboxConfig) error {
	if config.ProjectRoot == "" {
		return gerror.New(gerror.ErrCodeValidation, "project root cannot be empty", nil)
	}

	if !filepath.IsAbs(config.ProjectRoot) {
		return gerror.New(gerror.ErrCodeValidation, "project root must be absolute path", nil)
	}

	// Check if project root exists
	if _, err := os.Stat(config.ProjectRoot); os.IsNotExist(err) {
		return gerror.New(gerror.ErrCodeValidation, "project root directory does not exist", nil).
			WithDetails("project_root", config.ProjectRoot)
	}

	return nil
}

// readPipe reads all data from a pipe
func readPipe(pipe interface{}) ([]byte, error) {
	// This implementation reads from io.ReadCloser (stdout/stderr pipes)
	if reader, ok := pipe.(io.ReadCloser); ok {
		defer reader.Close()
		return io.ReadAll(reader)
	}
	return []byte{}, gerror.New(gerror.ErrCodeInternal, "invalid pipe type", nil)
}
