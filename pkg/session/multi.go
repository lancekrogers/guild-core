// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// MultiSessionManager manages multiple parallel chat sessions
type MultiSessionManager struct {
	manager     *SessionManager
	sessions    map[string]*SessionHandle
	activeID    string
	mu          sync.RWMutex
	maxSessions int
	resumer     *SessionResumer
}

// SessionHandle holds a session with its active state
type SessionHandle struct {
	Session      *Session
	LastAccessed time.Time
	Active       bool
	AutoSaveStop context.CancelFunc
}

// NewMultiSessionManager creates a manager for multiple sessions
func NewMultiSessionManager(manager *SessionManager, resumer *SessionResumer, maxSessions int) *MultiSessionManager {
	if maxSessions <= 0 {
		maxSessions = 10 // Default to 10 parallel sessions
	}

	return &MultiSessionManager{
		manager:     manager,
		sessions:    make(map[string]*SessionHandle),
		maxSessions: maxSessions,
		resumer:     resumer,
	}
}

// CreateSession creates a new session and makes it active
func (msm *MultiSessionManager) CreateSession(ctx context.Context, userID, campaignID string) (*Session, error) {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	// Check session limit
	if len(msm.sessions) >= msm.maxSessions {
		// Find and close the least recently used session
		if err := msm.closeLRUSessionLocked(ctx); err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeResourceLimit, "failed to make room for new session")
		}
	}

	// Create new session
	session, err := msm.manager.CreateSession(ctx, userID, campaignID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create session")
	}

	// Start auto-save
	saveCtx, cancel := context.WithCancel(context.Background())
	go msm.manager.StartAutoSave(saveCtx, session)

	// Add to active sessions
	handle := &SessionHandle{
		Session:      session,
		LastAccessed: time.Now(),
		Active:       true,
		AutoSaveStop: cancel,
	}
	msm.sessions[session.ID] = handle
	msm.activeID = session.ID

	return session, nil
}

// SwitchSession switches to a different session
func (msm *MultiSessionManager) SwitchSession(ctx context.Context, sessionID string) (*Session, error) {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	handle, exists := msm.sessions[sessionID]
	if !exists {
		// Try to load from storage
		session, err := msm.manager.LoadSession(ctx, sessionID)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "session not found")
		}

		// Check session limit
		if len(msm.sessions) >= msm.maxSessions {
			if err := msm.closeLRUSessionLocked(ctx); err != nil {
				return nil, gerror.Wrap(err, gerror.ErrCodeResourceLimit, "failed to make room for session")
			}
		}

		// Start auto-save
		saveCtx, cancel := context.WithCancel(context.Background())
		go msm.manager.StartAutoSave(saveCtx, session)

		handle = &SessionHandle{
			Session:      session,
			LastAccessed: time.Now(),
			Active:       true,
			AutoSaveStop: cancel,
		}
		msm.sessions[sessionID] = handle
	}

	// Update active session
	if msm.activeID != "" && msm.activeID != sessionID {
		if oldHandle, exists := msm.sessions[msm.activeID]; exists {
			oldHandle.Active = false
		}
	}

	handle.Active = true
	handle.LastAccessed = time.Now()
	msm.activeID = sessionID

	return handle.Session, nil
}

// GetActiveSession returns the currently active session
func (msm *MultiSessionManager) GetActiveSession() (*Session, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	if msm.activeID == "" {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no active session", nil)
	}

	handle, exists := msm.sessions[msm.activeID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "active session not found", nil)
	}

	handle.LastAccessed = time.Now()
	return handle.Session, nil
}

// ListActiveSessions returns all active sessions
func (msm *MultiSessionManager) ListActiveSessions() ([]*SessionInfo, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	var sessions []*SessionInfo
	for id, handle := range msm.sessions {
		info := &SessionInfo{
			ID:           id,
			Name:         msm.getSessionName(handle.Session),
			CampaignID:   handle.Session.CampaignID,
			Active:       handle.Active,
			LastAccessed: handle.LastAccessed,
			MessageCount: len(handle.Session.Messages),
			Status:       handle.Session.State.Status,
			CreatedAt:    handle.Session.StartTime,
		}
		sessions = append(sessions, info)
	}

	// Sort by last accessed time
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastAccessed.After(sessions[j].LastAccessed)
	})

	return sessions, nil
}

// SessionInfo provides summary information about a session
type SessionInfo struct {
	ID           string
	Name         string
	CampaignID   string
	Active       bool
	LastAccessed time.Time
	MessageCount int
	Status       SessionStatus
	CreatedAt    time.Time
}

// CloseSession closes a session and removes it from active sessions
func (msm *MultiSessionManager) CloseSession(ctx context.Context, sessionID string) error {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	handle, exists := msm.sessions[sessionID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "session not found", nil)
	}

	// Stop auto-save
	if handle.AutoSaveStop != nil {
		handle.AutoSaveStop()
	}

	// Update session status
	handle.Session.State.Status = SessionStatusClosed
	handle.Session.LastActiveTime = time.Now()

	// Save final state
	if err := msm.manager.SaveSession(ctx, handle.Session); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save session on close")
	}

	// Remove from active sessions
	delete(msm.sessions, sessionID)

	// If this was the active session, clear activeID
	if msm.activeID == sessionID {
		msm.activeID = ""
		// Try to activate the most recent session
		msm.activateMostRecentSessionLocked()
	}

	return nil
}

// CloseAllSessions closes all active sessions
func (msm *MultiSessionManager) CloseAllSessions(ctx context.Context) error {
	msm.mu.Lock()
	defer msm.mu.Unlock()

	var errors []error
	for id := range msm.sessions {
		if err := msm.closeSessionInternalLocked(ctx, id); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, fmt.Sprintf("%d sessions failed to close", len(errors)), errors[0])
	}

	return nil
}

// ResumeSession resumes a session from storage
func (msm *MultiSessionManager) ResumeSession(ctx context.Context, sessionID string) error {
	// First switch to the session (loads if needed)
	session, err := msm.SwitchSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Then resume its state
	if msm.resumer != nil {
		return msm.resumer.ResumeSession(ctx, session.ID)
	}

	return nil
}

// RecoverFromCrash attempts to recover sessions after a crash
func (msm *MultiSessionManager) RecoverFromCrash(ctx context.Context) ([]*Session, error) {
	// Find recent sessions that might need recovery
	options := ListOptions{
		OrderBy: "last_active_time DESC",
		Limit:   msm.maxSessions,
	}

	sessions, err := msm.manager.ListSessions(ctx, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions for recovery")
	}

	var recovered []*Session
	for _, session := range sessions {
		// Check if session was active within last hour
		if time.Since(session.LastActiveTime) < time.Hour &&
			session.State.Status == SessionStatusActive {
			// Load and activate the session
			if _, err := msm.SwitchSession(ctx, session.ID); err == nil {
				recovered = append(recovered, session)
			}
		}
	}

	return recovered, nil
}

// UpdateActiveSessionState updates the state of the active session
func (msm *MultiSessionManager) UpdateActiveSessionState(ctx context.Context, state SessionState) error {
	session, err := msm.GetActiveSession()
	if err != nil {
		return err
	}

	session.State = state
	session.LastActiveTime = time.Now()

	return msm.manager.SaveSession(ctx, session)
}

// AddMessageToActiveSession adds a message to the active session
func (msm *MultiSessionManager) AddMessageToActiveSession(ctx context.Context, message *Message) error {
	session, err := msm.GetActiveSession()
	if err != nil {
		return err
	}

	message.ID = generateMessageID()
	message.Timestamp = time.Now()

	session.Messages = append(session.Messages, *message)
	session.LastActiveTime = time.Now()

	return msm.manager.SaveSession(ctx, session)
}

// GetSessionMessages gets messages from a specific session
func (msm *MultiSessionManager) GetSessionMessages(sessionID string, limit int) ([]*Message, error) {
	msm.mu.RLock()
	defer msm.mu.RUnlock()

	handle, exists := msm.sessions[sessionID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "session not found", nil)
	}

	messages := make([]*Message, 0)
	start := len(handle.Session.Messages) - limit
	if start < 0 {
		start = 0
	}

	for i := start; i < len(handle.Session.Messages); i++ {
		msg := handle.Session.Messages[i]
		messages = append(messages, &msg)
	}

	return messages, nil
}

// Helper methods

func (msm *MultiSessionManager) getSessionName(session *Session) string {
	if name, exists := session.Metadata["name"]; exists {
		if str, ok := name.(string); ok {
			return str
		}
	}

	// Generate a name based on first message or time
	if len(session.Messages) > 0 {
		firstMsg := session.Messages[0].Content
		if len(firstMsg) > 30 {
			firstMsg = firstMsg[:30] + "..."
		}
		return firstMsg
	}

	return fmt.Sprintf("Session %s", session.StartTime.Format("Jan 2 15:04"))
}

func (msm *MultiSessionManager) closeLRUSessionLocked(ctx context.Context) error {
	if len(msm.sessions) == 0 {
		return nil
	}

	// Find least recently used session
	var lruID string
	var lruTime time.Time
	for id, handle := range msm.sessions {
		if !handle.Active && (lruID == "" || handle.LastAccessed.Before(lruTime)) {
			lruID = id
			lruTime = handle.LastAccessed
		}
	}

	// If all sessions are active, pick the oldest
	if lruID == "" {
		for id, handle := range msm.sessions {
			if lruID == "" || handle.LastAccessed.Before(lruTime) {
				lruID = id
				lruTime = handle.LastAccessed
			}
		}
	}

	if lruID != "" {
		return msm.closeSessionInternalLocked(ctx, lruID)
	}

	return nil
}

func (msm *MultiSessionManager) closeSessionInternalLocked(ctx context.Context, sessionID string) error {
	handle, exists := msm.sessions[sessionID]
	if !exists {
		return nil
	}

	// Stop auto-save
	if handle.AutoSaveStop != nil {
		handle.AutoSaveStop()
	}

	// Update session status
	handle.Session.State.Status = SessionStatusPaused // Use paused instead of closed for LRU
	handle.Session.LastActiveTime = time.Now()

	// Save final state
	if err := msm.manager.SaveSession(ctx, handle.Session); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save session")
	}

	// Remove from active sessions
	delete(msm.sessions, sessionID)

	return nil
}

func (msm *MultiSessionManager) activateMostRecentSessionLocked() {
	var mostRecentID string
	var mostRecentTime time.Time

	for id, handle := range msm.sessions {
		if mostRecentID == "" || handle.LastAccessed.After(mostRecentTime) {
			mostRecentID = id
			mostRecentTime = handle.LastAccessed
		}
	}

	if mostRecentID != "" {
		msm.activeID = mostRecentID
		if handle, exists := msm.sessions[mostRecentID]; exists {
			handle.Active = true
		}
	}
}

// GetResumableSessions returns sessions that can be resumed
func (msm *MultiSessionManager) GetResumableSessions(ctx context.Context, userID string) ([]*SessionInfo, error) {
	options := ListOptions{
		OrderBy: "last_active_time DESC",
		Limit:   50,
	}

	sessions, err := msm.manager.ListSessions(ctx, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions")
	}

	var resumable []*SessionInfo
	for _, session := range sessions {
		if session.UserID == userID &&
			(session.State.Status == SessionStatusActive ||
				session.State.Status == SessionStatusPaused) &&
			time.Since(session.LastActiveTime) < 24*time.Hour {

			info := &SessionInfo{
				ID:           session.ID,
				Name:         msm.getSessionName(session),
				CampaignID:   session.CampaignID,
				Active:       false,
				LastAccessed: session.LastActiveTime,
				MessageCount: len(session.Messages),
				Status:       session.State.Status,
				CreatedAt:    session.StartTime,
			}
			resumable = append(resumable, info)
		}
	}

	return resumable, nil
}
