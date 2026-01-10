// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
)

// MockManagerAgentClient is a mock implementation of ManagerAgentClient for testing
type MockManagerAgentClient struct {
	assignments      map[string]*TaskAssignment
	requestsReceived []AssignmentRequest
	mu               sync.Mutex

	// Control behavior
	shouldFail      bool
	failMessage     string
	assignmentDelay time.Duration
}

// AssignmentRequest tracks requests made to the mock
type AssignmentRequest struct {
	Task            *kanban.Task
	AvailableAgents []*AgentInfo
	Timestamp       time.Time
}

// NewMockManagerAgentClient creates a new mock manager agent client
func NewMockManagerAgentClient() *MockManagerAgentClient {
	return &MockManagerAgentClient{
		assignments:      make(map[string]*TaskAssignment),
		requestsReceived: make([]AssignmentRequest, 0),
	}
}

// RequestAssignment implements ManagerAgentClient interface
func (m *MockManagerAgentClient) RequestAssignment(ctx context.Context, task *kanban.Task, availableAgents []*AgentInfo) (*TaskAssignment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the request
	m.requestsReceived = append(m.requestsReceived, AssignmentRequest{
		Task:            task,
		AvailableAgents: availableAgents,
		Timestamp:       time.Now(),
	})

	// Simulate processing delay
	if m.assignmentDelay > 0 {
		select {
		case <-time.After(m.assignmentDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Check if should fail
	if m.shouldFail {
		return nil, fmt.Errorf("mock manager error: %s", m.failMessage)
	}

	// Check if pre-configured assignment exists
	if assignment, exists := m.assignments[task.ID]; exists {
		return assignment, nil
	}

	// Generate default assignment
	if len(availableAgents) == 0 {
		return nil, fmt.Errorf("no available agents")
	}

	// Pick first available agent
	selectedAgent := availableAgents[0]

	assignment := &TaskAssignment{
		TaskID:      task.ID,
		AgentID:     selectedAgent.AgentID,
		Reasoning:   fmt.Sprintf("Selected agent %s based on availability", selectedAgent.AgentID),
		APIProvider: "openai",
		Priority:    50,
		Deadline:    time.Now().Add(24 * time.Hour),
	}

	m.assignments[task.ID] = assignment
	return assignment, nil
}

// ReviewProgress implements ManagerAgentClient interface
func (m *MockManagerAgentClient) ReviewProgress(ctx context.Context, progress map[string]*CommissionProgress) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return nil, fmt.Errorf("mock manager error: %s", m.failMessage)
	}

	return []string{"Progress reviewed"}, nil
}

// SetAssignment pre-configures an assignment for a specific task
func (m *MockManagerAgentClient) SetAssignment(taskID string, assignment *TaskAssignment) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assignments[taskID] = assignment
}

// SetShouldFail configures the mock to fail
func (m *MockManagerAgentClient) SetShouldFail(shouldFail bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
	m.failMessage = message
}

// GetRequestsReceived returns all assignment requests received
func (m *MockManagerAgentClient) GetRequestsReceived() []AssignmentRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]AssignmentRequest{}, m.requestsReceived...)
}

// MockKanbanClient is a mock implementation of KanbanClient for testing
type MockKanbanClient struct {
	tasks           map[string]*kanban.Task
	taskLocks       map[string]bool
	assignmentCalls []AssignmentCall
	statusUpdates   []StatusUpdate
	mu              sync.Mutex

	// Control behavior
	shouldFailGet    bool
	shouldFailAssign bool
	shouldFailUpdate bool
	simulateConflict bool
}

// AssignmentCall tracks task assignment calls
type AssignmentCall struct {
	TaskID    string
	AgentID   string
	Timestamp time.Time
}

// StatusUpdate tracks status update calls
type StatusUpdate struct {
	TaskID    string
	Status    kanban.TaskStatus
	Timestamp time.Time
}

// NewMockKanbanClient creates a new mock kanban client
func NewMockKanbanClient() *MockKanbanClient {
	return &MockKanbanClient{
		tasks:           make(map[string]*kanban.Task),
		taskLocks:       make(map[string]bool),
		assignmentCalls: make([]AssignmentCall, 0),
		statusUpdates:   make([]StatusUpdate, 0),
	}
}

// GetTaskForUpdate implements KanbanClient interface
func (m *MockKanbanClient) GetTaskForUpdate(ctx context.Context, taskID string) (*kanban.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("mocks.kanban").
			WithOperation("GetTaskForUpdate")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFailGet {
		return nil, gerror.New(gerror.ErrCodeInternal, "mock get failure", nil).
			WithComponent("mocks.kanban").
			WithOperation("GetTaskForUpdate")
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
			WithComponent("mocks.kanban").
			WithOperation("GetTaskForUpdate").
			WithDetails("task_id", taskID)
	}

	// Simulate row lock
	if m.taskLocks[taskID] {
		return nil, gerror.New(gerror.ErrCodeConflict, "task is locked", nil).
			WithComponent("mocks.kanban").
			WithOperation("GetTaskForUpdate").
			WithDetails("task_id", taskID)
	}

	m.taskLocks[taskID] = true

	// Return a copy to prevent external modifications
	taskCopy := *task
	return &taskCopy, nil
}

// AssignTaskAtomic implements KanbanClient interface
func (m *MockKanbanClient) AssignTaskAtomic(ctx context.Context, taskID, agentID string) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("mocks.kanban").
			WithOperation("AssignTaskAtomic")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.assignmentCalls = append(m.assignmentCalls, AssignmentCall{
		TaskID:    taskID,
		AgentID:   agentID,
		Timestamp: time.Now(),
	})

	if m.shouldFailAssign {
		return gerror.New(gerror.ErrCodeInternal, "mock assign failure", nil).
			WithComponent("mocks.kanban").
			WithOperation("AssignTaskAtomic")
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
			WithComponent("mocks.kanban").
			WithOperation("AssignTaskAtomic").
			WithDetails("task_id", taskID)
	}

	// Check if already assigned
	if task.AssignedTo != "" {
		if m.simulateConflict {
			return gerror.New(gerror.ErrCodeConflict, "task already assigned", nil).
				WithComponent("mocks.kanban").
				WithOperation("AssignTaskAtomic").
				WithDetails("task_id", taskID).
				WithDetails("existing_assignee", task.AssignedTo)
		}
	}

	// Update task
	task.AssignedTo = agentID
	task.Status = kanban.StatusInProgress
	task.UpdatedAt = time.Now()

	// Release lock
	delete(m.taskLocks, taskID)

	return nil
}

// UpdateTaskStatusAtomic implements KanbanClient interface
func (m *MockKanbanClient) UpdateTaskStatusAtomic(ctx context.Context, taskID string, status kanban.TaskStatus) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("mocks.kanban").
			WithOperation("UpdateTaskStatusAtomic")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.statusUpdates = append(m.statusUpdates, StatusUpdate{
		TaskID:    taskID,
		Status:    status,
		Timestamp: time.Now(),
	})

	if m.shouldFailUpdate {
		return gerror.New(gerror.ErrCodeInternal, "mock update failure", nil).
			WithComponent("mocks.kanban").
			WithOperation("UpdateTaskStatusAtomic")
	}

	task, exists := m.tasks[taskID]
	if !exists {
		return gerror.New(gerror.ErrCodeNotFound, "task not found", nil).
			WithComponent("mocks.kanban").
			WithOperation("UpdateTaskStatusAtomic").
			WithDetails("task_id", taskID)
	}

	// Update task
	task.Status = status
	task.UpdatedAt = time.Now()

	return nil
}

// WithTransaction implements KanbanClient interface
func (m *MockKanbanClient) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled").
			WithComponent("mocks.kanban").
			WithOperation("WithTransaction")
	}

	// Simulate transaction by running the function
	return fn(ctx)
}

// AddTask adds a task to the mock store
func (m *MockKanbanClient) AddTask(task *kanban.Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[task.ID] = task
}

// GetAssignmentCalls returns all assignment calls made
func (m *MockKanbanClient) GetAssignmentCalls() []AssignmentCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]AssignmentCall{}, m.assignmentCalls...)
}

// GetStatusUpdates returns all status update calls made
func (m *MockKanbanClient) GetStatusUpdates() []StatusUpdate {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]StatusUpdate{}, m.statusUpdates...)
}

// SetSimulateConflict configures conflict simulation
func (m *MockKanbanClient) SetSimulateConflict(simulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateConflict = simulate
}
