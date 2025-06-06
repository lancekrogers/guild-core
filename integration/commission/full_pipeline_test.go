// +build integration

package commission_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/internal/config"
	"github.com/guild-ventures/guild-core/internal/kanban"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/internal/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullCommissionRefinementPipeline tests the complete commission refinement pipeline
func TestFullCommissionRefinementPipeline(t *testing.T) {
	// Setup test directory
	tempDir, err := os.MkdirTemp("", "guild-full-pipeline-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fmt.Printf("🏰 Starting FULL commission refinement pipeline test\n")
	fmt.Printf("📁 Test directory: %s\n", tempDir)

	// Step 1: Initialize component registry
	fmt.Printf("\n🔧 Step 1: Initializing component registry\n")
	
	reg := registry.NewComponentRegistry()
	ctx := context.Background()
	
	// Step 2: Setup mock provider with intelligent responses FIRST
	fmt.Printf("\n🤖 Step 2: Configuring mock AI provider\n")
	
	mockProvider := mock.NewProvider()
	
	// Configure mock to return file structure with proper ## File: format
	// This matches what the parser expects - a structured file response
	mockProvider.SetResponse("You are the Guild Master", `## File: commission_refined.md

# E-Commerce Platform Development

## Overview
Building a modern e-commerce platform requires coordinated effort across multiple domains. This commission breaks down the work into specialized tasks for our guild artisans.

## Task Breakdown

- BACKEND-001: Design Database Schema (priority: high, estimate: 6h)
- BACKEND-002: Implement Product Catalog API (priority: high, estimate: 8h, depends: BACKEND-001)
- BACKEND-003: Build Shopping Cart Service (priority: high, estimate: 6h, depends: BACKEND-001)
- FRONTEND-004: Create Product Listing UI (priority: medium, estimate: 8h, depends: BACKEND-002)
- FULLSTACK-005: Implement Checkout Flow (priority: high, estimate: 10h, depends: BACKEND-003;FRONTEND-004)

## Implementation Notes
- Use PostgreSQL for data persistence
- Implement Redis for session and cart caching
- Follow RESTful API design principles
- Ensure mobile-first responsive design
- Include comprehensive error handling and logging

## File: README.md

# E-Commerce Platform Development

This project implements a full-featured e-commerce platform.

## Features
- Product catalog with search
- Shopping cart functionality
- Secure checkout process
- Order management
- Responsive design

## Technology Stack
- Backend: Go with PostgreSQL
- Frontend: React with responsive design
- Caching: Redis for sessions and cart
- Architecture: RESTful API design`)

	// Register the mock provider
	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)
	fmt.Printf("  ✓ Mock provider configured with intelligent responses\n")

	// Step 3: Setup BoltDB with all required buckets BEFORE initializing registry
	fmt.Printf("\n💾 Step 3: Setting up database storage\n")
	
	dbPath := filepath.Join(tempDir, "guild.db")
	customBuckets := []string{
		"objectives",
		"tasks",
		"board_metadata", 
		"task_metadata",
		"task_events",
		"task_comments",
		"tasks_by_board_status",
		"board_events",
		"tasks_by_board",
	}
	
	store, err := boltdb.NewStore(dbPath, boltdb.WithCustomBuckets(customBuckets...))
	require.NoError(t, err)
	defer store.Close()
	fmt.Printf("  ✓ BoltDB initialized with all buckets\n")
	
	// Register the memory store with the registry
	err = reg.Memory().RegisterMemoryStore("boltdb", store)
	require.NoError(t, err)
	fmt.Printf("  ✓ Memory store registered\n")
	
	// Initialize registry with config AFTER registering providers and memory
	registryConfig := registry.Config{
		Agents: registry.AgentConfig{
			DefaultType: "worker",
		},
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
		Memory: registry.MemoryConfig{
			DefaultMemoryStore: "boltdb",
		},
	}
	err = reg.Initialize(ctx, registryConfig)
	require.NoError(t, err)
	fmt.Printf("  ✓ Component registry initialized\n")

	// Step 4: Initialize kanban system
	fmt.Printf("\n📋 Step 4: Setting up kanban board\n")
	
	kanbanMgr, err := kanban.NewManager(store)
	require.NoError(t, err)
	
	board, err := kanbanMgr.CreateBoard(ctx, "test-board", "Test Commission Board")
	require.NoError(t, err)
	fmt.Printf("  ✓ Kanban board created: %s\n", board.Name)

	// Create kanban manager adapter - use existing KanbanManager implementation
	kanbanManager := orchestrator.NewDefaultKanbanManager(board)

	// Step 5: Setup prompts system
	fmt.Printf("\n📝 Step 5: Initializing prompt system\n")
	
	// The CommissionIntegrationService will create its own prompt registry with built-in prompts
	// This is acceptable for the MVP since the integration service has built-in prompts
	fmt.Printf("  ✓ Prompt system ready (service will use built-in prompts)\n")

	// Step 6: Create commission integration service
	fmt.Printf("\n🔗 Step 6: Creating commission integration service\n")
	
	integrationService, err := orchestrator.NewCommissionIntegrationService(reg)
	require.NoError(t, err)
	fmt.Printf("  ✓ Integration service created\n")

	// Step 7: Create test commission
	fmt.Printf("\n📜 Step 7: Creating test commission\n")
	
	commission := manager.Commission{
		ID:          "test-commission-001",
		Title:       "E-Commerce Platform Development",
		Description: "Build a full-featured e-commerce platform with product catalog, shopping cart, and checkout functionality",
		Domain:      "web-development",
		Context: map[string]interface{}{
			"technology_stack": []string{"Go", "PostgreSQL", "React", "Redis"},
			"requirements": []string{
				"Product catalog with search",
				"Shopping cart functionality",
				"Secure checkout process",
				"Order management",
				"Responsive design",
			},
			"target_users": "Small to medium businesses",
			"timeline":     "4 weeks",
		},
	}
	fmt.Printf("  ✓ Created commission: %s\n", commission.Title)

	// Step 8: Create guild configuration with specialized artisans
	fmt.Printf("\n⚙️ Step 8: Configuring guild with specialized artisans\n")
	
	guildConfig := &config.GuildConfig{
		Name:    "Elite Development Guild",
		Version: "1.0.0",
		Agents: []config.AgentConfig{
			{
				ID:            "backend-master",
				Name:          "Backend Master Artisan",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"go", "api", "database", "postgresql", "redis"},
				Tools:         []string{"file", "shell", "git"},
				CostMagnitude: 5,
			},
			{
				ID:            "frontend-wizard",
				Name:          "Frontend Wizard",
				Type:          "specialist", 
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"react", "typescript", "ui", "ux", "responsive"},
				Tools:         []string{"file", "shell", "npm"},
				CostMagnitude: 5,
			},
			{
				ID:            "fullstack-generalist",
				Name:          "Fullstack Generalist",
				Type:          "worker",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"fullstack", "api", "frontend", "backend", "testing"},
				Tools:         []string{"file", "shell", "git", "docker"},
				CostMagnitude: 3,
			},
		},
		Manager: config.ManagerConfig{
			Default: "backend-master",
		},
	}
	fmt.Printf("  ✓ Configured guild with %d specialized artisans\n", len(guildConfig.Agents))

	// Step 9: Process commission through full pipeline
	fmt.Printf("\n🚀 Step 9: Processing commission through FULL refinement pipeline\n")
	
	startTime := time.Now()
	result, err := integrationService.ProcessCommissionToTasks(ctx, commission, guildConfig)
	processingTime := time.Since(startTime)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	fmt.Printf("  ✓ Commission processed in %v\n", processingTime)

	// Step 10: Verify comprehensive results
	fmt.Printf("\n✅ Step 10: Verifying pipeline results\n")
	
	// Check refined commission
	assert.NotNil(t, result.RefinedCommission)
	assert.Equal(t, commission.ID, result.RefinedCommission.CommissionID)
	assert.NotNil(t, result.RefinedCommission.Structure)
	fmt.Printf("  ✓ Commission refined successfully\n")
	
	// Check file structure was created
	assert.NotNil(t, result.RefinedCommission.Structure)
	assert.NotEmpty(t, result.RefinedCommission.Structure.Files)
	fmt.Printf("  ✓ File structure generated with %d files\n", len(result.RefinedCommission.Structure.Files))
	
	// Verify files were written to disk
	objectivesDir := filepath.Join(tempDir, ".guild", "objectives", "refined", commission.ID)
	entries, err := os.ReadDir(objectivesDir)
	if err == nil {
		fmt.Printf("  ✓ Refined files written to disk: %d files\n", len(entries))
		for _, entry := range entries {
			fmt.Printf("    - %s\n", entry.Name())
		}
	}
	
	// Check tasks were created
	assert.Equal(t, 5, len(result.Tasks), "Should create 5 tasks from mock response")
	fmt.Printf("  ✓ Created %d kanban tasks\n", len(result.Tasks))
	
	// Verify task details and assignments
	for i, task := range result.Tasks {
		assert.NotEmpty(t, task.ID)
		assert.NotEmpty(t, task.Title)
		assert.NotEmpty(t, task.Description)
		assert.Equal(t, kanban.StatusBacklog, task.Status)
		assert.NotEmpty(t, task.AssignedTo, "Task should be assigned to an artisan")
		
		// Check metadata
		assert.Contains(t, task.Metadata, "commission_id")
		assert.Equal(t, commission.ID, task.Metadata["commission_id"])
		assert.Contains(t, task.Metadata, "required_capabilities")
		assert.Contains(t, task.Metadata, "original_category")
		
		fmt.Printf("    Task %d: %s\n", i+1, task.Title)
		fmt.Printf("      - Priority: %s\n", task.Priority)
		fmt.Printf("      - Assigned: %s\n", task.AssignedTo)
		fmt.Printf("      - Category: %s\n", task.Metadata["original_category"])
		fmt.Printf("      - Estimate: %.1f hours\n", task.EstimatedHours)
		
		// Verify task exists in kanban
		kanbanTask, err := kanbanManager.GetTask(ctx, task.ID)
		assert.NoError(t, err)
		assert.Equal(t, task.Title, kanbanTask.Title)
	}
	
	// Check artisan assignments match capabilities
	backendTasks := result.GetTasksByArtisan("backend-master")
	frontendTasks := result.GetTasksByArtisan("frontend-wizard")
	fullstackTasks := result.GetTasksByArtisan("fullstack-generalist")
	
	fmt.Printf("\n  📊 Task Distribution:\n")
	fmt.Printf("    - Backend Master: %d tasks\n", len(backendTasks))
	fmt.Printf("    - Frontend Wizard: %d tasks\n", len(frontendTasks))
	fmt.Printf("    - Fullstack Generalist: %d tasks\n", len(fullstackTasks))
	
	// Verify intelligent assignment based on capabilities
	for _, task := range backendTasks {
		category := task.Metadata["original_category"]
		assert.Contains(t, []string{"backend", "database"}, category, 
			"Backend tasks should be assigned to backend master")
	}
	
	for _, task := range frontendTasks {
		category := task.Metadata["original_category"]
		assert.Equal(t, "frontend", category,
			"Frontend tasks should be assigned to frontend wizard")
	}
	
	// Check completion metrics
	assert.Equal(t, 5, result.GetTaskCount())
	assert.Equal(t, 3, result.GetAssignedArtisanCount())
	
	// Verify task dependencies were preserved
	checkoutTask := findTaskByTitle(result.Tasks, "Implement Checkout Flow")
	if checkoutTask != nil {
		assert.Len(t, checkoutTask.Dependencies, 2, "Checkout should depend on 2 tasks")
		fmt.Printf("  ✓ Task dependencies preserved\n")
	}
	
	fmt.Printf("\n🎉 FULL commission refinement pipeline completed successfully!\n")
	fmt.Printf("   The Guild Framework MVP is ready to orchestrate complex work!\n")
}

// TestCommissionPipelineErrorHandling tests error scenarios in the pipeline
func TestCommissionPipelineErrorHandling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-error-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	reg := registry.NewComponentRegistry()
	reg.Initialize(ctx, registry.Config{})

	// Test 1: Missing provider
	service, err := orchestrator.NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	commission := manager.Commission{
		Title: "Test Commission",
		Description: "Test",
	}
	
	guildConfig := &config.GuildConfig{
		Name: "Test Guild",
		Agents: []config.AgentConfig{
			{ID: "test", Provider: "nonexistent"},
		},
	}

	_, err = service.ProcessCommissionToTasks(ctx, commission, guildConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider")

	// Test 2: Empty agent list
	emptyGuildConfig := &config.GuildConfig{
		Name: "Empty Guild",
		Agents: []config.AgentConfig{},
	}

	// Register a mock provider
	mockProvider := mock.NewProvider()
	mockProvider.SetResponse("", "<task><id>t1</id><title>Task</title></task>")
	reg.Providers().RegisterProvider("mock", mockProvider)

	_, err = service.ProcessCommissionToTasks(ctx, commission, emptyGuildConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agents configured")
}

// TestLayeredPromptIntegration verifies the layered prompt system works with commission refinement
func TestLayeredPromptIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "guild-prompt-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	reg := registry.NewComponentRegistry()
	reg.Initialize(ctx, registry.Config{})

	// Setup mock provider that echoes the prompt to verify layers
	mockProvider := mock.NewProvider()
	var capturedPrompt string
	// Mock provider doesn't have SetResponseFunc, use SetResponse instead
	mockProvider.SetResponse("", "<task><id>test</id><title>Test Task</title></task>")
	reg.Providers().RegisterProvider("mock", mockProvider)

	// Create service and process commission
	service, _ := orchestrator.NewCommissionIntegrationService(reg)
	
	commission := manager.Commission{
		Title: "Test Layered Prompts",
		Description: "Verify prompt layers",
		Context: map[string]interface{}{
			"project_type": "golang",
			"requirements": []string{"testing", "documentation"},
		},
	}

	guildConfig := &config.GuildConfig{
		Name: "Prompt Test Guild",
		Agents: []config.AgentConfig{
			{ID: "test", Provider: "mock", Type: "manager"},
		},
	}

	_, err = service.ProcessCommissionToTasks(ctx, commission, guildConfig)
	require.NoError(t, err)

	// Verify prompt contains expected layers
	assert.Contains(t, capturedPrompt, "Guild Master")
	assert.Contains(t, capturedPrompt, "commission")
	assert.Contains(t, capturedPrompt, commission.Title)
	assert.Contains(t, capturedPrompt, commission.Description)
}

// Helper function to find task by title
func findTaskByTitle(tasks []*kanban.Task, title string) *kanban.Task {
	for _, task := range tasks {
		if task.Title == title {
			return task
		}
	}
	return nil
}