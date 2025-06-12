package interfaces

import (
	"time"
)

// EventType represents the type of event
type EventType string

const (
	// EventTypeTaskCreated is emitted when a new task is created
	EventTypeTaskCreated EventType = "task_created"

	// EventTypeTaskAssigned is emitted when a task is assigned to an agent
	EventTypeTaskAssigned EventType = "task_assigned"

	// EventTypeTaskStarted is emitted when a task is started by an agent
	EventTypeTaskStarted EventType = "task_started"

	// EventTypeTaskCompleted is emitted when a task is completed
	EventTypeTaskCompleted EventType = "task_completed"

	// EventTypeTaskFailed is emitted when a task fails
	EventTypeTaskFailed EventType = "task_failed"

	// EventTypeAgentCreated is emitted when a new agent is created
	EventTypeAgentCreated EventType = "agent_created"

	// EventTypeAgentStarted is emitted when an agent starts
	EventTypeAgentStarted EventType = "agent_started"

	// EventTypeAgentStopped is emitted when an agent stops
	EventTypeAgentStopped EventType = "agent_stopped"

	// EventTypeCommissionCreated is emitted when a new commission is created
	EventTypeCommissionCreated EventType = "commission_created"

	// EventTypeCommissionUpdated is emitted when a commission is updated
	EventTypeCommissionUpdated EventType = "commission_updated"

	// EventTypeCommissionCompleted is emitted when a commission is completed
	EventTypeCommissionCompleted EventType = "commission_completed"

	// EventTypeCommissionStatusChanged is emitted when a commission status changes
	EventTypeCommissionStatusChanged EventType = "commission_status_changed"
)

// Event represents an event in the system
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
}

// EventHandler is a function that handles events
type EventHandler func(event Event)
