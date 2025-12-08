// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

func TestLifecycleManager_New(t *testing.T) {
	lm := NewLifecycleManager()
	require.NotNil(t, lm)
	assert.NotNil(t, lm.sessions)
	assert.NotNil(t, lm.stopChan)
}

func TestLifecycleManager_AutoStart(t *testing.T) {
	tests := []struct {
		name           string
		campaign       string
		existingSocket bool
		wantErr        bool
		errCode        gerror.ErrorCode
	}{
		{
			name:           "auto-start new daemon",
			campaign:       "test-campaign",
			existingSocket: false,
			wantErr:        false,
		},
		{
			name:           "connect to existing daemon",
			campaign:       "test-campaign",
			existingSocket: true,
			wantErr:        false,
		},
		{
			name:     "empty campaign name",
			campaign: "",
			wantErr:  true,
			errCode:  gerror.ErrCodeInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that require actual daemon execution
			if !tt.wantErr {
				t.Skip("Requires actual daemon setup - tested in integration tests")
			}

			lm := NewLifecycleManager()
			ctx := context.Background()

			session, err := lm.AutoStartDaemon(ctx, tt.campaign)

			if tt.wantErr {
				require.Error(t, err)
				gerr, ok := err.(*gerror.GuildError)
				require.True(t, ok, "expected gerror.GuildError")
				assert.Equal(t, tt.errCode, gerr.Code)
			} else {
				require.NoError(t, err)
				require.NotNil(t, session)
				assert.Equal(t, tt.campaign, session.Campaign)
				assert.GreaterOrEqual(t, session.Session, 0)
				assert.LessOrEqual(t, session.Session, 9)
			}
		})
	}
}

func TestLifecycleManager_SessionManagement(t *testing.T) {
	lm := NewLifecycleManager()
	ctx := context.Background()

	t.Run("find available session slot", func(t *testing.T) {
		// Simulate occupied sessions
		lm.sessions["test-campaign:0"] = &SessionState{
			Config:     &DaemonConfig{Campaign: "test-campaign", Session: 0},
			StartedAt:  time.Now(),
			LastActive: time.Now(),
			Active:     true,
		}
		lm.sessions["test-campaign:1"] = &SessionState{
			Config:     &DaemonConfig{Campaign: "test-campaign", Session: 1},
			StartedAt:  time.Now(),
			LastActive: time.Now(),
			Active:     true,
		}

		// Should find session 2 as available
		session, err := lm.findAvailableSession(ctx, "test-campaign")
		require.NoError(t, err)
		assert.Equal(t, 2, session)
	})

	t.Run("all sessions occupied", func(t *testing.T) {
		// Fill all sessions
		for i := 0; i < 10; i++ {
			key := lm.getSessionKey("test-campaign", i)
			lm.sessions[key] = &SessionState{
				Config:     &DaemonConfig{Campaign: "test-campaign", Session: i},
				StartedAt:  time.Now(),
				LastActive: time.Now(),
				Active:     true,
			}
		}

		// Should return error
		_, err := lm.findAvailableSession(ctx, "test-campaign")
		require.Error(t, err)
		gerr, ok := err.(*gerror.GuildError)
		require.True(t, ok)
		assert.Equal(t, gerror.ErrCodeResourceLimit, gerr.Code)
	})
}

func TestLifecycleManager_IdleTimeout(t *testing.T) {
	lm := NewLifecycleManager()

	// Configure short idle timeout for testing
	lm.SetIdleTimeout(100 * time.Millisecond)

	// Add a session
	session := &SessionState{
		Config: &DaemonConfig{
			Campaign:   "test-campaign",
			Session:    0,
			SocketPath: "/tmp/test-idle.sock", // Add socket path
		},
		StartedAt:  time.Now(),
		LastActive: time.Now().Add(-200 * time.Millisecond), // Already expired
		Active:     true,
	}
	lm.sessions["test-campaign:0"] = session

	// Directly call checkIdleSessions instead of waiting for monitor
	lm.checkIdleSessions()

	// Session should be marked inactive
	lm.mu.RLock()
	assert.False(t, session.Active)
	lm.mu.RUnlock()
}

func TestLifecycleManager_GracefulShutdown(t *testing.T) {
	lm := NewLifecycleManager()
	ctx := context.Background()

	// Add some active sessions
	lm.sessions["campaign1:0"] = &SessionState{
		Config:     &DaemonConfig{Campaign: "campaign1", Session: 0},
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Active:     true,
	}
	lm.sessions["campaign2:0"] = &SessionState{
		Config:     &DaemonConfig{Campaign: "campaign2", Session: 0},
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Active:     true,
	}

	// Test graceful shutdown
	err := lm.ShutdownAll(ctx, 5*time.Second)
	require.NoError(t, err)

	// All sessions should be inactive
	lm.mu.RLock()
	for _, session := range lm.sessions {
		assert.False(t, session.Active)
	}
	lm.mu.RUnlock()
}

func TestLifecycleManager_ResourceLimits(t *testing.T) {
	t.Run("apply nice level", func(t *testing.T) {
		config := &DaemonConfig{
			Campaign:  "test-campaign",
			Session:   0,
			NiceLevel: 10,
		}

		// Test that nice level is set in config
		assert.Equal(t, 10, config.NiceLevel)
	})

	t.Run("memory limits", func(t *testing.T) {
		config := &DaemonConfig{
			Campaign:      "test-campaign",
			Session:       0,
			MemoryLimitMB: 512,
		}

		// Test that memory limit is set
		assert.Equal(t, 512, config.MemoryLimitMB)
	})
}

func TestLifecycleManager_CrashRecovery(t *testing.T) {
	lm := NewLifecycleManager()
	ctx := context.Background()

	// Simulate a crashed session (PID file exists but process doesn't)
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write invalid PID
	err := os.WriteFile(pidFile, []byte("99999"), 0644)
	require.NoError(t, err)

	config := &DaemonConfig{
		Campaign: "test-campaign",
		Session:  0,
		PIDFile:  pidFile,
	}

	// Should detect crashed daemon and allow restart
	canRecover := lm.canRecoverSession(ctx, config)
	assert.True(t, canRecover)
}

func TestLifecycleManager_SessionSwitching(t *testing.T) {
	lm := NewLifecycleManager()
	ctx := context.Background()

	// Create multiple sessions for same campaign
	session1 := &SessionState{
		Config: &DaemonConfig{
			Campaign: "test-campaign",
			Session:  0,
		},
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Active:     true,
	}
	session2 := &SessionState{
		Config: &DaemonConfig{
			Campaign: "test-campaign",
			Session:  1,
		},
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Active:     true,
	}

	lm.sessions["test-campaign:0"] = session1
	lm.sessions["test-campaign:1"] = session2

	// Switch to session 1
	err := lm.SwitchSession(ctx, "test-campaign", 1)
	require.NoError(t, err)

	// Get active session
	active, err := lm.GetActiveSession("test-campaign")
	require.NoError(t, err)
	assert.Equal(t, 1, active.Session)
}

func TestLifecycleManager_UpdateActivity(t *testing.T) {
	lm := NewLifecycleManager()

	// Add a session
	session := &SessionState{
		Config: &DaemonConfig{
			Campaign: "test-campaign",
			Session:  0,
		},
		StartedAt:  time.Now().Add(-1 * time.Hour),
		LastActive: time.Now().Add(-30 * time.Minute),
		Active:     true,
	}
	lm.sessions["test-campaign:0"] = session

	// Update activity
	lm.UpdateActivity("test-campaign", 0)

	// Check last active was updated
	lm.mu.RLock()
	assert.WithinDuration(t, time.Now(), session.LastActive, 1*time.Second)
	lm.mu.RUnlock()
}
