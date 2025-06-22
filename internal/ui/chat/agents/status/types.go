// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package status

import (
	"time"
)

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	// StatusIdle indicates the agent is available but not working
	StatusIdle AgentStatus = "idle"

	// StatusThinking indicates the agent is processing/planning
	StatusThinking AgentStatus = "thinking"

	// StatusWorking indicates the agent is actively executing tasks
	StatusWorking AgentStatus = "working"

	// StatusError indicates the agent encountered an error
	StatusError AgentStatus = "error"

	// StatusOffline indicates the agent is not available
	StatusOffline AgentStatus = "offline"

	// StatusStarting indicates the agent is initializing
	StatusStarting AgentStatus = "starting"

	// StatusStopping indicates the agent is shutting down
	StatusStopping AgentStatus = "stopping"
)

// AgentInfo represents detailed information about an agent
type AgentInfo struct {
	ID          string
	Name        string
	Type        string // manager, developer, reviewer, etc.
	Status      AgentStatus
	CurrentTask string
	TaskCount   int
	LastSeen    time.Time
	StartTime   time.Time
	ErrorCount  int
	LastError   string
	Metadata    map[string]interface{}
}

// AgentActivity represents a single activity log entry
type AgentActivity struct {
	AgentID   string
	Timestamp time.Time
	Status    AgentStatus
	Message   string
	TaskID    string
}

// StatusUpdate represents a status change event
type StatusUpdate struct {
	AgentID      string
	PreviousStatus AgentStatus
	NewStatus    AgentStatus
	Timestamp    time.Time
	Reason       string
}

// TrackerStats provides statistics about agent activity
type TrackerStats struct {
	TotalAgents      int
	ActiveAgents     int
	IdleAgents       int
	ErrorAgents      int
	OfflineAgents    int
	TotalTasks       int
	CompletedTasks   int
	FailedTasks      int
	AverageTaskTime  time.Duration
	UptimePercentage float64
}

// StatusTracker defines the interface for agent status tracking
type StatusTracker interface {
	// Agent management
	RegisterAgent(info AgentInfo) error
	UnregisterAgent(agentID string) error
	UpdateAgentStatus(agentID string, status AgentStatus, reason string) error
	
	// Agent information retrieval
	GetAgentInfo(agentID string) (*AgentInfo, error)
	GetAllAgents() ([]AgentInfo, error)
	GetAgentsByStatus(status AgentStatus) ([]AgentInfo, error)
	
	// Activity tracking
	LogActivity(activity AgentActivity) error
	GetAgentActivity(agentID string, limit int) ([]AgentActivity, error)
	
	// Statistics
	GetStats() TrackerStats
	GetAgentStats(agentID string) (map[string]interface{}, error)
	
	// Monitoring
	IsAgentActive(agentID string) bool
	GetLastStatusUpdate(agentID string) (*StatusUpdate, error)
	
	// Cleanup
	PurgeInactiveAgents(threshold time.Duration) ([]string, error)
}

// AgentDisplay defines formatting methods for agent status display
type AgentDisplay interface {
	// Format single agent status
	FormatAgentStatus(info AgentInfo) string
	FormatAgentCompact(info AgentInfo) string
	
	// Format multiple agents
	FormatAgentList(agents []AgentInfo) string
	FormatAgentSummary(agents []AgentInfo) string
	
	// Format with indicators
	GetStatusIcon(status AgentStatus) string
	GetStatusColor(status AgentStatus) string
	FormatWithIndicator(info AgentInfo) string
}

// IndicatorType represents different visual indicators
type IndicatorType string

const (
	IndicatorSpinner IndicatorType = "spinner"
	IndicatorPulse   IndicatorType = "pulse"
	IndicatorProgress IndicatorType = "progress"
	IndicatorDots    IndicatorType = "dots"
)

// AgentIndicator represents an animated indicator for agent activity
type AgentIndicator struct {
	Type      IndicatorType
	AgentID   string
	Status    AgentStatus
	Frame     int
	LastUpdate time.Time
}