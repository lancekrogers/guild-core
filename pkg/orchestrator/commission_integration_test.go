package orchestrator

import (
	"context"
	"fmt"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/config"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/storage"
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

	// Set up mock provider with realistic LLM response following Guild refinement format
	mockProvider := mock.NewProvider()
	mockProvider.SetDefaultResponse(`## File: commission_refined.md

# 🧠 Goal

Build a comprehensive web-based task management system with user authentication that enables teams to create, assign, track, and manage tasks efficiently using React frontend and Node.js backend.

# 📂 Context

This commission focuses on creating a modern web application for task management. The system will serve teams who need to organize work, track progress, and collaborate effectively. The technology stack leverages React for a responsive user interface and Node.js with Express for a scalable backend API.

# 🔧 Requirements

## Core Features
- User registration and authentication system
- Task creation, editing, and deletion
- Task assignment to team members
- Task status tracking (todo, in-progress, done)
- User dashboard with task overview
- Real-time updates for collaborative features

## Technical Requirements
- React-based frontend with responsive design
- Node.js/Express backend API
- Database integration for persistent storage
- RESTful API design
- Authentication middleware
- Input validation and sanitization
- Error handling and logging

## Implementation Tasks
- BACKEND-001: Set up Node.js server with Express (priority: high, estimate: 4h)
- BACKEND-002: Design database schema for tasks and users (priority: high, estimate: 3h, depends: BACKEND-001)
- BACKEND-003: Implement JWT authentication system (priority: high, estimate: 6h)
- API-001: Design and implement RESTful API endpoints (priority: high, estimate: 5h, depends: BACKEND-002)
- FRONTEND-001: Create React application structure and routing (priority: medium, estimate: 3h)
- FRONTEND-002: Build user authentication components (priority: high, estimate: 4h, depends: FRONTEND-001)
- FRONTEND-003: Implement task management interface (priority: high, estimate: 6h, depends: FRONTEND-002)
- TEST-001: Set up testing framework and write unit tests (priority: medium, estimate: 4h)
- DEPLOY-001: Configure production deployment (priority: low, estimate: 2h)

# 📌 Tags

web-app, task-management, react, nodejs, authentication, api, database

# 🔗 Related

- Web application development best practices
- React component architecture patterns  
- Node.js API security guidelines
- Database design for task management systems

## File: implementation/backend_plan.md

# Backend Implementation Plan

## Architecture Overview
The backend will be built using Node.js with Express.js framework, providing RESTful APIs for the frontend to consume.

## Database Design
- Users table: id, username, email, password_hash, created_at, updated_at
- Tasks table: id, title, description, status, priority, assigned_to, created_by, due_date, created_at, updated_at
- Task relationships and foreign keys

## API Endpoints
- POST /api/auth/register - User registration
- POST /api/auth/login - User authentication
- GET /api/tasks - Retrieve user tasks
- POST /api/tasks - Create new task
- PUT /api/tasks/:id - Update task
- DELETE /api/tasks/:id - Delete task

## Authentication Strategy
JWT-based authentication with refresh tokens for secure session management.

## File: implementation/frontend_plan.md

# Frontend Implementation Plan

## Component Architecture
React application with component-based architecture using functional components and hooks.

## Key Components
- AuthComponent: Login/register forms
- Dashboard: Main task overview
- TaskList: Display and filter tasks
- TaskForm: Create/edit task modal
- TaskItem: Individual task display
- Navigation: App navigation and user menu

## State Management
React Context API for global state management of user authentication and task data.

## Styling
Modern CSS with responsive design principles, potentially using a UI library like Material-UI or Tailwind CSS.

## File: README.md

# Task Management System Project

This project implements a comprehensive web-based task management system with user authentication.

## Overview
Build a modern web application for task management using React frontend and Node.js backend.

## Implementation Tasks
- BACKEND-001: Set up Node.js server with Express (priority: high, estimate: 4h)
- BACKEND-002: Design database schema for tasks and users (priority: high, estimate: 3h, depends: BACKEND-001)
- BACKEND-003: Implement JWT authentication system (priority: high, estimate: 6h)
- API-001: Design and implement RESTful API endpoints (priority: high, estimate: 5h, depends: BACKEND-002)
- FRONTEND-001: Create React application structure and routing (priority: medium, estimate: 3h)
- FRONTEND-002: Build user authentication components (priority: high, estimate: 4h, depends: FRONTEND-001)
- FRONTEND-003: Implement task management interface (priority: high, estimate: 6h, depends: FRONTEND-002)
- TEST-001: Set up testing framework and write unit tests (priority: medium, estimate: 4h)
- DEPLOY-001: Configure production deployment (priority: low, estimate: 2h)

## Getting Started
See the implementation plans in the implementation/ directory for detailed technical specifications.`)

	err := reg.Providers().RegisterProvider("mock", mockProvider)
	require.NoError(t, err)
	reg.Providers().SetDefaultProvider("mock")

	// Set up SQLite storage instead of mock BoltDB store
	err = setupSQLiteStorage(reg)
	require.NoError(t, err)

	// Set up prompts with realistic Guild Master system prompt
	promptRegistry := prompts.NewMemoryRegistry()
	
	// Register the actual Guild Master refinement prompt that guides LLM behavior
	guildMasterPrompt := `You are a Guild Master for the Guild Framework, responsible for refining commissions into detailed implementation plans.

## Your Role
You analyze commissions (objectives) and transform them into structured, actionable plans that can be assigned to specialized artisans through the Workshop Board (kanban system).

## Output Format
Create a hierarchical file structure with markdown files that follows this format:

### Main Commission File (commission_refined.md)
Use this structure for the refined commission:

# 🧠 Goal
[Clear, specific goal statement based on the commission]

# 📂 Context
[Enhanced context incorporating technical requirements and constraints]

# 🔧 Requirements
[Detailed requirements broken down into:]
## Core Features
[User-facing functionality]

## Technical Requirements
[Implementation specifics]

## Implementation Tasks
[Specific tasks in format: CATEGORY-NUMBER: Description (priority: X, estimate: Xh, depends: Y)]

# 📌 Tags
[Relevant tags for categorization]

# 🔗 Related
[Related documents, patterns, or references]

### Additional Implementation Files
Create supporting files like:
- implementation/backend_plan.md
- implementation/frontend_plan.md
- implementation/testing_plan.md

## Task Naming Convention
Use format: CATEGORY-NUMBER: Description
- BACKEND-001, BACKEND-002, etc.
- FRONTEND-001, FRONTEND-002, etc.
- API-001, API-002, etc.
- TEST-001, TEST-002, etc.

## Task Metadata Format
Include: (priority: high/medium/low, estimate: Xh, depends: TASK-ID)

Transform the given commission into this structured format that artisans can work with effectively.`

	promptRegistry.RegisterPrompt("manager", "default", guildMasterPrompt)
	promptRegistry.RegisterPrompt("manager", "web-app", guildMasterPrompt)

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
	if err != nil {
		t.Logf("Commission processing failed: %v", err)
	}
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
	mockProvider.SetResponse("", `- TASK-001: Test task`)
	require.NoError(t, reg.Providers().RegisterProvider("mock", mockProvider))
	reg.Providers().SetDefaultProvider("mock")

	// SQLite storage
	require.NoError(t, setupSQLiteStorage(reg))

	// Prompts
	promptRegistry := prompts.NewMemoryRegistry()
	promptRegistry.RegisterPrompt("manager", "default", "Test prompt")

	return reg
}

// setupSQLiteStorage initializes SQLite storage for the test registry
func setupSQLiteStorage(reg registry.ComponentRegistry) error {
	ctx := context.Background()
	
	// Initialize SQLite storage for tests
	storageReg, memoryStoreAdapter, err := storage.InitializeSQLiteStorageForTests(ctx)
	if err != nil {
		return err
	}
	
	// Cast to concrete registry to access SetStorageRegistry method
	if defaultReg, ok := reg.(*registry.DefaultComponentRegistry); ok {
		// Convert memory store adapter to the expected interface
		if memStore, ok := memoryStoreAdapter.(registry.MemoryStore); ok {
			// Create SQLite storage registry wrapper manually since there's no constructor
			sqliteStorageReg := &testSQLiteStorageRegistry{
				storageRegistry: storageReg,
				memoryStore:     memStore,
			}
			
			defaultReg.SetStorageRegistry(sqliteStorageReg, memStore)
			return nil
		}
		return fmt.Errorf("memory store adapter does not implement expected interface")
	}
	
	return fmt.Errorf("registry does not support SQLite initialization")
}

// testSQLiteStorageRegistry implements registry.StorageRegistry for testing
type testSQLiteStorageRegistry struct {
	storageRegistry storage.StorageRegistry
	memoryStore     registry.MemoryStore
}

func (t *testSQLiteStorageRegistry) RegisterTaskRepository(repo registry.TaskRepository) error {
	return nil // Not needed for SQLite
}

func (t *testSQLiteStorageRegistry) GetTaskRepository() registry.TaskRepository {
	return nil // Components should use type assertions to get the actual storage repos
}

func (t *testSQLiteStorageRegistry) RegisterCampaignRepository(repo registry.CampaignRepository) error {
	return nil // Not needed for SQLite
}

func (t *testSQLiteStorageRegistry) GetCampaignRepository() registry.CampaignRepository {
	return nil // Components should use type assertions to get the actual storage repos
}

func (t *testSQLiteStorageRegistry) RegisterCommissionRepository(repo registry.CommissionRepository) error {
	return nil // Not needed for SQLite
}

func (t *testSQLiteStorageRegistry) GetCommissionRepository() registry.CommissionRepository {
	// Return the actual commission repository from storage registry
	storageCommissionRepo := t.storageRegistry.GetCommissionRepository()
	if storageCommissionRepo == nil {
		return nil
	}
	
	// Create an adapter that implements registry.CommissionRepository interface
	return &testCommissionRepositoryAdapter{
		storageRepo: storageCommissionRepo,
	}
}

func (t *testSQLiteStorageRegistry) RegisterAgentRepository(repo registry.AgentRepository) error {
	return nil // Not needed for SQLite
}

func (t *testSQLiteStorageRegistry) GetAgentRepository() registry.AgentRepository {
	return nil // Components should use type assertions to get the actual storage repos
}

func (t *testSQLiteStorageRegistry) GetMemoryStore() registry.MemoryStore {
	return t.memoryStore
}

func (t *testSQLiteStorageRegistry) SetMemoryStore(store registry.MemoryStore) {
	t.memoryStore = store
}

// GetStorageRegistry returns the underlying storage.StorageRegistry for components that need it
func (t *testSQLiteStorageRegistry) GetStorageRegistry() storage.StorageRegistry {
	return t.storageRegistry
}

// testCommissionRepositoryAdapter adapts storage.CommissionRepository to registry.CommissionRepository
type testCommissionRepositoryAdapter struct {
	storageRepo storage.CommissionRepository
}

func (a *testCommissionRepositoryAdapter) CreateCommission(ctx context.Context, commission *registry.Commission) error {
	// Convert registry.Commission to storage.Commission
	storageCommission := &storage.Commission{
		ID:          commission.ID,
		CampaignID:  commission.CampaignID,
		Title:       commission.Title,
		Description: commission.Description,
		Domain:      commission.Domain,
		Context:     commission.Context,
		Status:      commission.Status,
		CreatedAt:   commission.CreatedAt,
	}
	return a.storageRepo.CreateCommission(ctx, storageCommission)
}

func (a *testCommissionRepositoryAdapter) GetCommission(ctx context.Context, id string) (*registry.Commission, error) {
	storageCommission, err := a.storageRepo.GetCommission(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// Convert storage.Commission to registry.Commission
	return &registry.Commission{
		ID:          storageCommission.ID,
		CampaignID:  storageCommission.CampaignID,
		Title:       storageCommission.Title,
		Description: storageCommission.Description,
		Domain:      storageCommission.Domain,
		Context:     storageCommission.Context,
		Status:      storageCommission.Status,
		CreatedAt:   storageCommission.CreatedAt,
	}, nil
}

func (a *testCommissionRepositoryAdapter) UpdateCommissionStatus(ctx context.Context, id, status string) error {
	return a.storageRepo.UpdateCommissionStatus(ctx, id, status)
}

func (a *testCommissionRepositoryAdapter) DeleteCommission(ctx context.Context, id string) error {
	return a.storageRepo.DeleteCommission(ctx, id)
}

func (a *testCommissionRepositoryAdapter) ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*registry.Commission, error) {
	storageCommissions, err := a.storageRepo.ListCommissionsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	
	registryCommissions := make([]*registry.Commission, len(storageCommissions))
	for i, sc := range storageCommissions {
		registryCommissions[i] = &registry.Commission{
			ID:          sc.ID,
			CampaignID:  sc.CampaignID,
			Title:       sc.Title,
			Description: sc.Description,
			Domain:      sc.Domain,
			Context:     sc.Context,
			Status:      sc.Status,
			CreatedAt:   sc.CreatedAt,
		}
	}
	return registryCommissions, nil
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