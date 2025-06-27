// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package lsp

import (
	"context"
	"time"

	"github.com/lancekrogers/guild/pkg/observability"
)

// LifecycleManager manages the lifecycle of language servers
type LifecycleManager struct {
	manager       *Manager
	cleanupTicker *time.Ticker
	done          chan bool
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(manager *Manager) *LifecycleManager {
	return &LifecycleManager{
		manager: manager,
		done:    make(chan bool),
	}
}

// Start starts the lifecycle manager
func (l *LifecycleManager) Start(ctx context.Context, cleanupInterval time.Duration, idleTimeout time.Duration) {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Starting LSP lifecycle manager",
		"cleanup_interval", cleanupInterval,
		"idle_timeout", idleTimeout)

	l.cleanupTicker = time.NewTicker(cleanupInterval)

	go func() {
		for {
			select {
			case <-l.cleanupTicker.C:
				if err := l.manager.serverManager.CleanupIdleServers(ctx, idleTimeout); err != nil {
					logger.ErrorContext(ctx, "Failed to cleanup idle servers",
						"error", err)
				}
			case <-l.done:
				logger.InfoContext(ctx, "Stopping LSP lifecycle manager")
				return
			case <-ctx.Done():
				logger.InfoContext(ctx, "Context cancelled, stopping LSP lifecycle manager")
				return
			}
		}
	}()
}

// Stop stops the lifecycle manager
func (l *LifecycleManager) Stop() {
	if l.cleanupTicker != nil {
		l.cleanupTicker.Stop()
	}
	close(l.done)
}

// RestartServer restarts a language server
func (l *LifecycleManager) RestartServer(ctx context.Context, language string, workspace string) error {
	logger := observability.GetLogger(ctx)
	logger.InfoContext(ctx, "Restarting language server",
		"language", language,
		"workspace", workspace)

	// Stop the server
	if err := l.manager.serverManager.StopServer(ctx, language, workspace); err != nil {
		logger.ErrorContext(ctx, "Failed to stop server for restart",
			"language", language,
			"workspace", workspace,
			"error", err)
	}

	// Give it a moment to clean up
	time.Sleep(100 * time.Millisecond)

	// Start it again
	_, err := l.manager.serverManager.GetServer(ctx, language, workspace)
	if err != nil {
		return err
	}

	logger.InfoContext(ctx, "Successfully restarted language server",
		"language", language,
		"workspace", workspace)

	return nil
}

// HealthCheck performs health checks on all active servers
func (l *LifecycleManager) HealthCheck(ctx context.Context) map[string]HealthStatus {
	logger := observability.GetLogger(ctx)
	servers := l.manager.GetActiveServers()

	results := make(map[string]HealthStatus)

	for _, server := range servers {
		healthy, err := l.manager.serverManager.CheckServerHealth(ctx, server.Language, server.Workspace)

		status := HealthStatus{
			Language:  server.Language,
			Workspace: server.Workspace,
			Healthy:   healthy,
			StartTime: server.StartTime,
			LastUsed:  server.LastUsed,
		}

		if err != nil {
			status.Error = err.Error()
		}

		results[server.Key] = status

		if !healthy {
			logger.WarnContext(ctx, "Unhealthy language server detected",
				"language", server.Language,
				"workspace", server.Workspace,
				"error", err)
		}
	}

	return results
}

// HealthStatus represents the health status of a language server
type HealthStatus struct {
	Language  string
	Workspace string
	Healthy   bool
	Error     string
	StartTime time.Time
	LastUsed  time.Time
}

// AutoRestart automatically restarts unhealthy servers
func (l *LifecycleManager) AutoRestart(ctx context.Context) error {
	logger := observability.GetLogger(ctx)
	healthStatus := l.HealthCheck(ctx)

	var lastErr error
	restarted := 0

	for key, status := range healthStatus {
		if !status.Healthy {
			logger.InfoContext(ctx, "Auto-restarting unhealthy server",
				"key", key,
				"language", status.Language,
				"workspace", status.Workspace)

			if err := l.RestartServer(ctx, status.Language, status.Workspace); err != nil {
				logger.ErrorContext(ctx, "Failed to auto-restart server",
					"language", status.Language,
					"workspace", status.Workspace,
					"error", err)
				lastErr = err
			} else {
				restarted++
			}
		}
	}

	if restarted > 0 {
		logger.InfoContext(ctx, "Auto-restarted servers",
			"count", restarted)
	}

	return lastErr
}

// MonitorServer monitors a specific server for health
func (l *LifecycleManager) MonitorServer(ctx context.Context, language string, workspace string, checkInterval time.Duration) {
	logger := observability.GetLogger(ctx)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			healthy, err := l.manager.serverManager.CheckServerHealth(ctx, language, workspace)
			if !healthy {
				logger.WarnContext(ctx, "Server health check failed",
					"language", language,
					"workspace", workspace,
					"error", err)

				// Attempt restart
				if err := l.RestartServer(ctx, language, workspace); err != nil {
					logger.ErrorContext(ctx, "Failed to restart unhealthy server",
						"language", language,
						"workspace", workspace,
						"error", err)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
