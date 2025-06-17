// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"fmt"
	"path/filepath"
	"time"

	daemonPkg "github.com/guild-ventures/guild-core/pkg/daemon"
	"github.com/guild-ventures/guild-core/pkg/paths"
)

// DaemonConfig represents the configuration for a Guild daemon instance
type DaemonConfig struct {
	// Campaign information
	Campaign     string `json:"campaign"`
	CampaignHash string `json:"campaign_hash"`
	Session      int    `json:"session"`

	// Transport configuration
	SocketPath   string `json:"socket_path"`   // Unix socket path

	// Logging and process management
	LogFile    string        `json:"log_file"`
	PIDFile    string        `json:"pid_file"`
	IdleTimeout time.Duration `json:"idle_timeout"`

	// Resource limits
	NiceLevel      int   `json:"nice_level"`       // Process priority adjustment
	MemoryLimit    int64 `json:"memory_limit"`    // Memory limit in bytes
	MemoryLimitMB  int   `json:"memory_limit_mb"` // Memory limit in MB (for easier config)
}

// GetDaemonConfig creates a daemon configuration for a campaign and session
func GetDaemonConfig(campaign string, requestedSession int) (*DaemonConfig, error) {
	config := &DaemonConfig{
		Campaign:     campaign,
		CampaignHash: paths.GetCampaignHash(campaign),
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


// setupFilePaths configures log and PID file paths based on campaign and session
func (c *DaemonConfig) setupFilePaths() error {
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
	return fmt.Sprintf("unix://%s", c.SocketPath)
}

// GetDisplayName returns a human-readable name for the daemon instance
func (c *DaemonConfig) GetDisplayName() string {
	if c.Session == 0 {
		return fmt.Sprintf("Guild Daemon (%s)", c.Campaign)
	}

	return fmt.Sprintf("Guild Daemon (%s-session-%d)", c.Campaign, c.Session)
}


// findAvailableSession finds the next available session for a campaign
func findAvailableSession(campaign string) (int, string, error) {
	return daemonPkg.FindAvailableSession(campaign)
}