// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package tools

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// HealthStatus represents the health status of a tool
type HealthStatus struct {
	Healthy          bool
	LastChecked      time.Time
	LastError        error
	ConsecutiveFails int
	ResponseTime     time.Duration
}

// HealthChecker manages health checks for tools with caching and observability
type HealthChecker struct {
	mu                  sync.RWMutex
	cache               map[string]*HealthStatus
	checkInterval       time.Duration
	timeout             time.Duration
	maxConsecutiveFails int
}

// NewHealthChecker creates a new health checker with sensible defaults
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		cache:               make(map[string]*HealthStatus),
		checkInterval:       30 * time.Second,
		timeout:             5 * time.Second,
		maxConsecutiveFails: 3,
	}
}

// CheckHealth performs a health check on a tool with caching
func (h *HealthChecker) CheckHealth(ctx context.Context, tool Tool) error {
	logger := observability.GetLogger(ctx).
		WithComponent("tools.health").
		WithOperation("CheckHealth").
		With("tool", tool.Name())

	// Check cache first
	h.mu.RLock()
	status, exists := h.cache[tool.Name()]
	h.mu.RUnlock()

	// If cached and recent, return cached result
	if exists && time.Since(status.LastChecked) < h.checkInterval {
		if !status.Healthy {
			return gerror.Wrap(status.LastError, gerror.ErrCodeInternal, "cached health check failure").
				WithComponent("tools.health").
				WithOperation("CheckHealth").
				WithDetails("tool", tool.Name()).
				WithDetails("consecutive_fails", status.ConsecutiveFails)
		}
		return nil
	}

	// Perform actual health check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	start := time.Now()
	err := h.performHealthCheck(checkCtx, tool)
	responseTime := time.Since(start)

	// Update cache
	h.mu.Lock()
	defer h.mu.Unlock()

	if status == nil {
		status = &HealthStatus{}
		h.cache[tool.Name()] = status
	}

	status.LastChecked = time.Now()
	status.ResponseTime = responseTime

	if err != nil {
		status.Healthy = false
		status.LastError = err
		status.ConsecutiveFails++

		// Log warning if multiple consecutive failures
		if status.ConsecutiveFails >= h.maxConsecutiveFails {
			logger.WithError(err).Warn("Tool has multiple consecutive health check failures",
				"consecutive_fails", status.ConsecutiveFails,
				"response_time_ms", responseTime.Milliseconds())
		}

		// Log metric
		logger.With(
			"metric", "tool.health_check.failed",
			"tool", tool.Name(),
			"category", tool.Category(),
		).Debug("Health check failed")
	} else {
		status.Healthy = true
		status.LastError = nil
		status.ConsecutiveFails = 0

		// Log success metric
		logger.With(
			"metric", "tool.health_check.success",
			"tool", tool.Name(),
			"category", tool.Category(),
			"response_time_ms", responseTime.Milliseconds(),
		).Debug("Health check succeeded")
	}

	return err
}

// performHealthCheck executes the actual health check
func (h *HealthChecker) performHealthCheck(ctx context.Context, tool Tool) error {
	// Create a channel to receive the result
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- tool.HealthCheck()
	}()

	select {
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "health check timed out").
			WithComponent("tools.health").
			WithOperation("performHealthCheck").
			WithDetails("tool", tool.Name()).
			WithDetails("timeout", h.timeout.String())
	case err := <-resultCh:
		return err
	}
}

// GetHealthStatus returns the current health status for a tool
func (h *HealthChecker) GetHealthStatus(toolName string) (*HealthStatus, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status, exists := h.cache[toolName]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	statusCopy := *status
	return &statusCopy, true
}

// ClearCache removes all cached health statuses
func (h *HealthChecker) ClearCache() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.cache = make(map[string]*HealthStatus)
}

// SetCheckInterval updates the cache duration for health checks
func (h *HealthChecker) SetCheckInterval(interval time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.checkInterval = interval
}

// HealthReport generates a summary of all tool health statuses
type HealthReport struct {
	TotalTools     int
	HealthyTools   int
	UnhealthyTools int
	Tools          []ToolHealthSummary
}

// ToolHealthSummary contains health information for a single tool
type ToolHealthSummary struct {
	Name             string
	Healthy          bool
	LastChecked      time.Time
	ResponseTime     time.Duration
	ConsecutiveFails int
	LastError        string
}

// GenerateHealthReport creates a comprehensive health report
func (h *HealthChecker) GenerateHealthReport() HealthReport {
	h.mu.RLock()
	defer h.mu.RUnlock()

	report := HealthReport{
		TotalTools: len(h.cache),
		Tools:      make([]ToolHealthSummary, 0, len(h.cache)),
	}

	for toolName, status := range h.cache {
		summary := ToolHealthSummary{
			Name:             toolName,
			Healthy:          status.Healthy,
			LastChecked:      status.LastChecked,
			ResponseTime:     status.ResponseTime,
			ConsecutiveFails: status.ConsecutiveFails,
		}

		if status.LastError != nil {
			summary.LastError = status.LastError.Error()
		}

		report.Tools = append(report.Tools, summary)

		if status.Healthy {
			report.HealthyTools++
		} else {
			report.UnhealthyTools++
		}
	}

	return report
}

// BackgroundHealthChecker runs periodic health checks on registered tools
type BackgroundHealthChecker struct {
	checker  *HealthChecker
	registry *ToolRegistry
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewBackgroundHealthChecker creates a background health checker
func NewBackgroundHealthChecker(registry *ToolRegistry, interval time.Duration) *BackgroundHealthChecker {
	return &BackgroundHealthChecker{
		checker:  NewHealthChecker(),
		registry: registry,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins background health checking
func (b *BackgroundHealthChecker) Start(ctx context.Context) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		ticker := time.NewTicker(b.interval)
		defer ticker.Stop()

		logger := observability.GetLogger(ctx).
			WithComponent("tools.health").
			WithOperation("BackgroundChecker")

		for {
			select {
			case <-ctx.Done():
				logger.Info("Background health checker stopping due to context cancellation")
				return
			case <-b.stopCh:
				logger.Info("Background health checker stopping")
				return
			case <-ticker.C:
				b.checkAllTools(ctx)
			}
		}
	}()
}

// checkAllTools performs health checks on all registered tools
func (b *BackgroundHealthChecker) checkAllTools(ctx context.Context) {
	tools := b.registry.ListTools()

	logger := observability.GetLogger(ctx).
		WithComponent("tools.health").
		WithOperation("checkAllTools")

	logger.Debug("Starting health check cycle", "tool_count", len(tools))

	for _, tool := range tools {
		// Check each tool in a separate goroutine with timeout
		go func(t Tool) {
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := b.checker.CheckHealth(checkCtx, t); err != nil {
				logger.WithError(err).Debug("Health check failed for tool", "tool", t.Name())
			}
		}(tool)
	}
}

// Stop halts the background health checker
func (b *BackgroundHealthChecker) Stop() {
	close(b.stopCh)
	b.wg.Wait()
}

// GetHealthReport returns the current health report
func (b *BackgroundHealthChecker) GetHealthReport() HealthReport {
	return b.checker.GenerateHealthReport()
}

// HealthCheckMiddleware creates a middleware that checks tool health before execution
func HealthCheckMiddleware(checker *HealthChecker) func(Tool) Tool {
	return func(tool Tool) Tool {
		return &healthCheckWrapper{
			Tool:    tool,
			checker: checker,
		}
	}
}

// healthCheckWrapper wraps a tool to perform health checks before execution
type healthCheckWrapper struct {
	Tool
	checker *HealthChecker
}

// Execute performs a health check before executing the tool
func (w *healthCheckWrapper) Execute(ctx context.Context, input string) (*ToolResult, error) {
	// Check health first
	if err := w.checker.CheckHealth(ctx, w.Tool); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "tool health check failed").
			WithComponent("tools.health").
			WithOperation("Execute").
			WithDetails("tool", w.Name())
	}

	// If healthy, proceed with execution
	return w.Tool.Execute(ctx, input)
}
