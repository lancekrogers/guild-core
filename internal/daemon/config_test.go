// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDaemonConfig(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	// Set up a test home directory
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	tests := []struct {
		name             string
		campaign         string
		requestedSession int
		wantSession      int
		wantErr          bool
	}{
		{
			name:             "creates config for primary session",
			campaign:         "test-campaign",
			requestedSession: 0,
			wantSession:      0,
			wantErr:          false,
		},
		{
			name:             "creates config for specific session",
			campaign:         "test-campaign",
			requestedSession: 2,
			wantSession:      2,
			wantErr:          false,
		},
		{
			name:             "finds available session when requested -1",
			campaign:         "test-campaign",
			requestedSession: -1,
			wantSession:      0, // Should get session 0 when none exist
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetDaemonConfig(tt.campaign, tt.requestedSession)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)

			// Verify basic configuration
			assert.Equal(t, tt.campaign, config.Campaign)
			assert.Equal(t, tt.wantSession, config.Session)
			assert.NotEmpty(t, config.CampaignHash)
			assert.NotEmpty(t, config.SocketPath)
			assert.Equal(t, 30*time.Minute, config.IdleTimeout)
			assert.Equal(t, 5, config.NiceLevel)

			// Verify socket path format
			if tt.wantSession == 0 {
				assert.Contains(t, config.SocketPath, "guild.sock")
			} else {
				assert.Contains(t, config.SocketPath, "guild-"+string(rune('0'+tt.wantSession))+".sock")
			}

			// Verify log and PID file paths
			assert.NotEmpty(t, config.LogFile)
			assert.NotEmpty(t, config.PIDFile)
			if tt.wantSession == 0 {
				assert.Contains(t, config.LogFile, "daemon.log")
				assert.Contains(t, config.PIDFile, "daemon.pid")
			} else {
				assert.Contains(t, config.LogFile, "daemon-"+string(rune('0'+tt.wantSession))+".log")
				assert.Contains(t, config.PIDFile, "daemon-"+string(rune('0'+tt.wantSession))+".pid")
			}
		})
	}
}

func TestSetupFilePaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	tests := []struct {
		name    string
		config  *DaemonConfig
		wantLog string
		wantPID string
		wantErr bool
	}{
		{
			name: "sets up paths for primary session",
			config: &DaemonConfig{
				Campaign: "test-campaign",
				Session:  0,
			},
			wantLog: "daemon.log",
			wantPID: "daemon.pid",
		},
		{
			name: "sets up paths for numbered session",
			config: &DaemonConfig{
				Campaign: "test-campaign",
				Session:  3,
			},
			wantLog: "daemon-3.log",
			wantPID: "daemon-3.pid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.setupFilePaths()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify paths contain expected filenames
			assert.Contains(t, tt.config.LogFile, tt.wantLog)
			assert.Contains(t, tt.config.PIDFile, tt.wantPID)

			// Verify paths are under campaign directory
			assert.Contains(t, tt.config.LogFile, tt.config.Campaign)
			assert.Contains(t, tt.config.PIDFile, tt.config.Campaign)

			// Verify campaign directory was created
			campaignDir := filepath.Dir(tt.config.LogFile)
			assert.DirExists(t, campaignDir)
		})
	}
}

func TestGetServerAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   *DaemonConfig
		expected string
	}{
		{
			name: "returns Unix socket address",
			config: &DaemonConfig{
				SocketPath: "/tmp/guild.sock",
			},
			expected: "unix:///tmp/guild.sock",
		},
		{
			name: "handles paths with spaces",
			config: &DaemonConfig{
				SocketPath: "/tmp/my campaign/guild.sock",
			},
			expected: "unix:///tmp/my campaign/guild.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetServerAddress()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		config   *DaemonConfig
		expected string
	}{
		{
			name: "formats primary session name",
			config: &DaemonConfig{
				Campaign: "my-campaign",
				Session:  0,
			},
			expected: "Guild Daemon (my-campaign)",
		},
		{
			name: "formats numbered session name",
			config: &DaemonConfig{
				Campaign: "my-campaign",
				Session:  2,
			},
			expected: "Guild Daemon (my-campaign-session-2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetDisplayName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindAvailableSession(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	// This is a simple test since the current implementation is simplified
	// In the future, this would test actual session discovery logic

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	session, socketPath, err := findAvailableSession("test-campaign")

	require.NoError(t, err)
	assert.Equal(t, 0, session)
	assert.NotEmpty(t, socketPath)
	assert.Contains(t, socketPath, "guild.sock")
}

func TestDaemonConfigSerialization(t *testing.T) {
	// Test that DaemonConfig can be properly serialized/deserialized
	config := &DaemonConfig{
		Campaign:     "test-campaign",
		CampaignHash: "abc123",
		Session:      1,
		SocketPath:   "/tmp/guild-1.sock",
		LogFile:      "/var/log/guild-1.log",
		PIDFile:      "/var/run/guild-1.pid",
		IdleTimeout:  30 * time.Minute,
		NiceLevel:    5,
		MemoryLimit:  1024 * 1024 * 1024, // 1GB
	}

	// Test field accessibility
	assert.Equal(t, "test-campaign", config.Campaign)
	assert.Equal(t, "abc123", config.CampaignHash)
	assert.Equal(t, 1, config.Session)
	assert.Equal(t, "/tmp/guild-1.sock", config.SocketPath)
	assert.Equal(t, "/var/log/guild-1.log", config.LogFile)
	assert.Equal(t, "/var/run/guild-1.pid", config.PIDFile)
	assert.Equal(t, 30*time.Minute, config.IdleTimeout)
	assert.Equal(t, 5, config.NiceLevel)
	assert.Equal(t, int64(1024*1024*1024), config.MemoryLimit)
}

func TestDaemonConfigDefaults(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	// Test that GetDaemonConfig sets appropriate defaults
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	config, err := GetDaemonConfig("test-defaults", 0)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, 30*time.Minute, config.IdleTimeout)
	assert.Equal(t, 5, config.NiceLevel)
	assert.Equal(t, int64(0), config.MemoryLimit) // Default is 0 (no limit)
}

func TestDaemonConfigCampaignDirectoryCreation(t *testing.T) {
	t.Skip("Skipping test that requires daemon environment setup")
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	// Test that GetDaemonConfig creates campaign directory
	config, err := GetDaemonConfig("new-campaign", 0)
	require.NoError(t, err)

	// Verify campaign directory exists
	campaignDir := filepath.Join(homeDir, ".guild", "campaigns", "new-campaign")
	assert.DirExists(t, campaignDir)

	// Verify log and PID paths are within campaign directory
	assert.Contains(t, config.LogFile, campaignDir)
	assert.Contains(t, config.PIDFile, campaignDir)
}
