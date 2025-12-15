// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// RecoveryManager handles crash recovery with checkpoint support
type RecoveryManager struct {
	checkpointDir string
	manager       *SessionManager
	checkpoints   map[string]*Checkpoint
	mu            sync.RWMutex
}

// Checkpoint represents a session checkpoint
type Checkpoint struct {
	SessionID      string                 `json:"session_id"`
	Timestamp      time.Time              `json:"timestamp"`
	State          SessionState           `json:"state"`
	LastMessages   []Message              `json:"last_messages"`
	Context        SessionContext         `json:"context"`
	UnsavedChanges []Change               `json:"unsaved_changes,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(manager *SessionManager, checkpointDir string) (*RecoveryManager, error) {
	// Ensure checkpoint directory exists
	if err := os.MkdirAll(checkpointDir, 0o755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create checkpoint directory")
	}

	rm := &RecoveryManager{
		checkpointDir: checkpointDir,
		manager:       manager,
		checkpoints:   make(map[string]*Checkpoint),
	}

	// Load existing checkpoints
	if err := rm.loadCheckpoints(); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to load checkpoints: %v\n", err)
	}

	return rm, nil
}

// CreateCheckpoint creates a checkpoint for a session
func (rm *RecoveryManager) CreateCheckpoint(ctx context.Context, session *Session) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Create checkpoint
	checkpoint := &Checkpoint{
		SessionID: session.ID,
		Timestamp: time.Now(),
		State:     session.State,
		Context:   session.Context,
		Metadata:  session.Metadata,
	}

	// Include last 10 messages for context
	msgCount := len(session.Messages)
	if msgCount > 0 {
		start := msgCount - 10
		if start < 0 {
			start = 0
		}
		checkpoint.LastMessages = session.Messages[start:]
	}

	// Save checkpoint
	if err := rm.saveCheckpoint(checkpoint); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save checkpoint")
	}

	rm.checkpoints[session.ID] = checkpoint
	return nil
}

// RecoverSession attempts to recover a session from checkpoint
func (rm *RecoveryManager) RecoverSession(ctx context.Context, sessionID string) (*Session, error) {
	rm.mu.RLock()
	checkpoint, exists := rm.checkpoints[sessionID]
	rm.mu.RUnlock()

	if !exists {
		// Try to load from disk
		var err error
		checkpoint, err = rm.loadCheckpoint(sessionID)
		if err != nil {
			return nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "no checkpoint found")
		}
	}

	// First try to load the session normally
	session, err := rm.manager.LoadSession(ctx, sessionID)
	if err != nil {
		// If session doesn't exist, create from checkpoint
		session = rm.createSessionFromCheckpoint(checkpoint)
	}

	// Apply any unsaved changes from checkpoint
	if len(checkpoint.UnsavedChanges) > 0 {
		for _, change := range checkpoint.UnsavedChanges {
			if err := rm.applyChange(session, change); err != nil {
				fmt.Printf("Warning: failed to apply change: %v\n", err)
			}
		}
	}

	// Update recovery metadata
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}
	session.Metadata["recovered_from_checkpoint"] = true
	session.Metadata["checkpoint_timestamp"] = checkpoint.Timestamp
	session.Metadata["recovery_timestamp"] = time.Now()

	// Save recovered session
	if err := rm.manager.SaveSession(ctx, session); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save recovered session")
	}

	// Remove checkpoint after successful recovery
	rm.removeCheckpoint(sessionID)

	return session, nil
}

// RecoverAllSessions attempts to recover all sessions with checkpoints
func (rm *RecoveryManager) RecoverAllSessions(ctx context.Context) ([]*Session, []error) {
	rm.mu.RLock()
	sessionIDs := make([]string, 0, len(rm.checkpoints))
	for id := range rm.checkpoints {
		sessionIDs = append(sessionIDs, id)
	}
	rm.mu.RUnlock()

	var recovered []*Session
	var errors []error

	for _, id := range sessionIDs {
		session, err := rm.RecoverSession(ctx, id)
		if err != nil {
			errors = append(errors, gerror.Wrap(err, gerror.ErrCodeInternal,
				fmt.Sprintf("failed to recover session %s", id)))
		} else {
			recovered = append(recovered, session)
		}
	}

	return recovered, errors
}

// AddUnsavedChange adds an unsaved change to the checkpoint
func (rm *RecoveryManager) AddUnsavedChange(sessionID string, change Change) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	checkpoint, exists := rm.checkpoints[sessionID]
	if !exists {
		// Create minimal checkpoint
		checkpoint = &Checkpoint{
			SessionID:      sessionID,
			Timestamp:      time.Now(),
			UnsavedChanges: []Change{},
		}
		rm.checkpoints[sessionID] = checkpoint
	}

	checkpoint.UnsavedChanges = append(checkpoint.UnsavedChanges, change)
	checkpoint.Timestamp = time.Now()

	// Save updated checkpoint
	return rm.saveCheckpoint(checkpoint)
}

// GetCheckpointInfo returns information about existing checkpoints
func (rm *RecoveryManager) GetCheckpointInfo() []CheckpointInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var info []CheckpointInfo
	for id, checkpoint := range rm.checkpoints {
		info = append(info, CheckpointInfo{
			SessionID:         id,
			Timestamp:         checkpoint.Timestamp,
			HasUnsavedChanges: len(checkpoint.UnsavedChanges) > 0,
			ChangeCount:       len(checkpoint.UnsavedChanges),
		})
	}

	return info
}

// CheckpointInfo provides summary information about a checkpoint
type CheckpointInfo struct {
	SessionID         string
	Timestamp         time.Time
	HasUnsavedChanges bool
	ChangeCount       int
}

// CleanOldCheckpoints removes checkpoints older than the specified duration
func (rm *RecoveryManager) CleanOldCheckpoints(maxAge time.Duration) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var toRemove []string

	for id, checkpoint := range rm.checkpoints {
		if checkpoint.Timestamp.Before(cutoff) {
			toRemove = append(toRemove, id)
		}
	}

	for _, id := range toRemove {
		rm.removeCheckpoint(id)
	}

	return nil
}

// Helper methods

func (rm *RecoveryManager) saveCheckpoint(checkpoint *Checkpoint) error {
	filename := filepath.Join(rm.checkpointDir, fmt.Sprintf("checkpoint_%s.json", checkpoint.SessionID))

	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to marshal checkpoint")
	}

	// Write atomically
	tmpFile := filename + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to write checkpoint")
	}

	if err := os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to save checkpoint")
	}

	return nil
}

func (rm *RecoveryManager) loadCheckpoint(sessionID string) (*Checkpoint, error) {
	filename := filepath.Join(rm.checkpointDir, fmt.Sprintf("checkpoint_%s.json", sessionID))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, gerror.New(gerror.ErrCodeNotFound, "checkpoint not found", err)
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read checkpoint")
	}

	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to unmarshal checkpoint")
	}

	return &checkpoint, nil
}

func (rm *RecoveryManager) loadCheckpoints() error {
	entries, err := os.ReadDir(rm.checkpointDir)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to read checkpoint directory")
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Extract session ID from filename
		filename := entry.Name()
		if len(filename) > 21 && filename[:11] == "checkpoint_" {
			sessionID := filename[11 : len(filename)-5]
			checkpoint, err := rm.loadCheckpoint(sessionID)
			if err != nil {
				fmt.Printf("Warning: failed to load checkpoint %s: %v\n", sessionID, err)
				continue
			}
			rm.checkpoints[sessionID] = checkpoint
		}
	}

	return nil
}

func (rm *RecoveryManager) removeCheckpoint(sessionID string) {
	delete(rm.checkpoints, sessionID)

	filename := filepath.Join(rm.checkpointDir, fmt.Sprintf("checkpoint_%s.json", sessionID))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: failed to remove checkpoint file: %v\n", err)
	}
}

func (rm *RecoveryManager) createSessionFromCheckpoint(checkpoint *Checkpoint) *Session {
	session := &Session{
		ID:             checkpoint.SessionID,
		StartTime:      checkpoint.Timestamp,
		LastActiveTime: checkpoint.Timestamp,
		State:          checkpoint.State,
		Context:        checkpoint.Context,
		Messages:       checkpoint.LastMessages,
		Metadata:       checkpoint.Metadata,
	}

	// Ensure required fields are initialized
	if session.State.ActiveAgents == nil {
		session.State.ActiveAgents = make(map[string]AgentState)
	}
	if session.State.Variables == nil {
		session.State.Variables = make(map[string]interface{})
	}
	if session.Metadata == nil {
		session.Metadata = make(map[string]interface{})
	}

	return session
}

func (rm *RecoveryManager) applyChange(session *Session, change Change) error {
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
		if stateData, ok := change.Data.(map[string]interface{}); ok {
			// Merge state data
			if _, exists := stateData["input_buffer"]; exists {
				session.State.InputBuffer = getString(stateData, "input_buffer")
			}
			if scrollPos, exists := stateData["scroll_position"]; exists {
				if pos, ok := scrollPos.(float64); ok {
					session.State.ScrollPosition = int(pos)
				}
			}
		}

	case "context_updated":
		if ctxData, ok := change.Data.(map[string]interface{}); ok {
			if _, exists := ctxData["working_directory"]; exists {
				session.Context.WorkingDirectory = getString(ctxData, "working_directory")
			}
			if _, exists := ctxData["git_branch"]; exists {
				session.Context.GitBranch = getString(ctxData, "git_branch")
			}
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
	}

	return nil
}
