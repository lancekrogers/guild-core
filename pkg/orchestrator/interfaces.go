package orchestrator

import (
	"context"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/orchestrator/interfaces"
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

