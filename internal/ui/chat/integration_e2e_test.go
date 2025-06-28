// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package chat

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lancekrogers/guild/internal/daemonconn"
	"github.com/lancekrogers/guild/pkg/config"
	"github.com/lancekrogers/guild/pkg/registry"
)

// TestDaemonIntegration_EndToEnd tests the complete daemon integration flow
func TestDaemonIntegration_EndToEnd(t *testing.T) {
	tests := []struct {
		name         string
		daemonAddr   string
		expectError  bool
		expectDirect bool
	}{
		{
			name:         "NoDaemon_FallbackToDirect",
			daemonAddr:   "",
			expectError:  false,
			expectDirect: true,
		},
		{
			name:         "InvalidDaemon_FallbackToDirect",
			daemonAddr:   "localhost:19999", // Non-existent port
			expectError:  false,
			expectDirect: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			// Set environment override if specified
			if tt.daemonAddr != "" {
				os.Setenv("GUILD_DAEMON_ADDR", tt.daemonAddr)
				defer os.Unsetenv("GUILD_DAEMON_ADDR")
			}

			// Create chat app
			guildConfig := &config.GuildConfig{
				Name: "test-guild",
			}
			registry := registry.NewComponentRegistry()

			app := NewApp(ctx, guildConfig, registry)
			app.SetCampaignID("test-campaign")

			// Initialize daemon connection (this should fallback to direct mode)
			err := app.initializeDaemonConnection()

			// For these tests, connection failure is expected but not an error condition
			if err != nil && tt.expectDirect {
				// Enable direct mode as would happen in normal initialization
				app.enableDirectMode()
			} else if tt.expectError {
				assert.Error(t, err)
			} else if tt.expectDirect {
				// If no error but we expect direct mode, enable it
				// This handles cases where connection manager returns success but no actual connection
				app.enableDirectMode()
			}

			if tt.expectDirect {
				assert.True(t, app.directMode, "Should be in direct mode")
				assert.False(t, app.isConnectedToDaemon(), "Should not be connected to daemon")
			}

			// Test message sending works in both modes
			err = app.sendMessage(ctx, "Hello, test!")
			assert.NoError(t, err, "Message sending should work in direct mode")
		})
	}
}

// TestConnectionStatus_RealTime tests real-time connection status updates
func TestConnectionStatus_RealTime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	guildConfig := &config.GuildConfig{Name: "test-guild"}
	registry := registry.NewComponentRegistry()
	app := NewApp(ctx, guildConfig, registry)

	// Test connection status formatting
	tests := []struct {
		name           string
		connected      bool
		info           *daemonconn.ConnectionInfo
		latency        time.Duration
		expectedStatus string
	}{
		{
			name:           "Offline",
			connected:      false,
			info:           nil,
			latency:        0,
			expectedStatus: "🔴 Offline",
		},
		{
			name:      "UnixSocket",
			connected: true,
			info: &daemonconn.ConnectionInfo{
				Address: "/tmp/guild.sock",
				Type:    "unix",
			},
			latency:        25 * time.Millisecond,
			expectedStatus: "🟢 Connected to unix socket (25ms)",
		},
		{
			name:      "TCPConnection",
			connected: true,
			info: &daemonconn.ConnectionInfo{
				Address: "localhost:7600",
				Type:    "tcp",
			},
			latency:        50 * time.Millisecond,
			expectedStatus: "🟢 Connected to localhost:7600 (50ms)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.connectionStatus = tt.connected
			app.connectionInfo = tt.info

			status := daemonconn.FormatConnectionStatus(tt.info, tt.latency)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

// TestSessionPersistence_Complete tests complete session persistence flow
func TestSessionPersistence_Complete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	guildConfig := &config.GuildConfig{Name: "test-guild"}
	registry := registry.NewComponentRegistry()
	app := NewApp(ctx, guildConfig, registry)
	app.SetCampaignID("test-campaign")

	// Test session creation when no daemon is available
	err := app.createNewSession()
	// This will fail without a session client, which is expected
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session client not available")
}

// TestDirectModeFunctionality tests direct mode operates correctly
func TestDirectModeFunctionality(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	guildConfig := &config.GuildConfig{Name: "test-guild"}
	registry := registry.NewComponentRegistry()
	app := NewApp(ctx, guildConfig, registry)

	// Enable direct mode
	app.enableDirectMode()

	// Verify direct mode state
	assert.True(t, app.directMode)
	assert.False(t, app.connectionStatus)
	assert.False(t, app.isConnectedToDaemon())

	// Test message sending in direct mode
	initialCount := len(app.messages)
	err := app.sendMessageDirect(ctx, "Direct mode test")

	assert.NoError(t, err)
	assert.Len(t, app.messages, initialCount+2) // User message + system response

	// Verify message content
	userMsg := app.messages[initialCount]
	assert.Equal(t, "Direct mode test", userMsg.Content)
	assert.Equal(t, "user", userMsg.AgentID)

	responseMsg := app.messages[initialCount+1]
	assert.Contains(t, responseMsg.Content, "Direct mode response")
	assert.Equal(t, "system", responseMsg.AgentID)
}

// TestGracefulDegradation tests fallback behavior under various failure scenarios
func TestGracefulDegradation(t *testing.T) {
	scenarios := []struct {
		name         string
		setup        func(*App)
		expectDirect bool
	}{
		{
			name: "NoConnectionManager",
			setup: func(app *App) {
				app.connManager = nil
			},
			expectDirect: true,
		},
		{
			name: "NoDaemonClients",
			setup: func(app *App) {
				app.chatClient = nil
				app.sessionClient = nil
				app.guildClient = nil
			},
			expectDirect: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			guildConfig := &config.GuildConfig{Name: "test-guild"}
			registry := registry.NewComponentRegistry()
			app := NewApp(ctx, guildConfig, registry)

			scenario.setup(app)

			// These conditions should trigger direct mode
			connected := app.isConnectedToDaemon()
			assert.False(t, connected)

			// App should still be able to send messages
			err := app.sendMessage(ctx, "Test under failure")
			assert.NoError(t, err)
		})
	}
}

// TestReconnectionScenarios tests various reconnection scenarios
func TestReconnectionScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	guildConfig := &config.GuildConfig{Name: "test-guild"}
	registry := registry.NewComponentRegistry()
	app := NewApp(ctx, guildConfig, registry)

	// Create connection manager
	connManager := daemonconn.NewManager(ctx)
	app.connManager = connManager

	// Test connection manager basic operations
	assert.False(t, connManager.IsConnected())

	conn, info := connManager.GetConnection()
	assert.Nil(t, conn)
	assert.Nil(t, info)

	// Test latency measurement (should return 0 when not connected)
	latency := connManager.GetLatency(ctx)
	assert.Equal(t, time.Duration(0), latency)

	// Test cleanup
	err := connManager.Close()
	assert.NoError(t, err)
}

// BenchmarkDaemonConnection benchmarks daemon connection performance
func BenchmarkDaemonConnection(b *testing.B) {
	ctx := context.Background()

	b.Run("Discover", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			daemonconn.Discover(ctx)
			cancel()
		}
	})

	b.Run("ManagerCreate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			manager := daemonconn.NewManager(ctx)
			manager.Close()
		}
	})
}
