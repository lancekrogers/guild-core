// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package kanban

import (
	"context"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// TestKanbanSQLiteIntegration tests the end-to-end kanban SQLite integration
func TestKanbanSQLiteIntegration(t *testing.T) {
	ctx := context.Background()

	// Initialize SQLite storage for testing
	storageReg, memoryStore, err := storage.InitializeSQLiteStorageForTests(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize SQLite storage: %v", err)
	}

	// Create a mock ComponentRegistry for testing
	componentRegistry := &MockComponentRegistry{
		storageReg:  storageReg,
		memoryStore: memoryStore,
	}

	// Test kanban manager creation with registry
	manager, err := NewManagerWithRegistry(ctx, componentRegistry)
	if err != nil {
		t.Fatalf("Failed to create kanban manager: %v", err)
	}
	defer manager.Close()

	// Test board creation
	board, err := manager.CreateBoard(ctx, "Test Board", "A test board for SQLite integration")
	if err != nil {
		t.Fatalf("Failed to create board: %v", err)
	}

	if board.ID == "" {
		t.Error("Board ID should not be empty")
	}

	if board.Name != "Test Board" {
		t.Errorf("Expected board name 'Test Board', got '%s'", board.Name)
	}

	// Test task creation
	task, err := board.CreateTask(ctx, "Test Task", "A test task for SQLite integration")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task.ID == "" {
		t.Error("Task ID should not be empty")
	}

	if task.Title != "Test Task" {
		t.Errorf("Expected task title 'Test Task', got '%s'", task.Title)
	}

	if task.Status != StatusBacklog {
		t.Errorf("Expected task status '%s', got '%s'", StatusBacklog, task.Status)
	}

	// Test task retrieval
	retrievedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve task: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID '%s', got '%s'", task.ID, retrievedTask.ID)
	}

	// Test task status update
	err = board.UpdateTaskStatus(ctx, task.ID, StatusInProgress, "test-agent", "Starting work on task")
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Verify status was updated
	updatedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated task: %v", err)
	}

	if updatedTask.Status != StatusInProgress {
		t.Errorf("Expected task status '%s', got '%s'", StatusInProgress, updatedTask.Status)
	}

	// Test task assignment
	err = board.AssignTask(ctx, task.ID, "test-agent", "manager", "Assigning task to agent")
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Test board listing
	boards, err := manager.ListBoards(ctx)
	if err != nil {
		t.Fatalf("Failed to list boards: %v", err)
	}

	if len(boards) != 1 {
		t.Errorf("Expected 1 board, got %d", len(boards))
	}

	if boards[0].ID != board.ID {
		t.Errorf("Expected board ID '%s', got '%s'", board.ID, boards[0].ID)
	}

	// Test task deletion
	err = board.DeleteTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Verify task was deleted (should fail to retrieve)
	_, err = board.GetTask(ctx, task.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted task, but got none")
	}

	// Test board deletion
	err = manager.DeleteBoard(ctx, board.ID)
	if err != nil {
		t.Fatalf("Failed to delete board: %v", err)
	}

	// Verify board was deleted
	boards, err = manager.ListBoards(ctx)
	if err != nil {
		t.Fatalf("Failed to list boards after deletion: %v", err)
	}

	if len(boards) != 0 {
		t.Errorf("Expected 0 boards after deletion, got %d", len(boards))
	}
}

// TestKanbanAdapterConversions tests the adapter type conversions
func TestKanbanAdapterConversions(t *testing.T) {
	ctx := context.Background()

	// Initialize SQLite storage for testing
	storageReg, _, err := storage.InitializeSQLiteStorageForTests(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize SQLite storage: %v", err)
	}

	// Create dependencies first: Campaign -> Commission -> Board

	// Test campaign adapter
	campaignAdapter := storage.NewKanbanCampaignRepositoryAdapter(storageReg.GetCampaignRepository())

	// Test map to campaign conversion
	campaignMap := map[string]interface{}{
		"ID":     "test-campaign-1",
		"Name":   "Test Campaign",
		"Status": "active",
	}

	err = campaignAdapter.CreateCampaign(ctx, campaignMap)
	if err != nil {
		t.Fatalf("Failed to create campaign from map: %v", err)
	}

	// Test commission adapter
	commissionAdapter := storage.NewKanbanCommissionRepositoryAdapter(storageReg.GetCommissionRepository())

	// Test map to commission conversion
	commissionMap := map[string]interface{}{
		"ID":         "test-commission-1",
		"CampaignID": "test-campaign-1",
		"Title":      "Test Commission",
		"Status":     "active",
	}

	err = commissionAdapter.CreateCommission(ctx, commissionMap)
	if err != nil {
		t.Fatalf("Failed to create commission from map: %v", err)
	}

	// Test board adapter
	boardAdapter := storage.NewKanbanBoardRepositoryAdapter(storageReg.GetBoardRepository())

	// Test map to board conversion
	boardMap := map[string]interface{}{
		"ID":           "test-board-1",
		"CommissionID": "test-commission-1",
		"Name":         "Test Board",
		"Description":  "Test board description",
		"Status":       "active",
	}

	err = boardAdapter.CreateBoard(ctx, boardMap)
	if err != nil {
		t.Fatalf("Failed to create board from map: %v", err)
	}

	// Test task adapter
	taskAdapter := storage.NewKanbanTaskRepositoryAdapter(storageReg.GetTaskRepository())

	// Test map to task conversion
	taskMap := map[string]interface{}{
		"ID":           "test-task-1",
		"BoardID":      "test-board-1",
		"CommissionID": "test-commission-1",
		"Title":        "Test Task",
		"Description":  "Test task description",
		"Status":       "todo",
		"StoryPoints":  int32(2),
		"Metadata":     map[string]interface{}{"test": "value"},
	}

	err = taskAdapter.CreateTask(ctx, taskMap)
	if err != nil {
		t.Fatalf("Failed to create task from map: %v", err)
	}
}

// MockComponentRegistry is a simple mock implementation for testing
type MockComponentRegistry struct {
	storageReg  storage.StorageRegistry
	memoryStore interface{}
}

func (m *MockComponentRegistry) Storage() StorageRegistry {
	return &MockStorageRegistry{
		storageReg:  m.storageReg,
		memoryStore: m.memoryStore,
	}
}

// Implement other ComponentRegistry methods as no-ops
func (m *MockComponentRegistry) Agents() registry.AgentRegistry { return nil }

func (m *MockComponentRegistry) Tools() registry.ToolRegistry { return nil }

func (m *MockComponentRegistry) Providers() registry.ProviderRegistry { return nil }

func (m *MockComponentRegistry) Memory() registry.MemoryRegistry { return nil }

func (m *MockComponentRegistry) Project() registry.ProjectRegistry { return nil }

func (m *MockComponentRegistry) Prompts() *registry.PromptRegistry { return nil }

func (m *MockComponentRegistry) Orchestrator() interface{} { return nil }

func (m *MockComponentRegistry) Initialize(ctx context.Context, config registry.Config) error {
	return nil
}

func (m *MockComponentRegistry) Shutdown(ctx context.Context) error { return nil }

func (m *MockComponentRegistry) GetAgentsByCost(maxCost int) []registry.AgentInfo { return nil }

func (m *MockComponentRegistry) GetCheapestAgentByCapability(capability string) (*registry.AgentInfo, error) {
	return nil, nil
}

func (m *MockComponentRegistry) GetToolsByCost(maxCost int) []registry.ToolInfo { return nil }

func (m *MockComponentRegistry) GetCheapestToolByCapability(capability string) (*registry.ToolInfo, error) {
	return nil, nil
}

func (m *MockComponentRegistry) GetAgentsByCapability(capability string) []registry.AgentInfo {
	return nil
}

// MockStorageRegistry is a mock storage registry for testing
type MockStorageRegistry struct {
	storageReg  storage.StorageRegistry
	memoryStore interface{}
}

func (m *MockStorageRegistry) GetKanbanCampaignRepository() CampaignRepository {
	return storage.NewKanbanCampaignRepositoryAdapter(m.storageReg.GetCampaignRepository())
}

func (m *MockStorageRegistry) GetKanbanCommissionRepository() CommissionRepository {
	return storage.NewKanbanCommissionRepositoryAdapter(m.storageReg.GetCommissionRepository())
}

func (m *MockStorageRegistry) GetBoardRepository() BoardRepository {
	return storage.NewKanbanBoardRepositoryAdapter(m.storageReg.GetBoardRepository())
}

func (m *MockStorageRegistry) GetKanbanTaskRepository() TaskRepository {
	return storage.NewKanbanTaskRepositoryAdapter(m.storageReg.GetTaskRepository())
}

func (m *MockStorageRegistry) GetMemoryStore() MemoryStore {
	if store, ok := m.memoryStore.(MemoryStore); ok {
		return store
	}
	return nil
}
