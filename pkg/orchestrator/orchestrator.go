// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/commission"
)

// Status represents the status of the orchestrator
type Status string

const (
	// StatusIdle indicates the orchestrator is not running
	StatusIdle Status = "idle"

	// StatusRunning indicates the orchestrator is running
	StatusRunning Status = "running"

	// StatusPaused indicates the orchestrator is paused
	StatusPaused Status = "paused"

	// StatusError indicates the orchestrator encountered an error
	StatusError Status = "error"
)

// Use Event and EventHandler from interfaces package
// These are re-exported in eventbus.go

// Orchestrator coordinates multiple agents to complete commissions
type Orchestrator interface {
	// Start starts the orchestrator
	Start(ctx context.Context) error

	// Stop stops the orchestrator
	Stop(ctx context.Context) error

	// Pause pauses the orchestrator
	Pause(ctx context.Context) error

	// Resume resumes the orchestrator
	Resume(ctx context.Context) error

	// Status returns the current status
	Status() Status

	// AddAgent adds an agent to the orchestrator
	AddAgent(agent agent.Agent) error

	// RemoveAgent removes an agent from the orchestrator
	RemoveAgent(agentID string) error

	// GetAgent gets an agent by ID
	GetAgent(agentID string) (agent.Agent, bool)

	// SetCommission sets the current commission
	SetCommission(commission *commission.Commission) error

	// GetCommission gets the current commission
	GetCommission() *commission.Commission

	// AddEventHandler adds an event handler
	AddEventHandler(handler EventHandler)

	// EmitEvent emits an event
	EmitEvent(event Event)
}

// Config represents the configuration for the orchestrator
type Config struct {
	MaxConcurrentAgents int    `json:"max_concurrent_agents"`
	ManagerAgentID      string `json:"manager_agent_id"`
	KanbanBoardID       string `json:"kanban_board_id"`
	CommissionID        string `json:"commission_id,omitempty"`
	ExecutionMode       string `json:"execution_mode"` // "sequential", "parallel", "managed"
}

// OrchestrationEvent types
const (
	// EventAgentAdded is emitted when an agent is added
	EventAgentAdded = "agent.added"

	// EventAgentRemoved is emitted when an agent is removed
	EventAgentRemoved = "agent.removed"

	// EventAgentStarted is emitted when an agent starts execution
	EventAgentStarted = "agent.started"

	// EventAgentCompleted is emitted when an agent completes execution
	EventAgentCompleted = "agent.completed"

	// EventAgentFailed is emitted when an agent fails
	EventAgentFailed = "agent.failed"

	// EventTaskCreated is emitted when a task is created
	EventTaskCreated = "task.created"

	// EventTaskUpdated is emitted when a task is updated
	EventTaskUpdated = "task.updated"

	// EventTaskAssigned is emitted when a task is assigned
	EventTaskAssigned = "task.assigned"

	// EventTaskCompleted is emitted when a task is completed
	EventTaskCompleted = "task.completed"

	// EventCommissionSet is emitted when a commission is set
	EventCommissionSet = "commission.set"

	// EventCommissionCompleted is emitted when a commission is completed
	EventCommissionCompleted = "commission.completed"

	// EventCommissionStatusChanged is emitted when a commission status changes
	EventCommissionStatusChanged = "commission.status.changed"

	// EventOrchestratorStarted is emitted when the orchestrator starts
	EventOrchestratorStarted = "orchestrator.started"

	// EventOrchestratorStopped is emitted when the orchestrator stops
	EventOrchestratorStopped = "orchestrator.stopped"

	// EventOrchestratorPaused is emitted when the orchestrator pauses
	EventOrchestratorPaused = "orchestrator.paused"

	// EventOrchestratorResumed is emitted when the orchestrator resumes
	EventOrchestratorResumed = "orchestrator.resumed"

	// EventOrchestratorError is emitted when the orchestrator encounters an error
	EventOrchestratorError = "orchestrator.error"
)
