// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/gerror"
)

// agentStatusTracker implements the StatusTracker interface
type agentStatusTracker struct {
	mu            sync.RWMutex
	agents        map[string]*AgentInfo
	activities    map[string][]AgentActivity
	statusUpdates map[string]*StatusUpdate
	stats         TrackerStats
	maxActivities int
	ctx           context.Context
}

// NewStatusTracker creates a new agent status tracker
func NewStatusTracker(ctx context.Context) StatusTracker {
	if ctx == nil {
		ctx = context.Background()
	}

	return &agentStatusTracker{
		agents:        make(map[string]*AgentInfo),
		activities:    make(map[string][]AgentActivity),
		statusUpdates: make(map[string]*StatusUpdate),
		maxActivities: 100, // Keep last 100 activities per agent
		ctx:           ctx,
	}
}

// RegisterAgent registers a new agent in the tracker
func (t *agentStatusTracker) RegisterAgent(info AgentInfo) error {
	if err := t.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentStatusTracker").
			WithOperation("RegisterAgent")
	}

	if info.ID == "" {
		return gerror.New(gerror.ErrCodeInvalidInput, "agent ID cannot be empty", nil).
			WithComponent("AgentStatusTracker").
			WithOperation("RegisterAgent")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Initialize timestamps if not set
	now := time.Now()
	if info.StartTime.IsZero() {
		info.StartTime = now
	}
	if info.LastSeen.IsZero() {
		info.LastSeen = now
	}

	// Initialize metadata if nil
	if info.Metadata == nil {
		info.Metadata = make(map[string]interface{})
	}

	t.agents[info.ID] = &info
	t.updateStatsLocked()

	// Log registration activity
	activity := AgentActivity{
		AgentID:   info.ID,
		Timestamp: now,
		Status:    info.Status,
		Message:   fmt.Sprintf("Agent %s registered", info.Name),
	}
	t.logActivityLocked(activity)

	return nil
}

// UnregisterAgent removes an agent from the tracker
func (t *agentStatusTracker) UnregisterAgent(agentID string) error {
	if err := t.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentStatusTracker").
			WithOperation("UnregisterAgent")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	agent, exists := t.agents[agentID]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("AgentStatusTracker").
			WithOperation("UnregisterAgent")
	}

	// Log unregistration activity
	activity := AgentActivity{
		AgentID:   agentID,
		Timestamp: time.Now(),
		Status:    StatusOffline,
		Message:   fmt.Sprintf("Agent %s unregistered", agent.Name),
	}
	t.logActivityLocked(activity)

	delete(t.agents, agentID)
	delete(t.activities, agentID)
	delete(t.statusUpdates, agentID)
	t.updateStatsLocked()

	return nil
}

// UpdateAgentStatus updates the status of an agent
func (t *agentStatusTracker) UpdateAgentStatus(agentID string, status AgentStatus, reason string) error {
	if err := t.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentStatusTracker").
			WithOperation("UpdateAgentStatus")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	agent, exists := t.agents[agentID]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("AgentStatusTracker").
			WithOperation("UpdateAgentStatus")
	}

	previousStatus := agent.Status
	now := time.Now()

	// Update agent info
	agent.Status = status
	agent.LastSeen = now

	// Track errors
	if status == StatusError {
		agent.ErrorCount++
		if reason != "" {
			agent.LastError = reason
		}
	}

	// Record status update
	update := &StatusUpdate{
		AgentID:        agentID,
		PreviousStatus: previousStatus,
		NewStatus:      status,
		Timestamp:      now,
		Reason:         reason,
	}
	t.statusUpdates[agentID] = update

	// Log activity
	activity := AgentActivity{
		AgentID:   agentID,
		Timestamp: now,
		Status:    status,
		Message:   fmt.Sprintf("Status changed from %s to %s: %s", previousStatus, status, reason),
	}
	t.logActivityLocked(activity)

	t.updateStatsLocked()
	return nil
}

// GetAgentInfo retrieves information about a specific agent
func (t *agentStatusTracker) GetAgentInfo(agentID string) (*AgentInfo, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	agent, exists := t.agents[agentID]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("AgentStatusTracker").
			WithOperation("GetAgentInfo")
	}

	// Return a copy to prevent external modification
	info := *agent
	return &info, nil
}

// GetAllAgents returns information about all agents
func (t *agentStatusTracker) GetAllAgents() ([]AgentInfo, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	agents := make([]AgentInfo, 0, len(t.agents))
	for _, agent := range t.agents {
		agents = append(agents, *agent)
	}

	return agents, nil
}

// GetAgentsByStatus returns agents with a specific status
func (t *agentStatusTracker) GetAgentsByStatus(status AgentStatus) ([]AgentInfo, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var agents []AgentInfo
	for _, agent := range t.agents {
		if agent.Status == status {
			agents = append(agents, *agent)
		}
	}

	return agents, nil
}

// LogActivity logs an activity for an agent
func (t *agentStatusTracker) LogActivity(activity AgentActivity) error {
	if err := t.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentStatusTracker").
			WithOperation("LogActivity")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	return t.logActivityLocked(activity)
}

// logActivityLocked logs an activity (must be called with lock held)
func (t *agentStatusTracker) logActivityLocked(activity AgentActivity) error {
	if activity.Timestamp.IsZero() {
		activity.Timestamp = time.Now()
	}

	activities := t.activities[activity.AgentID]
	activities = append(activities, activity)

	// Trim to max activities
	if len(activities) > t.maxActivities {
		activities = activities[len(activities)-t.maxActivities:]
	}

	t.activities[activity.AgentID] = activities
	return nil
}

// GetAgentActivity retrieves recent activities for an agent
func (t *agentStatusTracker) GetAgentActivity(agentID string, limit int) ([]AgentActivity, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activities, exists := t.activities[agentID]
	if !exists {
		return []AgentActivity{}, nil
	}

	// Return most recent activities up to limit
	if limit <= 0 || limit > len(activities) {
		limit = len(activities)
	}

	start := len(activities) - limit
	if start < 0 {
		start = 0
	}

	// Return a copy to prevent external modification
	result := make([]AgentActivity, limit)
	copy(result, activities[start:])
	return result, nil
}

// GetStats returns tracker statistics
func (t *agentStatusTracker) GetStats() TrackerStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.stats
}

// GetAgentStats returns statistics for a specific agent
func (t *agentStatusTracker) GetAgentStats(agentID string) (map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	agent, exists := t.agents[agentID]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("AgentStatusTracker").
			WithOperation("GetAgentStats")
	}

	stats := make(map[string]interface{})
	stats["id"] = agent.ID
	stats["name"] = agent.Name
	stats["type"] = agent.Type
	stats["status"] = agent.Status
	stats["task_count"] = agent.TaskCount
	stats["error_count"] = agent.ErrorCount
	stats["uptime"] = time.Since(agent.StartTime)
	stats["last_seen"] = agent.LastSeen
	stats["idle_time"] = time.Since(agent.LastSeen)

	// Activity stats
	activities := t.activities[agentID]
	stats["activity_count"] = len(activities)

	return stats, nil
}

// IsAgentActive checks if an agent is active
func (t *agentStatusTracker) IsAgentActive(agentID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	agent, exists := t.agents[agentID]
	if !exists {
		return false
	}

	return agent.Status != StatusOffline && agent.Status != StatusError
}

// GetLastStatusUpdate retrieves the last status update for an agent
func (t *agentStatusTracker) GetLastStatusUpdate(agentID string) (*StatusUpdate, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	update, exists := t.statusUpdates[agentID]
	if !exists {
		return nil, gerror.Newf(gerror.ErrCodeNotFound, "no status updates for agent %s", agentID).
			WithComponent("AgentStatusTracker").
			WithOperation("GetLastStatusUpdate")
	}

	// Return a copy
	updateCopy := *update
	return &updateCopy, nil
}

// UpdateProcessingState updates the processing state of an agent
func (t *agentStatusTracker) UpdateProcessingState(agentID string, state ProcessingState) error {
	if err := t.ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentStatusTracker").
			WithOperation("UpdateProcessingState")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	agent, exists := t.agents[agentID]
	if !exists {
		return gerror.Newf(gerror.ErrCodeNotFound, "agent %s not found", agentID).
			WithComponent("AgentStatusTracker").
			WithOperation("UpdateProcessingState")
	}

	// Update processing state
	previousState := agent.ProcessingState
	agent.ProcessingState = state
	agent.LastSeen = time.Now()

	// If moving from idle to active processing, record start time
	if previousState == ProcessingIdle && state != ProcessingIdle {
		agent.ProcessingStart = time.Now()
	}

	// Log activity for state change
	return t.logActivityLocked(AgentActivity{
		AgentID:   agentID,
		Timestamp: time.Now(),
		Status:    agent.Status,
		Message:   fmt.Sprintf("Processing state changed from %s to %s", previousState, state),
		TaskID:    agent.CurrentTask,
	})
}

// PurgeInactiveAgents removes agents that haven't been seen within the threshold
func (t *agentStatusTracker) PurgeInactiveAgents(threshold time.Duration) ([]string, error) {
	if err := t.ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("AgentStatusTracker").
			WithOperation("PurgeInactiveAgents")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	cutoff := time.Now().Add(-threshold)
	var purged []string

	for id, agent := range t.agents {
		if agent.LastSeen.Before(cutoff) {
			purged = append(purged, id)
			delete(t.agents, id)
			delete(t.activities, id)
			delete(t.statusUpdates, id)
		}
	}

	if len(purged) > 0 {
		t.updateStatsLocked()
	}

	return purged, nil
}

// updateStatsLocked updates the tracker statistics (must be called with lock held)
func (t *agentStatusTracker) updateStatsLocked() {
	t.stats.TotalAgents = len(t.agents)
	t.stats.ActiveAgents = 0
	t.stats.IdleAgents = 0
	t.stats.ErrorAgents = 0
	t.stats.OfflineAgents = 0

	for _, agent := range t.agents {
		switch agent.Status {
		case StatusWorking, StatusThinking:
			t.stats.ActiveAgents++
		case StatusIdle:
			t.stats.IdleAgents++
		case StatusError:
			t.stats.ErrorAgents++
		case StatusOffline:
			t.stats.OfflineAgents++
		}
	}

	// Calculate uptime percentage
	if t.stats.TotalAgents > 0 {
		t.stats.UptimePercentage = float64(t.stats.TotalAgents-t.stats.OfflineAgents) / float64(t.stats.TotalAgents) * 100
	}
}
