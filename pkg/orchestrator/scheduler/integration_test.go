// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationSchedulerFullWorkflow tests the complete scheduler workflow
func TestIntegrationSchedulerFullWorkflow(t *testing.T) {
	ctx := context.Background()

	// Create dependencies
	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	// Create scheduler with custom config
	config := &SchedulerConfig{
		MaxConcurrentTasks: 3,
		ScheduleInterval:   10 * time.Millisecond,
		DefaultTimeout:     30 * time.Second,
		EnableMetrics:      true,
	}

	scheduler, err := NewTaskScheduler(ctx, config, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Register multiple agents with different capabilities
	agents := []struct {
		id           string
		capabilities []AgentCapability
	}{
		{"agent-code-1", []AgentCapability{CapabilityCode}},
		{"agent-code-2", []AgentCapability{CapabilityCode}},
		{"agent-test-1", []AgentCapability{CapabilityTest}},
		{"agent-review-1", []AgentCapability{CapabilityReview}},
		{"agent-multi-1", []AgentCapability{CapabilityCode, CapabilityTest, CapabilityReview}},
	}

	executionCounts := make(map[string]*atomic.Int32)
	for _, agent := range agents {
		count := &atomic.Int32{}
		executionCounts[agent.id] = count

		executor := &mockAgentExecutor{
			agentID:      agent.id,
			capabilities: convertCapabilities(agent.capabilities),
			isAvailable:  true,
			executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
				count.Add(1)
				// Simulate work
				select {
				case <-time.After(50 * time.Millisecond):
					return fmt.Sprintf("Task %s completed by %s", taskID, agent.id), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		}

		err = scheduler.RegisterAgent(ctx, agent.id, executor, agent.capabilities)
		require.NoError(t, err)
	}

	// Register commission
	commissionID := "test-commission-1"
	err = scheduler.RegisterCommission(ctx, commissionID, 10)
	require.NoError(t, err)

	// Create tasks with dependencies
	taskGraph := []struct {
		id           string
		priority     int
		dependencies []string
		capability   string
	}{
		{"task-1", 100, []string{}, "code"},
		{"task-2", 90, []string{}, "code"},
		{"task-3", 80, []string{"task-1"}, "test"},
		{"task-4", 70, []string{"task-2"}, "test"},
		{"task-5", 60, []string{"task-3", "task-4"}, "review"},
		{"task-6", 50, []string{}, "code"},
		{"task-7", 40, []string{"task-6"}, "test"},
		{"task-8", 30, []string{"task-7"}, "review"},
		{"task-9", 20, []string{"task-5", "task-8"}, "code"},
		{"task-10", 10, []string{"task-9"}, "review"},
	}

	// Pre-configure assignments in manager
	for _, task := range taskGraph {
		// Select appropriate agent based on capability
		var agentID string
		switch task.capability {
		case "code":
			if task.priority > 50 {
				agentID = "agent-code-1"
			} else {
				agentID = "agent-code-2"
			}
		case "test":
			agentID = "agent-test-1"
		case "review":
			agentID = "agent-review-1"
		}

		managerAgent.SetAssignment(task.id, &TaskAssignment{
			TaskID:      task.id,
			AgentID:     agentID,
			Reasoning:   fmt.Sprintf("Selected %s for %s capability", agentID, task.capability),
			APIProvider: "openai",
			Priority:    task.priority,
			Deadline:    time.Now().Add(5 * time.Minute),
		})

		// Add to kanban
		kanbanClient.AddTask(&kanban.Task{
			ID:          task.id,
			Title:       fmt.Sprintf("Task %s", task.id),
			Description: fmt.Sprintf("Test task requiring %s capability", task.capability),
			Status:      kanban.StatusTodo,
			Priority:    mapToKanbanPriority(task.priority),
		})
	}

	// Submit all tasks
	for _, task := range taskGraph {
		schedulableTask := &SchedulableTask{
			ID:           task.id,
			CommissionID: commissionID,
			Priority:     task.priority,
			Dependencies: task.dependencies,
			Resources: ResourceRequirements{
				APIQuotas: map[string]int{"openai": 1},
			},
		}

		err = scheduler.SubmitTask(ctx, schedulableTask)
		require.NoError(t, err)
	}

	// Subscribe to progress updates
	progressChan := scheduler.SubscribeToProgress(ctx)

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Track completion
	completedTasks := make(map[string]bool)
	taskResults := make(map[string]*TaskResult)
	resultChan := scheduler.GetResults()

	// Wait for all tasks to complete with timeout
	done := make(chan struct{})
	go func() {
		for {
			select {
			case result := <-resultChan:
				completedTasks[result.TaskID] = true
				taskResults[result.TaskID] = &result

				if len(completedTasks) == len(taskGraph) {
					close(done)
					return
				}
			case <-time.After(10 * time.Second):
				close(done)
				return
			}
		}
	}()

	// Monitor progress updates
	var progressSnapshots []ProgressSnapshot
	go func() {
		for snapshot := range progressChan {
			progressSnapshots = append(progressSnapshots, snapshot)
		}
	}()

	<-done

	// Verify all tasks completed
	assert.Len(t, completedTasks, len(taskGraph), "All tasks should complete")

	// Verify task results
	for taskID, result := range taskResults {
		assert.Equal(t, TaskStatusCompleted, result.Status, "Task %s should complete successfully", taskID)
		assert.NoError(t, result.Error, "Task %s should not have errors", taskID)
		assert.NotNil(t, result.Output, "Task %s should have output", taskID)
	}

	// Verify dependency order
	for _, task := range taskGraph {
		result := taskResults[task.id]
		for _, dep := range task.dependencies {
			depResult := taskResults[dep]
			assert.True(t, depResult.EndTime.Before(result.StartTime) || depResult.EndTime.Equal(result.StartTime),
				"Dependency %s should complete before %s starts", dep, task.id)
		}
	}

	// Verify agent distribution
	totalExecuted := int32(0)
	for agentID, count := range executionCounts {
		t.Logf("Agent %s executed %d tasks", agentID, count.Load())
		totalExecuted += count.Load()
	}
	// Verify that tasks were distributed among agents
	assert.Equal(t, int32(len(taskGraph)), totalExecuted, "All tasks should be executed")
	// At least 3 different agents should have executed tasks
	activeAgents := 0
	for _, count := range executionCounts {
		if count.Load() > 0 {
			activeAgents++
		}
	}
	assert.GreaterOrEqual(t, activeAgents, 3, "At least 3 agents should have executed tasks")

	// Verify progress tracking
	finalProgress, err := scheduler.GetCommissionProgress(commissionID)
	require.NoError(t, err)
	assert.Equal(t, 10, finalProgress.TotalTasks)
	assert.Equal(t, 10, finalProgress.CompletedTasks)
	assert.Equal(t, 0, finalProgress.FailedTasks)
	assert.Equal(t, float64(100), finalProgress.OverallProgress)

	// Verify scheduler stats
	stats := scheduler.GetSchedulerStats()
	assert.Equal(t, 0, stats["running_tasks"])
	assert.Equal(t, 0, stats["queued_tasks"])
	assert.Equal(t, int64(10), stats["tasks_completed"])

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

// TestIntegrationSchedulerWithFailures tests scheduler behavior with task failures
func TestIntegrationSchedulerWithFailures(t *testing.T) {
	ctx := context.Background()

	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	scheduler, err := NewTaskScheduler(ctx, nil, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Register agents with controlled failure behavior
	failureCount := &atomic.Int32{}
	successCount := &atomic.Int32{}

	failingExecutor := &mockAgentExecutor{
		agentID:     "failing-agent",
		isAvailable: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			failureCount.Add(1)
			return nil, gerror.New(gerror.ErrCodeInternal, "simulated failure", nil).
				WithComponent("test").
				WithOperation("execute")
		},
	}

	successExecutor := &mockAgentExecutor{
		agentID:     "success-agent",
		isAvailable: true,
		executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
			successCount.Add(1)
			time.Sleep(20 * time.Millisecond)
			return "success", nil
		},
	}

	err = scheduler.RegisterAgent(ctx, "failing-agent", failingExecutor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)
	err = scheduler.RegisterAgent(ctx, "success-agent", successExecutor, []AgentCapability{CapabilityCode})
	require.NoError(t, err)

	// Create tasks that will fail and block dependencies
	tasks := []struct {
		id           string
		agentID      string
		dependencies []string
	}{
		{"fail-1", "failing-agent", []string{}},
		{"fail-2", "failing-agent", []string{}},
		{"blocked-1", "success-agent", []string{"fail-1"}},
		{"blocked-2", "success-agent", []string{"fail-2"}},
		{"success-1", "success-agent", []string{}},
		{"success-2", "success-agent", []string{"success-1"}},
	}

	// Configure assignments and add to kanban
	for _, task := range tasks {
		managerAgent.SetAssignment(task.id, &TaskAssignment{
			TaskID:  task.id,
			AgentID: task.agentID,
		})

		kanbanClient.AddTask(&kanban.Task{
			ID:     task.id,
			Title:  task.id,
			Status: kanban.StatusTodo,
		})
	}

	// Submit tasks
	for _, task := range tasks {
		err = scheduler.SubmitTask(ctx, &SchedulableTask{
			ID:           task.id,
			CommissionID: "test-commission",
			Priority:     50,
			Dependencies: task.dependencies,
		})
		require.NoError(t, err)
	}

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Collect results
	results := make(map[string]*TaskResult)
	resultChan := scheduler.GetResults()
	
	done := make(chan struct{})
	go func() {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case result := <-resultChan:
				results[result.TaskID] = &result
				// We expect 4 results: 2 failures and 2 successes
				if len(results) == 4 {
					close(done)
					return
				}
			case <-timeout:
				close(done)
				return
			}
		}
	}()

	<-done

	// Verify results
	assert.Equal(t, 4, len(results), "Should have 4 task results")

	// Check failed tasks
	assert.Equal(t, TaskStatusFailed, results["fail-1"].Status)
	assert.Error(t, results["fail-1"].Error)
	assert.Equal(t, TaskStatusFailed, results["fail-2"].Status)
	assert.Error(t, results["fail-2"].Error)

	// Check successful tasks
	assert.Equal(t, TaskStatusCompleted, results["success-1"].Status)
	assert.NoError(t, results["success-1"].Error)
	assert.Equal(t, TaskStatusCompleted, results["success-2"].Status)
	assert.NoError(t, results["success-2"].Error)

	// Verify blocked tasks are still queued
	status1, err := scheduler.GetTaskStatus("blocked-1")
	require.NoError(t, err)
	assert.Equal(t, TaskStatusPending, status1)

	status2, err := scheduler.GetTaskStatus("blocked-2")
	require.NoError(t, err)
	assert.Equal(t, TaskStatusPending, status2)

	// Verify kanban status updates
	statusUpdates := kanbanClient.GetStatusUpdates()
	blockedCount := 0
	for _, update := range statusUpdates {
		if update.Status == kanban.StatusBlocked && (update.TaskID == "fail-1" || update.TaskID == "fail-2") {
			blockedCount++
		}
	}
	assert.Equal(t, 2, blockedCount, "Failed tasks should be marked as blocked in kanban")

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

// TestIntegrationSchedulerConcurrency tests concurrent task execution limits
func TestIntegrationSchedulerConcurrency(t *testing.T) {
	ctx := context.Background()

	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	// Configure low concurrency limit
	config := &SchedulerConfig{
		MaxConcurrentTasks: 2,
		ScheduleInterval:   5 * time.Millisecond,
		DefaultTimeout:     30 * time.Second,
	}

	scheduler, err := NewTaskScheduler(ctx, config, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Track concurrent executions
	var maxConcurrent atomic.Int32
	var currentConcurrent atomic.Int32

	// Register multiple agents
	for i := 1; i <= 5; i++ {
		agentID := fmt.Sprintf("agent-%d", i)
		executor := &mockAgentExecutor{
			agentID:     agentID,
			isAvailable: true,
			executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
				// Increment concurrent count
				current := currentConcurrent.Add(1)
				
				// Update max if needed
				for {
					max := maxConcurrent.Load()
					if current <= max || maxConcurrent.CompareAndSwap(max, current) {
						break
					}
				}

				// Simulate work
				time.Sleep(100 * time.Millisecond)

				// Decrement concurrent count
				currentConcurrent.Add(-1)

				return "done", nil
			},
		}

		err = scheduler.RegisterAgent(ctx, agentID, executor, []AgentCapability{CapabilityCode})
		require.NoError(t, err)
	}

	// Submit many tasks
	taskCount := 10
	for i := 1; i <= taskCount; i++ {
		taskID := fmt.Sprintf("task-%d", i)
		agentID := fmt.Sprintf("agent-%d", ((i-1)%5)+1)

		managerAgent.SetAssignment(taskID, &TaskAssignment{
			TaskID:  taskID,
			AgentID: agentID,
		})

		kanbanClient.AddTask(&kanban.Task{
			ID:     taskID,
			Title:  taskID,
			Status: kanban.StatusTodo,
		})

		err = scheduler.SubmitTask(ctx, &SchedulableTask{
			ID:           taskID,
			CommissionID: "test-commission",
			Priority:     100 - i, // Higher priority for earlier tasks
		})
		require.NoError(t, err)
	}

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Wait for some tasks to start executing
	time.Sleep(200 * time.Millisecond)

	// Check that concurrency limit is respected
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(config.MaxConcurrentTasks),
		"Should not exceed max concurrent tasks limit")

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}

// TestIntegrationSchedulerRaceConditions tests for race conditions
func TestIntegrationSchedulerRaceConditions(t *testing.T) {
	ctx := context.Background()

	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()
	kanbanClient.SetSimulateConflict(true) // Enable conflict simulation

	scheduler, err := NewTaskScheduler(ctx, nil, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Register multiple agents
	var wg sync.WaitGroup
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			agentID := fmt.Sprintf("agent-%d", id)
			executor := &mockAgentExecutor{
				agentID:     agentID,
				isAvailable: true,
				executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
					return "done", nil
				},
			}
			err := scheduler.RegisterAgent(ctx, agentID, executor, []AgentCapability{CapabilityCode})
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Submit tasks concurrently
	taskCount := 20
	for i := 1; i <= taskCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("task-%d", id)
			
			// Configure assignment
			managerAgent.SetAssignment(taskID, &TaskAssignment{
				TaskID:  taskID,
				AgentID: fmt.Sprintf("agent-%d", ((id-1)%5)+1),
			})

			// Add to kanban
			kanbanClient.AddTask(&kanban.Task{
				ID:     taskID,
				Title:  taskID,
				Status: kanban.StatusTodo,
			})

			// Submit task
			err := scheduler.SubmitTask(ctx, &SchedulableTask{
				ID:           taskID,
				CommissionID: "test-commission",
				Priority:     id,
			})
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Create a done channel to coordinate goroutines
	done := make(chan struct{})
	
	// Query status concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				select {
				case <-done:
					return
				default:
					stats := scheduler.GetSchedulerStats()
					_ = stats // Use stats
					
					// Get random task status
					taskID := fmt.Sprintf("task-%d", (j%taskCount)+1)
					_, _ = scheduler.GetTaskStatus(taskID)
					
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}

	// Let it run for a bit
	time.Sleep(100 * time.Millisecond)

	// Signal goroutines to stop
	close(done)

	// Stop scheduler after signaling goroutines
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)

	// Wait for all goroutines to finish
	wg.Wait()
}

// TestIntegrationSchedulerManagerAgentIntegration tests manager agent decision making
func TestIntegrationSchedulerManagerAgentIntegration(t *testing.T) {
	ctx := context.Background()

	// Create a more sophisticated mock manager that makes decisions
	managerAgent := &smartMockManagerAgent{
		MockManagerAgentClient: NewMockManagerAgentClient(),
		agentLoad:              make(map[string]int),
	}
	kanbanClient := NewMockKanbanClient()

	scheduler, err := NewTaskScheduler(ctx, nil, managerAgent, kanbanClient)
	require.NoError(t, err)

	// Register agents with varying capabilities and performance
	agentConfigs := []struct {
		id           string
		capabilities []AgentCapability
		performance  float64 // Success rate
	}{
		{"expert-coder", []AgentCapability{CapabilityCode}, 0.95},
		{"junior-coder", []AgentCapability{CapabilityCode}, 0.70},
		{"senior-tester", []AgentCapability{CapabilityTest}, 0.90},
		{"versatile-agent", []AgentCapability{CapabilityCode, CapabilityTest, CapabilityReview}, 0.85},
	}

	for _, config := range agentConfigs {
		executor := &mockAgentExecutor{
			agentID:      config.id,
			capabilities: convertCapabilities(config.capabilities),
			isAvailable:  true,
			executeFunc: func(ctx context.Context, taskID string, payload interface{}) (interface{}, error) {
				// Simulate performance-based success
				if time.Now().UnixNano()%100 < int64(config.performance*100) {
					return fmt.Sprintf("Success from %s", config.id), nil
				}
				return nil, fmt.Errorf("simulated failure from %s", config.id)
			},
		}

		err = scheduler.RegisterAgent(ctx, config.id, executor, config.capabilities)
		require.NoError(t, err)
	}

	// Submit various tasks
	tasks := []struct {
		id         string
		priority   int
		complexity string // high, medium, low
		capability string
	}{
		{"critical-feature", 100, "high", "code"},
		{"bug-fix", 90, "medium", "code"},
		{"unit-tests", 80, "medium", "test"},
		{"integration-tests", 70, "high", "test"},
		{"code-review", 60, "low", "review"},
		{"refactor", 50, "high", "code"},
		{"documentation", 40, "low", "code"},
	}

	for _, task := range tasks {
		kanbanClient.AddTask(&kanban.Task{
			ID:          task.id,
			Title:       task.id,
			Description: fmt.Sprintf("Complexity: %s", task.complexity),
			Status:      kanban.StatusTodo,
			Priority:    mapToKanbanPriority(task.priority),
		})

		err = scheduler.SubmitTask(ctx, &SchedulableTask{
			ID:           task.id,
			CommissionID: "test-commission",
			Priority:     task.priority,
			Payload: map[string]interface{}{
				"complexity": task.complexity,
				"capability": task.capability,
			},
		})
		require.NoError(t, err)
	}

	// Start scheduler
	err = scheduler.Start(ctx)
	require.NoError(t, err)

	// Collect results
	time.Sleep(500 * time.Millisecond)

	// Verify manager made intelligent assignments
	requests := managerAgent.GetRequestsReceived()
	assert.Greater(t, len(requests), 0, "Manager should receive assignment requests")

	// Check assignment patterns
	assignments := managerAgent.GetAssignments()
	for taskID, assignment := range assignments {
		t.Logf("Task %s assigned to %s: %s", taskID, assignment.AgentID, assignment.Reasoning)
		
		// Verify critical tasks got assigned to high-performance agents
		for _, task := range tasks {
			if task.id == taskID && task.priority >= 90 {
				assert.Contains(t, []string{"expert-coder", "senior-tester", "versatile-agent"}, 
					assignment.AgentID, "High priority task should go to reliable agent")
			}
		}
	}

	// Stop scheduler
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err = scheduler.Stop(stopCtx)
	require.NoError(t, err)
}


// smartMockManagerAgent extends MockManagerAgentClient with intelligent assignment logic
type smartMockManagerAgent struct {
	*MockManagerAgentClient
	agentLoad map[string]int
	mu        sync.Mutex
}

func (s *smartMockManagerAgent) RequestAssignment(ctx context.Context, task *kanban.Task, availableAgents []*AgentInfo) (*TaskAssignment, error) {
	// Need to call parent's lock for requestsReceived
	s.MockManagerAgentClient.mu.Lock()
	s.MockManagerAgentClient.requestsReceived = append(s.MockManagerAgentClient.requestsReceived, AssignmentRequest{
		Task:            task,
		AvailableAgents: availableAgents,
		Timestamp:       time.Now(),
	})
	s.MockManagerAgentClient.mu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Make intelligent assignment decision
	var bestAgent *AgentInfo
	minLoad := int(^uint(0) >> 1) // Max int

	for _, agent := range availableAgents {
		load := s.agentLoad[agent.AgentID]
		
		// Prefer agents with lower load
		if load < minLoad {
			bestAgent = agent
			minLoad = load
		}
		
		// Special handling for high priority tasks
		if task.Priority == kanban.PriorityHigh {
			// Prefer "expert" or "senior" agents
			if (agent.AgentID == "expert-coder" || agent.AgentID == "senior-tester") && load < minLoad+2 {
				bestAgent = agent
				break
			}
		}
	}

	if bestAgent == nil && len(availableAgents) > 0 {
		bestAgent = availableAgents[0]
	}

	if bestAgent == nil {
		return nil, fmt.Errorf("no available agents")
	}

	// Update load
	s.agentLoad[bestAgent.AgentID]++

	assignment := &TaskAssignment{
		TaskID:      task.ID,
		AgentID:     bestAgent.AgentID,
		Reasoning:   fmt.Sprintf("Selected %s based on load balancing (current load: %d)", bestAgent.AgentID, s.agentLoad[bestAgent.AgentID]),
		APIProvider: "openai",
		Priority:    50,
		Deadline:    time.Now().Add(10 * time.Minute),
	}

	// Store assignment in parent's map with proper locking
	s.MockManagerAgentClient.mu.Lock()
	s.MockManagerAgentClient.assignments[task.ID] = assignment
	s.MockManagerAgentClient.mu.Unlock()
	
	return assignment, nil
}

func (s *smartMockManagerAgent) GetAssignments() map[string]*TaskAssignment {
	s.MockManagerAgentClient.mu.Lock()
	defer s.MockManagerAgentClient.mu.Unlock()
	
	result := make(map[string]*TaskAssignment)
	for k, v := range s.MockManagerAgentClient.assignments {
		result[k] = v
	}
	return result
}