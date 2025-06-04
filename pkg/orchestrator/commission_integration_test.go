package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/memory/mocks"
	// "github.com/guild-ventures/guild-core/pkg/objective"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommissionIntegrationService_FullPipeline tests the complete commission refinement pipeline
func TestCommissionIntegrationService_FullPipeline(t *testing.T) {
	ctx := context.Background()

	// Create a test registry
	reg := registry.NewComponentRegistry()

	// Set up mock provider
	mockProvider := mock.NewProvider()
	mockProvider.SetResponse(`## File: README.md

# Task Management System

A comprehensive web-based task management system.

## Implementation Tasks

- BACKEND-001: Set up Node.js server with Express (priority: high, estimate: 4h)
- BACKEND-002: Design database schema for tasks (priority: high, estimate: 3h, depends: BACKEND-001)
- BACKEND-003: Implement authentication system (priority: high, estimate: 6h)
- FRONTEND-001: Create React application structure (priority: medium, estimate: 2h)
- FRONTEND-002: Build task list component (priority: medium, estimate: 4h, depends: FRONTEND-001)
- API-001: Design RESTful API endpoints (priority: high, estimate: 3h)
- TEST-001: Set up testing framework (priority: medium, estimate: 2h)

## File: tasks/backend_tasks.md

# Backend Development Tasks

Detailed tasks for backend implementation...`)

	err := reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)
	reg.Providers().SetDefaultProvider("mock")

	// Set up memory
	memStore := mocks.NewMockStore()
	err = reg.Memory().RegisterMemoryStore("default", memStore)
	require.NoError(t, err)

	// Set up prompts
	promptRegistry := prompts.NewMemoryRegistry()
	promptManager := prompts.NewDefaultManager(promptRegistry, nil)
	// Register prompt with registry directly
	promptRegistry.RegisterPrompt("manager", "default", "Test prompt")
	require.NoError(t, err)

	// Register a test prompt
	promptManager.Registry.RegisterPrompt("manager", "default", "You are a Guild Master refining commissions into tasks.")

	// Create integration service
	service, err := NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	// Create test guild config
	guildConfig := &config.GuildConfig{
		Name: "Test Guild",
		Agents: []config.AgentConfig{
			{
				ID:           "backend-artisan",
				Name:         "Backend Artisan",
				Type:         "specialist",
				Capabilities: []string{"backend", "api"},
			},
			{
				ID:           "frontend-artisan",
				Name:         "Frontend Artisan",
				Type:         "specialist",
				Capabilities: []string{"frontend", "ui"},
			},
			{
				ID:           "test-artisan",
				Name:         "Test Artisan",
				Type:         "specialist",
				Capabilities: []string{"test", "qa"},
			},
		},
	}

	// Create test commission
	commission := manager.Commission{
		ID:          "test-001",
		Title:       "Build Task Management System",
		Description: "Create a web-based task management system with user authentication",
		Domain:      "web-app",
		Context: map[string]interface{}{
			"tech_stack": "React, Node.js",
		},
	}

	// Process commission
	result, err := service.ProcessCommissionToTasks(ctx, commission, guildConfig)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify results
	assert.Equal(t, commission.ID, result.Commission.ID)
	assert.NotNil(t, result.RefinedCommission)
	assert.Greater(t, len(result.Tasks), 0)

	// Check that tasks were created
	assert.GreaterOrEqual(t, len(result.Tasks), 7, "Should have created at least 7 tasks")

	// Verify task properties
	foundBackendTask := false
	foundFrontendTask := false
	for _, task := range result.Tasks {
		assert.NotEmpty(t, task.ID)
		assert.NotEmpty(t, task.Title)
		assert.NotEmpty(t, task.AssignedTo)

		// Check assignments
		if task.Metadata["category"] == "BACKEND" {
			foundBackendTask = true
			assert.Equal(t, "backend-artisan", task.AssignedTo)
		} else if task.Metadata["category"] == "FRONTEND" {
			foundFrontendTask = true
			assert.Equal(t, "frontend-artisan", task.AssignedTo)
		}
	}

	assert.True(t, foundBackendTask, "Should have found backend tasks")
	assert.True(t, foundFrontendTask, "Should have found frontend tasks")

	// Check assigned artisans
	assert.GreaterOrEqual(t, len(result.AssignedArtisans), 2)
}

// TestCommissionIntegrationService_DirectRefiner tests using the refiner directly
func TestCommissionIntegrationService_DirectRefiner(t *testing.T) {
	ctx := context.Background()

	// Set up test infrastructure
	reg := setupTestRegistry(t)
	service, err := NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	// Get the guild master factory
	factory := service.GetGuildMasterFactory()
	require.NotNil(t, factory)

	// Create a refiner
	refiner, err := factory.CreateCommissionRefinerWithDefaults()
	require.NoError(t, err)
	require.NotNil(t, refiner)

	// Test simple refinement
	guildMasterRefiner, ok := refiner.(*manager.GuildMasterRefiner)
	require.True(t, ok)

	refinedContent, err := guildMasterRefiner.RefineCommissionSimple(
		ctx,
		"Create a REST API for user management",
		"microservice",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, refinedContent)
	assert.Contains(t, refinedContent, "API")
}

// TestCommissionIntegrationService_TaskBridge tests the task bridge functionality
func TestCommissionIntegrationService_TaskBridge(t *testing.T) {
	ctx := context.Background()

	// Set up test infrastructure
	reg := setupTestRegistry(t)
	service, err := NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	// Get task bridge
	taskBridge := service.GetTaskBridge()
	require.NotNil(t, taskBridge)

	// Create tasks from refined content
	taskIDs, err := taskBridge.CreateTasksFromRefinedContent(
		ctx,
		"test-commission-001",
		`# Test Commission

## Tasks

- TASK-001: First test task
- TASK-002: Second test task`,
	)
	require.NoError(t, err)
	assert.Len(t, taskIDs, 2)
}

// setupTestRegistry creates a test registry with all required components
func setupTestRegistry(t *testing.T) registry.ComponentRegistry {
	reg := registry.NewComponentRegistry()

	// Mock provider
	mockProvider := mock.NewProvider()
	mockProvider.SetResponse(`- TASK-001: Test task`)
	require.NoError(t, reg.Providers().RegisterProvider("mock", mockProvider))
	reg.Providers().SetDefaultProvider("mock")

	// Memory
	memStore := mocks.NewMockStore()
	require.NoError(t, reg.Memory().RegisterStore("default", memStore))

	// Prompts
	promptRegistry := prompts.NewMemoryRegistry()
	promptManager := prompts.NewDefaultManager(promptRegistry, nil)
	promptManager.Registry.RegisterPrompt("manager", "default", "Test prompt")
	// Prompt registry doesn't have RegisterManager

	return reg
}

// TestCommissionProcessingResult_Methods tests the result helper methods
func TestCommissionProcessingResult_Methods(t *testing.T) {
	result := &CommissionProcessingResult{
		Commission: manager.Commission{
			ID:    "test-001",
			Title: "Test Commission",
		},
		Tasks: []*kanban.Task{
			{
				ID:         "task-1",
				Title:      "Backend Task",
				Status:     kanban.StatusTodo,
				AssignedTo: "backend-artisan",
			},
			{
				ID:         "task-2",
				Title:      "Frontend Task",
				Status:     kanban.StatusInProgress,
				AssignedTo: "frontend-artisan",
			},
			{
				ID:         "task-3",
				Title:      "Another Backend Task",
				Status:     kanban.StatusTodo,
				AssignedTo: "backend-artisan",
			},
		},
		AssignedArtisans: []string{"backend-artisan", "frontend-artisan"},
	}

	// Test GetTasksByStatus
	todoTasks := result.GetTasksByStatus(kanban.StatusTodo)
	assert.Len(t, todoTasks, 2)

	inProgressTasks := result.GetTasksByStatus(kanban.StatusInProgress)
	assert.Len(t, inProgressTasks, 1)

	// Test GetTasksByArtisan
	backendTasks := result.GetTasksByArtisan("backend-artisan")
	assert.Len(t, backendTasks, 2)

	frontendTasks := result.GetTasksByArtisan("frontend-artisan")
	assert.Len(t, frontendTasks, 1)

	// Test counts
	assert.Equal(t, 3, result.GetTaskCount())
	assert.Equal(t, 2, result.GetAssignedArtisanCount())
}