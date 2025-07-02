// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressAggregator_RegisterCommission(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Register commission
	err := pa.RegisterCommission(ctx, "commission-1", 10)
	require.NoError(t, err)

	// Verify commission was registered
	progress, err := pa.GetCommissionProgress("commission-1")
	require.NoError(t, err)
	assert.Equal(t, "commission-1", progress.CommissionID)
	assert.Equal(t, 10, progress.TotalTasks)
	assert.Equal(t, 10, progress.PendingTasks)
	assert.Equal(t, 0, progress.CompletedTasks)
	assert.NotNil(t, progress.StartTime)
}

func TestProgressAggregator_UpdateTaskProgress(t *testing.T) {
	pa := NewProgressAggregator()

	// Update task progress
	update := ProgressUpdate{
		TaskID:     "task-1",
		Percentage: 50.0,
		Message:    "Processing data",
		Timestamp:  time.Now(),
		Details: map[string]interface{}{
			"subtasks": []SubTaskProgress{
				{ID: "subtask-1", Name: "Load data", Percentage: 100.0, Status: "completed"},
				{ID: "subtask-2", Name: "Process data", Percentage: 25.0, Status: "running"},
			},
		},
	}

	err := pa.UpdateTaskProgress(update)
	assert.NoError(t, err)

	// Verify task progress
	task, err := pa.GetTaskProgress("task-1")
	require.NoError(t, err)
	assert.Equal(t, 50.0, task.Percentage)
	assert.Equal(t, "Processing data", task.Message)
	assert.Len(t, task.SubTasks, 2)
}

func TestProgressAggregator_UpdateTaskStatus(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Register commission
	err := pa.RegisterCommission(ctx, "commission-1", 3)
	require.NoError(t, err)

	// Update task statuses
	pa.UpdateTaskStatus("task-1", "commission-1", TaskStatusRunning)
	pa.UpdateTaskStatus("task-2", "commission-1", TaskStatusRunning)
	pa.UpdateTaskStatus("task-1", "commission-1", TaskStatusCompleted)

	// Verify commission progress
	progress, err := pa.GetCommissionProgress("commission-1")
	require.NoError(t, err)
	assert.Equal(t, 1, progress.CompletedTasks)
	assert.Equal(t, 1, progress.RunningTasks)
	assert.Equal(t, 1, progress.PendingTasks)
	assert.InDelta(t, float64(100.0/3.0), progress.OverallProgress, 0.0000001)
}

func TestProgressAggregator_Subscribe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pa := NewProgressAggregator()

	// Subscribe to updates
	ch := pa.Subscribe(ctx)

	// Should receive initial snapshot
	select {
	case snapshot := <-ch:
		assert.NotNil(t, snapshot)
		assert.NotZero(t, snapshot.Timestamp)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for initial snapshot")
	}

	// Register commission and update task
	err := pa.RegisterCommission(ctx, "commission-1", 1)
	require.NoError(t, err)
	pa.UpdateTaskStatus("task-1", "commission-1", TaskStatusRunning)

	// Should receive update
	select {
	case snapshot := <-ch:
		assert.Len(t, snapshot.Commissions, 1)
		assert.Len(t, snapshot.RunningTasks, 1)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for update")
	}

	// Cancel context should close channel
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok)
}

func TestProgressAggregator_ProgressSnapshot(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Set up test data
	err := pa.RegisterCommission(ctx, "commission-1", 10)
	require.NoError(t, err)

	// Add some completed tasks
	for i := 1; i <= 5; i++ {
		taskID := "task-" + string(rune('0'+i))
		pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusRunning)
		time.Sleep(10 * time.Millisecond)
		pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusCompleted)
	}

	// Add running tasks
	pa.UpdateTaskStatus("task-6", "commission-1", TaskStatusRunning)
	pa.UpdateTaskProgress(ProgressUpdate{
		TaskID:     "task-6",
		Percentage: 75.0,
		Message:    "Processing",
		Timestamp:  time.Now(),
	})

	// Get snapshot
	snapshot := pa.GetProgressSnapshot()

	// Verify snapshot
	assert.NotZero(t, snapshot.Timestamp)
	assert.Len(t, snapshot.Commissions, 1)
	assert.Len(t, snapshot.RunningTasks, 1)

	commission := snapshot.Commissions["commission-1"]
	assert.Equal(t, 50.0, commission.OverallProgress)
	assert.Equal(t, 5, commission.TaskCounts.Completed)
	assert.Equal(t, 1, commission.TaskCounts.Running)
	assert.Equal(t, 4, commission.TaskCounts.Pending)

	task := snapshot.RunningTasks[0]
	assert.Equal(t, "task-6", task.TaskID)
	assert.Equal(t, 75.0, task.Progress)
	assert.Equal(t, "Processing", task.Message)

	// Verify metrics
	assert.Equal(t, 5, snapshot.Metrics.TotalTasksProcessed)
	assert.Equal(t, 100.0, snapshot.Metrics.SuccessRate)
	assert.Greater(t, snapshot.Metrics.TasksPerMinute, 0.0)
}

func TestProgressAggregator_EstimatedCompletion(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Register commission
	err := pa.RegisterCommission(ctx, "commission-1", 10)
	require.NoError(t, err)

	// Complete tasks with consistent timing
	for i := 1; i <= 5; i++ {
		taskID := "task-" + string(rune('0'+i))
		pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusRunning)
		time.Sleep(50 * time.Millisecond)
		pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusCompleted)
	}

	// Check estimated end time
	progress, err := pa.GetCommissionProgress("commission-1")
	require.NoError(t, err)
	assert.NotNil(t, progress.EstimatedEnd)
	assert.True(t, progress.EstimatedEnd.After(time.Now()))
}

func TestProgressAggregator_ConcurrentUpdates(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Register commission
	err := pa.RegisterCommission(ctx, "commission-1", 100)
	require.NoError(t, err)

	// Subscribe to updates
	ch := pa.Subscribe(ctx)

	// Drain initial snapshot
	<-ch

	// Concurrent updates
	var wg sync.WaitGroup
	numGoroutines := 10
	tasksPerGoroutine := 10

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutine int) {
			defer wg.Done()

			for i := 0; i < tasksPerGoroutine; i++ {
				taskID := "task-" + string(rune('0'+goroutine)) + "-" + string(rune('0'+i))

				// Update status
				pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusRunning)

				// Update progress
				pa.UpdateTaskProgress(ProgressUpdate{
					TaskID:     taskID,
					Percentage: float64(i) * 10.0,
					Message:    "Processing",
					Timestamp:  time.Now(),
				})

				// Complete task
				pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusCompleted)
			}
		}(g)
	}

	// Collect updates
	updateCount := 0
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ch:
				updateCount++
			case <-done:
				return
			}
		}
	}()

	// Wait for all updates
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	close(done)

	// Verify final state
	progress, err := pa.GetCommissionProgress("commission-1")
	require.NoError(t, err)
	assert.Equal(t, 100, progress.CompletedTasks)
	assert.Equal(t, 100.0, progress.OverallProgress)
	assert.Greater(t, updateCount, 0)
}

func TestProgressAggregator_Clear(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Add data
	err := pa.RegisterCommission(ctx, "commission-1", 10)
	require.NoError(t, err)
	pa.UpdateTaskStatus("task-1", "commission-1", TaskStatusRunning)

	// Clear
	pa.Clear()

	// Verify cleared
	_, err = pa.GetCommissionProgress("commission-1")
	assert.Error(t, err)

	_, err = pa.GetTaskProgress("task-1")
	assert.Error(t, err)
}

func TestProgressAggregator_SubTasks(t *testing.T) {
	pa := NewProgressAggregator()

	// Create task with subtasks
	update := ProgressUpdate{
		TaskID:     "main-task",
		Percentage: 0.0,
		Message:    "Starting",
		Timestamp:  time.Now(),
		Details: map[string]interface{}{
			"subtasks": []SubTaskProgress{
				{ID: "sub-1", Name: "Initialize", Percentage: 0.0, Status: "pending"},
				{ID: "sub-2", Name: "Process", Percentage: 0.0, Status: "pending"},
				{ID: "sub-3", Name: "Finalize", Percentage: 0.0, Status: "pending"},
			},
		},
	}

	err := pa.UpdateTaskProgress(update)
	require.NoError(t, err)

	// Update subtasks progressively
	subtaskUpdates := []struct {
		percentage float64
		subtasks   []SubTaskProgress
	}{
		{
			percentage: 33.3,
			subtasks: []SubTaskProgress{
				{ID: "sub-1", Name: "Initialize", Percentage: 100.0, Status: "completed"},
				{ID: "sub-2", Name: "Process", Percentage: 0.0, Status: "running"},
				{ID: "sub-3", Name: "Finalize", Percentage: 0.0, Status: "pending"},
			},
		},
		{
			percentage: 66.6,
			subtasks: []SubTaskProgress{
				{ID: "sub-1", Name: "Initialize", Percentage: 100.0, Status: "completed"},
				{ID: "sub-2", Name: "Process", Percentage: 100.0, Status: "completed"},
				{ID: "sub-3", Name: "Finalize", Percentage: 0.0, Status: "running"},
			},
		},
		{
			percentage: 100.0,
			subtasks: []SubTaskProgress{
				{ID: "sub-1", Name: "Initialize", Percentage: 100.0, Status: "completed"},
				{ID: "sub-2", Name: "Process", Percentage: 100.0, Status: "completed"},
				{ID: "sub-3", Name: "Finalize", Percentage: 100.0, Status: "completed"},
			},
		},
	}

	for _, update := range subtaskUpdates {
		err := pa.UpdateTaskProgress(ProgressUpdate{
			TaskID:     "main-task",
			Percentage: update.percentage,
			Message:    "Processing",
			Timestamp:  time.Now(),
			Details: map[string]interface{}{
				"subtasks": update.subtasks,
			},
		})
		require.NoError(t, err)

		// Verify subtask state
		task, err := pa.GetTaskProgress("main-task")
		require.NoError(t, err)
		assert.Equal(t, update.percentage, task.Percentage)
		assert.Len(t, task.SubTasks, 3)
	}
}

func TestProgressAggregator_Metrics(t *testing.T) {
	pa := NewProgressAggregator()
	ctx := context.Background()

	// Create multiple commissions
	err := pa.RegisterCommission(ctx, "commission-1", 10)
	require.NoError(t, err)
	err = pa.RegisterCommission(ctx, "commission-2", 5)
	require.NoError(t, err)

	// Complete tasks with various outcomes
	for i := 1; i <= 10; i++ {
		taskID := "task-1-" + string(rune('0'+i))
		pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusRunning)
		time.Sleep(10 * time.Millisecond)

		if i <= 8 {
			pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusCompleted)
		} else {
			pa.UpdateTaskStatus(taskID, "commission-1", TaskStatusFailed)
		}
	}

	for i := 1; i <= 5; i++ {
		taskID := "task-2-" + string(rune('0'+i))
		pa.UpdateTaskStatus(taskID, "commission-2", TaskStatusRunning)
		time.Sleep(10 * time.Millisecond)
		pa.UpdateTaskStatus(taskID, "commission-2", TaskStatusCompleted)
	}

	// Get snapshot and verify metrics
	snapshot := pa.GetProgressSnapshot()

	assert.Equal(t, 15, snapshot.Metrics.TotalTasksProcessed)    // 10 + 5
	assert.Equal(t, 13.0/15.0*100, snapshot.Metrics.SuccessRate) // 13 completed / 15 total
	assert.Greater(t, snapshot.Metrics.TasksPerMinute, 0.0)
	assert.Greater(t, snapshot.Metrics.AverageTaskDuration, time.Duration(0))
}

func TestProgressAggregator_ContextCancellation(t *testing.T) {
	pa := NewProgressAggregator()

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// RegisterCommission should fail with cancelled context
	err := pa.RegisterCommission(ctx, "commission-1", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}
