// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/guild-ventures/guild-core/pkg/paths"
)

// DaemonConfig represents the configuration for a Guild daemon instance
type DaemonConfig struct {
	// Campaign information
	Campaign     string `json:"campaign"`
	CampaignHash string `json:"campaign_hash"`
	Session      int    `json:"session"`

	// Transport configuration
	UseSocket    bool   `json:"use_socket"`     // true for Unix sockets, false for TCP
	SocketPath   string `json:"socket_path"`   // Unix socket path
	Port         int    `json:"port"`          // TCP port (fallback)
	Host         string `json:"host"`          // TCP host (fallback)

	// Logging and process management
	LogFile    string        `json:"log_file"`
	PIDFile    string        `json:"pid_file"`
	IdleTimeout time.Duration `json:"idle_timeout"`

	// Resource limits
	NiceLevel    int   `json:"nice_level"`     // Process priority adjustment
	MemoryLimit  int64 `json:"memory_limit"`  // Memory limit in bytes
}

// GetDaemonConfig creates a daemon configuration for a campaign and session
func GetDaemonConfig(campaign string, requestedSession int) (*DaemonConfig, error) {
	config := &DaemonConfig{
		Campaign:     campaign,
		CampaignHash: paths.GetCampaignHash(campaign),
		UseSocket:    true, // Default to Unix sockets
		Host:         "localhost",
		Port:         9090, // Default TCP port for fallback
		IdleTimeout:  30 * time.Minute,
		NiceLevel:    5, // Lower priority
	}

	// Determine session
	if requestedSession < 0 {
		// Find available session
		session, socketPath, err := findAvailableSession(campaign)
		if err != nil {
			return nil, err
		}
		config.Session = session
		config.SocketPath = socketPath
	} else {
		// Use specific session
		config.Session = requestedSession
		socketPath, err := paths.GetCampaignSocket(campaign, requestedSession)
		if err != nil {
			return nil, err
		}
		config.SocketPath = socketPath
	}

	// Set up file paths
	if err := config.setupFilePaths(); err != nil {
		return nil, err
	}

	return config, nil
}

// GetLegacyDaemonConfig creates a configuration compatible with the existing single-instance daemon
func GetLegacyDaemonConfig() *DaemonConfig {
	return &DaemonConfig{
		Campaign:    "default",
		Session:     0,
		UseSocket:   false, // Use TCP for backward compatibility
		Host:        "localhost",
		Port:        9090,
		LogFile:     GetLogFilePath(),
		PIDFile:     GetPIDFilePath(),
		IdleTimeout: 0, // No idle timeout for legacy mode
		NiceLevel:   0, // Normal priority
	}
}

// setupFilePaths configures log and PID file paths based on campaign and session
func (c *DaemonConfig) setupFilePaths() error {
	if c.Campaign == "" || c.Campaign == "default" {
		// Legacy mode - use existing paths
		c.LogFile = GetLogFilePath()
		c.PIDFile = GetPIDFilePath()
		return nil
	}

	// Campaign-specific paths
	campaignDir, err := paths.GetCampaignDir(c.Campaign)
	if err != nil {
		return err
	}

	// Create campaign directory if it doesn't exist
	if _, err := paths.EnsureCampaignDir(c.Campaign); err != nil {
		return err
	}

	// Set file paths
	if c.Session == 0 {
		c.LogFile = filepath.Join(campaignDir, "daemon.log")
		c.PIDFile = filepath.Join(campaignDir, "daemon.pid")
	} else {
		c.LogFile = filepath.Join(campaignDir, fmt.Sprintf("daemon-%d.log", c.Session))
		c.PIDFile = filepath.Join(campaignDir, fmt.Sprintf("daemon-%d.pid", c.Session))
	}

	return nil
}

// GetServerAddress returns the address string for the daemon
func (c *DaemonConfig) GetServerAddress() string {
	if c.UseSocket {
		return fmt.Sprintf("unix://%s", c.SocketPath)
	}
	return fmt.Sprintf("tcp://%s:%d", c.Host, c.Port)
}

// GetDisplayName returns a human-readable name for the daemon instance
func (c *DaemonConfig) GetDisplayName() string {
	if c.Campaign == "" || c.Campaign == "default" {
		return "Guild Daemon"
	}

	if c.Session == 0 {
		return fmt.Sprintf("Guild Daemon (%s)", c.Campaign)
	}

	return fmt.Sprintf("Guild Daemon (%s-session-%d)", c.Campaign, c.Session)
}

// IsLegacy returns true if this is a legacy single-instance configuration
func (c *DaemonConfig) IsLegacy() bool {
	return !c.UseSocket || c.Campaign == "" || c.Campaign == "default"
}

// findAvailableSession finds the next available session for a campaign
// This is a simplified version that will be replaced by the socket registry implementation
func findAvailableSession(campaign string) (int, string, error) {
	// Try primary session first
	socketPath, err := paths.GetCampaignSocket(campaign, 0)
	if err != nil {
		return 0, "", err
	}

	// For now, just return session 0
	// This will be replaced by proper session discovery
	return 0, socketPath, nil
}