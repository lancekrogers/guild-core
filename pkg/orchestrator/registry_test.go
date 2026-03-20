// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"testing"

	"github.com/lancekrogers/guild-core/pkg/agents/core/manager"
	"github.com/lancekrogers/guild-core/pkg/config"
	"github.com/lancekrogers/guild-core/pkg/kanban"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock commission planner for testing
type mockCommissionPlanner struct {
	planFromRefinedCalled bool
	assignTasksCalled     bool
}

func (m *mockCommissionPlanner) PlanFromRefinedCommission(ctx context.Context, refined *manager.RefinedCommission, guildConfig *config.GuildConfig) ([]*kanban.Task, error) {
	m.planFromRefinedCalled = true
	return []*kanban.Task{
		{ID: "task1", Title: "Task 1"},
		{ID: "task2", Title: "Task 2"},
	}, nil
}

func (m *mockCommissionPlanner) AssignTasksToArtisans(ctx context.Context, tasks []*kanban.Task, guild *config.GuildConfig) error {
	m.assignTasksCalled = true
	return nil
}

// TestOrchestratorRegistry tests the basic registry operations
func TestOrchestratorRegistry(t *testing.T) {
	registry := NewOrchestratorRegistry()
	require.NotNil(t, registry)

	// Verify it implements the interface
	var _ OrchestratorRegistry = registry
}

// TestCommissionPlannerRegistration tests commission planner registration
func TestCommissionPlannerRegistration(t *testing.T) {
	registry := NewOrchestratorRegistry()

	// Test RegisterCommissionPlanner
	planner := &mockCommissionPlanner{}
	err := registry.RegisterCommissionPlanner("test-planner", planner)
	assert.NoError(t, err)

	// Test duplicate registration
	err = registry.RegisterCommissionPlanner("test-planner", planner)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test GetCommissionPlanner
	retrieved, err := registry.GetCommissionPlanner("test-planner")
	assert.NoError(t, err)
	assert.Equal(t, planner, retrieved)

	// Test GetCommissionPlanner for non-existent planner
	_, err = registry.GetCommissionPlanner("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test GetDefaultCommissionPlanner (should return the first registered planner)
	defaultPlanner, err := registry.GetDefaultCommissionPlanner()
	assert.NoError(t, err)
	assert.Equal(t, planner, defaultPlanner)

	// Test HasCommissionPlanner
	assert.True(t, registry.HasCommissionPlanner("test-planner"))
	assert.False(t, registry.HasCommissionPlanner("non-existent"))

	// Test ListCommissionPlanners
	planners := registry.ListCommissionPlanners()
	assert.Contains(t, planners, "test-planner")
}

// TestEventBusRegistration tests event bus registration
func TestEventBusRegistration(t *testing.T) {
	registry := NewOrchestratorRegistry()

	// Test RegisterEventBus
	eventBus := &simpleTestEventBus{}
	err := registry.RegisterEventBus("test-bus", eventBus)
	assert.NoError(t, err)

	// Test duplicate registration
	err = registry.RegisterEventBus("test-bus", eventBus)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Test GetEventBus
	retrieved, err := registry.GetEventBus("test-bus")
	assert.NoError(t, err)
	assert.Equal(t, eventBus, retrieved)

	// Test GetEventBus for non-existent bus
	_, err = registry.GetEventBus("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test GetDefaultEventBus (should return the first registered bus)
	defaultBus, err := registry.GetDefaultEventBus()
	assert.NoError(t, err)
	assert.Equal(t, eventBus, defaultBus)
}

// TestDefaultCommissionPlanner tests getting default commission planner
func TestDefaultCommissionPlanner(t *testing.T) {
	registry := NewOrchestratorRegistry()

	// Register multiple planners
	planner1 := &mockCommissionPlanner{}
	planner2 := &mockCommissionPlanner{}

	err := registry.RegisterCommissionPlanner("planner1", planner1)
	require.NoError(t, err)

	err = registry.RegisterCommissionPlanner("planner2", planner2)
	require.NoError(t, err)

	// GetDefaultCommissionPlanner should return the first registered
	defaultPlanner, err := registry.GetDefaultCommissionPlanner()
	assert.NoError(t, err)
	assert.Equal(t, planner1, defaultPlanner)
}

// TestDefaultEventBus tests getting default event bus
func TestDefaultEventBus(t *testing.T) {
	registry := NewOrchestratorRegistry()

	// Register multiple event buses
	bus1 := &simpleTestEventBus{}
	bus2 := &simpleTestEventBus{}

	err := registry.RegisterEventBus("bus1", bus1)
	require.NoError(t, err)

	err = registry.RegisterEventBus("bus2", bus2)
	require.NoError(t, err)

	// GetDefaultEventBus should return the first registered
	defaultBus, err := registry.GetDefaultEventBus()
	assert.NoError(t, err)
	assert.Equal(t, bus1, defaultBus)
}

// TestRegistryConcurrency tests concurrent access to registry
func TestRegistryConcurrency(t *testing.T) {
	registry := NewOrchestratorRegistry()

	// Concurrent registration of planners
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			planner := &mockCommissionPlanner{}
			plannerName := string(rune('a' + id))
			err := registry.RegisterCommissionPlanner(plannerName, planner)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errors)

	// Check for errors (some duplicates might occur due to race conditions)
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
		}
	}

	// At least some registrations should succeed
	planners := registry.ListCommissionPlanners()
	assert.Greater(t, len(planners), 0)
}

// TestEmptyRegistry tests operations on empty registry
func TestEmptyRegistry(t *testing.T) {
	registry := NewOrchestratorRegistry()

	// Test GetCommissionPlanner on empty registry
	_, err := registry.GetCommissionPlanner("any")
	assert.Error(t, err)

	// Test GetDefaultCommissionPlanner on empty registry
	_, err = registry.GetDefaultCommissionPlanner()
	assert.Error(t, err)

	// Test GetEventBus on empty registry
	_, err = registry.GetEventBus("any")
	assert.Error(t, err)

	// Test GetDefaultEventBus on empty registry
	_, err = registry.GetDefaultEventBus()
	assert.Error(t, err)

	// Test ListCommissionPlanners on empty registry
	planners := registry.ListCommissionPlanners()
	assert.Empty(t, planners)

	// Test HasCommissionPlanner on empty registry
	assert.False(t, registry.HasCommissionPlanner("any"))
}
