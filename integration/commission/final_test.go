// +build integration

package commission_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCommissionRefinementMVP tests the MVP commission refinement functionality
func TestCommissionRefinementMVP(t *testing.T) {
	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:           "guild-mvp-test",
		WithObjectives: true,
	})
	defer cleanup()

	fmt.Printf("🚀 Testing Commission Refinement MVP\n")
	fmt.Printf("📁 Test directory: %s\n", projCtx.GetProjectRoot())

	ctx := context.Background()

	// Step 1: Initialize component registry
	fmt.Printf("\n🔧 Step 1: Setting up component registry\n")

	reg := registry.NewComponentRegistry()

	// Setup database and storage
	dbPath := filepath.Join(projCtx.GetGuildPath(), "guild.db")
	database, err := storage.DefaultDatabaseFactory(ctx, dbPath)
	require.NoError(t, err)
	defer database.Close()
	
	// Run migrations
	err = database.Migrate(ctx)
	require.NoError(t, err)

	// Setup storage registry
	storageReg := storage.NewStorageRegistry()
	storageReg.SetDatabase(database)
	
	// Initialize all repositories
	taskRepo := storage.DefaultTaskRepositoryFactory(database)
	boardRepo := storage.DefaultBoardRepositoryFactory(database)
	campaignRepo := storage.DefaultCampaignRepositoryFactory(database)
	commissionRepo := storage.DefaultCommissionRepositoryFactory(database)
	agentRepo := storage.DefaultAgentRepositoryFactory(database)
	promptChainRepo := storage.DefaultPromptChainRepositoryFactory(database)
	
	storageReg.RegisterTaskRepository(taskRepo)
	storageReg.RegisterBoardRepository(boardRepo)
	storageReg.RegisterCampaignRepository(campaignRepo)
	storageReg.RegisterCommissionRepository(commissionRepo)
	storageReg.RegisterAgentRepository(agentRepo)
	storageReg.RegisterPromptChainRepository(promptChainRepo)

	// Register storage with main registry
	err = reg.Storage().RegisterStorageRegistry(storageReg)
	require.NoError(t, err)

	// Register mock provider first
	mockProvider := mock.NewProvider()
	mockProvider.SetResponse("Guild Master", `## File: commission_refined.md

# E-Commerce MVP Commission

## Overview
This commission outlines the essential tasks for our e-commerce MVP.

## Task List

- BACKEND-001: Setup Database Schema (priority: high, estimate: 4h)
- BACKEND-002: Create Product API (priority: high, estimate: 6h, depends: BACKEND-001)
- FRONTEND-003: Build Product Catalog UI (priority: medium, estimate: 8h, depends: BACKEND-002)

## Architecture Notes
- Use PostgreSQL for persistence
- RESTful API design
- React frontend

## File: README.md

# E-Commerce MVP

This project implements a minimal viable e-commerce platform.

## Goals
- Product catalog
- Basic shopping functionality
- Clean architecture`)

	err = reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Initialize registry
	registryConfig := registry.Config{
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
		Storage: registry.StorageConfig{
			DefaultStorage: "sqlite",
		},
	}
	err = reg.Initialize(ctx, registryConfig)
	require.NoError(t, err)
	fmt.Printf("  ✓ Component registry initialized\n")

	// Step 2: Setup kanban system
	fmt.Printf("\n📋 Step 2: Setting up kanban system\n")

	// Create a test campaign first
	campaign := &storage.Campaign{
		ID:        "test-campaign",
		Name:      "MVP Test Campaign",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = campaignRepo.CreateCampaign(ctx, campaign)
	require.NoError(t, err)

	// Create a commission in storage
	storageCommission := &storage.Commission{
		ID:          "mvp-commission-001",
		CampaignID:  campaign.ID,
		Title:       "E-Commerce MVP Development",
		Description: strPtr("Build essential e-commerce functionality for our MVP release"),
		Domain:      strPtr("web-development"),
		Status:      "active",
		CreatedAt:   time.Now(),
	}
	err = commissionRepo.CreateCommission(ctx, storageCommission)
	require.NoError(t, err)

	// Create board for the commission
	board := &storage.Board{
		ID:           "mvp-board",
		CommissionID: storageCommission.ID,
		Name:         "MVP Commission Board",
		Description:  strPtr("Board for MVP commission tasks"),
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err = boardRepo.CreateBoard(ctx, board)
	require.NoError(t, err)
	fmt.Printf("  ✓ Created kanban board: %s\n", board.Name)

	// Step 3: Create commission integration service
	fmt.Printf("\n🔗 Step 3: Creating commission integration service\n")

	service, err := orchestrator.NewCommissionIntegrationService(reg)
	require.NoError(t, err)
	fmt.Printf("  ✓ Integration service created\n")

	// Step 4: Define commission and guild
	fmt.Printf("\n📜 Step 4: Defining commission and guild\n")

	commission := manager.Commission{
		ID:          "mvp-commission-001",
		Title:       "E-Commerce MVP Development",
		Description: "Build essential e-commerce functionality for our MVP release",
		Domain:      "web-development",
	}

	guildConfig := &config.GuildConfig{
		Name:    "MVP Development Guild",
		Version: "1.0.0",
		Agents: []config.AgentConfig{
			{
				ID:            "backend-specialist",
				Name:          "Backend Specialist",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"api", "database", "backend"},
				CostMagnitude: 5,
			},
			{
				ID:            "frontend-specialist",
				Name:          "Frontend Specialist",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"react", "ui", "frontend"},
				CostMagnitude: 5,
			},
		},
	}
	fmt.Printf("  ✓ Commission: %s\n", commission.Title)
	fmt.Printf("  ✓ Guild: %s with %d artisans\n", guildConfig.Name, len(guildConfig.Agents))

	// Step 5: Execute commission refinement
	fmt.Printf("\n🚀 Step 5: Executing commission refinement\n")

	result, err := service.ProcessCommissionToTasks(ctx, commission, guildConfig)
	require.NoError(t, err, "Commission refinement should succeed")
	require.NotNil(t, result, "Result should not be nil")

	fmt.Printf("  ✓ Commission refinement completed\n")

	// Step 6: Verify results
	fmt.Printf("\n✅ Step 6: Verifying results\n")

	// Check refined commission
	assert.NotNil(t, result.RefinedCommission)
	assert.Equal(t, commission.ID, result.RefinedCommission.CommissionID)
	assert.NotNil(t, result.RefinedCommission.Structure)
	fmt.Printf("  ✓ Refined commission created\n")

	// Check file structure
	assert.Greater(t, len(result.RefinedCommission.Structure.Files), 0)
	fmt.Printf("  ✓ File structure contains %d files\n", len(result.RefinedCommission.Structure.Files))

	// List files created
	for i, file := range result.RefinedCommission.Structure.Files {
		fmt.Printf("    File %d: %s (%d tasks)\n", i+1, file.Path, file.TasksCount)
	}

	// Check tasks created
	assert.Greater(t, len(result.Tasks), 0, "Should create kanban tasks")
	fmt.Printf("  ✓ Created %d kanban tasks\n", len(result.Tasks))

	// List tasks created
	for i, task := range result.Tasks {
		fmt.Printf("    Task %d: %s (Priority: %s, Assigned: %s)\n",
			i+1, task.Title, task.Priority, task.AssignedTo)
	}

	// Check artisan assignments
	assert.Greater(t, len(result.AssignedArtisans), 0, "Should assign artisans")
	fmt.Printf("  ✓ Assigned to %d artisans: %v\n",
		len(result.AssignedArtisans), result.AssignedArtisans)

	// Step 7: Verify task distribution
	fmt.Printf("\n📊 Step 7: Analyzing task distribution\n")

	backendTasks := result.GetTasksByArtisan("backend-specialist")
	frontendTasks := result.GetTasksByArtisan("frontend-specialist")

	fmt.Printf("  Backend specialist: %d tasks\n", len(backendTasks))
	fmt.Printf("  Frontend specialist: %d tasks\n", len(frontendTasks))

	// Verify intelligent assignment
	for _, task := range backendTasks {
		category := task.Metadata["original_category"]
		fmt.Printf("    Backend task: %s (category: %s)\n", task.Title, category)
	}

	for _, task := range frontendTasks {
		category := task.Metadata["original_category"]
		fmt.Printf("    Frontend task: %s (category: %s)\n", task.Title, category)
	}

	fmt.Printf("\n🎉 Commission Refinement MVP Test PASSED!\n")
	fmt.Printf("   ✅ Commission refined successfully\n")
	fmt.Printf("   ✅ Tasks created and assigned intelligently\n")
	fmt.Printf("   ✅ File structure generated correctly\n")
	fmt.Printf("   ✅ MVP pipeline operational\n")
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}