// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scheduler

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lancekrogers/guild-core/pkg/kanban"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentOrchestrator(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		config := &OrchestratorConfig{
			MaxConcurrentTasks:  5,
			DefaultTaskTimeout:  30 * time.Minute,
			ManagerAgentTimeout: 5 * time.Second,
			RateLimitConfigs: map[string]RateLimitConfig{
				"openai": {
					Provider:    "openai",
					MaxRequests: 100,
					Window:      time.Minute,
				},
			},
		}

		managerAgent := NewMockManagerAgentClient()
		kanbanClient := NewMockKanbanClient()

		ao, err := NewAgentOrchestrator(ctx, config, managerAgent, kanbanClient)
		require.NoError(t, err)
		assert.NotNil(t, ao)
		assert.NotNil(t, ao.agentPool)
		assert.NotNil(t, ao.rateLimiters)
		assert.Len(t, ao.rateLimiters, 1)
	})

	t.Run("nil config", func(t *testing.T) {
		managerAgent := NewMockManagerAgentClient()
		kanbanClient := NewMockKanbanClient()

		ao, err := NewAgentOrchestrator(ctx, nil, managerAgent, kanbanClient)
		assert.Error(t, err)
		assert.Nil(t, ao)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("nil manager agent", func(t *testing.T) {
		config := &OrchestratorConfig{MaxConcurrentTasks: 5}
		kanbanClient := NewMockKanbanClient()

		ao, err := NewAgentOrchestrator(ctx, config, nil, kanbanClient)
		assert.Error(t, err)
		assert.Nil(t, ao)
		assert.Contains(t, err.Error(), "managerAgent cannot be nil")
	})

	t.Run("cancelled context", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		config := &OrchestratorConfig{MaxConcurrentTasks: 5}
		managerAgent := NewMockManagerAgentClient()
		kanbanClient := NewMockKanbanClient()

		ao, err := NewAgentOrchestrator(cancelledCtx, config, managerAgent, kanbanClient)
		assert.Error(t, err)
		assert.Nil(t, ao)
		assert.Contains(t, err.Error(), "context cancelled")
	})
}

func TestRegisterAgent(t *testing.T) {
	ctx := context.Background()
	ao := createTestOrchestrator(t)

	t.Run("successful registration", func(t *testing.T) {
		executor := &mockAgentExecutor{
			agentID:      "agent1",
			capabilities: []string{"code", "test"},
			isAvailable:  true,
		}

		err := ao.RegisterAgent(ctx, "agent1", executor, []AgentCapability{CapabilityCode, CapabilityTest})
		require.NoError(t, err)

		// Verify agent was registered
		ao.agentPool.mu.RLock()
		agentInfo, exists := ao.agentPool.agents["agent1"]
		ao.agentPool.mu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, "agent1", agentInfo.AgentID)
		assert.Len(t, agentInfo.Capabilities, 2)
		assert.True(t, agentInfo.IsAvailable)
	})

	t.Run("empty agent ID", func(t *testing.T) {
		executor := &mockAgentExecutor{agentID: ""}

		err := ao.RegisterAgent(ctx, "", executor, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agentID cannot be empty")
	})

	t.Run("nil executor", func(t *testing.T) {
		err := ao.RegisterAgent(ctx, "agent1", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "executor cannot be nil")
	})
}

func TestRequestTaskAssignment(t *testing.T) {
	ctx := context.Background()

	t.Run("successful assignment", func(t *testing.T) {
		ao := createTestOrchestrator(t)
		managerAgent := ao.managerAgent.(*MockManagerAgentClient)

		// Register agents
		registerTestAgents(t, ao, 3)

		// Create test task
		task := &kanban.Task{
			ID:       "task1",
			Title:    "Test Task",
			Priority: kanban.PriorityHigh,
		}

		// Pre-configure assignment
		expectedAssignment := &TaskAssignment{
			TaskID:      "task1",
			AgentID:     "agent1",
			Reasoning:   "Best match for task",
			APIProvider: "openai",
			Priority:    100,
			Deadline:    time.Now().Add(24 * time.Hour),
		}
		managerAgent.SetAssignment("task1", expectedAssignment)

		assignment, err := ao.RequestTaskAssignment(ctx, task)
		require.NoError(t, err)
		assert.NotNil(t, assignment)
		assert.Equal(t, "task1", assignment.TaskID)
		assert.Equal(t, "agent1", assignment.AgentID)

		// Verify request was recorded
		requests := managerAgent.GetRequestsReceived()
		assert.Len(t, requests, 1)
		assert.Equal(t, "task1", requests[0].Task.ID)
		assert.Len(t, requests[0].AvailableAgents, 3)
	})

	t.Run("no available agents", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		task := &kanban.Task{
			ID:    "task1",
			Title: "Test Task",
		}

		assignment, err := ao.RequestTaskAssignment(ctx, task)
		assert.Error(t, err)
		assert.Nil(t, assignment)
		assert.Contains(t, err.Error(), "no available agents")
	})

	t.Run("manager agent failure", func(t *testing.T) {
		ao := createTestOrchestrator(t)
		managerAgent := ao.managerAgent.(*MockManagerAgentClient)

		// Register agent
		registerTestAgents(t, ao, 1)

		// Configure manager to fail
		managerAgent.SetShouldFail(true, "manager unavailable")

		task := &kanban.Task{
			ID:    "task1",
			Title: "Test Task",
		}

		assignment, err := ao.RequestTaskAssignment(ctx, task)
		assert.Error(t, err)
		assert.Nil(t, assignment)
		assert.Contains(t, err.Error(), "manager agent assignment failed")
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		config := &OrchestratorConfig{
			MaxConcurrentTasks: 5,
			RateLimitConfigs: map[string]RateLimitConfig{
				"openai": {
					Provider:    "openai",
					MaxRequests: 1,
					Window:      time.Hour,
				},
			},
		}

		managerAgent := NewMockManagerAgentClient()
		kanbanClient := NewMockKanbanClient()

		ao, err := NewAgentOrchestrator(ctx, config, managerAgent, kanbanClient)
		require.NoError(t, err)

		// Register agent
		registerTestAgents(t, ao, 1)

		// Use up rate limit
		ao.checkRateLimit("openai")

		task := &kanban.Task{
			ID:    "task1",
			Title: "Test Task",
		}

		// Configure assignment with rate-limited provider
		managerAgent.SetAssignment("task1", &TaskAssignment{
			TaskID:      "task1",
			AgentID:     "agent1",
			APIProvider: "openai",
		})

		assignment, err := ao.RequestTaskAssignment(ctx, task)
		assert.Error(t, err)
		assert.Nil(t, assignment)
		assert.Contains(t, err.Error(), "API rate limit exceeded")
	})
}

func TestAssignTask(t *testing.T) {
	ctx := context.Background()

	t.Run("successful atomic assignment", func(t *testing.T) {
		ao := createTestOrchestrator(t)
		kanbanClient := ao.kanbanClient.(*MockKanbanClient)

		// Register agent
		registerTestAgents(t, ao, 1)

		// Add task to kanban
		task := &kanban.Task{
			ID:     "task1",
			Title:  "Test Task",
			Status: kanban.StatusTodo,
		}
		kanbanClient.AddTask(task)

		assignment := &TaskAssignment{
			TaskID:  "task1",
			AgentID: "agent1",
		}

		err := ao.AssignTask(ctx, assignment)
		require.NoError(t, err)

		// Verify assignment was made
		calls := kanbanClient.GetAssignmentCalls()
		assert.Len(t, calls, 1)
		assert.Equal(t, "task1", calls[0].TaskID)
		assert.Equal(t, "agent1", calls[0].AgentID)

		// Verify agent availability updated
		ao.agentPool.mu.RLock()
		agentInfo := ao.agentPool.agents["agent1"]
		ao.agentPool.mu.RUnlock()

		assert.False(t, agentInfo.IsAvailable)
		assert.Equal(t, "task1", agentInfo.CurrentTask)
	})

	t.Run("task already assigned", func(t *testing.T) {
		ao := createTestOrchestrator(t)
		kanbanClient := ao.kanbanClient.(*MockKanbanClient)

		// Add already assigned task
		task := &kanban.Task{
			ID:         "task1",
			Title:      "Test Task",
			AssignedTo: "agent2",
		}
		kanbanClient.AddTask(task)
		kanbanClient.SetSimulateConflict(true)

		assignment := &TaskAssignment{
			TaskID:  "task1",
			AgentID: "agent1",
		}

		err := ao.AssignTask(ctx, assignment)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task already assigned")
	})

	t.Run("nil assignment", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		err := ao.AssignTask(ctx, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "assignment cannot be nil")
	})
}

func TestReleaseAgent(t *testing.T) {
	ctx := context.Background()

	t.Run("successful release with completion", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		// Register and assign agent
		registerTestAgents(t, ao, 1)
		ao.agentPool.mu.Lock()
		agentInfo := ao.agentPool.agents["agent1"]
		agentInfo.IsAvailable = false
		agentInfo.CurrentTask = "task1"
		agentInfo.TasksHandled = 5
		agentInfo.ErrorRate = 0.1
		ao.agentPool.mu.Unlock()

		err := ao.ReleaseAgent(ctx, "agent1", true)
		require.NoError(t, err)

		// Verify agent state
		ao.agentPool.mu.RLock()
		agentInfo = ao.agentPool.agents["agent1"]
		ao.agentPool.mu.RUnlock()

		assert.True(t, agentInfo.IsAvailable)
		assert.Empty(t, agentInfo.CurrentTask)
		assert.Equal(t, 6, agentInfo.TasksHandled)
		assert.Equal(t, 0.1, agentInfo.ErrorRate) // No change on success

		// Verify metrics
		metrics := ao.GetMetrics()
		assert.Equal(t, int64(1), metrics.TasksCompleted)
		assert.Equal(t, int64(0), metrics.TasksFailed)
	})

	t.Run("release with failure", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		// Register agent
		registerTestAgents(t, ao, 1)
		ao.agentPool.mu.Lock()
		agentInfo := ao.agentPool.agents["agent1"]
		agentInfo.IsAvailable = false
		agentInfo.TasksHandled = 4
		agentInfo.ErrorRate = 0.0
		ao.agentPool.mu.Unlock()

		err := ao.ReleaseAgent(ctx, "agent1", false)
		require.NoError(t, err)

		// Verify error rate updated
		ao.agentPool.mu.RLock()
		agentInfo = ao.agentPool.agents["agent1"]
		ao.agentPool.mu.RUnlock()

		assert.True(t, agentInfo.IsAvailable)
		assert.Equal(t, 5, agentInfo.TasksHandled)
		assert.Equal(t, 0.2, agentInfo.ErrorRate) // 1 failure out of 5 tasks

		// Verify metrics
		metrics := ao.GetMetrics()
		assert.Equal(t, int64(0), metrics.TasksCompleted)
		assert.Equal(t, int64(1), metrics.TasksFailed)
	})

	t.Run("agent not found", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		err := ao.ReleaseAgent(ctx, "nonexistent", true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent not found")
	})
}

func TestRateLimiting(t *testing.T) {
	ctx := context.Background()

	t.Run("rate limit enforcement", func(t *testing.T) {
		config := &OrchestratorConfig{
			MaxConcurrentTasks: 5,
			RateLimitConfigs: map[string]RateLimitConfig{
				"openai": {
					Provider:    "openai",
					MaxRequests: 3,
					Window:      100 * time.Millisecond,
				},
			},
		}

		managerAgent := NewMockManagerAgentClient()
		kanbanClient := NewMockKanbanClient()

		ao, err := NewAgentOrchestrator(ctx, config, managerAgent, kanbanClient)
		require.NoError(t, err)

		// Make requests up to limit
		for i := 0; i < 3; i++ {
			assert.True(t, ao.checkRateLimit("openai"))
		}

		// Next request should be rate limited
		assert.False(t, ao.checkRateLimit("openai"))

		// Wait for window to pass
		time.Sleep(150 * time.Millisecond)

		// Should be able to make request again
		assert.True(t, ao.checkRateLimit("openai"))
	})

	t.Run("no limit for unconfigured provider", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		// Should always allow unconfigured providers
		for i := 0; i < 100; i++ {
			assert.True(t, ao.checkRateLimit("anthropic"))
		}
	})
}

func TestConcurrentOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("concurrent agent registration", func(t *testing.T) {
		ao := createTestOrchestrator(t)

		var wg sync.WaitGroup
		errors := make([]error, 10)

		// Register 10 agents concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				executor := &mockAgentExecutor{
					agentID:     fmt.Sprintf("agent%d", idx),
					isAvailable: true,
				}

				errors[idx] = ao.RegisterAgent(ctx, executor.agentID, executor, []AgentCapability{CapabilityCode})
			}(i)
		}

		wg.Wait()

		// Verify all registrations succeeded
		for _, err := range errors {
			assert.NoError(t, err)
		}

		// Verify all agents registered
		ao.agentPool.mu.RLock()
		assert.Len(t, ao.agentPool.agents, 10)
		ao.agentPool.mu.RUnlock()
	})

	t.Run("concurrent task assignments", func(t *testing.T) {
		ao := createTestOrchestrator(t)
		managerAgent := ao.managerAgent.(*MockManagerAgentClient)

		// Register multiple agents
		registerTestAgents(t, ao, 5)

		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		// Try to assign 10 tasks concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				task := &kanban.Task{
					ID:       fmt.Sprintf("task%d", idx),
					Title:    fmt.Sprintf("Test Task %d", idx),
					Priority: kanban.PriorityMedium,
				}

				// Pre-configure assignment
				managerAgent.SetAssignment(task.ID, &TaskAssignment{
					TaskID:      task.ID,
					AgentID:     fmt.Sprintf("agent%d", (idx%5)+1),
					APIProvider: "openai",
				})

				_, err := ao.RequestTaskAssignment(ctx, task)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		// Should have successful assignments
		assert.Greater(t, successCount, 0)

		// Verify metrics
		metrics := ao.GetMetrics()
		assert.Equal(t, int64(successCount), metrics.TasksAssigned)
	})
}

// Helper functions

func createTestOrchestrator(t *testing.T) *AgentOrchestrator {
	config := &OrchestratorConfig{
		MaxConcurrentTasks:  10,
		DefaultTaskTimeout:  30 * time.Minute,
		ManagerAgentTimeout: 5 * time.Second,
		RateLimitConfigs: map[string]RateLimitConfig{
			"openai": {
				Provider:    "openai",
				MaxRequests: 100,
				Window:      time.Minute,
			},
		},
	}

	managerAgent := NewMockManagerAgentClient()
	kanbanClient := NewMockKanbanClient()

	ao, err := NewAgentOrchestrator(context.Background(), config, managerAgent, kanbanClient)
	require.NoError(t, err)

	return ao
}

func registerTestAgents(t *testing.T, ao *AgentOrchestrator, count int) {
	for i := 0; i < count; i++ {
		executor := &mockAgentExecutor{
			agentID:      fmt.Sprintf("agent%d", i+1), // Start from agent1
			capabilities: []string{"code", "test"},
			isAvailable:  true,
		}

		err := ao.RegisterAgent(context.Background(), executor.agentID, executor,
			[]AgentCapability{CapabilityCode, CapabilityTest})
		require.NoError(t, err)
	}
}
