package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/commission"
	"github.com/guild-ventures/guild-core/pkg/orchestrator/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrchestratorLifecycle tests the basic lifecycle operations
func TestOrchestratorLifecycle(t *testing.T) {
	// Create test dependencies
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{
		MaxConcurrentAgents: 5,
		ManagerAgentID:      "test-manager",
		KanbanBoardID:       "test-board",
	}

	// Create orchestrator
	orch := newOrchestrator(&config, dispatcher, eventBus)
	require.NotNil(t, orch)

	// Test initial status
	assert.Equal(t, StatusIdle, orch.Status())

	// Test Start
	ctx := context.Background()
	err := orch.Start(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, orch.Status())

	// Test Start when already running
	err = orch.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test Pause
	err = orch.Pause(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StatusPaused, orch.Status())

	// Test Pause when already paused
	err = orch.Pause(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	// Test Resume
	err = orch.Resume(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, orch.Status())

	// Test Resume when not paused
	err = orch.Resume(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not paused")

	// Test Stop
	err = orch.Stop(ctx)
	assert.NoError(t, err)
	assert.Equal(t, StatusIdle, orch.Status())

	// Test Stop when already stopped
	err = orch.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestOrchestratorAgentManagement tests agent management operations
func TestOrchestratorAgentManagement(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{
		MaxConcurrentAgents: 5,
	}

	orch := newOrchestrator(&config, dispatcher, eventBus)

	// Test AddAgent
	agent1 := &simpleTestAgent{id: "agent1", name: "Agent 1"}
	err := orch.AddAgent(agent1)
	assert.NoError(t, err)

	// Test GetAgent
	retrieved, exists := orch.GetAgent("agent1")
	assert.True(t, exists)
	assert.Equal(t, agent1, retrieved)

	// Test GetAgent for non-existent agent
	_, exists = orch.GetAgent("nonexistent")
	assert.False(t, exists)

	// Test AddAgent with duplicate ID
	agent2 := &simpleTestAgent{id: "agent1", name: "Duplicate"}
	err = orch.AddAgent(agent2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test RemoveAgent
	err = orch.RemoveAgent("agent1")
	assert.NoError(t, err)

	// Verify agent was removed
	_, exists = orch.GetAgent("agent1")
	assert.False(t, exists)

	// Test RemoveAgent for non-existent agent
	err = orch.RemoveAgent("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestOrchestratorObjectiveManagement tests objective management
func TestOrchestratorObjectiveManagement(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{}

	orch := newOrchestrator(&config, dispatcher, eventBus)

	// Test initial state - no objective
	assert.Nil(t, orch.GetObjective())

	// Test SetObjective
	objective := &commission.Commission{
		ID:          "obj1",
		Title:       "Test Objective",
		Description: "Test Description",
	}

	err := orch.SetObjective(objective)
	assert.NoError(t, err)

	// Test GetObjective
	retrieved := orch.GetObjective()
	assert.NotNil(t, retrieved)
	assert.Equal(t, objective.ID, retrieved.ID)
	assert.Equal(t, objective.Title, retrieved.Title)

	// Test SetObjective while running (current implementation allows this)
	ctx := context.Background()
	err = orch.Start(ctx)
	require.NoError(t, err)

	newObjective := &commission.Commission{
		ID:    "obj2",
		Title: "New Objective",
	}
	err = orch.SetObjective(newObjective)
	assert.NoError(t, err) // Current implementation allows setting objective while running
	
	// Verify new objective was set
	retrieved = orch.GetObjective()
	assert.NotNil(t, retrieved)
	assert.Equal(t, newObjective.ID, retrieved.ID)

	// Stop orchestrator
	err = orch.Stop(ctx)
	require.NoError(t, err)
}

// TestOrchestratorEventHandling tests event handler management
func TestOrchestratorEventHandling(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{}

	orch := newOrchestrator(&config, dispatcher, eventBus)

	// Test AddEventHandler
	handler := func(event interfaces.Event) {
		// Handler would be called in a real implementation
	}

	orch.AddEventHandler(interfaces.EventHandler(handler))

	// Test EmitEvent
	event := interfaces.Event{
		Type: "test",
		Data: map[string]interface{}{"message": "test"},
	}
	orch.EmitEvent(event)

	// Note: With our simple mock, we can't verify the handler was called
	// In a real test, we'd use a more sophisticated mock
}

// TestDefaultOrchestratorFactory tests the factory function
func TestDefaultOrchestratorFactory(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{
		MaxConcurrentAgents: 10,
		ManagerAgentID:      "manager",
		KanbanBoardID:       "board",
	}

	orch := DefaultOrchestratorFactory(&config, dispatcher, eventBus)
	require.NotNil(t, orch)

	// Verify it's a BaseOrchestrator
	assert.IsType(t, &BaseOrchestrator{}, orch)

	// Verify initial status
	assert.Equal(t, StatusIdle, orch.Status())
}

// TestOrchestratorConcurrentOperations tests concurrent access
func TestOrchestratorConcurrentOperations(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{
		MaxConcurrentAgents: 5,
	}

	orch := newOrchestrator(&config, dispatcher, eventBus)

	// Add multiple agents concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			agent := &simpleTestAgent{
				id:   string(rune('a' + id)),
				name: string(rune('A' + id)),
			}
			orch.AddAgent(agent)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify agents were added (at least some should succeed)
	agentCount := 0
	for i := 0; i < 10; i++ {
		if _, exists := orch.GetAgent(string(rune('a' + i))); exists {
			agentCount++
		}
	}
	assert.Greater(t, agentCount, 0)
}

// TestOrchestratorStopWithTimeout tests stopping with context timeout
func TestOrchestratorStopWithTimeout(t *testing.T) {
	dispatcher := &simpleTestDispatcher{}
	eventBus := &simpleTestEventBus{}
	config := Config{}

	orch := newOrchestrator(&config, dispatcher, eventBus)

	// Start orchestrator
	ctx := context.Background()
	err := orch.Start(ctx)
	require.NoError(t, err)

	// Create a context with very short timeout
	stopCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	// Stop should still succeed (our mock dispatcher stops immediately)
	err = orch.Stop(stopCtx)
	assert.NoError(t, err)
	assert.Equal(t, StatusIdle, orch.Status())
}

// TestOrchestratorErrorStatus tests error status handling
func TestOrchestratorErrorStatus(t *testing.T) {
	// This test would require a dispatcher that can simulate errors
	// For now, we just verify the error status constant exists
	assert.Equal(t, Status("error"), StatusError)
}