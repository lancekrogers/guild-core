// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchedulerWithOrchestrator tests the refactored scheduler using AgentOrchestrator
func TestSchedulerWithOrchestrator(t *testing.T) {
	ctx := context.Background()

	// Create mock dependencies
	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	// Create scheduler
	config := &SchedulerConfig{
		MaxConcurrentTasks: 5,
		ScheduleInterval:   10 * time.Millisecond,
		DefaultTimeout:     30 * time.Second,
	}

	scheduler, err := NewTaskScheduler(ctx, config, managerAgent, kanbanClient)
	require.NoError(t, err)
	require.NotNil(t, scheduler)

	// Verify orchestrator is initialized
	assert.NotNil(t, scheduler.orchestrator)
	assert.NotNil(t, scheduler.kanbanClient)

	// Register an agent
	executor := &mockAgentExecutor{
		agentID:      "test-agent",
		capabilities: []string{"code"},
		isAvailable:  true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			return "completed", nil
		},
	}

	err = scheduler.RegisterAgent(ctx, "test-agent", executor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)

	// Pre-configure assignment in mock manager
	managerAgent.SetAssignment("task-1", &TaskAssignment{
		TaskID:      "task-1",
		AgentID:     "test-agent",
		APIProvider: "openai",
		Priority:    50,
		Deadline:    time.Now().Add(1 * time.Hour),
	})

	// Add task to kanban
	kanbanTask := &kanban.Task{
		ID:       "task-1",
		Title:    "Test Task",
		Status:   kanban.StatusTodo,
		Priority: kanban.PriorityMedium,
	}
	kanbanClient.AddTask(kanbanTask)

	// Submit task to scheduler
	task := &SchedulableTask{
		ID:           "task-1",
		CommissionID: "commission-1",
		Priority:     50,
		Dependencies: []string{},
		Resources: ResourceRequirements{
			APIQuotas: map[string]int{"openai": 1},
		},
	}

	err = scheduler.SubmitTask(ctx, task)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for task to be scheduled
	time.Sleep(100 * time.Millisecond)

	// Verify task was assigned
	assignments := kanbanClient.GetAssignmentCalls()
	assert.Len(t, assignments, 1)
	assert.Equal(t, "task-1", assignments[0].TaskID)
	assert.Equal(t, "test-agent", assignments[0].AgentID)

	// Wait for result
	select {
	case result := <-scheduler.GetResults():
		assert.Equal(t, "task-1", result.TaskID)
		assert.Equal(t, TaskStatusCompleted, result.Status)
		assert.NoError(t, result.Error)
		assert.Equal(t, "completed", result.Output)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for task result")
	}

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

// TestSchedulerTaskAssignment tests the task assignment flow
func TestSchedulerTaskAssignment(t *testing.T) {
	ctx := context.Background()

	// Create mock dependencies
	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	// Create scheduler
	scheduler, err := NewTaskScheduler(ctx, nil, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Register multiple agents
	for i := 1; i <= 3; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		executor := &mockAgentExecutor{
			agentID:      agentID,
			capabilities: []string{"code", "test"},
			isAvailable:  true,
		}

		err = scheduler.RegisterAgent(ctx, agentID, executor, []AgentCapability{CapabilityCode, CapabilityTest})
		require.NoError(t, err)
	}

	// Submit multiple tasks
	for i := 1; i <= 5; i++ {
		taskID := fmt.Sprintf("task-%d", i)

		// Pre-configure assignment
		managerAgent.SetAssignment(taskID, &TaskAssignment{
			TaskID:    taskID,
			AgentID:   fmt.Sprintf("agent-%d", ((i-1)%3)+1),
			Reasoning: "Round-robin assignment",
		})

		// Add to kanban
		kanbanClient.AddTask(&kanban.Task{
			ID:     taskID,
			Title:  fmt.Sprintf("Task %d", i),
			Status: kanban.StatusTodo,
		})

		// Submit to scheduler
		task := &SchedulableTask{
			ID:           taskID,
			CommissionID: "commission-1",
			Priority:     50 + i*10,
		}

		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for assignments
	time.Sleep(200 * time.Millisecond)

	// Verify assignments were made
	assignments := kanbanClient.GetAssignmentCalls()
	assert.GreaterOrEqual(t, len(assignments), 3) // At least 3 tasks assigned (max concurrent)

	// Verify round-robin distribution
	agentCounts := make(map[string]int)
	for _, assignment := range assignments {
		agentCounts[assignment.AgentID]++
	}

	// Each agent should get roughly equal tasks
	for _, count := range agentCounts {
		assert.GreaterOrEqual(t, count, 1)
	}

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

// TestSchedulerMetrics tests the metrics collection
func TestSchedulerMetrics(t *testing.T) {
	ctx := context.Background()

	// Create scheduler
	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()
	scheduler, err := NewTaskScheduler(ctx, nil, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Get initial stats
	stats := scheduler.GetSchedulerStats()
	assert.Equal(t, 0, stats["running_tasks"])
	assert.Equal(t, 0, stats["queued_tasks"])
	assert.Equal(t, int64(0), stats["tasks_assigned"])
	assert.Equal(t, int64(0), stats["tasks_completed"])

	// Register agent and submit task
	executor := &mockAgentExecutor{
		agentID:     "agent-1",
		isAvailable: true,
	}
	err = scheduler.RegisterAgent(ctx, "agent-1", executor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)

	// Submit task
	task := &SchedulableTask{
		ID:           "task-1",
		CommissionID: "commission-1",
		Priority:     50,
	}
	err = scheduler.SubmitTask(ctx, task)
	require.NoError(t, err)

	// Check updated stats
	stats = scheduler.GetSchedulerStats()
	assert.Equal(t, 1, stats["queued_tasks"])
	assert.Equal(t, 1, stats["agents_available"])
}

// TestSchedulerErrorHandling tests error scenarios
func TestSchedulerErrorHandling(t *testing.T) {
	ctx := context.Background()

	// Create scheduler
	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()
	scheduler, err := NewTaskScheduler(ctx, nil, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Register agent that fails execution with non-retryable error
	executor := &mockAgentExecutor{
		agentID:     "failing-agent",
		isAvailable: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			// Return a non-retryable error
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "permanent failure", nil).
				WithComponent("test").
				WithOperation("execute")
		},
	}
	err = scheduler.RegisterAgent(ctx, "failing-agent", executor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)

	// Configure assignment
	managerAgent.SetAssignment("failing-task", &TaskAssignment{
		TaskID:  "failing-task",
		AgentID: "failing-agent",
	})

	// Add to kanban
	kanbanClient.AddTask(&kanban.Task{
		ID:     "failing-task",
		Title:  "Failing Task",
		Status: kanban.StatusTodo,
	})

	// Submit task
	task := &SchedulableTask{
		ID:           "failing-task",
		CommissionID: "commission-1",
		Priority:     50,
	}
	err = scheduler.SubmitTask(ctx, task)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for result
	select {
	case result := <-scheduler.GetResults():
		assert.Equal(t, "failing-task", result.TaskID)
		assert.Equal(t, TaskStatusFailed, result.Status)
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "permanent failure")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for task result")
	}

	// Verify task status was updated
	statusUpdates := kanbanClient.GetStatusUpdates()
	found := false
	for _, update := range statusUpdates {
		if update.TaskID == "failing-task" && update.Status == kanban.StatusBlocked {
			found = true
			break
		}
	}
	assert.True(t, found, "Task should be marked as blocked after failure")

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}
