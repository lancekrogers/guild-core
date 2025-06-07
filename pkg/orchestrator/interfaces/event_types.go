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

	// EventTypeObjectiveCreated is emitted when a new objective is created
	EventTypeObjectiveCreated EventType = "objective_created"

	// EventTypeObjectiveUpdated is emitted when an objective is updated
	EventTypeObjectiveUpdated EventType = "objective_updated"

	// EventTypeObjectiveCompleted is emitted when an objective is completed
	EventTypeObjectiveCompleted EventType = "objective_completed"

	// EventTypeObjectiveStatusChanged is emitted when an objective status changes
	EventTypeObjectiveStatusChanged EventType = "objective_status_changed"
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
