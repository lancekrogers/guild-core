package executor

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// mockAgent is a simple mock implementation for testing
type mockAgent struct {
	id   string
	name string
}

func (m *mockAgent) GetID() string   { return m.id }
func (m *mockAgent) GetName() string { return m.name }
func (m *mockAgent) Execute(ctx context.Context, request string) (string, error) {
	// Mock execution
	return "mock result", nil
}

// mockKanbanBoard is a simple mock for testing
type mockKanbanBoard struct {
	tasks map[string]*kanban.Task
}

func newMockKanbanBoard() *mockKanbanBoard {
	return &mockKanbanBoard{
		tasks: make(map[string]*kanban.Task),
	}
}

func (m *mockKanbanBoard) UpdateTaskStatus(ctx context.Context, taskID string, status kanban.TaskStatus, assignee, comment string) error {
	if task, exists := m.tasks[taskID]; exists {
		task.Status = status
		task.AssignedTo = assignee
	}
	return nil
}

func (m *mockKanbanBoard) GetTask(ctx context.Context, taskID string) (*kanban.Task, error) {
	return m.tasks[taskID], nil
}

func TestBasicTaskExecutor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		task           *kanban.Task
		expectedStatus ExecutionStatus
		expectedError  bool
		contextTimeout time.Duration
	}{
		{
			name: "successful execution",
			task: &kanban.Task{
				ID:          "test-task-1",
				Title:       "Test Task",
				Description: "A test task for execution",
				Status:      kanban.StatusTodo,
			},
			expectedStatus: StatusCompleted,
			expectedError:  false,
		},
		{
			name: "execution with context cancellation",
			task: &kanban.Task{
				ID:          "test-task-2",
				Title:       "Cancelled Task",
				Description: "A task that gets cancelled",
				Status:      kanban.StatusTodo,
			},
			expectedStatus: StatusFailed,
			expectedError:  true,
			contextTimeout: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock dependencies
			mockAgent := &mockAgent{id: "test-agent", name: "Test Agent"}
			mockBoard := newMockKanbanBoard()
			mockBoard.tasks[tt.task.ID] = tt.task

			// Create executor
			execContext := &ExecutionContext{
				WorkspaceDir: "/tmp/test-workspace",
				ProjectRoot:  "/tmp/test-project",
				AgentID:      "test-agent",
				AgentType:    "worker",
				Capabilities: []string{"testing"},
				Tools:        []string{"mock-tool"},
			}

			// Create executor (nil workspace manager for tests)
			executor, err := NewBasicTaskExecutor(mockAgent, nil, nil, execContext, nil)
			assert.NoError(t, err)

			// Create context
			ctx := context.Background()
			if tt.contextTimeout > 0 {
				cancelCtx, cancel := context.WithTimeout(ctx, tt.contextTimeout)
				defer cancel()
				ctx = cancelCtx
			}

			// Execute task
			result, err := executor.Execute(ctx, tt.task)

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if result != nil {
				assert.Equal(t, tt.expectedStatus, result.Status)
				assert.Equal(t, tt.task.ID, result.TaskID)
				assert.NotZero(t, result.Duration)
			}

			// Verify final status
			assert.Equal(t, tt.expectedStatus, executor.GetStatus())
		})
	}
}

func TestBasicTaskExecutor_Progress(t *testing.T) {
	// Create minimal test setup
	execContext := &ExecutionContext{
		AgentID: "test-agent",
	}
	mockAgent := &mockAgent{id: "test-agent", name: "Test Agent"}

	executor, err := NewBasicTaskExecutor(mockAgent, nil, nil, execContext, nil)
	assert.NoError(t, err)

	// Initial progress should be 0
	assert.Equal(t, 0.0, executor.GetProgress())

	// Update progress
	executor.updateProgress(0.5, "test phase")
	assert.Equal(t, 0.5, executor.GetProgress())

	// Update to 100%
	executor.updateProgress(1.0, "complete")
	assert.Equal(t, 1.0, executor.GetProgress())
}

func TestBasicTaskExecutor_Stop(t *testing.T) {
	// Create minimal test setup
	execContext := &ExecutionContext{
		AgentID: "test-agent",
	}
	mockAgent := &mockAgent{id: "test-agent", name: "Test Agent"}

	executor, err := NewBasicTaskExecutor(mockAgent, nil, nil, execContext, nil)
	assert.NoError(t, err)

	// Initial status
	executor.status = StatusRunning

	// Stop should work without error
	stopErr := executor.Stop()
	assert.NoError(t, stopErr)

	// Status should be stopped
	assert.Equal(t, StatusStopped, executor.GetStatus())

	// Multiple stops should be idempotent
	stopErr2 := executor.Stop()
	assert.NoError(t, stopErr2)
}

func TestBasicTaskExecutor_StateTransitions(t *testing.T) {
	mockAgent := &mockAgent{id: "test-agent", name: "Test Agent"}
	execContext := &ExecutionContext{AgentID: "test-agent"}
	executor, err := NewBasicTaskExecutor(mockAgent, nil, nil, execContext, nil)
	assert.NoError(t, err)

	// Test state transitions - executor starts in initializing state
	assert.Equal(t, StatusInitializing, executor.GetStatus())

	executor.status = StatusInitializing
	assert.Equal(t, StatusInitializing, executor.GetStatus())

	executor.status = StatusRunning
	assert.Equal(t, StatusRunning, executor.GetStatus())

	executor.status = StatusCompleted
	assert.Equal(t, StatusCompleted, executor.GetStatus())
}

func TestExecutionResult_Duration(t *testing.T) {
	start := time.Now()
	end := start.Add(5 * time.Second)

	result := &ExecutionResult{
		StartTime: start,
		EndTime:   end,
	}

	result.Duration = result.EndTime.Sub(result.StartTime)
	// Use InDelta for time comparison to handle minor variations
	assert.InDelta(t, float64(5*time.Second), float64(result.Duration), float64(time.Millisecond))
}

func TestExecutionContext_Fields(t *testing.T) {
	ctx := &ExecutionContext{
		WorkspaceDir: "/workspace",
		ProjectRoot:  "/project",
		Objective:    "Test objective",
		AgentID:      "agent-123",
		AgentType:    "worker",
		Capabilities: []string{"coding", "testing"},
		Tools:        []string{"file_system", "shell"},
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	assert.Equal(t, "/workspace", ctx.WorkspaceDir)
	assert.Equal(t, "agent-123", ctx.AgentID)
	assert.Contains(t, ctx.Capabilities, "coding")
	assert.Contains(t, ctx.Tools, "shell")
	assert.Equal(t, "value", ctx.Metadata["key"])
}
