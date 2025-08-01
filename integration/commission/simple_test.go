// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration

package commission_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lancekrogers/guild/internal/testutil"
	"github.com/lancekrogers/guild/pkg/agents/core/manager"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/prompts/layered"
	"github.com/lancekrogers/guild/pkg/registry"
	"github.com/lancekrogers/guild/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreComponentInitialization tests basic component initialization
func TestCoreComponentInitialization(t *testing.T) {
	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "simple-component-test",
	})
	defer cleanup()

	// Use project context for proper initialization
	ctx := context.Background()
	_ = projCtx // Project context is available if needed

	// Test 1: Registry
	reg := registry.NewComponentRegistry()
	require.NotNil(t, reg)
	fmt.Println("✓ Component registry created")

	// Test 2: SQLite database and storage registry
	storageReg, memStore, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)
	require.NotNil(t, storageReg)
	require.NotNil(t, memStore)
	fmt.Println("✓ SQLite database created and storage registry initialized")

	// Test 3: Verify repositories are available
	taskRepo := storageReg.GetTaskRepository()
	boardRepo := storageReg.GetBoardRepository()
	campaignRepo := storageReg.GetCampaignRepository()
	commissionRepo := storageReg.GetCommissionRepository()

	require.NotNil(t, taskRepo)
	require.NotNil(t, boardRepo)
	require.NotNil(t, campaignRepo)
	require.NotNil(t, commissionRepo)
	fmt.Println("✓ All repositories available")

	// Test 4: Kanban manager with SQLite
	// Create kanban registry adapter
	kanbanAdapter := &testKanbanRegistryAdapter{storageRegistry: storageReg}
	kanbanManager, err := kanban.NewManagerWithRegistry(ctx, kanbanAdapter)
	require.NoError(t, err)
	require.NotNil(t, kanbanManager)
	fmt.Println("✓ Kanban manager created with SQLite backend")

	// Test 5: Create a board
	board, err := kanbanManager.CreateBoard(ctx, "test-board", "Test Board")
	require.NoError(t, err)
	assert.NotEmpty(t, board.ID)
	fmt.Println("✓ Kanban board created")

	// Test 6: Response parser
	parser := manager.NewResponseParser()
	require.NotNil(t, parser)
	fmt.Println("✓ Response parser created")

	// Test 7: Prompt registry
	promptRegistry := layered.NewMemoryRegistry()
	require.NotNil(t, promptRegistry)

	// Register a test prompt
	err = promptRegistry.RegisterPrompt("test", "default", "This is a test prompt")
	require.NoError(t, err)

	// Retrieve the prompt
	prompt, err := promptRegistry.GetPrompt("test", "default")
	require.NoError(t, err)
	assert.Equal(t, "This is a test prompt", prompt)
	fmt.Println("✓ Prompt registry working")

	fmt.Println("\n✅ All core components initialized successfully")
}

// TestKanbanWorkflow tests a basic kanban workflow
func TestKanbanWorkflow(t *testing.T) {
	// Setup test project
	projCtx, cleanup := testutil.SetupTestProject(t, testutil.TestProjectOptions{
		Name: "kanban-workflow-test",
	})
	defer cleanup()

	// Use project context for proper initialization
	ctx := context.Background()
	_ = projCtx // Project context is available if needed

	// Setup database and storage
	storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
	require.NoError(t, err)

	// Create kanban manager with SQLite backend
	kanbanAdapter := &testKanbanRegistryAdapter{storageRegistry: storageReg}
	kanbanManager, err := kanban.NewManagerWithRegistry(ctx, kanbanAdapter)
	require.NoError(t, err)

	// Create board
	board, err := kanbanManager.CreateBoard(ctx, "workflow-board", "Workflow Test Board")
	require.NoError(t, err)

	// Create multiple tasks
	taskIDs := []string{}
	taskTitles := []string{
		"Setup project structure",
		"Implement core features",
		"Write tests",
		"Documentation",
	}

	for _, title := range taskTitles {
		task, err := kanbanManager.CreateTask(ctx, board.ID, title, fmt.Sprintf("Description for: %s", title))
		require.NoError(t, err)
		taskIDs = append(taskIDs, task.ID)
		fmt.Printf("✓ Created task: %s\n", title)
	}

	// Move first task to in-progress
	err = kanbanManager.UpdateTaskStatus(ctx, taskIDs[0], kanban.StatusInProgress, "test-user", "Starting work")
	require.NoError(t, err)

	// Move second task to in-progress
	err = kanbanManager.UpdateTaskStatus(ctx, taskIDs[1], kanban.StatusInProgress, "test-user", "Starting implementation")
	require.NoError(t, err)

	// Complete first task
	err = kanbanManager.UpdateTaskStatus(ctx, taskIDs[0], kanban.StatusDone, "test-user", "Completed setup")
	require.NoError(t, err)

	// Verify task statuses
	task0, err := kanbanManager.GetTask(ctx, taskIDs[0])
	require.NoError(t, err)
	assert.Equal(t, kanban.StatusDone, task0.Status)

	task1, err := kanbanManager.GetTask(ctx, taskIDs[1])
	require.NoError(t, err)
	assert.Equal(t, kanban.StatusInProgress, task1.Status)

	// List tasks by status
	// Note: StatusBacklog is mapped to "todo" in storage due to database constraints
	todoTasks, err := kanbanManager.ListTasksByStatus(ctx, kanban.StatusTodo)
	require.NoError(t, err)
	assert.Len(t, todoTasks, 2) // Tasks 2 and 3 (still in todo/backlog)

	inProgressTasks, err := kanbanManager.ListTasksByStatus(ctx, kanban.StatusInProgress)
	require.NoError(t, err)
	assert.Len(t, inProgressTasks, 1) // Task 1

	doneTasks, err := kanbanManager.ListTasksByStatus(ctx, kanban.StatusDone)
	require.NoError(t, err)
	assert.Len(t, doneTasks, 1) // Task 0

	fmt.Println("\n✅ Kanban workflow completed successfully")
}

// TestResponseParserWithStructure tests parsing structured responses
func TestResponseParserWithStructure(t *testing.T) {
	parser := manager.NewResponseParser()

	// Test with a more complex structure
	response := &manager.ArtisanResponse{
		Content: `Based on the commission, I'll create a comprehensive project structure.

## File: README.md
# User Management System
A complete user management system with authentication and profiles.

## Features
- JWT Authentication
- User Profiles
- Role-based Access Control

## File: docs/architecture.md
# System Architecture

The system follows a layered architecture:
1. API Gateway Layer
2. Business Logic Layer
3. Data Access Layer

## File: src/models/user.go
` + "package models" + `

type User struct {
    ID       string
    Email    string
    Username string
    Profile  UserProfile
}

## File: src/handlers/auth.go
` + "package handlers" + `

import "net/http"

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    // Authentication logic here
}`,
	}

	structure, err := parser.ParseResponse(context.Background(), response)
	require.NoError(t, err)
	require.NotNil(t, structure)

	// Verify all files were parsed
	assert.Len(t, structure.Files, 4)

	// Check file paths
	expectedPaths := []string{"README.md", "docs/architecture.md", "src/models/user.go", "src/handlers/auth.go"}
	actualPaths := []string{}
	for _, file := range structure.Files {
		actualPaths = append(actualPaths, file.Path)
	}

	// Sort for comparison
	assert.ElementsMatch(t, expectedPaths, actualPaths)

	// Verify content is not empty
	for _, file := range structure.Files {
		assert.NotEmpty(t, file.Content)
		fmt.Printf("✓ Parsed file: %s (%d bytes)\n", file.Path, len(file.Content))
	}

	fmt.Println("\n✅ Response parser tested successfully")
}

// testKanbanRegistryAdapter adapts storage.StorageRegistry to kanban.ComponentRegistry
type testKanbanRegistryAdapter struct {
	storageRegistry storage.StorageRegistry
}

// Storage returns a kanban.StorageRegistry implementation
func (t *testKanbanRegistryAdapter) Storage() kanban.StorageRegistry {
	return &testKanbanStorageAdapter{storageRegistry: t.storageRegistry}
}

// testKanbanStorageAdapter adapts storage.StorageRegistry to kanban.StorageRegistry
type testKanbanStorageAdapter struct {
	storageRegistry storage.StorageRegistry
}

func (t *testKanbanStorageAdapter) GetKanbanCampaignRepository() kanban.CampaignRepository {
	// Create adapter for storage.CampaignRepository to kanban.CampaignRepository
	return &kanbanCampaignRepoAdapter{repo: t.storageRegistry.GetCampaignRepository()}
}

func (t *testKanbanStorageAdapter) GetKanbanCommissionRepository() kanban.CommissionRepository {
	// Create adapter for storage.CommissionRepository to kanban.CommissionRepository
	return &kanbanCommissionRepoAdapter{repo: t.storageRegistry.GetCommissionRepository()}
}

func (t *testKanbanStorageAdapter) GetKanbanTaskRepository() kanban.TaskRepository {
	// Create adapter for storage.TaskRepository to kanban.TaskRepository
	return &kanbanTaskRepoAdapter{repo: t.storageRegistry.GetTaskRepository()}
}

func (t *testKanbanStorageAdapter) GetBoardRepository() kanban.BoardRepository {
	// Create adapter for storage.BoardRepository to kanban.BoardRepository
	return &kanbanBoardRepoAdapter{repo: t.storageRegistry.GetBoardRepository()}
}

func (t *testKanbanStorageAdapter) GetMemoryStore() kanban.MemoryStore {
	// Get the memory store from the storage registry
	memStore := t.storageRegistry.GetMemoryStore()
	if memStore == nil {
		return nil
	}
	// Cast to kanban.MemoryStore interface if possible
	if kanbanStore, ok := memStore.(kanban.MemoryStore); ok {
		return kanbanStore
	}
	// Otherwise wrap it
	return &memoryStoreAdapter{memStore: memStore}
}

// Simple adapters to bridge interface differences
type kanbanTaskRepoAdapter struct {
	repo storage.TaskRepository
}

func (k *kanbanTaskRepoAdapter) CreateTask(ctx context.Context, task interface{}) error {
	// Convert to storage.Task
	if storageTask, ok := task.(*storage.Task); ok {
		return k.repo.CreateTask(ctx, storageTask)
	}
	// Handle map[string]interface{} case for kanban compatibility
	if taskMap, ok := task.(map[string]interface{}); ok {
		// Convert map to storage.Task
		storageTask := &storage.Task{
			ID:          taskMap["ID"].(string),
			Title:       taskMap["Title"].(string),
			Status:      taskMap["Status"].(string),
			StoryPoints: taskMap["StoryPoints"].(int32),
			CreatedAt:   taskMap["CreatedAt"].(time.Time),
			UpdatedAt:   taskMap["UpdatedAt"].(time.Time),
		}
		// Handle nullable fields
		if boardID, ok := taskMap["BoardID"].(string); ok {
			storageTask.BoardID = &boardID
		}
		if commissionID, ok := taskMap["CommissionID"].(string); ok {
			storageTask.CommissionID = commissionID
		}
		if agentID, ok := taskMap["AssignedAgentID"].(*string); ok {
			storageTask.AssignedAgentID = agentID
		}
		if desc, ok := taskMap["Description"].(*string); ok {
			storageTask.Description = desc
		}
		if metadata, ok := taskMap["Metadata"].(map[string]interface{}); ok {
			storageTask.Metadata = metadata
		}
		// Set default column
		storageTask.Column = "backlog"
		return k.repo.CreateTask(ctx, storageTask)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid task type", nil).
		WithComponent("kanbanTaskRepoAdapter").
		WithOperation("CreateTask")
}

func (k *kanbanTaskRepoAdapter) UpdateTask(ctx context.Context, task interface{}) error {
	if storageTask, ok := task.(*storage.Task); ok {
		return k.repo.UpdateTask(ctx, storageTask)
	}
	// Handle map[string]interface{} case for kanban compatibility
	if taskMap, ok := task.(map[string]interface{}); ok {
		// Get the task from storage first to preserve all fields
		taskID := taskMap["ID"].(string)
		existingTask, err := k.repo.GetTask(ctx, taskID)
		if err != nil {
			return err
		}

		// Update fields from the map
		if title, ok := taskMap["Title"].(string); ok {
			existingTask.Title = title
		}
		if status, ok := taskMap["Status"].(string); ok {
			existingTask.Status = status
		}
		if agentID, ok := taskMap["AssignedAgentID"].(*string); ok {
			existingTask.AssignedAgentID = agentID
		}
		if desc, ok := taskMap["Description"].(*string); ok {
			existingTask.Description = desc
		}
		if metadata, ok := taskMap["Metadata"].(map[string]interface{}); ok {
			existingTask.Metadata = metadata
		}
		if updatedAt, ok := taskMap["UpdatedAt"].(time.Time); ok {
			existingTask.UpdatedAt = updatedAt
		}

		return k.repo.UpdateTask(ctx, existingTask)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid task type", nil).
		WithComponent("kanbanTaskRepoAdapter").
		WithOperation("UpdateTask")
}

func (k *kanbanTaskRepoAdapter) DeleteTask(ctx context.Context, id string) error {
	return k.repo.DeleteTask(ctx, id)
}

func (k *kanbanTaskRepoAdapter) ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error) {
	tasks, err := k.repo.ListTasksByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	// Convert to []interface{}
	result := make([]interface{}, len(tasks))
	for i, task := range tasks {
		result[i] = task
	}
	return result, nil
}

func (k *kanbanTaskRepoAdapter) RecordTaskEvent(ctx context.Context, event interface{}) error {
	if storageEvent, ok := event.(*storage.TaskEvent); ok {
		return k.repo.RecordTaskEvent(ctx, storageEvent)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid event type", nil).
		WithComponent("kanbanTaskRepoAdapter").
		WithOperation("RecordTaskEvent")
}

type kanbanBoardRepoAdapter struct {
	repo storage.BoardRepository
}

func (k *kanbanBoardRepoAdapter) CreateBoard(ctx context.Context, board interface{}) error {
	if storageBoard, ok := board.(*storage.Board); ok {
		return k.repo.CreateBoard(ctx, storageBoard)
	}
	// Handle map[string]interface{} case for kanban compatibility
	if boardMap, ok := board.(map[string]interface{}); ok {
		// Convert map to storage.Board
		storageBoard := &storage.Board{
			ID:           boardMap["ID"].(string),
			CommissionID: boardMap["CommissionID"].(string),
			Name:         boardMap["Name"].(string),
			Status:       boardMap["Status"].(string),
		}
		if desc, ok := boardMap["Description"].(*string); ok && desc != nil {
			storageBoard.Description = desc
		}
		if createdAt, ok := boardMap["CreatedAt"].(time.Time); ok {
			storageBoard.CreatedAt = createdAt
		}
		if updatedAt, ok := boardMap["UpdatedAt"].(time.Time); ok {
			storageBoard.UpdatedAt = updatedAt
		}
		return k.repo.CreateBoard(ctx, storageBoard)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid board type", nil).
		WithComponent("kanbanBoardRepoAdapter").
		WithOperation("CreateBoard")
}

func (k *kanbanBoardRepoAdapter) GetBoard(ctx context.Context, id string) (interface{}, error) {
	return k.repo.GetBoard(ctx, id)
}

func (k *kanbanBoardRepoAdapter) GetBoardByCommission(ctx context.Context, commissionID string) (interface{}, error) {
	return k.repo.GetBoardByCommission(ctx, commissionID)
}

func (k *kanbanBoardRepoAdapter) UpdateBoard(ctx context.Context, board interface{}) error {
	if storageBoard, ok := board.(*storage.Board); ok {
		return k.repo.UpdateBoard(ctx, storageBoard)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid board type", nil).
		WithComponent("kanbanBoardRepoAdapter").
		WithOperation("UpdateBoard")
}

func (k *kanbanBoardRepoAdapter) ListBoards(ctx context.Context) ([]interface{}, error) {
	boards, err := k.repo.ListBoards(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to []interface{}
	result := make([]interface{}, len(boards))
	for i, board := range boards {
		result[i] = board
	}
	return result, nil
}

func (k *kanbanBoardRepoAdapter) DeleteBoard(ctx context.Context, id string) error {
	// Storage interface doesn't have DeleteBoard, so we return an error
	return gerror.New(gerror.ErrCodeNotImplemented, "DeleteBoard not implemented in storage layer", nil).
		WithComponent("kanbanBoardRepoAdapter").
		WithOperation("DeleteBoard")
}

// kanbanCampaignRepoAdapter adapts storage.CampaignRepository to kanban.CampaignRepository
type kanbanCampaignRepoAdapter struct {
	repo storage.CampaignRepository
}

// kanbanCommissionRepoAdapter adapts storage.CommissionRepository to kanban.CommissionRepository
type kanbanCommissionRepoAdapter struct {
	repo storage.CommissionRepository
}

func (k *kanbanCommissionRepoAdapter) CreateCommission(ctx context.Context, commission interface{}) error {
	if storageCommission, ok := commission.(*storage.Commission); ok {
		return k.repo.CreateCommission(ctx, storageCommission)
	}
	// Handle map[string]interface{} case for kanban compatibility
	if commissionMap, ok := commission.(map[string]interface{}); ok {
		// Convert map to storage.Commission
		storageCommission := &storage.Commission{
			ID:         commissionMap["ID"].(string),
			CampaignID: commissionMap["CampaignID"].(string),
			Title:      commissionMap["Title"].(string),
			Status:     commissionMap["Status"].(string),
		}
		if createdAt, ok := commissionMap["CreatedAt"].(time.Time); ok {
			storageCommission.CreatedAt = createdAt
		}
		return k.repo.CreateCommission(ctx, storageCommission)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid commission type", nil).
		WithComponent("kanbanCommissionRepoAdapter").
		WithOperation("CreateCommission")
}

func (k *kanbanCommissionRepoAdapter) GetCommission(ctx context.Context, id string) (interface{}, error) {
	return k.repo.GetCommission(ctx, id)
}

func (k *kanbanCampaignRepoAdapter) CreateCampaign(ctx context.Context, campaign interface{}) error {
	if storageCampaign, ok := campaign.(*storage.Campaign); ok {
		return k.repo.CreateCampaign(ctx, storageCampaign)
	}
	// Handle map[string]interface{} case for kanban compatibility
	if campaignMap, ok := campaign.(map[string]interface{}); ok {
		// Convert map to storage.Campaign
		storageCampaign := &storage.Campaign{
			ID:     campaignMap["ID"].(string),
			Name:   campaignMap["Name"].(string),
			Status: campaignMap["Status"].(string),
		}
		if createdAt, ok := campaignMap["CreatedAt"].(time.Time); ok {
			storageCampaign.CreatedAt = createdAt
		}
		if updatedAt, ok := campaignMap["UpdatedAt"].(time.Time); ok {
			storageCampaign.UpdatedAt = updatedAt
		}
		return k.repo.CreateCampaign(ctx, storageCampaign)
	}
	return gerror.New(gerror.ErrCodeValidation, "invalid campaign type", nil).
		WithComponent("kanbanCampaignRepoAdapter").
		WithOperation("CreateCampaign")
}

// memoryStoreAdapter wraps generic memory store to implement kanban.MemoryStore
type memoryStoreAdapter struct {
	memStore interface{}
}

func (m *memoryStoreAdapter) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	// This is a simplified adapter - real implementation would need to match interfaces
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "memory store adapter not fully implemented", nil).
		WithComponent("memoryStoreAdapter").
		WithOperation("Get")
}

func (m *memoryStoreAdapter) Put(ctx context.Context, bucket, key string, value []byte) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "memory store adapter not fully implemented", nil).
		WithComponent("memoryStoreAdapter").
		WithOperation("Put")
}

func (m *memoryStoreAdapter) Delete(ctx context.Context, bucket, key string) error {
	return gerror.New(gerror.ErrCodeNotImplemented, "memory store adapter not fully implemented", nil).
		WithComponent("memoryStoreAdapter").
		WithOperation("Delete")
}

func (m *memoryStoreAdapter) List(ctx context.Context, bucket string) ([]string, error) {
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "memory store adapter not fully implemented", nil).
		WithComponent("memoryStoreAdapter").
		WithOperation("List")
}
