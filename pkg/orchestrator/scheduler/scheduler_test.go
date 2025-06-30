// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor implements interfaces.AgentExecutor for testing
type mockExecutor struct {
	id          string
	available   bool
	executeFunc func(ctx context.Context, taskID string, payload interface{}) (interface{}, error)
	mu          sync.Mutex
}

func (m *mockExecutor) Execute(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
	m.mu.Lock()
	m.available = false
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.available = true
		m.mu.Unlock()
	}()

	if m.executeFunc != nil {
		return m.executeFunc(ctx, taskID, payload)
	}

	// Simulate work
	select {
	case <-time.After(100 * time.Millisecond):
		return "completed", nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockExecutor) GetAgentID() string {
	return m.id
}

func (m *mockExecutor) GetCapabilities() []string {
	return []string{"test"}
}

func (m *mockExecutor) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

func TestTaskScheduler_Creation(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	
	scheduler, err := NewTaskScheduler(ctx, config)
	require.NoError(t, err)
	assert.NotNil(t, scheduler)
	
	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()
	
	_, err = NewTaskScheduler(cancelledCtx, config)
	assert.Error(t, err)
}

func TestTaskScheduler_RegisterExecutor(t *testing.T) {
	ctx := context.Background()
	scheduler, err := NewTaskScheduler(ctx, DefaultSchedulerConfig())
	require.NoError(t, err)
	
	executor := &mockExecutor{id: "agent-1", available: true}
	
	// Register executor
	err = scheduler.RegisterExecutor("agent-1", executor)
	assert.NoError(t, err)
	
	// Test nil executor
	err = scheduler.RegisterExecutor("agent-2", nil)
	assert.Error(t, err)
}

func TestTaskScheduler_SubmitTask(t *testing.T) {
	ctx := context.Background()
	scheduler, err := NewTaskScheduler(ctx, DefaultSchedulerConfig())
	require.NoError(t, err)
	
	task := &SchedulableTask{
		ID:           "task-1",
		CommissionID: "commission-1",
		Priority:     10,
		Dependencies: []string{},
		Resources: ResourceRequirements{
			CPUCores: 1.0,
			MemoryMB: 512,
		},
		Agent: "agent-1",
	}
	
	// Submit task
	err = scheduler.SubmitTask(ctx, task)
	assert.NoError(t, err)
	
	// Verify task is queued
	queued := scheduler.GetQueuedTasks()
	assert.Len(t, queued, 1)
	assert.Equal(t, "task-1", queued[0].ID)
	
	// Test nil task
	err = scheduler.SubmitTask(ctx, nil)
	assert.Error(t, err)
}

func TestTaskScheduler_SimpleExecution(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.ScheduleInterval = 10 * time.Millisecond
	
	scheduler, err := NewTaskScheduler(ctx, config)
	require.NoError(t, err)
	
	// Register executor
	executor := &mockExecutor{
		id:        "agent-1",
		available: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			return "task-result", nil
		},
	}
	
	err = scheduler.RegisterExecutor("agent-1", executor)
	require.NoError(t, err)
	
	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	
	// Submit task
	task := &SchedulableTask{
		ID:           "task-1",
		Priority:     10,
		Dependencies: []string{},
		Resources: ResourceRequirements{
			CPUCores: 1.0,
			MemoryMB: 512,
		},
		Agent: "agent-1",
	}
	
	err = scheduler.SubmitTask(ctx, task)
	require.NoError(t, err)
	
	// Wait for task completion
	var result TaskResult
	select {
	case result = <-scheduler.GetResults():
		assert.Equal(t, "task-1", result.TaskID)
		assert.Equal(t, TaskStatusCompleted, result.Status)
		assert.Equal(t, "task-result", result.Output)
		assert.NoError(t, result.Error)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for task result")
	}
	
	// Verify task status
	status, err := scheduler.GetTaskStatus("task-1")
	assert.NoError(t, err)
	assert.Equal(t, TaskStatusCompleted, status)
	
	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestTaskScheduler_Dependencies(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.ScheduleInterval = 10 * time.Millisecond
	
	scheduler, err := NewTaskScheduler(ctx, config)
	require.NoError(t, err)
	
	// Register executors
	for i := 1; i <= 3; i++ {
		executor := &mockExecutor{
			id:        fmt.Sprintf("agent-%d", i),
			available: true,
		}
		err = scheduler.RegisterExecutor(executor.id, executor)
		require.NoError(t, err)
	}
	
	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	
	// Submit tasks with dependencies
	// task-3 depends on task-2, which depends on task-1
	tasks := []*SchedulableTask{
		{
			ID:           "task-1",
			Priority:     10,
			Dependencies: []string{},
			Resources:    ResourceRequirements{CPUCores: 1.0},
			Agent:        "agent-1",
		},
		{
			ID:           "task-2",
			Priority:     10,
			Dependencies: []string{"task-1"},
			Resources:    ResourceRequirements{CPUCores: 1.0},
			Agent:        "agent-2",
		},
		{
			ID:           "task-3",
			Priority:     10,
			Dependencies: []string{"task-2"},
			Resources:    ResourceRequirements{CPUCores: 1.0},
			Agent:        "agent-3",
		},
	}
	
	for _, task := range tasks {
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}
	
	// Collect results
	results := make(map[string]TaskResult)
	for i := 0; i < 3; i++ {
		select {
		case result := <-scheduler.GetResults():
			results[result.TaskID] = result
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}
	
	// Verify all tasks completed
	assert.Len(t, results, 3)
	for _, task := range tasks {
		result, ok := results[task.ID]
		assert.True(t, ok, "missing result for task %s", task.ID)
		assert.Equal(t, TaskStatusCompleted, result.Status)
	}
	
	// Verify execution order
	assert.True(t, results["task-1"].EndTime.Before(results["task-2"].StartTime) ||
		results["task-1"].EndTime.Equal(results["task-2"].StartTime))
	assert.True(t, results["task-2"].EndTime.Before(results["task-3"].StartTime) ||
		results["task-2"].EndTime.Equal(results["task-3"].StartTime))
	
	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestTaskScheduler_Priority(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.ScheduleInterval = 10 * time.Millisecond
	config.MaxConcurrentTasks = 1 // Force sequential execution
	
	scheduler, err := NewTaskScheduler(ctx, config)
	require.NoError(t, err)
	
	// Register single executor
	executor := &mockExecutor{
		id:        "agent-1",
		available: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			// Quick execution to test priority
			time.Sleep(20 * time.Millisecond)
			return taskID, nil
		},
	}
	
	err = scheduler.RegisterExecutor("agent-1", executor)
	require.NoError(t, err)
	
	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	
	// Submit tasks with different priorities
	tasks := []*SchedulableTask{
		{ID: "low-priority", Priority: 1, Resources: ResourceRequirements{CPUCores: 1.0}},
		{ID: "high-priority", Priority: 100, Resources: ResourceRequirements{CPUCores: 1.0}},
		{ID: "medium-priority", Priority: 50, Resources: ResourceRequirements{CPUCores: 1.0}},
	}
	
	for _, task := range tasks {
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}
	
	// Collect execution order
	var executionOrder []string
	for i := 0; i < 3; i++ {
		select {
		case result := <-scheduler.GetResults():
			executionOrder = append(executionOrder, result.TaskID)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}
	
	// Verify high priority executed first (after any already running task)
	assert.Contains(t, executionOrder[:2], "high-priority", "high priority task should execute early")
	
	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestTaskScheduler_ResourceAllocation(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.ScheduleInterval = 10 * time.Millisecond
	
	scheduler, err := NewTaskScheduler(ctx, config)
	require.NoError(t, err)
	
	// Set limited resources
	err = scheduler.resources.SetSystemResources(ctx, SystemResources{
		CPUCores: 2.0,
		MemoryMB: 1024,
	})
	require.NoError(t, err)
	
	// Register executors
	for i := 1; i <= 3; i++ {
		executor := &mockExecutor{
			id:        fmt.Sprintf("agent-%d", i),
			available: true,
		}
		err = scheduler.RegisterExecutor(executor.id, executor)
		require.NoError(t, err)
	}
	
	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	
	// Submit tasks that exceed available resources
	tasks := []*SchedulableTask{
		{
			ID:       "task-1",
			Priority: 10,
			Resources: ResourceRequirements{
				CPUCores: 1.5,
				MemoryMB: 512,
			},
		},
		{
			ID:       "task-2",
			Priority: 10,
			Resources: ResourceRequirements{
				CPUCores: 1.5,
				MemoryMB: 512,
			},
		},
	}
	
	for _, task := range tasks {
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}
	
	// Wait a bit for scheduling attempts
	time.Sleep(100 * time.Millisecond)
	
	// Verify only one task is running due to resource constraints
	running := scheduler.GetRunningTasks()
	assert.LessOrEqual(t, len(running), 1, "should not exceed resource limits")
	
	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestTaskScheduler_ConcurrentExecution(t *testing.T) {
	ctx := context.Background()
	config := DefaultSchedulerConfig()
	config.ScheduleInterval = 10 * time.Millisecond
	config.MaxConcurrentTasks = 5
	
	scheduler, err := NewTaskScheduler(ctx, config)
	require.NoError(t, err)
	
	// Register multiple executors
	numExecutors := 5
	for i := 1; i <= numExecutors; i++ {
		executor := &mockExecutor{
			id:        fmt.Sprintf("agent-%d", i),
			available: true,
			executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
				// Simulate concurrent work
				time.Sleep(50 * time.Millisecond)
				return taskID, nil
			},
		}
		err = scheduler.RegisterExecutor(executor.id, executor)
		require.NoError(t, err)
	}
	
	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)
	
	// Submit multiple tasks
	numTasks := 10
	startTime := time.Now()
	
	for i := 1; i <= numTasks; i++ {
		task := &SchedulableTask{
			ID:        fmt.Sprintf("task-%d", i),
			Priority:  10,
			Resources: ResourceRequirements{CPUCores: 0.5},
		}
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}
	
	// Collect results
	results := make([]TaskResult, 0, numTasks)
	for i := 0; i < numTasks; i++ {
		select {
		case result := <-scheduler.GetResults():
			results = append(results, result)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for results")
		}
	}
	
	duration := time.Since(startTime)
	
	// Verify all tasks completed
	assert.Len(t, results, numTasks)
	
	// Verify concurrent execution (should be faster than sequential)
	expectedSequentialTime := time.Duration(numTasks) * 50 * time.Millisecond
	assert.Less(t, duration, expectedSequentialTime/2, "execution should be concurrent")
	
	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	assert.NoError(t, err)
}

func TestTaskScheduler_GetStats(t *testing.T) {
	ctx := context.Background()
	scheduler, err := NewTaskScheduler(ctx, DefaultSchedulerConfig())
	require.NoError(t, err)
	
	// Register executor
	executor := &mockExecutor{id: "agent-1", available: true}
	err = scheduler.RegisterExecutor("agent-1", executor)
	require.NoError(t, err)
	
	// Submit tasks
	for i := 1; i <= 3; i++ {
		task := &SchedulableTask{
			ID:       fmt.Sprintf("task-%d", i),
			Priority: i * 10,
		}
		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}
	
	// Get stats
	stats := scheduler.GetSchedulerStats()
	assert.Equal(t, 0, stats["running_tasks"])
	assert.Equal(t, 3, stats["queued_tasks"])
	assert.Equal(t, 1, stats["total_executors"])
	assert.NotNil(t, stats["resource_usage"])
}