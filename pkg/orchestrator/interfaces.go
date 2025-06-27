// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"time"

	"github.com/lancekrogers/guild/pkg/agent"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/orchestrator/interfaces"
)

// KanbanManager interface for kanban operations needed by orchestrator
type KanbanManager interface {
	ListTasksByStatus(ctx context.Context, boardID string, status kanban.TaskStatus) ([]*kanban.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID, status, assignee, comment string) error
	GetTask(ctx context.Context, taskID string) (*kanban.Task, error)
	CreateTask(ctx context.Context, title, description string) (*kanban.Task, error)
	UpdateTask(ctx context.Context, task *kanban.Task) error
}

// AgentFactory interface for agent creation needed by orchestrator
type AgentFactory interface {
	CreateAgent(agentType, name string, options ...interface{}) (agent.Agent, error)
}

// TaskDispatcher defines the interface for dispatching tasks to agents
type TaskDispatcher interface {
	// RegisterAgent adds an agent to the dispatcher's pool
	RegisterAgent(agent agent.Agent)

	// UnregisterAgent removes an agent from the dispatcher's pool
	UnregisterAgent(agentID string)

	// Dispatch assigns a task to an available agent
	Dispatch(ctx context.Context, task *kanban.Task) error

	// GetTaskStatus returns the current status of a task
	GetTaskStatus(ctx context.Context, taskID string) (TaskStatus, error)

	// GetAgentStatus returns the current status of an agent
	GetAgentStatus(agentID string) AgentStatus

	// ListAvailableAgents returns agents that can accept tasks
	ListAvailableAgents() []agent.Agent

	// Stop gracefully shuts down the dispatcher
	Stop(ctx context.Context) error
}

// TaskStatus represents the execution status of a task
type TaskStatus struct {
	TaskID    string
	AgentID   string
	Status    string
	StartTime time.Time
	Error     error
}

// AgentStatus represents the current state of an agent
type AgentStatus struct {
	AgentID      string
	Available    bool
	CurrentTask  string
	TasksHandled int
}

// EventBus defines the interface for publishing and subscribing to events
type EventBus interface {
	// Subscribe registers a handler for a specific event type
	Subscribe(eventType interfaces.EventType, handler interfaces.EventHandler)

	// SubscribeAll registers a handler for all event types
	SubscribeAll(handler interfaces.EventHandler)

	// Unsubscribe removes a handler for a specific event type
	Unsubscribe(eventType interfaces.EventType, handler interfaces.EventHandler)

	// Publish sends an event to all subscribers
	Publish(event interfaces.Event)

	// PublishJSON publishes an event from a JSON string
	PublishJSON(jsonEvent string) error
}
