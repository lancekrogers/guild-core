package orchestrator

import (
	"context"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/internal/kanban"
)

// defaultKanbanManager implements KanbanManager using the kanban.Board
type defaultKanbanManager struct {
	board *kanban.Board
}

// newDefaultKanbanManager creates a new default kanban manager (private constructor)
func newDefaultKanbanManager(board *kanban.Board) *defaultKanbanManager {
	return &defaultKanbanManager{
		board: board,
	}
}

// DefaultKanbanManagerFactory creates a kanban manager for registry use
func DefaultKanbanManagerFactory(board *kanban.Board) KanbanManager {
	return newDefaultKanbanManager(board)
}

// CreateTask creates a new task on the kanban board
func (m *defaultKanbanManager) CreateTask(ctx context.Context, title, description string) (*kanban.Task, error) {
	return m.board.CreateTask(ctx, title, description)
}

// UpdateTask updates an existing task
func (m *defaultKanbanManager) UpdateTask(ctx context.Context, task *kanban.Task) error {
	return m.board.UpdateTask(ctx, task)
}

// GetTask retrieves a task by ID
func (m *defaultKanbanManager) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	return m.board.GetTask(ctx, taskID)
}


// ListTasksByStatus retrieves tasks by status (implements KanbanManager interface)
func (m *defaultKanbanManager) ListTasksByStatus(ctx context.Context, boardID string, status kanban.TaskStatus) ([]*kanban.Task, error) {
	// Note: boardID is ignored as we have a single board
	allTasks, err := m.board.GetAllTasks(ctx)
	if err != nil {
		return nil, err
	}

	var filteredTasks []*kanban.Task
	for _, task := range allTasks {
		if task.Status == status {
			filteredTasks = append(filteredTasks, task)
		}
	}

	return filteredTasks, nil
}

// UpdateTaskStatus updates a task's status (implements KanbanManager interface)
func (m *defaultKanbanManager) UpdateTaskStatus(ctx context.Context, taskID, status, assignee, comment string) error {
	task, err := m.board.GetTask(ctx, taskID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeOrchestration, "failed to get task").
			WithComponent("orchestrator").
			WithOperation("UpdateTaskStatus")
	}

	// Parse status string to TaskStatus
	var taskStatus kanban.TaskStatus
	switch status {
	case "todo":
		taskStatus = kanban.StatusTodo
	case "in_progress":
		taskStatus = kanban.StatusInProgress
	case "review", "ready_for_review":
		taskStatus = kanban.StatusReadyForReview
	case "done":
		taskStatus = kanban.StatusDone
	case "blocked":
		taskStatus = kanban.StatusBlocked
	default:
		return gerror.New(gerror.ErrCodeValidation, "invalid status", nil).
			WithComponent("orchestrator").
			WithOperation("UpdateTaskStatus").
			WithDetails("status", status)
	}

	task.Status = taskStatus
	if assignee != "" {
		task.AssignedTo = assignee
	}
	if comment != "" {
		task.AddNote(comment, assignee)
	}

	return m.board.UpdateTask(ctx, task)
}

// GetAllTasks retrieves all tasks (helper method)
func (m *defaultKanbanManager) GetAllTasks(ctx context.Context) ([]*kanban.Task, error) {
	return m.board.GetAllTasks(ctx)
}

// GetBoard returns the underlying kanban board
func (m *defaultKanbanManager) GetBoard() *kanban.Board {
	return m.board
}