// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package services

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lancekrogers/guild/internal/daemon"
	pkgDaemon "github.com/lancekrogers/guild/pkg/daemon"
	"github.com/lancekrogers/guild/pkg/gerror"
)

// DaemonService monitors and manages the Guild daemon
type DaemonService struct {
	ctx        context.Context
	campaignID string

	// Daemon management
	daemonConfig *daemon.DaemonConfig
	lifecycle    *daemon.LifecycleManager

	// Monitoring state
	lastPing     time.Time
	isConnected  bool
	pingInterval time.Duration

	// Statistics
	uptime       time.Duration
	startTime    time.Time
	restartCount int
}

// NewDaemonService creates a new daemon service
func NewDaemonService(ctx context.Context, campaignID string) (*DaemonService, error) {
	if campaignID == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "campaign ID cannot be empty", nil).
			WithComponent("services.daemon").
			WithOperation("NewDaemonService")
	}

	// Get daemon configuration
	config, err := daemon.GetDaemonConfig(campaignID, 0)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get daemon config").
			WithComponent("services.daemon").
			WithOperation("NewDaemonService")
	}

	return &DaemonService{
		ctx:          ctx,
		campaignID:   campaignID,
		daemonConfig: config,
		lifecycle:    daemon.DefaultLifecycleManager,
		pingInterval: 5 * time.Second,
		startTime:    time.Now(),
	}, nil
}

// Start initializes the daemon service and begins monitoring
func (ds *DaemonService) Start() tea.Cmd {
	return func() tea.Msg {
		// Check if daemon is already running
		isRunning := pkgDaemon.CanConnect(ds.daemonConfig.SocketPath)

		if !isRunning {
			// Try to start the daemon
			_, err := ds.lifecycle.AutoStartDaemon(ds.ctx, ds.campaignID)
			if err != nil {
				return DaemonServiceErrorMsg{
					Operation: "auto_start",
					Error: gerror.Wrap(err, gerror.ErrCodeInternal, "failed to auto-start daemon").
						WithComponent("services.daemon").
						WithOperation("Start"),
				}
			}

			// Wait a moment for daemon to initialize
			time.Sleep(500 * time.Millisecond)

			// Verify it started
			isRunning = pkgDaemon.CanConnect(ds.daemonConfig.SocketPath)
			if !isRunning {
				return DaemonServiceErrorMsg{
					Operation: "verify_start",
					Error: gerror.New(gerror.ErrCodeConnection, "daemon failed to start", nil).
						WithComponent("services.daemon").
						WithOperation("Start"),
				}
			}
		}

		ds.isConnected = isRunning
		ds.lastPing = time.Now()

		// Start monitoring
		ds.lifecycle.MonitorSessions(ds.ctx)

		return DaemonServiceStartedMsg{
			CampaignID: ds.campaignID,
			SocketPath: ds.daemonConfig.SocketPath,
			PID:        0, // PID not available in current config
		}
	}
}

// Ping checks if the daemon is still responsive
func (ds *DaemonService) Ping() tea.Cmd {
	return func() tea.Msg {
		isConnected := pkgDaemon.CanConnect(ds.daemonConfig.SocketPath)

		if isConnected != ds.isConnected {
			// Connection status changed
			ds.isConnected = isConnected

			if isConnected {
				return DaemonReconnectedMsg{
					CampaignID: ds.campaignID,
					Downtime:   time.Since(ds.lastPing),
				}
			} else {
				return DaemonDisconnectedMsg{
					CampaignID: ds.campaignID,
					LastSeen:   ds.lastPing,
				}
			}
		}

		if isConnected {
			ds.lastPing = time.Now()
			return DaemonPingMsg{
				CampaignID:   ds.campaignID,
				ResponseTime: time.Since(ds.lastPing),
			}
		}

		return DaemonServiceErrorMsg{
			Operation: "ping",
			Error: gerror.New(gerror.ErrCodeConnection, "daemon not responding", nil).
				WithComponent("services.daemon").
				WithOperation("Ping"),
		}
	}
}

// StartPeriodicPing starts periodic health checks
func (ds *DaemonService) StartPeriodicPing() tea.Cmd {
	return tea.Tick(ds.pingInterval, func(t time.Time) tea.Msg {
		return PingRequestMsg{Timestamp: t}
	})
}

// Stop stops the daemon service
func (ds *DaemonService) Stop() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement daemon stopping when StopDaemon method is available
		ds.isConnected = false

		return DaemonStoppedMsg{
			CampaignID: ds.campaignID,
		}
	}
}

// Restart restarts the daemon
func (ds *DaemonService) Restart() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement daemon restart when lifecycle methods are available
		ds.restartCount++
		ds.isConnected = true
		ds.lastPing = time.Now()

		return DaemonRestartedMsg{
			CampaignID:   ds.campaignID,
			RestartCount: ds.restartCount,
		}
	}
}

// GetStatus returns the current daemon status
func (ds *DaemonService) GetStatus() DaemonStatus {
	return DaemonStatus{
		CampaignID:   ds.campaignID,
		IsConnected:  ds.isConnected,
		LastPing:     ds.lastPing,
		Uptime:       time.Since(ds.startTime),
		RestartCount: ds.restartCount,
		SocketPath:   ds.daemonConfig.SocketPath,
		PID:          0, // PID not available in current config
	}
}

// GetConfig returns the daemon configuration
func (ds *DaemonService) GetConfig() *daemon.DaemonConfig {
	return ds.daemonConfig
}

// SetPingInterval sets the ping interval for health checks
func (ds *DaemonService) SetPingInterval(interval time.Duration) {
	ds.pingInterval = interval
}

// IsConnected returns whether the daemon is currently connected
func (ds *DaemonService) IsConnected() bool {
	return ds.isConnected
}

// GetUptime returns how long the daemon has been running
func (ds *DaemonService) GetUptime() time.Duration {
	return time.Since(ds.startTime)
}

// GetRestartCount returns the number of times the daemon has been restarted
func (ds *DaemonService) GetRestartCount() int {
	return ds.restartCount
}

// GetStats returns statistics about the daemon service
func (ds *DaemonService) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	status := ds.GetStatus()
	stats["campaign_id"] = status.CampaignID
	stats["is_connected"] = status.IsConnected
	stats["last_ping"] = status.LastPing.Format(time.RFC3339)
	stats["uptime"] = status.Uptime.String()
	stats["restart_count"] = status.RestartCount
	stats["socket_path"] = status.SocketPath
	stats["pid"] = status.PID
	stats["ping_interval"] = ds.pingInterval.String()

	return stats
}

// DaemonStatus represents the current status of the daemon
type DaemonStatus struct {
	CampaignID   string
	IsConnected  bool
	LastPing     time.Time
	Uptime       time.Duration
	RestartCount int
	SocketPath   string
	PID          int
}

// Message types for daemon service communication

// DaemonServiceStartedMsg indicates the daemon service has started
type DaemonServiceStartedMsg struct {
	CampaignID string
	SocketPath string
	PID        int
}

// DaemonServiceErrorMsg represents a daemon service error
type DaemonServiceErrorMsg struct {
	Operation string
	Error     error
}

// DaemonPingMsg represents a successful daemon ping
type DaemonPingMsg struct {
	CampaignID   string
	ResponseTime time.Duration
}

// DaemonDisconnectedMsg indicates the daemon has disconnected
type DaemonDisconnectedMsg struct {
	CampaignID string
	LastSeen   time.Time
}

// DaemonReconnectedMsg indicates the daemon has reconnected
type DaemonReconnectedMsg struct {
	CampaignID string
	Downtime   time.Duration
}

// DaemonStoppedMsg indicates the daemon has been stopped
type DaemonStoppedMsg struct {
	CampaignID string
}

// DaemonRestartedMsg indicates the daemon has been restarted
type DaemonRestartedMsg struct {
	CampaignID   string
	RestartCount int
}

// PingRequestMsg triggers a ping check
type PingRequestMsg struct {
	Timestamp time.Time
}

// FormatStatus returns a human-readable status string
func (ds *DaemonService) FormatStatus() string {
	status := ds.GetStatus()

	if status.IsConnected {
		return fmt.Sprintf("🟢 Connected (uptime: %s)", formatDuration(status.Uptime))
	}

	downtime := time.Since(status.LastPing)
	return fmt.Sprintf("🔴 Disconnected (down: %s)", formatDuration(downtime))
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else {
		hours := int(d.Hours())
		minutes := int((d - time.Duration(hours)*time.Hour).Minutes())
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}
