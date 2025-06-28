// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemonconn

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDiscover_WithTimeout tests the discovery function with timeout
func TestDiscover_WithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test discovery - may succeed if daemon is running, or fail if not
	conn, info, err := Discover(ctx)

	if err != nil {
		// No daemon running - expected in CI/test environments
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "failed to connect to daemon")
	} else {
		// Daemon is running - validate the connection
		assert.NoError(t, err)
		assert.NotNil(t, conn)
		assert.NotNil(t, info)

		// Clean up connection
		if conn != nil {
			conn.Close()
		}

		// Validate connection info
		assert.NotEmpty(t, info.Address)
		assert.NotEmpty(t, info.Type)
		assert.Contains(t, []string{"unix", "tcp"}, info.Type)
	}
}

// TestDiscover_WithEnvironmentOverride tests env var override
func TestDiscover_WithEnvironmentOverride(t *testing.T) {
	// Set environment variable
	os.Setenv("GUILD_DAEMON_ADDR", "localhost:9999")
	defer os.Unsetenv("GUILD_DAEMON_ADDR")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Test discovery with env override (should attempt to connect to 9999)
	conn, info, err := Discover(ctx)

	// gRPC dial may succeed even if nothing is listening, so we handle both cases
	if err != nil {
		// Expected case - no daemon listening on 9999
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Nil(t, info)
		assert.Contains(t, err.Error(), "failed to connect to daemon at override address")
	} else {
		// gRPC connection created but may not be functional
		assert.NotNil(t, conn)
		assert.NotNil(t, info)
		assert.Equal(t, "localhost:9999", info.Address)
		assert.Equal(t, "tcp", info.Type)

		// Clean up
		if conn != nil {
			conn.Close()
		}
	}
}

// TestConnectionInfo_FormatStatus tests the status formatting
func TestConnectionInfo_FormatStatus(t *testing.T) {
	tests := []struct {
		name     string
		info     *ConnectionInfo
		latency  time.Duration
		expected string
	}{
		{
			name:     "nil info",
			info:     nil,
			latency:  0,
			expected: "🔴 Offline",
		},
		{
			name: "unix socket",
			info: &ConnectionInfo{
				Address: "/tmp/guild.sock",
				Type:    "unix",
			},
			latency:  25 * time.Millisecond,
			expected: "🟢 Connected to unix socket (25ms)",
		},
		{
			name: "tcp connection",
			info: &ConnectionInfo{
				Address: "localhost:7600",
				Type:    "tcp",
			},
			latency:  50 * time.Millisecond,
			expected: "🟢 Connected to localhost:7600 (50ms)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatConnectionStatus(tt.info, tt.latency)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestManager_BasicOperations tests basic manager operations
func TestManager_BasicOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	manager := NewManager(ctx)

	// Test initial state
	assert.False(t, manager.IsConnected())

	conn, info := manager.GetConnection()
	assert.Nil(t, conn)
	assert.Nil(t, info)

	// Test connection attempt (may succeed if daemon is running)
	err := manager.Connect(ctx)
	// Don't assert on error since daemon might be running

	// Test cleanup
	err = manager.Close()
	assert.NoError(t, err)

	cancel()
}
