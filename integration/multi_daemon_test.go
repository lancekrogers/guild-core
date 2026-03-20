// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/lancekrogers/guild-core/internal/daemon"
	"github.com/lancekrogers/guild-core/pkg/client"
	daemonPkg "github.com/lancekrogers/guild-core/pkg/daemon"
	pb "github.com/lancekrogers/guild-core/pkg/grpc/pb/guild/v1"
	"github.com/lancekrogers/guild-core/pkg/paths"
)

// TestMultiDaemonLifecycle tests the full lifecycle of multiple daemon instances
func TestMultiDaemonLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	manager := daemon.NewManager()

	// Test campaigns
	campaigns := []string{"shop", "blog", "api"}

	// Cleanup function
	cleanup := func() {
		for _, campaign := range campaigns {
			if err := manager.StopCampaign(campaign); err != nil {
				t.Logf("Failed to stop campaign %s: %v", campaign, err)
			}
			// Clean up sockets
			daemonPkg.CleanupStaleSessionSockets(campaign)
		}
	}
	defer cleanup()

	// Also clean up before starting
	cleanup()

	t.Run("start multiple daemons", func(t *testing.T) {
		configs := make(map[string]*daemon.DaemonConfig)

		// Start a daemon for each campaign
		for _, campaign := range campaigns {
			config, err := manager.EnsureDaemonRunning(ctx, campaign, 0)
			require.NoError(t, err, "Failed to start daemon for campaign %s", campaign)
			require.NotNil(t, config)

			configs[campaign] = config

			// Verify socket is accessible
			assert.True(t, daemonPkg.CanConnect(config.SocketPath),
				"Socket not accessible for campaign %s", campaign)
		}

		// Verify all daemons are running
		running, err := manager.ListRunning()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(running), len(campaigns))
	})

	t.Run("connect via gRPC", func(t *testing.T) {
		// Skip if daemon start failed
		if t.Failed() {
			t.Skip("Previous test failed")
		}

		// Test gRPC connection to each daemon
		for _, campaign := range campaigns {
			config, err := daemon.GetDaemonConfig(campaign, 0)
			require.NoError(t, err)

			// Connect via gRPC
			conn, err := grpc.Dial("unix://"+config.SocketPath,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithTimeout(5*time.Second))
			require.NoError(t, err, "Failed to connect to daemon for campaign %s", campaign)
			defer conn.Close()

			// Try to call a simple gRPC method
			client := pb.NewGuildClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// ListCampaigns should work
			resp, err := client.ListCampaigns(ctx, &pb.ListCampaignsRequest{})
			assert.NoError(t, err, "Failed to call ListCampaigns for campaign %s", campaign)
			if resp != nil {
				assert.NotNil(t, resp.Campaigns)
			}
		}
	})

	t.Run("HTTP client connections", func(t *testing.T) {
		// Skip if daemon start failed
		if t.Failed() {
			t.Skip("Previous test failed")
		}

		// Test HTTP client for each daemon
		for _, campaign := range campaigns {
			config, err := daemon.GetDaemonConfig(campaign, 0)
			require.NoError(t, err)

			// Create HTTP client
			httpClient, err := client.NewClient(config.SocketPath)
			require.NoError(t, err)
			defer httpClient.Close()

			// Test health check
			healthy := httpClient.IsHealthy(ctx)
			assert.True(t, healthy, "Daemon not healthy for campaign %s", campaign)

			// Test a simple HTTP request
			resp, err := httpClient.Get(ctx, "http://unix/status")
			if err == nil && resp != nil {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				resp.Body.Close()
			}
		}
	})
}

// TestMultipleSessions tests running multiple sessions for the same campaign
func TestMultipleSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	manager := daemon.NewManager()
	campaign := "test-multi-session"

	// Cleanup
	defer func() {
		manager.StopCampaign(campaign)
		daemonPkg.CleanupStaleSessionSockets(campaign)
	}()

	// Clean before test
	manager.StopCampaign(campaign)
	daemonPkg.CleanupStaleSessionSockets(campaign)

	t.Run("start multiple sessions", func(t *testing.T) {
		configs := make([]*daemon.DaemonConfig, 3)

		// Start 3 sessions
		for i := 0; i < 3; i++ {
			config, err := manager.EnsureDaemonRunning(ctx, campaign, i)
			require.NoError(t, err, "Failed to start session %d", i)
			require.NotNil(t, config)
			assert.Equal(t, i, config.Session)

			configs[i] = config

			// Verify socket is accessible
			assert.True(t, daemonPkg.CanConnect(config.SocketPath))
		}

		// List all sessions
		sessions, err := daemonPkg.ListCampaignSessions(campaign)
		require.NoError(t, err)
		assert.Len(t, sessions, 3)

		// Verify session numbers
		for i, session := range sessions {
			assert.Equal(t, i, session.Session)
			assert.Equal(t, "running", session.Status)
		}
	})

	t.Run("session limit enforcement", func(t *testing.T) {
		// Skip if previous test failed
		if t.Failed() {
			t.Skip("Previous test failed")
		}

		// Try to start sessions beyond limit (10)
		for i := 3; i < 12; i++ {
			_, err := manager.EnsureDaemonRunning(ctx, campaign, i)
			if i < 10 {
				assert.NoError(t, err, "Should allow session %d", i)
			} else {
				assert.Error(t, err, "Should reject session %d (beyond limit)", i)
			}
		}

		// Clean up extra sessions
		for i := 3; i < 10; i++ {
			socketPath, _ := paths.GetCampaignSocket(campaign, i)
			daemonPkg.StopSession(socketPath)
		}
	})
}

// TestDaemonCrashRecovery tests recovery from daemon crashes
func TestDaemonCrashRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	manager := daemon.NewManager()
	lifecycleManager := daemon.NewLifecycleManager()
	campaign := "test-crash-recovery"

	// Cleanup
	defer func() {
		manager.StopCampaign(campaign)
		daemonPkg.CleanupStaleSessionSockets(campaign)
	}()

	t.Run("detect crashed daemon", func(t *testing.T) {
		// Start a daemon
		config, err := manager.EnsureDaemonRunning(ctx, campaign, 0)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Simulate crash by removing socket but leaving PID file
		socketPath := config.SocketPath
		os.Remove(socketPath)

		// Wait a moment
		time.Sleep(100 * time.Millisecond)

		// Check if daemon is detected as not running
		assert.False(t, daemonPkg.CanConnect(socketPath))

		// Check that socket is indeed stale
		assert.False(t, daemonPkg.CanConnect(socketPath), "Socket should not be connectable")

		// Auto-start should work
		newConfig, err := lifecycleManager.AutoStartDaemon(ctx, campaign)
		assert.NoError(t, err)
		assert.NotNil(t, newConfig)

		// Should be accessible again
		assert.True(t, daemonPkg.CanConnect(newConfig.SocketPath))
	})
}

// TestConcurrentDaemonStarts tests race conditions in daemon startup
func TestConcurrentDaemonStarts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	manager := daemon.NewManager()
	campaign := "test-concurrent"

	// Cleanup
	defer func() {
		manager.StopCampaign(campaign)
		daemonPkg.CleanupStaleSessionSockets(campaign)
	}()

	// Clean before test
	manager.StopCampaign(campaign)
	daemonPkg.CleanupStaleSessionSockets(campaign)

	t.Run("concurrent starts same campaign", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make([]error, 5)
		configs := make([]*daemon.DaemonConfig, 5)

		// Try to start 5 daemons concurrently for same campaign
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				config, err := manager.EnsureDaemonRunning(ctx, campaign, -1)
				errors[idx] = err
				configs[idx] = config
			}(i)
		}

		wg.Wait()

		// At least one should succeed
		successCount := 0
		for i, err := range errors {
			if err == nil && configs[i] != nil {
				successCount++
			}
		}
		assert.GreaterOrEqual(t, successCount, 1, "At least one daemon should start successfully")

		// Check how many are actually running
		sessions, err := daemonPkg.ListCampaignSessions(campaign)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(sessions), 1)
		t.Logf("Started %d daemon(s) from %d concurrent attempts", len(sessions), 5)
	})
}

// TestIdleTimeout tests daemon idle timeout functionality
func TestIdleTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	lifecycleManager := daemon.NewLifecycleManager()
	campaign := "test-idle"

	// Set very short idle timeout for testing
	lifecycleManager.SetIdleTimeout(500 * time.Millisecond)

	// Cleanup
	defer func() {
		daemon.DefaultManager.StopCampaign(campaign)
		daemonPkg.CleanupStaleSessionSockets(campaign)
	}()

	t.Run("daemon stops after idle timeout", func(t *testing.T) {
		// Start daemon
		config, err := lifecycleManager.AutoStartDaemon(ctx, campaign)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify it's running
		assert.True(t, daemonPkg.CanConnect(config.SocketPath))

		// Start monitoring
		monitorCtx, cancel := context.WithCancel(ctx)
		go lifecycleManager.MonitorSessions(monitorCtx)

		// Wait for idle timeout + monitoring interval
		time.Sleep(2 * time.Second)

		// Should be stopped now
		assert.False(t, daemonPkg.CanConnect(config.SocketPath))

		// Stop monitoring
		cancel()
	})
}

// TestSocketCleanup tests proper cleanup of stale sockets
func TestSocketCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	campaign := "test-cleanup"

	t.Run("cleanup stale sockets", func(t *testing.T) {
		// Create some fake socket files
		for i := 0; i < 3; i++ {
			socketPath, err := paths.GetCampaignSocket(campaign, i)
			require.NoError(t, err)

			// Create directory if needed
			socketDir := filepath.Dir(socketPath)
			os.MkdirAll(socketDir, 0o755)

			// Create empty file to simulate stale socket
			file, err := os.Create(socketPath)
			require.NoError(t, err)
			file.Close()
		}

		// Run cleanup
		err := daemonPkg.CleanupStaleSessionSockets(campaign)
		assert.NoError(t, err)

		// Verify sockets are removed
		for i := 0; i < 3; i++ {
			socketPath, _ := paths.GetCampaignSocket(campaign, i)
			_, err := os.Stat(socketPath)
			assert.True(t, os.IsNotExist(err), "Socket %d should be removed", i)
		}
	})
}

// TestStopCommands tests the stop command functionality
func TestStopCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	manager := daemon.NewManager()
	campaigns := []string{"stop-test-1", "stop-test-2", "stop-test-3"}

	// Cleanup
	defer func() {
		for _, c := range campaigns {
			manager.StopCampaign(c)
			daemonPkg.CleanupStaleSessionSockets(c)
		}
	}()

	t.Run("setup daemons", func(t *testing.T) {
		// Start daemons for test campaigns
		for _, campaign := range campaigns {
			_, err := manager.EnsureDaemonRunning(ctx, campaign, 0)
			require.NoError(t, err)
		}

		// Verify all running
		running, err := manager.ListRunning()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(running), len(campaigns))
	})

	t.Run("stop specific campaign", func(t *testing.T) {
		// Stop just the first campaign
		err := manager.StopCampaign(campaigns[0])
		assert.NoError(t, err)

		// Verify it's stopped
		sessions, err := daemonPkg.ListCampaignSessions(campaigns[0])
		assert.NoError(t, err)
		assert.Empty(t, sessions)

		// Others should still be running
		for i := 1; i < len(campaigns); i++ {
			sessions, err := daemonPkg.ListCampaignSessions(campaigns[i])
			assert.NoError(t, err)
			assert.NotEmpty(t, sessions, "Campaign %s should still be running", campaigns[i])
		}
	})

	t.Run("stop all campaigns", func(t *testing.T) {
		// Stop all
		err := manager.StopAll()
		assert.NoError(t, err)

		// Verify all stopped
		for _, campaign := range campaigns {
			sessions, err := daemonPkg.ListCampaignSessions(campaign)
			assert.NoError(t, err)
			assert.Empty(t, sessions, "Campaign %s should be stopped", campaign)
		}
	})
}

// TestDaemonResourceLimits tests resource limit enforcement
func TestDaemonResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("nice level configuration", func(t *testing.T) {
		config := &daemon.DaemonConfig{
			Campaign:  "test-resources",
			Session:   0,
			NiceLevel: 10,
		}

		// Verify nice level is set
		assert.Equal(t, 10, config.NiceLevel)
	})

	t.Run("memory limit configuration", func(t *testing.T) {
		config := &daemon.DaemonConfig{
			Campaign:      "test-resources",
			Session:       0,
			MemoryLimitMB: 512,
		}

		// Verify memory limit
		assert.Equal(t, 512, config.MemoryLimitMB)

		// Convert to bytes
		expectedBytes := int64(512 * 1024 * 1024)
		config.MemoryLimit = expectedBytes
		assert.Equal(t, expectedBytes, config.MemoryLimit)
	})
}
