package mocks

import (
	"context"
	"fmt"
	"sync"

	"github.com/guild-ventures/guild-core/internal/kanban"
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

// ListTasksByStatus lists tasks by their status
func (m *MockKanbanManager) ListTasksByStatus(ctx context.Context, boardID string, status kanban.TaskStatus) ([]*kanban.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listError != nil {
		return nil, m.listError
	}

	tasks, ok := m.tasksByStatus[string(status)]
	if !ok {
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
	if oldTasks, ok := m.tasksByStatus[string(task.Status)]; ok {
		var newTasks []*kanban.Task
		for _, t := range oldTasks {
			if t.ID != taskID {
				newTasks = append(newTasks, t)
			}
		}
		m.tasksByStatus[string(task.Status)] = newTasks
	}

	// Update task using the proper method
	task.UpdateStatus(kanban.TaskStatus(status), assignee, comment)

	// Add task to new status list
	if _, ok := m.tasksByStatus[status]; !ok {
		m.tasksByStatus[status] = []*kanban.Task{}
	}
	m.tasksByStatus[status] = append(m.tasksByStatus[status], task)

	return nil
}

// CreateTask creates a new task
func (m *MockKanbanManager) CreateTask(ctx context.Context, title, description string) (*kanban.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createTaskError != nil {
		return nil, m.createTaskError
	}

	// Create new task
	task := &kanban.Task{
		ID:          fmt.Sprintf("TASK-%d", len(m.tasks)+1),
		Title:       title,
		Description: description,
		Status:      kanban.StatusTodo,
		Metadata:    make(map[string]string),
	}

	// Store task
	m.tasks[task.ID] = task

	// Add to status list
	statusKey := string(task.Status)
	if _, ok := m.tasksByStatus[statusKey]; !ok {
		m.tasksByStatus[statusKey] = []*kanban.Task{}
	}
	m.tasksByStatus[statusKey] = append(m.tasksByStatus[statusKey], task)

	return task, nil
}

// GetTask retrieves a task by ID
func (m *MockKanbanManager) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, nil
	}

	// Return a copy to avoid race conditions
	taskCopy := *task
	return &taskCopy, nil
}

// UpdateTask updates an existing task
func (m *MockKanbanManager) UpdateTask(ctx context.Context, task *kanban.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.updateTaskError != nil {
		return m.updateTaskError
	}

	existingTask, exists := m.tasks[task.ID]
	if !exists {
		return nil
	}

	// If status changed, update the status lists
	if existingTask.Status != task.Status {
		// Remove from old status list
		if tasks, ok := m.tasksByStatus[string(existingTask.Status)]; ok {
			var newTasks []*kanban.Task
			for _, t := range tasks {
				if t.ID != task.ID {
					newTasks = append(newTasks, t)
				}
			}
			m.tasksByStatus[string(existingTask.Status)] = newTasks
		}

		// Add to new status list
		statusKey := string(task.Status)
		if _, ok := m.tasksByStatus[statusKey]; !ok {
			m.tasksByStatus[statusKey] = []*kanban.Task{}
		}
		m.tasksByStatus[statusKey] = append(m.tasksByStatus[statusKey], task)
	}

	// Update the task
	m.tasks[task.ID] = task

	return nil
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

	// Remove from status list
	if tasks, ok := m.tasksByStatus[string(task.Status)]; ok {
		var newTasks []*kanban.Task
		for _, t := range tasks {
			if t.ID != taskID {
				newTasks = append(newTasks, t)
			}
		}
		m.tasksByStatus[string(task.Status)] = newTasks
	}

	// Delete the task
	delete(m.tasks, taskID)

	return nil
}

// AddTasks adds multiple tasks to the mock (for testing)
func (m *MockKanbanManager) AddTasks(tasks ...*kanban.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, task := range tasks {
		m.tasks[task.ID] = task

		// Add to status list
		statusKey := string(task.Status)
		if _, ok := m.tasksByStatus[statusKey]; !ok {
			m.tasksByStatus[statusKey] = []*kanban.Task{}
		}
		m.tasksByStatus[statusKey] = append(m.tasksByStatus[statusKey], task)
	}
}

// SetListError sets the error to be returned by List operations
func (m *MockKanbanManager) SetListError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listError = err
}

// SetUpdateError sets the error to be returned by Update operations
func (m *MockKanbanManager) SetUpdateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateError = err
}

// SetCreateTaskError sets the error to be returned by CreateTask
func (m *MockKanbanManager) SetCreateTaskError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createTaskError = err
}

// SetUpdateTaskError sets the error to be returned by UpdateTask
func (m *MockKanbanManager) SetUpdateTaskError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateTaskError = err
}

// SetDeleteTaskError sets the error to be returned by DeleteTask
func (m *MockKanbanManager) SetDeleteTaskError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteTaskError = err
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

// GetBoard retrieves a board by ID
func (m *MockKanbanManager) GetBoard(ctx context.Context, boardID string) (*kanban.Board, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	board, exists := m.boards[boardID]
	if !exists {
		return nil, nil
	}

	// Return a copy
	boardCopy := *board
	return &boardCopy, nil
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

// ListBoards lists all boards
func (m *MockKanbanManager) ListBoards(ctx context.Context) ([]*kanban.Board, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var boards []*kanban.Board
	for _, board := range m.boards {
		boardCopy := *board
		boards = append(boards, &boardCopy)
	}

	return boards, nil
}

// Reset resets the mock
func (m *MockKanbanManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks = make(map[string]*kanban.Task)
	m.tasksByStatus = make(map[string][]*kanban.Task)
	m.boards = make(map[string]*kanban.Board)
	m.listError = nil
	m.updateError = nil
	m.createTaskError = nil
	m.updateTaskError = nil
	m.deleteTaskError = nil
	m.createBoardError = nil
	m.deleteBoardError = nil
}