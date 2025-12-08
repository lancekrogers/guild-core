// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package daemon

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	daemonPkg "github.com/guild-framework/guild-core/pkg/daemon"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

// SessionState tracks the state of a daemon session
type SessionState struct {
	Config       *DaemonConfig
	StartedAt    time.Time
	LastActive   time.Time
	Active       bool
	ProcessID    int
	RestartCount int
}

// LifecycleManager manages daemon lifecycle including auto-start, idle timeout, and resource management
type LifecycleManager struct {
	mu            sync.RWMutex
	sessions      map[string]*SessionState // key: campaign:session
	activeSession map[string]int           // campaign -> active session number
	idleTimeout   time.Duration
	stopChan      chan struct{}
	manager       *Manager
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		sessions:      make(map[string]*SessionState),
		activeSession: make(map[string]int),
		idleTimeout:   15 * time.Minute, // Default idle timeout
		stopChan:      make(chan struct{}),
		manager:       DefaultManager,
	}
}

// AutoStartDaemon automatically starts a daemon for the campaign if not running
func (lm *LifecycleManager) AutoStartDaemon(ctx context.Context, campaign string) (*DaemonConfig, error) {
	if campaign == "" {
		return nil, gerror.New(gerror.ErrCodeInvalidInput, "campaign name cannot be empty", nil).
			WithComponent("daemon").
			WithOperation("LifecycleManager.AutoStartDaemon")
	}

	// Check if we have an active session
	lm.mu.RLock()
	if activeSession, exists := lm.activeSession[campaign]; exists {
		session := lm.sessions[lm.getSessionKey(campaign, activeSession)]
		if session != nil && session.Active {
			lm.mu.RUnlock()
			// Update activity
			lm.UpdateActivity(campaign, activeSession)
			return session.Config, nil
		}
	}
	lm.mu.RUnlock()

	// Find available session slot
	sessionNum, err := lm.findAvailableSession(ctx, campaign)
	if err != nil {
		return nil, err
	}

	// Use the manager to ensure daemon is running
	config, err := lm.manager.EnsureDaemonRunning(ctx, campaign, sessionNum)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to start daemon").
			WithComponent("daemon").
			WithOperation("LifecycleManager.AutoStartDaemon").
			WithDetails("campaign", campaign).
			WithDetails("session", sessionNum)
	}

	// Track the session
	lm.mu.Lock()
	key := lm.getSessionKey(campaign, sessionNum)
	lm.sessions[key] = &SessionState{
		Config:     config,
		StartedAt:  time.Now(),
		LastActive: time.Now(),
		Active:     true,
		ProcessID:  lm.getProcessID(config),
	}
	lm.activeSession[campaign] = sessionNum
	lm.mu.Unlock()

	return config, nil
}

// findAvailableSession finds an available session slot for a campaign
func (lm *LifecycleManager) findAvailableSession(ctx context.Context, campaign string) (int, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Check existing sessions
	for session := 0; session < 10; session++ {
		key := lm.getSessionKey(campaign, session)
		state, exists := lm.sessions[key]

		// Slot is available if:
		// 1. No session exists
		// 2. Session is inactive
		// 3. Session process has crashed
		if !exists || !state.Active {
			return session, nil
		}

		// Check if process is still running
		if state.ProcessID > 0 && !lm.isProcessRunning(state.ProcessID) {
			// Process crashed, mark as inactive
			state.Active = false
			return session, nil
		}
	}

	return 0, gerror.New(gerror.ErrCodeResourceLimit, "maximum sessions reached for campaign", nil).
		WithComponent("daemon").
		WithOperation("LifecycleManager.findAvailableSession").
		WithDetails("campaign", campaign).
		WithDetails("max_sessions", 10)
}

// UpdateActivity updates the last active time for a session
func (lm *LifecycleManager) UpdateActivity(campaign string, session int) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	key := lm.getSessionKey(campaign, session)
	if state, exists := lm.sessions[key]; exists {
		state.LastActive = time.Now()
	}
}

// MonitorSessions starts monitoring sessions for idle timeout and crashes
func (lm *LifecycleManager) MonitorSessions(ctx context.Context) {
	go lm.monitorIdleSessions(ctx)
	go lm.monitorCrashedSessions(ctx)
}

// monitorIdleSessions checks for idle sessions and stops them
func (lm *LifecycleManager) monitorIdleSessions(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lm.stopChan:
			return
		case <-ticker.C:
			lm.checkIdleSessions()
		}
	}
}

// checkIdleSessions checks and stops idle sessions
func (lm *LifecycleManager) checkIdleSessions() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	for _, session := range lm.sessions {
		if !session.Active {
			continue
		}

		// Check if session is idle
		if now.Sub(session.LastActive) > lm.idleTimeout {
			// Stop idle session
			if err := lm.stopSessionInternal(session); err == nil {
				session.Active = false
			}
		}
	}
}

// monitorCrashedSessions checks for crashed daemons and marks them as inactive
func (lm *LifecycleManager) monitorCrashedSessions(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-lm.stopChan:
			return
		case <-ticker.C:
			lm.checkCrashedSessions()
		}
	}
}

// checkCrashedSessions checks for crashed processes
func (lm *LifecycleManager) checkCrashedSessions() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for _, session := range lm.sessions {
		if !session.Active {
			continue
		}

		// Check if process is still running
		if session.ProcessID > 0 && !lm.isProcessRunning(session.ProcessID) {
			session.Active = false
			// Clean up socket if it exists
			if session.Config != nil {
				daemonPkg.UnlinkIfStale(session.Config.SocketPath)
			}
		}
	}
}

// SwitchSession switches the active session for a campaign
func (lm *LifecycleManager) SwitchSession(ctx context.Context, campaign string, session int) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Validate session exists and is active
	key := lm.getSessionKey(campaign, session)
	state, exists := lm.sessions[key]
	if !exists || !state.Active {
		return gerror.New(gerror.ErrCodeNotFound, "session not found or inactive", nil).
			WithComponent("daemon").
			WithOperation("LifecycleManager.SwitchSession").
			WithDetails("campaign", campaign).
			WithDetails("session", session)
	}

	// Update active session
	lm.activeSession[campaign] = session
	state.LastActive = time.Now()

	return nil
}

// GetActiveSession returns the active session for a campaign
func (lm *LifecycleManager) GetActiveSession(campaign string) (*DaemonConfig, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	session, exists := lm.activeSession[campaign]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no active session for campaign", nil).
			WithComponent("daemon").
			WithOperation("LifecycleManager.GetActiveSession").
			WithDetails("campaign", campaign)
	}

	key := lm.getSessionKey(campaign, session)
	state, exists := lm.sessions[key]
	if !exists || !state.Active {
		return nil, gerror.New(gerror.ErrCodeNotFound, "active session not found", nil).
			WithComponent("daemon").
			WithOperation("LifecycleManager.GetActiveSession").
			WithDetails("campaign", campaign).
			WithDetails("session", session)
	}

	return state.Config, nil
}

// ShutdownAll gracefully shuts down all managed sessions
func (lm *LifecycleManager) ShutdownAll(ctx context.Context, timeout time.Duration) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Stop monitoring
	close(lm.stopChan)

	// Shutdown all active sessions
	var errors []error
	for _, session := range lm.sessions {
		if !session.Active {
			continue
		}

		if err := lm.stopSessionInternal(session); err != nil {
			errors = append(errors, err)
		} else {
			session.Active = false
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, "failed to shutdown some sessions", nil).
			WithComponent("daemon").
			WithOperation("LifecycleManager.ShutdownAll").
			WithDetails("error_count", len(errors))
	}

	return nil
}

// SetIdleTimeout sets the idle timeout duration
func (lm *LifecycleManager) SetIdleTimeout(timeout time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.idleTimeout = timeout
}

// canRecoverSession checks if a session can be recovered after crash
func (lm *LifecycleManager) canRecoverSession(ctx context.Context, config *DaemonConfig) bool {
	// Check if PID file exists
	if config.PIDFile == "" {
		return true // No PID file, can start fresh
	}

	pidData, err := os.ReadFile(config.PIDFile)
	if err != nil {
		return true // No PID file, can start fresh
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return true // Invalid PID, can start fresh
	}

	// Check if process is running
	return !lm.isProcessRunning(pid)
}

// Helper methods

func (lm *LifecycleManager) getSessionKey(campaign string, session int) string {
	return fmt.Sprintf("%s:%d", campaign, session)
}

func (lm *LifecycleManager) getProcessID(config *DaemonConfig) int {
	if config.PIDFile == "" {
		return 0
	}

	pidData, err := os.ReadFile(config.PIDFile)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return 0
	}

	return pid
}

func (lm *LifecycleManager) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (lm *LifecycleManager) stopSessionInternal(session *SessionState) error {
	if session.Config == nil {
		return nil
	}

	// Try to stop via socket first
	if err := daemonPkg.StopSession(session.Config.SocketPath); err == nil {
		return nil
	}

	// If socket stop failed and we have PID, try kill
	if session.ProcessID > 0 {
		process, err := os.FindProcess(session.ProcessID)
		if err == nil {
			// Try graceful shutdown first
			if err := process.Signal(syscall.SIGTERM); err == nil {
				// Wait a bit for graceful shutdown
				time.Sleep(2 * time.Second)

				// Check if still running
				if lm.isProcessRunning(session.ProcessID) {
					// Force kill
					process.Signal(syscall.SIGKILL)
				}
			}
		}
	}

	// Clean up socket file
	os.Remove(session.Config.SocketPath)

	// Clean up PID file
	if session.Config.PIDFile != "" {
		os.Remove(session.Config.PIDFile)
	}

	return nil
}

// Global lifecycle manager instance
var DefaultLifecycleManager = NewLifecycleManager()
