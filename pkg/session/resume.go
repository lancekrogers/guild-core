// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
)

// SessionResumer handles seamless session restoration
type SessionResumer struct {
	manager      *SessionManager
	ui           UIRestorer
	orchestrator OrchestratorInterface
	corpus       CorpusInterface
}

// UIRestorer defines the interface for UI state restoration
type UIRestorer interface {
	AddMessage(msg Message) error
	SetView(view string) error
	SetScrollPosition(position int) error
	SetInput(input string) error
	SetCommandHistory(history []string) error
	ShowNotification(message string) error
	ShowRetryPrompt(task Task) error
}

// OrchestratorInterface defines the interface for agent orchestration
type OrchestratorInterface interface {
	ConnectAgent(ctx context.Context, agentID string) (AgentInterface, error)
	GetTaskStatus(taskID string) TaskStatus
	ResumeTask(taskID string) error
}

// AgentInterface defines the interface for agents
type AgentInterface interface {
	RestoreState(ctx context.Context, state AgentState) error
	SetContext(ctx context.Context, context AgentContext) error
}

// CorpusInterface defines the interface for corpus operations
type CorpusInterface interface {
	BuildAgentContext(ctx context.Context, session *Session, agentID string) (AgentContext, error)
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCompleted TaskStatus = "completed"
)

// Task represents a task in the system
type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AgentContext represents the context provided to an agent
type AgentContext struct {
	WorkingDirectory string                 `json:"working_directory"`
	OpenFiles        []string               `json:"open_files"`
	Variables        map[string]interface{} `json:"variables"`
	History          []Message              `json:"history"`
}

// ResumeContext contains information needed for session resumption
type ResumeContext struct {
	Session         *Session  `json:"session"`
	ActiveTasks     []Task    `json:"active_tasks"`
	PendingMessages []Message `json:"pending_messages"`
	LastActivity    time.Time `json:"last_activity"`
}

// NewSessionResumer creates a new session resumer
func NewSessionResumer(manager *SessionManager, ui UIRestorer, orchestrator OrchestratorInterface, corpus CorpusInterface) *SessionResumer {
	return &SessionResumer{
		manager:      manager,
		ui:           ui,
		orchestrator: orchestrator,
		corpus:       corpus,
	}
}

// ResumeSession restores a session to its previous state
func (sr *SessionResumer) ResumeSession(ctx context.Context, sessionID string) error {
	// Load session
	session, err := sr.manager.LoadSession(ctx, sessionID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load session for resume")
	}

	// Build resume context
	resumeCtx, err := sr.buildResumeContext(ctx, session)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to build resume context")
	}

	// Restore UI state
	if err := sr.restoreUIState(ctx, session); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to restore UI state")
	}

	// Reconnect agents
	if err := sr.reconnectAgents(ctx, session); err != nil {
		log.Printf("Warning: failed to reconnect some agents: %v", err)
		// Continue with resume even if some agents fail to reconnect
	}

	// Resume active tasks
	if err := sr.resumeTasks(ctx, resumeCtx); err != nil {
		log.Printf("Warning: failed to resume some tasks: %v", err)
		// Continue with resume even if some tasks fail to resume
	}

	// Show resume summary
	if err := sr.showResumeSummary(ctx, resumeCtx); err != nil {
		log.Printf("Warning: failed to show resume summary: %v", err)
		// This is not critical for resume functionality
	}

	return nil
}

// buildResumeContext creates the context needed for session resumption
func (sr *SessionResumer) buildResumeContext(ctx context.Context, session *Session) (*ResumeContext, error) {
	resumeCtx := &ResumeContext{
		Session:      session,
		LastActivity: session.LastActiveTime,
	}

	// Find active tasks (this would be enhanced to query actual task system)
	activeTasks := []Task{}
	for _, runningTaskID := range session.Context.RunningTasks {
		status := sr.orchestrator.GetTaskStatus(runningTaskID)
		if status == TaskStatusRunning || status == TaskStatusPaused {
			task := Task{
				ID:     runningTaskID,
				Title:  fmt.Sprintf("Task %s", runningTaskID),
				Status: status,
			}
			activeTasks = append(activeTasks, task)
		}
	}
	resumeCtx.ActiveTasks = activeTasks

	// Find pending messages (messages created while session was inactive)
	pendingMessages := []Message{}
	for _, msg := range session.Messages {
		if msg.Timestamp.After(session.LastActiveTime) {
			pendingMessages = append(pendingMessages, msg)
		}
	}
	resumeCtx.PendingMessages = pendingMessages

	return resumeCtx, nil
}

// restoreUIState restores the UI to its previous state
func (sr *SessionResumer) restoreUIState(ctx context.Context, session *Session) error {
	// Restore messages
	for _, msg := range session.Messages {
		if err := sr.ui.AddMessage(msg); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to restore message")
		}
	}

	// Restore view
	if session.State.CurrentView != "" {
		if err := sr.ui.SetView(session.State.CurrentView); err != nil {
			log.Printf("Warning: failed to restore view %s: %v", session.State.CurrentView, err)
		}
	}

	// Restore scroll position
	if session.State.ScrollPosition > 0 {
		if err := sr.ui.SetScrollPosition(session.State.ScrollPosition); err != nil {
			log.Printf("Warning: failed to restore scroll position: %v", err)
		}
	}

	// Restore input buffer
	if session.State.InputBuffer != "" {
		if err := sr.ui.SetInput(session.State.InputBuffer); err != nil {
			log.Printf("Warning: failed to restore input buffer: %v", err)
		}
	}

	// Restore command history
	if len(session.State.CommandHistory) > 0 {
		if err := sr.ui.SetCommandHistory(session.State.CommandHistory); err != nil {
			log.Printf("Warning: failed to restore command history: %v", err)
		}
	}

	return nil
}

// reconnectAgents reconnects all agents that were active in the session
func (sr *SessionResumer) reconnectAgents(ctx context.Context, session *Session) error {
	var errors []string

	for agentID, state := range session.State.ActiveAgents {
		// Recreate agent connection
		agent, err := sr.orchestrator.ConnectAgent(ctx, agentID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to reconnect agent %s: %v", agentID, err))
			continue
		}

		// Restore agent state
		if err := agent.RestoreState(ctx, state); err != nil {
			errors = append(errors, fmt.Sprintf("failed to restore agent %s state: %v", agentID, err))
			continue
		}

		// Re-inject context
		agentContext, err := sr.corpus.BuildAgentContext(ctx, session, agentID)
		if err != nil {
			log.Printf("Warning: failed to build context for agent %s: %v", agentID, err)
			// Use basic context if corpus fails
			agentContext = AgentContext{
				WorkingDirectory: session.Context.WorkingDirectory,
				OpenFiles:        session.Context.OpenFiles,
				Variables:        session.State.Variables,
			}
		}

		if err := agent.SetContext(ctx, agentContext); err != nil {
			errors = append(errors, fmt.Sprintf("failed to set context for agent %s: %v", agentID, err))
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, strings.Join(errors, "; "), nil)
	}

	return nil
}

// resumeTasks resumes active tasks from the session
func (sr *SessionResumer) resumeTasks(ctx context.Context, resumeCtx *ResumeContext) error {
	var errors []string

	for _, task := range resumeCtx.ActiveTasks {
		// Check current task status
		status := sr.orchestrator.GetTaskStatus(task.ID)

		switch status {
		case TaskStatusRunning:
			// Already running, just show notification
			err := sr.ui.ShowNotification(fmt.Sprintf("Task %s is still running", task.ID))
			if err != nil {
				log.Printf("Warning: failed to show notification: %v", err)
			}

		case TaskStatusPaused:
			// Resume paused task
			if err := sr.orchestrator.ResumeTask(task.ID); err != nil {
				errors = append(errors, fmt.Sprintf("failed to resume task %s: %v", task.ID, err))
			} else {
				err := sr.ui.ShowNotification(fmt.Sprintf("Resumed task %s", task.ID))
				if err != nil {
					log.Printf("Warning: failed to show notification: %v", err)
				}
			}

		case TaskStatusFailed:
			// Offer to retry
			if err := sr.ui.ShowRetryPrompt(task); err != nil {
				log.Printf("Warning: failed to show retry prompt for task %s: %v", task.ID, err)
			}

		case TaskStatusCompleted:
			// Task completed while away, show notification
			err := sr.ui.ShowNotification(fmt.Sprintf("Task %s completed while you were away", task.ID))
			if err != nil {
				log.Printf("Warning: failed to show notification: %v", err)
			}
		}
	}

	if len(errors) > 0 {
		return gerror.New(gerror.ErrCodeInternal, strings.Join(errors, "; "), nil)
	}

	return nil
}

// showResumeSummary displays a summary of the session resumption
func (sr *SessionResumer) showResumeSummary(ctx context.Context, resumeCtx *ResumeContext) error {
	summary := sr.generateSummary(resumeCtx)

	// Create system message
	msg := Message{
		ID:        generateMessageID(),
		Agent:     "system",
		Content:   summary,
		Timestamp: time.Now(),
		Type:      MessageTypeSystem,
		Metadata: map[string]interface{}{
			"type": "resume_summary",
		},
	}

	return sr.ui.AddMessage(msg)
}

// generateSummary creates a formatted summary of the session resumption
func (sr *SessionResumer) generateSummary(resumeCtx *ResumeContext) string {
	timeSince := time.Since(resumeCtx.LastActivity)

	var summary strings.Builder
	summary.WriteString("### Session Resumed\n\n")

	summary.WriteString(fmt.Sprintf("Welcome back! You were away for %s.\n\n",
		sr.formatDuration(timeSince)))

	if len(resumeCtx.ActiveTasks) > 0 {
		summary.WriteString("**Active Tasks:**\n")
		for _, task := range resumeCtx.ActiveTasks {
			status := sr.orchestrator.GetTaskStatus(task.ID)
			summary.WriteString(fmt.Sprintf("- %s: %s (%s)\n",
				task.ID, task.Title, string(status)))
		}
		summary.WriteString("\n")
	}

	if len(resumeCtx.PendingMessages) > 0 {
		summary.WriteString("**While you were away:**\n")
		for _, msg := range resumeCtx.PendingMessages {
			summary.WriteString(fmt.Sprintf("- %s: %s\n",
				msg.Agent, sr.truncate(msg.Content, 50)))
		}
		summary.WriteString("\n")
	}

	if len(resumeCtx.Session.State.ActiveAgents) > 0 {
		summary.WriteString("**Active Agents:**\n")
		for agentID, state := range resumeCtx.Session.State.ActiveAgents {
			summary.WriteString(fmt.Sprintf("- %s (%s)\n", state.Name, agentID))
		}
		summary.WriteString("\n")
	}

	summary.WriteString("Ready to continue where you left off!")

	return summary.String()
}

// formatDuration formats a duration in a human-readable way
func (sr *SessionResumer) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		return fmt.Sprintf("%d minute%s", minutes, pluralize(minutes))
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%d hour%s", hours, pluralize(hours))
		}
		return fmt.Sprintf("%d hour%s %d minute%s", hours, pluralize(hours), minutes, pluralize(minutes))
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	if hours == 0 {
		return fmt.Sprintf("%d day%s", days, pluralize(days))
	}
	return fmt.Sprintf("%d day%s %d hour%s", days, pluralize(days), hours, pluralize(hours))
}

// truncate truncates a string to the specified length
func (sr *SessionResumer) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// pluralize returns "s" if count != 1, empty string otherwise
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// SessionRecovery handles crash recovery scenarios
type SessionRecovery struct {
	manager *SessionManager
	backup  *BackupManager
}

// BackupManager handles session backups
type BackupManager struct {
	unsavedChanges map[string][]Change
	mu             sync.RWMutex
}

// Change represents a change that can be applied to a session
type Change struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewSessionRecovery creates a new session recovery instance
func NewSessionRecovery(manager *SessionManager, backup *BackupManager) *SessionRecovery {
	return &SessionRecovery{
		manager: manager,
		backup:  backup,
	}
}

// NewBackupManager creates a new backup manager
func NewBackupManager() *BackupManager {
	return &BackupManager{
		unsavedChanges: make(map[string][]Change),
	}
}

// RecoverFromCrash attempts to recover the most recent session after a crash
func (sr *SessionRecovery) RecoverFromCrash(ctx context.Context) (*Session, error) {
	// Find most recent session
	sessions, err := sr.manager.ListSessions(ctx, ListOptions{
		OrderBy: "last_active_time DESC",
		Limit:   1,
	})

	if err != nil || len(sessions) == 0 {
		return nil, gerror.New(gerror.ErrCodeNotFound, "no sessions to recover", nil)
	}

	lastSession := sessions[0]

	// Check if session was properly closed
	if lastSession.State.Status == SessionStatusClosed {
		return nil, nil // No recovery needed
	}

	log.Printf("Recovering session %s", lastSession.ID)

	// Check for unsaved changes
	unsaved := sr.backup.GetUnsavedChanges(lastSession.ID)
	if len(unsaved) > 0 {
		// Apply unsaved changes
		for _, change := range unsaved {
			if err := sr.applyChange(ctx, lastSession, change); err != nil {
				log.Printf("Failed to apply change: %v", err)
			}
		}
	}

	// Mark as recovered
	if lastSession.Metadata == nil {
		lastSession.Metadata = make(map[string]interface{})
	}
	lastSession.Metadata["recovered"] = true
	lastSession.Metadata["recovery_time"] = time.Now()

	// Save the recovered session
	if err := sr.manager.SaveSession(ctx, lastSession); err != nil {
		log.Printf("Warning: failed to save recovered session: %v", err)
	}

	return lastSession, nil
}

// GetUnsavedChanges returns unsaved changes for a session
func (bm *BackupManager) GetUnsavedChanges(sessionID string) []Change {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if changes, exists := bm.unsavedChanges[sessionID]; exists {
		return changes
	}
	return []Change{}
}

// AddUnsavedChange adds an unsaved change to the backup
func (bm *BackupManager) AddUnsavedChange(sessionID string, change Change) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.unsavedChanges[sessionID] == nil {
		bm.unsavedChanges[sessionID] = []Change{}
	}
	bm.unsavedChanges[sessionID] = append(bm.unsavedChanges[sessionID], change)
}

// ClearUnsavedChanges clears unsaved changes for a session
func (bm *BackupManager) ClearUnsavedChanges(sessionID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	delete(bm.unsavedChanges, sessionID)
}

// applyChange applies a change to a session
func (sr *SessionRecovery) applyChange(ctx context.Context, session *Session, change Change) error {
	switch change.Type {
	case "message_added":
		if msgData, ok := change.Data.(map[string]interface{}); ok {
			msg := Message{
				ID:        getString(msgData, "id"),
				Agent:     getString(msgData, "agent"),
				Content:   getString(msgData, "content"),
				Timestamp: change.Timestamp,
				Type:      MessageType(getString(msgData, "type")),
			}
			session.Messages = append(session.Messages, msg)
		}
	case "state_updated":
		if stateData, ok := change.Data.(SessionState); ok {
			session.State = stateData
		}
	case "variable_set":
		if varData, ok := change.Data.(map[string]interface{}); ok {
			key := getString(varData, "key")
			value := varData["value"]
			if session.State.Variables == nil {
				session.State.Variables = make(map[string]interface{})
			}
			session.State.Variables[key] = value
		}
	default:
		log.Printf("Unknown change type: %s", change.Type)
	}

	return nil
}

// CanResumeSession checks if a session can be resumed
func (sr *SessionResumer) CanResumeSession(ctx context.Context, sessionID string) (bool, error) {
	session, err := sr.manager.LoadSession(ctx, sessionID)
	if err != nil {
		return false, gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load session")
	}

	// Check if session is in a resumable state
	if session.State.Status != SessionStatusActive && session.State.Status != SessionStatusPaused {
		return false, nil
	}

	// Check if session is not too old (24 hours limit)
	if time.Since(session.LastActiveTime) > 24*time.Hour {
		return false, nil
	}

	return true, nil
}

// GetResumableSessions returns sessions that can be resumed for a user
func (sr *SessionResumer) GetResumableSessions(ctx context.Context, userID string) ([]*Session, error) {
	options := ListOptions{
		OrderBy: "last_active_time DESC",
		Limit:   50,
	}

	sessions, err := sr.manager.ListSessions(ctx, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions")
	}

	var resumable []*Session
	for _, session := range sessions {
		if session.UserID == userID {
			canResume, err := sr.CanResumeSession(ctx, session.ID)
			if err != nil {
				continue // Skip sessions with errors
			}
			if canResume {
				resumable = append(resumable, session)
			}
		}
	}

	return resumable, nil
}

// RecoverFromCrash attempts to recover from a crash
func (sr *SessionResumer) RecoverFromCrash(ctx context.Context) (*Session, error) {
	// Look for sessions that were active recently
	options := ListOptions{
		OrderBy: "last_active_time DESC",
		Limit:   10,
	}

	sessions, err := sr.manager.ListSessions(ctx, options)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list sessions for recovery")
	}

	for _, session := range sessions {
		// Look for recently active sessions (within last hour)
		if time.Since(session.LastActiveTime) < time.Hour {
			if session.State.Status == SessionStatusActive {
				return session, nil
			}
		}
	}

	return nil, gerror.New(gerror.ErrCodeNotFound, "no sessions to recover", nil)
}

// CreateRecoveryPoint creates a recovery point for a session
func (sr *SessionResumer) CreateRecoveryPoint(ctx context.Context, session *Session) error {
	// For now, just save the current session state
	// A full implementation might save to a separate recovery store
	return sr.manager.SaveSession(ctx, session)
}

// getString safely extracts a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// sync.RWMutex import is needed for BackupManager
