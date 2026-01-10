// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/kanban"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreaker tests circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	t.Run("opens after threshold failures", func(t *testing.T) {
		cb := NewCircuitBreaker("test-agent", 3, 100*time.Millisecond)

		// Simulate failures
		for i := 0; i < 3; i++ {
			err := cb.Call(context.Background(), func() error {
				return fmt.Errorf("failure %d", i)
			})
			assert.Error(t, err)
		}

		// Circuit should be open
		assert.Equal(t, CircuitOpen, cb.GetState())

		// Next call should fail immediately
		err := cb.Call(context.Background(), func() error {
			t.Fatal("should not be called")
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")
	})

	t.Run("transitions to half-open after timeout", func(t *testing.T) {
		cb := NewCircuitBreaker("test-agent", 2, 50*time.Millisecond)

		// Open the circuit
		for i := 0; i < 2; i++ {
			_ = cb.Call(context.Background(), func() error {
				return fmt.Errorf("failure")
			})
		}

		assert.Equal(t, CircuitOpen, cb.GetState())

		// Wait for reset timeout
		time.Sleep(60 * time.Millisecond)

		// Should allow one request in half-open state
		called := false
		err := cb.Call(context.Background(), func() error {
			called = true
			return nil
		})
		assert.NoError(t, err)
		assert.True(t, called)

		// Circuit should be closed after success
		assert.Equal(t, CircuitClosed, cb.GetState())
	})
}

// TestRetryPolicy tests retry logic
func TestRetryPolicy(t *testing.T) {
	t.Run("retries transient errors", func(t *testing.T) {
		rp := DefaultRetryPolicy()
		attempts := 0

		err := rp.Retry(context.Background(), "test-op", func() error {
			attempts++
			if attempts < 3 {
				return gerror.New(gerror.ErrCodeTimeout, "timeout", nil)
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})

	t.Run("does not retry permanent errors", func(t *testing.T) {
		rp := DefaultRetryPolicy()
		attempts := 0

		err := rp.Retry(context.Background(), "test-op", func() error {
			attempts++
			return gerror.New(gerror.ErrCodeInvalidInput, "bad input", nil)
		})

		assert.Error(t, err)
		assert.Equal(t, 1, attempts)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		rp := DefaultRetryPolicy()
		ctx, cancel := context.WithCancel(context.Background())

		attempts := 0
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := rp.Retry(ctx, "test-op", func() error {
			attempts++
			return gerror.New(gerror.ErrCodeTimeout, "timeout", nil)
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context cancelled")
		assert.Less(t, attempts, 3)
	})
}

// TestHealthMonitor tests health monitoring
func TestHealthMonitor(t *testing.T) {
	hm := NewHealthMonitor(5 * time.Minute)
	defer hm.Stop()

	// Record some successes
	for i := 0; i < 10; i++ {
		hm.RecordSuccess("agent-1", 100*time.Millisecond)
	}

	// Record some failures
	for i := 0; i < 3; i++ {
		hm.RecordFailure("agent-1", fmt.Sprintf("task-%d", i), fmt.Errorf("test error"))
	}

	// Get health report
	report := hm.GetHealthReport()
	assert.Len(t, report, 1)

	metrics := report["agent-1"]
	assert.Equal(t, int64(13), metrics.TotalRequests)
	assert.Equal(t, int64(3), metrics.FailedRequests)
	assert.InDelta(t, 76.92, metrics.SuccessRate, 0.01)
	assert.Equal(t, 100*time.Millisecond, metrics.AverageLatency)
}

// TestTaskRecovery tests task recovery and dead letter queue
func TestTaskRecovery(t *testing.T) {
	tr := NewTaskRecovery(3)

	task := &SchedulableTask{
		ID:           "test-task",
		CommissionID: "test-commission",
		Priority:     50,
	}

	// First few failures should retry
	for i := 1; i <= 2; i++ {
		shouldRetry := tr.HandleFailedTask(task, "agent-1", fmt.Errorf("failure %d", i), i)
		assert.True(t, shouldRetry)
	}

	// After max retries, should go to dead letter queue
	shouldRetry := tr.HandleFailedTask(task, "agent-1", fmt.Errorf("final failure"), 3)
	assert.False(t, shouldRetry)

	// Check dead letter queue
	dlq := tr.GetDeadLetterQueue()
	require.Len(t, dlq, 1)
	assert.Equal(t, "test-task", dlq[0].Task.ID)
	assert.Equal(t, 3, dlq[0].FailureCount)

	// Resubmit from dead letter queue
	resubmitted, err := tr.ResubmitDeadLetterTask(0)
	require.NoError(t, err)
	assert.Equal(t, "test-task", resubmitted.ID)

	// Dead letter queue should be empty
	dlq = tr.GetDeadLetterQueue()
	assert.Len(t, dlq, 0)
}

// TestProductionMetricsCollector tests metrics collection
func TestProductionMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	// Record various metrics
	mc.RecordTaskSubmitted()
	mc.RecordTaskStarted("task-1", "agent-1", 100*time.Millisecond)
	mc.RecordTaskCompleted("task-1", "agent-1", 500*time.Millisecond)

	mc.RecordTaskSubmitted()
	mc.RecordTaskStarted("task-2", "agent-2", 50*time.Millisecond)
	mc.RecordTaskFailed("task-2", "agent-2", gerror.New(gerror.ErrCodeTimeout, "timeout", nil), 200*time.Millisecond)

	// Get metrics snapshot
	snapshot := mc.GetMetrics()

	assert.Equal(t, int64(2), snapshot.Tasks.Submitted)
	assert.Equal(t, int64(1), snapshot.Tasks.Completed)
	assert.Equal(t, int64(1), snapshot.Tasks.Failed)
	assert.Equal(t, float64(50), snapshot.Tasks.SuccessRate)

	// Check latency stats
	assert.Equal(t, int64(2), snapshot.Latencies.QueueWaitTime.Count)
	assert.Equal(t, int64(2), snapshot.Latencies.ExecutionTime.Count)

	// Check agent metrics
	assert.Len(t, snapshot.Agents, 2)
	assert.Equal(t, int64(1), snapshot.Agents["agent-1"].TasksCompleted)
	assert.Equal(t, int64(1), snapshot.Agents["agent-2"].TasksFailed)

	// Check error metrics
	assert.Equal(t, int64(1), snapshot.Errors[string(gerror.ErrCodeTimeout)])
}

// TestProductionScheduler tests scheduler with all production features
func TestProductionScheduler(t *testing.T) {
	ctx := context.Background()

	// Create scheduler with production config
	config := &SchedulerConfig{
		MaxConcurrentTasks:      5,
		ScheduleInterval:        10 * time.Millisecond,
		DefaultTimeout:          30 * time.Second,
		EnableMetrics:           true,
		EnableTracing:           false,
		MaxTaskRetries:          2,
		HealthCheckWindow:       1 * time.Minute,
		CircuitBreakerThreshold: 3,
		CircuitBreakerTimeout:   100 * time.Millisecond,
	}

	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	scheduler, err := NewTaskScheduler(ctx, config, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Create agents with varying reliability
	var reliableCount, flakyCount, failingCount atomic.Int32

	// Reliable agent
	reliableExecutor := &mockAgentExecutor{
		agentID:     "reliable-agent",
		isAvailable: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			reliableCount.Add(1)
			time.Sleep(20 * time.Millisecond)
			return "success", nil
		},
	}

	// Flaky agent (fails sometimes)
	flakyExecutor := &mockAgentExecutor{
		agentID:     "flaky-agent",
		isAvailable: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			count := flakyCount.Add(1)
			time.Sleep(20 * time.Millisecond)
			if count%3 == 0 {
				return nil, gerror.New(gerror.ErrCodeTimeout, "flaky timeout", nil)
			}
			return "success", nil
		},
	}

	// Failing agent (always fails)
	failingExecutor := &mockAgentExecutor{
		agentID:     "failing-agent",
		isAvailable: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			failingCount.Add(1)
			return nil, gerror.New(gerror.ErrCodeInternal, "always fails", nil)
		},
	}

	// Register agents
	err = scheduler.RegisterAgent(ctx, "reliable-agent", reliableExecutor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)
	err = scheduler.RegisterAgent(ctx, "flaky-agent", flakyExecutor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)
	err = scheduler.RegisterAgent(ctx, "failing-agent", failingExecutor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Submit tasks
	taskCount := 15
	for i := 0; i < taskCount; i++ {
		taskID := fmt.Sprintf("task-%d", i)

		// Assign tasks to different agents
		var agentID string
		switch i % 3 {
		case 0:
			agentID = "reliable-agent"
		case 1:
			agentID = "flaky-agent"
		case 2:
			agentID = "failing-agent"
		}

		managerAgent.SetAssignment(taskID, &TaskAssignment{
			TaskID:  taskID,
			AgentID: agentID,
		})

		kanbanClient.AddTask(&kanban.Task{
			ID:     taskID,
			Title:  taskID,
			Status: kanban.StatusTodo,
		})

		task := &SchedulableTask{
			ID:           taskID,
			CommissionID: "test-commission",
			Priority:     50,
		}

		err = scheduler.SubmitTask(ctx, task)
		require.NoError(t, err)
	}

	// Wait for tasks to process
	time.Sleep(500 * time.Millisecond)

	// Get metrics
	metrics := scheduler.GetMetricsSnapshot()
	require.NotNil(t, metrics)

	// Verify metrics
	assert.Equal(t, int64(taskCount), metrics.Tasks.Submitted)
	assert.Greater(t, metrics.Tasks.Completed, int64(0))
	assert.Greater(t, metrics.Tasks.Failed, int64(0))

	// Get health report
	health := scheduler.GetHealthReport()
	require.NotNil(t, health)

	// Check agent health
	assert.Contains(t, health, "reliable-agent")
	assert.Contains(t, health, "flaky-agent")
	assert.Contains(t, health, "failing-agent")

	// Reliable agent should have 100% success rate
	assert.Equal(t, float64(100), health["reliable-agent"].SuccessRate)

	// Failing agent should have 0% success rate
	assert.Equal(t, float64(0), health["failing-agent"].SuccessRate)

	// Check dead letter queue
	dlq := scheduler.GetDeadLetterQueue()
	// Note: tasks might still be retrying, so dead letter queue might be empty
	t.Logf("Dead letter queue has %d tasks", len(dlq))

	// Test resubmitting from dead letter queue
	if len(dlq) > 0 {
		err = scheduler.ResubmitDeadLetterTask(ctx, 0)
		assert.NoError(t, err)

		// Dead letter queue should have one less item
		newDlq := scheduler.GetDeadLetterQueue()
		assert.Equal(t, len(dlq)-1, len(newDlq))
	}

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Generate final report
	finalMetrics := scheduler.GetMetricsSnapshot()
	if finalMetrics != nil {
		report := FormatMetricsAsText(finalMetrics)
		t.Logf("Final metrics report:\n%s", report)
	}
}

// TestConcurrentMetricsAccess tests concurrent access to metrics
func TestConcurrentMetricsAccess(t *testing.T) {
	mc := NewMetricsCollector()

	var wg sync.WaitGroup

	// Concurrent metric updates
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%d", id%3)

			for j := 0; j < 100; j++ {
				taskID := fmt.Sprintf("task-%d-%d", id, j)

				mc.RecordTaskSubmitted()
				mc.RecordTaskStarted(taskID, agentID, time.Duration(j)*time.Millisecond)

				if j%5 == 0 {
					mc.RecordTaskFailed(taskID, agentID, fmt.Errorf("error"), time.Duration(j)*time.Millisecond)
				} else {
					mc.RecordTaskCompleted(taskID, agentID, time.Duration(j)*time.Millisecond)
				}
			}
		}(i)
	}

	// Concurrent metric reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				snapshot := mc.GetMetrics()
				_ = snapshot // Use snapshot
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Verify final metrics
	snapshot := mc.GetMetrics()
	assert.Equal(t, int64(1000), snapshot.Tasks.Submitted)
	assert.Equal(t, int64(800), snapshot.Tasks.Completed)
	assert.Equal(t, int64(200), snapshot.Tasks.Failed)
}
