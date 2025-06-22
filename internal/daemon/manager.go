// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	daemonPkg "github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// Manager handles multiple daemon instances
type Manager struct {
	configs map[string]*DaemonConfig // keyed by campaign:session
}

// NewManager creates a new daemon manager
func NewManager() *Manager {
	return &Manager{
		configs: make(map[string]*DaemonConfig),
	}
}

// EnsureDaemonRunning starts a daemon for the specified campaign if not already running
func (m *Manager) EnsureDaemonRunning(ctx context.Context, campaign string, preferredSession int) (*DaemonConfig, error) {
	// Clean up any stale sockets first
	if err := daemonPkg.CleanupStaleSessionSockets(campaign); err != nil {
		// Log warning but don't fail
	}

	config, err := GetDaemonConfig(campaign, preferredSession)
	if err != nil {
		return nil, err
	}

	// Check if socket exists and is responsive
	if daemonPkg.CanConnect(config.SocketPath) {
		return config, nil // Already running
	}

	// Clean any stale socket
	if err := daemonPkg.UnlinkIfStale(config.SocketPath); err != nil {
		return nil, err
	}

	// Start new daemon
	if err := m.startCampaignDaemon(ctx, config); err != nil {
		return nil, err
	}

	// Store config for management
	key := m.getConfigKey(config)
	m.configs[key] = config

	return config, nil
}

// startCampaignDaemon starts a new daemon instance for a campaign
func (m *Manager) startCampaignDaemon(ctx context.Context, config *DaemonConfig) error {
	// Build command arguments
	args := []string{"serve", "--daemon"}

	// Add campaign
	args = append(args, "--campaign", config.Campaign)

	// Add session if not primary
	if config.Session > 0 {
		args = append(args, "--session", strconv.Itoa(config.Session))
	}

	// Add socket path
	args = append(args, "--socket", config.SocketPath)

	// Get guild executable path
	guildPath, err := getExecutablePath()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to find guild executable").
			WithComponent("daemon").
			WithOperation("startCampaignDaemon")
	}

	// Prepare command
	cmd := exec.Command(guildPath, args...)

	// Set process to run with lower priority and in a new session
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true, // New session
		Setpgid: true, // New process group
	}

	// Set up environment for resource limits
	if runtime.GOOS != "windows" && config.NiceLevel > 0 {
		cmd.Env = append(os.Environ(), "NICE_LEVEL="+strconv.Itoa(config.NiceLevel))
	}

	// Set up logging
	if config.LogFile != "" {
		logDir := getDirFromPath(config.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create log directory").
				WithComponent("daemon").
				WithOperation("startCampaignDaemon").
				WithDetails("directory", logDir)
		}

		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to open log file").
				WithComponent("daemon").
				WithOperation("startCampaignDaemon").
				WithDetails("log_file", config.LogFile)
		}
		defer logFile.Close()

		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start daemon").
			WithComponent("daemon").
			WithOperation("startCampaignDaemon").
			WithDetails("campaign", config.Campaign)
	}

	// Write PID file if configured
	if config.PIDFile != "" {
		pidDir := getDirFromPath(config.PIDFile)
		if err := os.MkdirAll(pidDir, 0755); err != nil {
			cmd.Process.Kill()
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create PID directory").
				WithComponent("daemon").
				WithOperation("startCampaignDaemon").
				WithDetails("directory", pidDir)
		}

		if err := os.WriteFile(config.PIDFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
			cmd.Process.Kill()
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write PID file").
				WithComponent("daemon").
				WithOperation("startCampaignDaemon").
				WithDetails("pid_file", config.PIDFile)
		}
	}

	// Release the process so it continues after parent exits
	if err := cmd.Process.Release(); err != nil {
		// Log warning but don't fail - process might still work
	}

	// Wait for daemon to be ready
	return m.waitForSocket(ctx, config.SocketPath, 10*time.Second)
}

// waitForSocket waits for a Unix socket to become responsive
func (m *Manager) waitForSocket(ctx context.Context, socketPath string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return gerror.New(gerror.ErrCodeTimeout, "daemon failed to start within timeout", nil).
				WithComponent("daemon").
				WithOperation("waitForSocket").
				WithDetails("socket", socketPath).
				WithDetails("timeout", timeout.String()).
				FromContext(ctx)
		case <-ticker.C:
			if daemonPkg.CanConnect(socketPath) {
				return nil
			}
		}
	}
}

// StopCampaign stops all sessions for a campaign
func (m *Manager) StopCampaign(campaign string) error {
	sessions, err := daemonPkg.ListCampaignSessions(campaign)
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return gerror.New(gerror.ErrCodeNotFound, "no running sessions for campaign", nil).
			WithComponent("daemon").
			WithOperation("StopCampaign").
			WithDetails("campaign", campaign)
	}

	var errors []error
	for _, session := range sessions {
		if err := daemonPkg.StopSession(session.Socket); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to stop some sessions", nil).
			WithComponent("daemon").
			WithOperation("StopCampaign").
			WithDetails("campaign", campaign).
			WithDetails("error_count", len(errors))
	}

	return nil
}

// StopAll stops all managed daemon instances
func (m *Manager) StopAll() error {
	allSessions, err := daemonPkg.DiscoverAllRunningSessions()
	if err != nil {
		return err
	}

	var errors []error
	for _, sessions := range allSessions {
		for _, session := range sessions {
			if err := daemonPkg.StopSession(session.Socket); err != nil {
				errors = append(errors, err)
			}
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to stop some daemons", nil).
			WithComponent("daemon").
			WithOperation("StopAll").
			WithDetails("error_count", len(errors))
	}

	return nil
}

// ListRunning returns all currently running daemon instances
func (m *Manager) ListRunning() (map[string][]daemonPkg.SessionInfo, error) {
	return daemonPkg.DiscoverAllRunningSessions()
}

// Helper functions

func (m *Manager) getConfigKey(config *DaemonConfig) string {
	return config.Campaign + ":" + strconv.Itoa(config.Session)
}

func getDirFromPath(filePath string) string {
	if filePath == "" {
		return ""
	}
	return filepath.Dir(filePath)
}

// Global manager instance for CLI commands
var DefaultManager = NewManager()
