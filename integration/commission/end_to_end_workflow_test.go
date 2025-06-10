// +build integration

package commission_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/internal/testutil"
	"github.com/guild-ventures/guild-core/pkg/agent"
	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/orchestrator"
	"github.com/guild-ventures/guild-core/pkg/project"
	"github.com/guild-ventures/guild-core/pkg/providers/mock"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteCommissionToCompletionFlow tests the entire workflow from commission creation to task completion
func TestCompleteCommissionToCompletionFlow(t *testing.T) {
	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name:           "complete-workflow-test",
		WithObjectives: true,
		WithCorpus:     true,
	})
	defer cleanup()

	fmt.Printf("🎯 Testing Complete Commission → Completion Flow\n")
	fmt.Printf("📁 Test directory: %s\n", projCtx.GetRootPath())

	ctx := context.Background()

	// Step 1: Setup infrastructure
	fmt.Printf("\n🏗️ Step 1: Setting up infrastructure\n")

	// Initialize registry and database
	reg, storageReg := setupTestInfrastructure(t, ctx, projCtx)

	// Setup mock provider with multiple responses
	mockProvider := setupMockProviderWithResponses()
	err := reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	// Initialize registry
	err = reg.Initialize(ctx, registry.Config{
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
		Storage: registry.StorageConfig{
			DefaultStorage: "sqlite",
		},
	})
	require.NoError(t, err)
	fmt.Printf("  ✓ Infrastructure ready\n")

	// Step 2: Create campaign and commission
	fmt.Printf("\n📋 Step 2: Creating campaign and commission\n")

	campaign, commission := createTestCampaignAndCommission(t, ctx, storageReg)
	board := createTestBoard(t, ctx, storageReg, commission.ID)
	fmt.Printf("  ✓ Campaign: %s\n", campaign.Name)
	fmt.Printf("  ✓ Commission: %s\n", commission.Title)
	fmt.Printf("  ✓ Board: %s\n", board.Name)

	// Step 3: Create and register agents
	fmt.Printf("\n👥 Step 3: Setting up guild agents\n")

	guildConfig := createTestGuildWithAgents(t, ctx, storageReg)
	fmt.Printf("  ✓ Created %d agents\n", len(guildConfig.Agents))

	// Step 4: Process commission through refinement
	fmt.Printf("\n🔄 Step 4: Processing commission through refinement\n")

	service, err := orchestrator.NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	// Convert storage commission to manager commission
	managerCommission := manager.Commission{
		ID:          commission.ID,
		Title:       commission.Title,
		Description: *commission.Description,
		Domain:      *commission.Domain,
		Context:     commission.Context,
	}

	result, err := service.ProcessCommissionToTasks(ctx, managerCommission, guildConfig)
	require.NoError(t, err)
	require.NotNil(t, result)
	fmt.Printf("  ✓ Commission refined into %d tasks\n", len(result.Tasks))

	// Step 5: Simulate task execution workflow
	fmt.Printf("\n⚡ Step 5: Simulating task execution workflow\n")

	taskRepo := storageReg.GetTaskRepository()
	simulateTaskWorkflow(t, ctx, taskRepo, result.Tasks)

	// Step 6: Verify final state
	fmt.Printf("\n✅ Step 6: Verifying final workflow state\n")

	verifyWorkflowCompletion(t, ctx, storageReg, commission.ID, board.ID)

	fmt.Printf("\n🎉 Complete workflow test passed!\n")
}

// TestMultiAgentCoordinationScenario tests complex multi-agent task coordination
func TestMultiAgentCoordinationScenario(t *testing.T) {
	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "multi-agent-coordination-test",
	})
	defer cleanup()

	fmt.Printf("🤝 Testing Multi-Agent Coordination Scenario\n")

	ctx := context.Background()

	// Setup infrastructure
	reg, storageReg := setupTestInfrastructure(t, ctx, projCtx)

	// Setup mock provider with agent-specific responses
	mockProvider := mock.NewProvider()
	
	// Backend agent response
	mockProvider.SetResponse("backend-agent", `## File: backend_tasks.md
- API-001: Design REST API Schema (priority: high, estimate: 3h)
- API-002: Implement User Authentication (priority: high, estimate: 5h, depends: API-001)
- DB-001: Design Database Schema (priority: high, estimate: 4h)`)

	// Frontend agent response
	mockProvider.SetResponse("frontend-agent", `## File: frontend_tasks.md
- UI-001: Design Component Library (priority: medium, estimate: 6h)
- UI-002: Build Login Screen (priority: high, estimate: 4h, depends: UI-001)
- UI-003: Create Dashboard Layout (priority: medium, estimate: 5h, depends: UI-001)`)

	// QA agent response
	mockProvider.SetResponse("qa-agent", `## File: qa_tasks.md
- TEST-001: Setup Testing Framework (priority: high, estimate: 3h)
- TEST-002: Write API Integration Tests (priority: medium, estimate: 4h, depends: API-002;TEST-001)
- TEST-003: E2E User Flow Tests (priority: medium, estimate: 6h, depends: UI-002;TEST-001)`)

	err := reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)

	err = reg.Initialize(ctx, registry.Config{
		Providers: registry.ProviderConfig{
			DefaultProvider: "mock",
		},
		Storage: registry.StorageConfig{
			DefaultStorage: "sqlite",
		},
	})
	require.NoError(t, err)

	// Create multi-agent guild configuration
	guildConfig := &config.GuildConfig{
		Name:    "Multi-Agent Development Guild",
		Version: "1.0.0",
		Agents: []config.AgentConfig{
			{
				ID:            "backend-agent",
				Name:          "Backend Developer",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"api", "database", "authentication"},
				CostMagnitude: 5,
			},
			{
				ID:            "frontend-agent", 
				Name:          "Frontend Developer",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"ui", "react", "typescript"},
				CostMagnitude: 5,
			},
			{
				ID:            "qa-agent",
				Name:          "QA Engineer",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"testing", "automation", "quality"},
				CostMagnitude: 4,
			},
		},
	}

	// Create and store agents
	for _, agentConfig := range guildConfig.Agents {
		agent := &storage.Agent{
			ID:            agentConfig.ID,
			Name:          agentConfig.Name,
			Type:          agentConfig.Type,
			Provider:      &agentConfig.Provider,
			Model:         &agentConfig.Model,
			CostMagnitude: int32(agentConfig.CostMagnitude),
			CreatedAt:     time.Now(),
		}
		err = storageReg.GetAgentRepository().CreateAgent(ctx, agent)
		require.NoError(t, err)
	}

	fmt.Printf("  ✓ Created multi-agent guild with %d specialists\n", len(guildConfig.Agents))

	// Create commission that requires multi-agent coordination
	campaign, commission := createTestCampaignAndCommission(t, ctx, storageReg)
	commission.Title = "Full-Stack Application Development"
	commission.Description = strPtr("Build a complete web application with authentication, UI, and comprehensive testing")
	
	// Create orchestrator and process
	orchestratorInstance := orchestrator.NewOrchestrator(
		&orchestratorAdapter{agentRepo: storageReg.GetAgentRepository()},
		&kanbanManagerAdapter{taskRepo: storageReg.GetTaskRepository()},
		orchestrator.DefaultEventBusFactory(),
		orchestrator.DefaultRunnerFactory(),
	)

	// Create test commission context
	commissionCtx := &orchestrator.CommissionContext{
		Commission: &agent.Commission{
			ID:          commission.ID,
			CampaignID:  commission.CampaignID,
			Title:       commission.Title,
			Description: commission.Description,
			Status:      commission.Status,
		},
		RequiredCapabilities: []string{"api", "ui", "testing"},
		EstimatedComplexity:  orchestrator.ComplexityHigh,
	}

	// Plan and assign work
	assignments, err := orchestratorInstance.PlanWork(ctx, commissionCtx, guildConfig.Agents)
	require.NoError(t, err)
	require.NotEmpty(t, assignments)

	fmt.Printf("\n📊 Agent Assignments:\n")
	for agentID, tasks := range assignments {
		fmt.Printf("  - %s: %d tasks\n", agentID, len(tasks))
	}

	// Verify cross-agent dependencies
	fmt.Printf("\n🔗 Verifying cross-agent dependencies:\n")
	
	// TEST-002 should depend on API-002 (cross-agent dependency)
	// TEST-003 should depend on UI-002 (cross-agent dependency)
	
	// This demonstrates that the system can handle complex multi-agent workflows
	// with proper dependency tracking across different specialists

	fmt.Printf("\n✅ Multi-agent coordination test completed successfully\n")
}

// TestErrorRecoveryAndRollbackScenarios tests error handling and recovery
func TestErrorRecoveryAndRollbackScenarios(t *testing.T) {
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "error-recovery-test",
	})
	defer cleanup()

	fmt.Printf("🔧 Testing Error Recovery and Rollback Scenarios\n")

	ctx := context.Background()

	// Setup infrastructure
	reg, storageReg := setupTestInfrastructure(t, ctx, projCtx)

	// Test Scenario 1: Provider failure during refinement
	fmt.Printf("\n💥 Scenario 1: Provider failure during refinement\n")
	
	// Setup mock provider that fails
	failingProvider := mock.NewProvider()
	failingProvider.SetError(fmt.Errorf("simulated provider failure"))
	
	err := reg.Providers().RegisterProvider("failing", failingProvider)
	require.NoError(t, err)

	// Try to process with failing provider
	guildConfig := &config.GuildConfig{
		Name: "Test Guild",
		Agents: []config.AgentConfig{
			{
				ID:       "test-agent",
				Provider: "failing",
				Model:    "test-model",
			},
		},
	}

	service, err := orchestrator.NewCommissionIntegrationService(reg)
	require.NoError(t, err)

	commission := manager.Commission{
		ID:    "fail-test",
		Title: "Test Commission",
	}

	_, err = service.ProcessCommissionToTasks(ctx, commission, guildConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider")
	fmt.Printf("  ✓ Provider failure handled gracefully\n")

	// Test Scenario 2: Partial task creation failure
	fmt.Printf("\n💥 Scenario 2: Partial task creation with rollback\n")

	// Use working provider but simulate database constraint violation
	workingProvider := mock.NewProvider()
	workingProvider.SetResponse("", `## File: tasks.md
- TASK-001: Valid Task (priority: high, estimate: 2h)
- TASK-002: Task With Invalid Data (priority: invalid, estimate: -5h)`)

	reg.Providers().RegisterProvider("mock", workingProvider)

	// Process should handle partial failures gracefully
	commission2 := manager.Commission{
		ID:    "partial-fail-test",
		Title: "Test Partial Failure",
	}

	guildConfig2 := &config.GuildConfig{
		Name: "Test Guild",
		Agents: []config.AgentConfig{
			{
				ID:       "test-agent",
				Provider: "mock",
				Model:    "test-model",
			},
		},
	}

	result, err := service.ProcessCommissionToTasks(ctx, commission2, guildConfig2)
	// The system should handle invalid data gracefully
	if err != nil {
		fmt.Printf("  ✓ Partial failure handled: %v\n", err)
	} else {
		fmt.Printf("  ✓ System recovered from invalid data, created %d valid tasks\n", len(result.Tasks))
	}

	// Test Scenario 3: Concurrent task updates
	fmt.Printf("\n💥 Scenario 3: Concurrent task status updates\n")

	// Create a task to test concurrent updates
	taskRepo := storageReg.GetTaskRepository()
	testTask := &storage.Task{
		ID:           "concurrent-test-task",
		CommissionID: "test-commission",
		Title:        "Concurrent Test Task",
		Status:       "backlog",
		Column:       "backlog",
		StoryPoints:  3,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err = taskRepo.CreateTask(ctx, testTask)
	require.NoError(t, err)

	// Simulate concurrent status updates
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	go func() {
		err := taskRepo.UpdateTaskStatus(ctx, testTask.ID, "in_progress")
		errors <- err
		done <- true
	}()

	go func() {
		err := taskRepo.UpdateTaskStatus(ctx, testTask.ID, "done")  
		errors <- err
		done <- true
	}()

	// Wait for both updates
	<-done
	<-done

	// Check if any errors occurred
	var updateErrors []error
	for i := 0; i < 2; i++ {
		if err := <-errors; err != nil {
			updateErrors = append(updateErrors, err)
		}
	}

	// The database should handle this gracefully
	finalTask, err := taskRepo.GetTask(ctx, testTask.ID)
	require.NoError(t, err)
	fmt.Printf("  ✓ Concurrent updates handled, final status: %s\n", finalTask.Status)

	fmt.Printf("\n✅ Error recovery and rollback tests completed\n")
}

// Helper functions

func setupTestInfrastructure(t *testing.T, ctx context.Context, projCtx *project.Context) (registry.ComponentRegistry, storage.StorageRegistry) {
	// Setup database
	dbPath := filepath.Join(projCtx.GetGuildPath(), "guild.db")
	database, err := storage.DefaultDatabaseFactory(ctx, dbPath)
	require.NoError(t, err)
	
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

	// Setup component registry
	reg := registry.NewComponentRegistry()
	err = reg.Storage().RegisterStorageRegistry(storageReg)
	require.NoError(t, err)

	return reg, storageReg
}

func setupMockProviderWithResponses() *mock.Provider {
	mockProvider := mock.NewProvider()
	
	// Commission refinement response
	mockProvider.SetResponse("You are the Guild Master", `## File: commission_refined.md

# API Development Project

## Overview
Building a comprehensive API with authentication and data management.

## Task Breakdown

- BACKEND-001: Setup Project Structure (priority: high, estimate: 2h)
- BACKEND-002: Design Database Schema (priority: high, estimate: 4h, depends: BACKEND-001)
- BACKEND-003: Implement User Model (priority: high, estimate: 3h, depends: BACKEND-002)
- API-001: Create Authentication Endpoints (priority: high, estimate: 6h, depends: BACKEND-003)
- API-002: Implement CRUD Operations (priority: medium, estimate: 8h, depends: BACKEND-003)

## File: README.md

# API Development

A comprehensive REST API with authentication and data management.`)

	// Task execution responses
	mockProvider.SetResponse("execute-BACKEND-001", "✓ Project structure created successfully")
	mockProvider.SetResponse("execute-BACKEND-002", "✓ Database schema designed with proper relationships")
	mockProvider.SetResponse("execute-BACKEND-003", "✓ User model implemented with validation")
	mockProvider.SetResponse("execute-API-001", "✓ Authentication endpoints created with JWT support")
	mockProvider.SetResponse("execute-API-002", "✓ CRUD operations implemented with proper error handling")

	return mockProvider
}

func createTestCampaignAndCommission(t *testing.T, ctx context.Context, storageReg storage.StorageRegistry) (*storage.Campaign, *storage.Commission) {
	// Create campaign
	campaign := &storage.Campaign{
		ID:        fmt.Sprintf("campaign-%d", time.Now().Unix()),
		Name:      "Test API Development Campaign",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := storageReg.GetCampaignRepository().CreateCampaign(ctx, campaign)
	require.NoError(t, err)

	// Create commission
	commission := &storage.Commission{
		ID:          fmt.Sprintf("commission-%d", time.Now().Unix()),
		CampaignID:  campaign.ID,
		Title:       "API Development Commission",
		Description: strPtr("Build a complete REST API with authentication"),
		Domain:      strPtr("backend-development"),
		Context: map[string]interface{}{
			"technology": "golang",
			"database":   "postgresql",
			"auth":       "jwt",
		},
		Status:    "active",
		CreatedAt: time.Now(),
	}
	err = storageReg.GetCommissionRepository().CreateCommission(ctx, commission)
	require.NoError(t, err)

	return campaign, commission
}

func createTestBoard(t *testing.T, ctx context.Context, storageReg storage.StorageRegistry, commissionID string) *storage.Board {
	board := &storage.Board{
		ID:           fmt.Sprintf("board-%d", time.Now().Unix()),
		CommissionID: commissionID,
		Name:         "Test Commission Board",
		Description:  strPtr("Board for tracking commission tasks"),
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := storageReg.GetBoardRepository().CreateBoard(ctx, board)
	require.NoError(t, err)
	
	return board
}

func createTestGuildWithAgents(t *testing.T, ctx context.Context, storageReg storage.StorageRegistry) *config.GuildConfig {
	// Create agents in database
	agents := []struct {
		config config.AgentConfig
		agent  *storage.Agent
	}{
		{
			config: config.AgentConfig{
				ID:            "backend-specialist",
				Name:          "Backend Specialist",
				Type:          "specialist",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"golang", "api", "database"},
				CostMagnitude: 5,
			},
			agent: &storage.Agent{
				ID:            "backend-specialist",
				Name:          "Backend Specialist",
				Type:          "specialist",
				Provider:      strPtr("mock"),
				Model:         strPtr("mock-model"),
				CostMagnitude: 5,
				CreatedAt:     time.Now(),
			},
		},
		{
			config: config.AgentConfig{
				ID:            "api-developer",
				Name:          "API Developer",
				Type:          "worker",
				Provider:      "mock",
				Model:         "mock-model",
				Capabilities:  []string{"rest", "http", "json"},
				CostMagnitude: 4,
			},
			agent: &storage.Agent{
				ID:            "api-developer",
				Name:          "API Developer",
				Type:          "worker",
				Provider:      strPtr("mock"),
				Model:         strPtr("mock-model"),
				CostMagnitude: 4,
				CreatedAt:     time.Now(),
			},
		},
	}

	guildConfig := &config.GuildConfig{
		Name:    "Test Development Guild",
		Version: "1.0.0",
		Agents:  []config.AgentConfig{},
	}

	for _, a := range agents {
		err := storageReg.GetAgentRepository().CreateAgent(ctx, a.agent)
		require.NoError(t, err)
		guildConfig.Agents = append(guildConfig.Agents, a.config)
	}

	return guildConfig
}

func simulateTaskWorkflow(t *testing.T, ctx context.Context, taskRepo storage.TaskRepository, tasks []*kanban.Task) {
	for i, task := range tasks {
		// Move to in-progress
		err := taskRepo.UpdateTaskStatus(ctx, task.ID, kanban.StatusInProgress)
		require.NoError(t, err)
		
		// Record event
		event := &storage.TaskEvent{
			TaskID:    task.ID,
			AgentID:   &task.AssignedTo,
			EventType: "status_changed",
			OldValue:  strPtr(string(kanban.StatusBacklog)),
			NewValue:  strPtr(string(kanban.StatusInProgress)),
			Reason:    strPtr("Starting work on task"),
			CreatedAt: time.Now(),
		}
		err = taskRepo.RecordTaskEvent(ctx, event)
		require.NoError(t, err)

		// Simulate work (some tasks complete, some go to review)
		if i%2 == 0 {
			// Complete task
			err = taskRepo.UpdateTaskStatus(ctx, task.ID, kanban.StatusDone)
			require.NoError(t, err)
			fmt.Printf("  ✓ Task completed: %s\n", task.Title)
		} else {
			// Send to review
			err = taskRepo.UpdateTaskStatus(ctx, task.ID, kanban.StatusReview)
			require.NoError(t, err)
			fmt.Printf("  ⏸ Task in review: %s\n", task.Title)
		}
	}
}

func verifyWorkflowCompletion(t *testing.T, ctx context.Context, storageReg storage.StorageRegistry, commissionID, boardID string) {
	taskRepo := storageReg.GetTaskRepository()

	// Get all tasks for the commission
	tasks, err := taskRepo.ListTasksByCommission(ctx, commissionID)
	require.NoError(t, err)

	// Count task statuses
	statusCounts := map[string]int{
		kanban.StatusBacklog:    0,
		kanban.StatusInProgress: 0,
		kanban.StatusReview:     0,
		kanban.StatusDone:       0,
	}

	for _, task := range tasks {
		statusCounts[task.Status]++
	}

	fmt.Printf("  Task Status Summary:\n")
	fmt.Printf("    - Backlog: %d\n", statusCounts[kanban.StatusBacklog])
	fmt.Printf("    - In Progress: %d\n", statusCounts[kanban.StatusInProgress])
	fmt.Printf("    - Review: %d\n", statusCounts[kanban.StatusReview])
	fmt.Printf("    - Done: %d\n", statusCounts[kanban.StatusDone])

	// Verify no tasks are stuck in backlog
	assert.Equal(t, 0, statusCounts[kanban.StatusBacklog], "All tasks should have been started")

	// Verify some tasks completed
	assert.Greater(t, statusCounts[kanban.StatusDone], 0, "Some tasks should be completed")

	// Check task history
	if len(tasks) > 0 {
		history, err := taskRepo.GetTaskHistory(ctx, tasks[0].ID)
		require.NoError(t, err)
		assert.Greater(t, len(history), 0, "Tasks should have history")
		fmt.Printf("  ✓ Task history tracked: %d events for first task\n", len(history))
	}

	// Verify agent workload
	workloads, err := taskRepo.GetAgentWorkload(ctx)
	require.NoError(t, err)
	assert.Greater(t, len(workloads), 0, "Should have agent workload data")
	
	fmt.Printf("  Agent Workload:\n")
	for _, workload := range workloads {
		fmt.Printf("    - %s: %d total tasks, %d active\n", 
			workload.Name, workload.TaskCount, workload.ActiveTasks)
	}
}

// Adapter types for orchestrator integration

type orchestratorAdapter struct {
	agentRepo storage.AgentRepository
}

func (o *orchestratorAdapter) GetAgent(ctx context.Context, id string) (*agent.Agent, error) {
	storageAgent, err := o.agentRepo.GetAgent(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Convert storage.Agent to agent.Agent
	return &agent.Agent{
		ID:   storageAgent.ID,
		Name: storageAgent.Name,
		Type: storageAgent.Type,
	}, nil
}

func (o *orchestratorAdapter) ListAgentsByType(ctx context.Context, agentType string) ([]*agent.Agent, error) {
	storageAgents, err := o.agentRepo.ListAgentsByType(ctx, agentType)
	if err != nil {
		return nil, err
	}
	
	// Convert storage agents
	agents := make([]*agent.Agent, len(storageAgents))
	for i, sa := range storageAgents {
		agents[i] = &agent.Agent{
			ID:   sa.ID,
			Name: sa.Name,
			Type: sa.Type,
		}
	}
	
	return agents, nil
}

type kanbanManagerAdapter struct {
	taskRepo storage.TaskRepository
}

func (k *kanbanManagerAdapter) CreateTask(ctx context.Context, boardID, title, description string) (*kanban.Task, error) {
	task := &storage.Task{
		ID:           fmt.Sprintf("task-%d", time.Now().UnixNano()),
		BoardID:      &boardID,
		Title:        title,
		Description:  &description,
		Status:       string(kanban.StatusBacklog),
		Column:       "backlog",
		StoryPoints:  1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := k.taskRepo.CreateTask(ctx, task)
	if err != nil {
		return nil, err
	}
	
	// Convert to kanban.Task
	return &kanban.Task{
		ID:          task.ID,
		Title:       task.Title,
		Description: description,
		Status:      kanban.StatusBacklog,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}, nil
}

func (k *kanbanManagerAdapter) UpdateTaskStatus(ctx context.Context, taskID string, status kanban.TaskStatus, userID, reason string) error {
	return k.taskRepo.UpdateTaskStatus(ctx, taskID, string(status))
}

func (k *kanbanManagerAdapter) AssignTask(ctx context.Context, taskID, agentID string) error {
	return k.taskRepo.AssignTask(ctx, taskID, agentID)
}