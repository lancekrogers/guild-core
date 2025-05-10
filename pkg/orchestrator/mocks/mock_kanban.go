package mocks

import (
	"context"
	"sync"

	"github.com/blockhead-consulting/guild/pkg/kanban"
)

// MockKanbanManager is a mock implementation of the kanban.Manager
type MockKanbanManager struct {
	tasks            map[string]*kanban.Task
	tasksByStatus    map[string][]*kanban.Task
	boards           map[string]*kanban.Board
	mu               sync.Mutex
	listError        error
	updateError      error
	createTaskError  error
	updateTaskError  error
	deleteTaskError  error
	createBoardError error
	deleteBoardError error
}

// NewMockKanbanManager creates a new mock kanban manager
func NewMockKanbanManager() *MockKanbanManager {
	return &MockKanbanManager{
		tasks:         make(map[string]*kanban.Task),
		tasksByStatus: make(map[string][]*kanban.Task),
		boards:        make(map[string]*kanban.Board),
	}
}

// ListTasksByStatus lists tasks by status
func (m *MockKanbanManager) ListTasksByStatus(ctx context.Context, status string) ([]*kanban.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listError != nil {
		return nil, m.listError
	}

	tasks, exists := m.tasksByStatus[status]
	if !exists {
		return []*kanban.Task{}, nil
	}

	// Create a copy to avoid race conditions
	result := make([]*kanban.Task, len(tasks))
	copy(result, tasks)

	return result, nil
}

// UpdateTaskStatus updates a task's status
func (m *MockKanbanManager) UpdateTaskStatus(ctx context.Context, taskID, status, assignee, comment string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateTaskError != nil {
		return m.updateTaskError
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil // Simulating that the task was updated even if it doesn't exist
	}

	// Remove task from old status list
	if oldTasks, ok := m.tasksByStatus[task.Status]; ok {
		var newTasks []*kanban.Task
		for _, t := range oldTasks {
			if t.ID != taskID {
				newTasks = append(newTasks, t)
			}
		}
		m.tasksByStatus[task.Status] = newTasks
	}

	// Update task
	oldStatus := task.Status
	task.Status = status
	task.Assignee = assignee

	// Add comment if provided
	if comment != "" {
		task.Comments = append(task.Comments, &kanban.Comment{
			Author:  assignee,
			Content: comment,
		})
	}

	// Add task to new status list
	if _, ok := m.tasksByStatus[status]; !ok {
		m.tasksByStatus[status] = []*kanban.Task{}
	}
	m.tasksByStatus[status] = append(m.tasksByStatus[status], task)

	// Add status transition to history
	task.History = append(task.History, &kanban.TaskHistory{
		FromStatus: oldStatus,
		ToStatus:   status,
		Assignee:   assignee,
		Comment:    comment,
	})

	return nil
}

// CreateTask creates a new task
func (m *MockKanbanManager) CreateTask(ctx context.Context, boardID string, task *kanban.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createTaskError != nil {
		return m.createTaskError
	}

	// Store task
	m.tasks[task.ID] = task

	// Add task to status list
	if _, ok := m.tasksByStatus[task.Status]; !ok {
		m.tasksByStatus[task.Status] = []*kanban.Task{}
	}
	m.tasksByStatus[task.Status] = append(m.tasksByStatus[task.Status], task)

	return nil
}

// GetTask gets a task by ID
func (m *MockKanbanManager) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, nil
	}

	return task, nil
}

// DeleteTask deletes a task
func (m *MockKanbanManager) DeleteTask(ctx context.Context, taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteTaskError != nil {
		return m.deleteTaskError
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil
	}

	// Remove task from status list
	if tasks, ok := m.tasksByStatus[task.Status]; ok {
		var newTasks []*kanban.Task
		for _, t := range tasks {
			if t.ID != taskID {
				newTasks = append(newTasks, t)
			}
		}
		m.tasksByStatus[task.Status] = newTasks
	}

	// Remove task
	delete(m.tasks, taskID)

	return nil
}

// CreateBoard creates a new board
func (m *MockKanbanManager) CreateBoard(ctx context.Context, board *kanban.Board) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createBoardError != nil {
		return m.createBoardError
	}

	m.boards[board.ID] = board

	return nil
}

// DeleteBoard deletes a board
func (m *MockKanbanManager) DeleteBoard(ctx context.Context, boardID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.deleteBoardError != nil {
		return m.deleteBoardError
	}

	delete(m.boards, boardID)

	return nil
}

// AddTasks adds tasks to the mock
func (m *MockKanbanManager) AddTasks(tasks ...*kanban.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range tasks {
		m.tasks[task.ID] = task

		// Add task to status list
		if _, ok := m.tasksByStatus[task.Status]; !ok {
			m.tasksByStatus[task.Status] = []*kanban.Task{}
		}
		m.tasksByStatus[task.Status] = append(m.tasksByStatus[task.Status], task)
	}
}

// SetListError sets the error to be returned by ListTasksByStatus
func (m *MockKanbanManager) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.listError = err
}

// SetUpdateTaskError sets the error to be returned by UpdateTaskStatus
func (m *MockKanbanManager) SetUpdateTaskError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.updateTaskError = err
}

// SetCreateTaskError sets the error to be returned by CreateTask
func (m *MockKanbanManager) SetCreateTaskError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.createTaskError = err
}

// SetDeleteTaskError sets the error to be returned by DeleteTask
func (m *MockKanbanManager) SetDeleteTaskError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deleteTaskError = err
}

// Reset resets the mock
func (m *MockKanbanManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = make(map[string]*kanban.Task)
	m.tasksByStatus = make(map[string][]*kanban.Task)
	m.boards = make(map[string]*kanban.Board)
	m.listError = nil
	m.updateTaskError = nil
	m.createTaskError = nil
	m.deleteTaskError = nil
	m.createBoardError = nil
	m.deleteBoardError = nil
}

