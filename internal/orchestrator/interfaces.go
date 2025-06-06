package orchestrator

import (
	"context"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/kanban"
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

