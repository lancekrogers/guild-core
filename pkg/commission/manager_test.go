package commission

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	
	"github.com/guild-ventures/guild-core/pkg/storage"
)

// mockCommissionRepository is a simple in-memory implementation for testing
type mockCommissionRepository struct {
	commissions map[string]*storage.Commission
}

func newMockCommissionRepository() *mockCommissionRepository {
	return &mockCommissionRepository{
		commissions: make(map[string]*storage.Commission),
	}
}

func (m *mockCommissionRepository) CreateCommission(ctx context.Context, commission *storage.Commission) error {
	m.commissions[commission.ID] = commission
	return nil
}

func (m *mockCommissionRepository) GetCommission(ctx context.Context, id string) (*storage.Commission, error) {
	commission, exists := m.commissions[id]
	if !exists {
		return nil, storage.ErrNotFound
	}
	return commission, nil
}

func (m *mockCommissionRepository) UpdateCommissionStatus(ctx context.Context, id, status string) error {
	commission, exists := m.commissions[id]
	if !exists {
		return storage.ErrNotFound
	}
	commission.Status = status
	return nil
}

func (m *mockCommissionRepository) DeleteCommission(ctx context.Context, id string) error {
	delete(m.commissions, id)
	return nil
}

func (m *mockCommissionRepository) ListCommissionsByCampaign(ctx context.Context, campaignID string) ([]*storage.Commission, error) {
	var result []*storage.Commission
	for _, commission := range m.commissions {
		if commission.CampaignID == campaignID {
			result = append(result, commission)
		}
	}
	return result, nil
}

func setupTestManager(t *testing.T) (*Manager, func()) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "commission-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create mock commission repository
	mockRepo := newMockCommissionRepository()
	
	// Create a commission manager using the factory
	managerInterface, err := DefaultCommissionManagerFactory(mockRepo, tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create commission manager: %v", err)
	}
	
	// Type assert to get concrete manager
	manager, ok := managerInterface.(*Manager)
	if !ok {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to type assert to *Manager")
	}
	
	// Initialize the manager
	if err := manager.Init(context.Background()); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// Return the manager and a cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}
	
	return manager, cleanup
}

func TestManager_SaveAndGetCommission(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	// Create a test commission
	commission := NewCommission("Test Commission", "This is a test commission")
	commission.Tags = []string{"test", "example"}
	commission.Priority = "high"
	
	// Save the commission
	ctx := context.Background()
	if err := manager.SaveCommission(ctx, commission); err != nil {
		t.Fatalf("Failed to save commissionective: %v", err)
	}
	
	// Get the commissionective
	retrievedCommission, err := manager.GetCommission(ctx, commission.ID)
	if err != nil {
		t.Fatalf("Failed to get commissionective: %v", err)
	}
	
	// Check properties
	if retrievedCommission.Title != commission.Title {
		t.Errorf("Expected title '%s', got '%s'", commission.Title, retrievedCommission.Title)
	}
	
	if retrievedCommission.Priority != commission.Priority {
		t.Errorf("Expected priority '%s', got '%s'", commission.Priority, retrievedCommission.Priority)
	}
	
	if len(retrievedCommission.Tags) != len(commission.Tags) {
		t.Errorf("Expected %d tags, got %d", len(commission.Tags), len(retrievedCommission.Tags))
	}
}

func TestManager_DeleteCommission(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	// Create a test commissionective
	commission := NewCommission("Test Commission", "This is a test commissionective")
	
	// Save the commissionective
	ctx := context.Background()
	if err := manager.SaveCommission(ctx, commission); err != nil {
		t.Fatalf("Failed to save commissionective: %v", err)
	}
	
	// Delete the commissionective
	if err := manager.DeleteCommission(ctx, commission.ID); err != nil {
		t.Fatalf("Failed to delete commissionective: %v", err)
	}
	
	// Try to get the deleted commissionective
	_, err := manager.GetCommission(ctx, commission.ID)
	if err == nil {
		t.Error("Expected error when getting deleted commissionective, got nil")
	}
}

func TestManager_ListCommissions(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test commissionectives
	commission1 := NewCommission("Test Commission 1", "First test commissionective")
	commission2 := NewCommission("Test Commission 2", "Second test commissionective")
	commission3 := NewCommission("Test Commission 3", "Third test commissionective")
	
	// Save commissionectives
	if err := manager.SaveCommission(ctx, commission1); err != nil {
		t.Fatalf("Failed to save commissionective 1: %v", err)
	}
	if err := manager.SaveCommission(ctx, commission2); err != nil {
		t.Fatalf("Failed to save commissionective 2: %v", err)
	}
	if err := manager.SaveCommission(ctx, commission3); err != nil {
		t.Fatalf("Failed to save commissionective 3: %v", err)
	}
	
	// List commissionectives
	commissionectives, err := manager.ListCommissions(ctx)
	if err != nil {
		t.Fatalf("Failed to list commissionectives: %v", err)
	}
	
	// Check count
	if len(commissionectives) != 3 {
		t.Errorf("Expected 3 commissionectives, got %d", len(commissionectives))
	}
}

func TestManager_FindCommissionsByTags(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test commissionectives with tags
	commission1 := NewCommission("Test Commission 1", "First test commissionective")
	commission1.Tags = []string{"test", "important"}
	
	commission2 := NewCommission("Test Commission 2", "Second test commissionective")
	commission2.Tags = []string{"test", "example"}
	
	commission3 := NewCommission("Test Commission 3", "Third test commissionective")
	commission3.Tags = []string{"example", "low-priority"}
	
	// Save commissionectives
	if err := manager.SaveCommission(ctx, commission1); err != nil {
		t.Fatalf("Failed to save commissionective 1: %v", err)
	}
	if err := manager.SaveCommission(ctx, commission2); err != nil {
		t.Fatalf("Failed to save commissionective 2: %v", err)
	}
	if err := manager.SaveCommission(ctx, commission3); err != nil {
		t.Fatalf("Failed to save commissionective 3: %v", err)
	}
	
	// Find commissionectives with a single tag
	commissionectives, err := manager.FindCommissionsByTags(ctx, []string{"test"})
	if err != nil {
		t.Fatalf("Failed to find commissionectives by tag: %v", err)
	}
	
	if len(commissionectives) != 2 {
		t.Errorf("Expected 2 commissionectives with tag 'test', got %d", len(commissionectives))
	}
	
	// Find commissionectives with multiple tags
	commissionectives, err = manager.FindCommissionsByTags(ctx, []string{"test", "example"})
	if err != nil {
		t.Fatalf("Failed to find commissionectives by tags: %v", err)
	}
	
	if len(commissionectives) != 1 {
		t.Errorf("Expected 1 commissionective with tags 'test' and 'example', got %d", len(commissionectives))
	}
	
	// Find commissionectives with a non-existent tag
	commissionectives, err = manager.FindCommissionsByTags(ctx, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Failed to find commissionectives by tag: %v", err)
	}
	
	if len(commissionectives) != 0 {
		t.Errorf("Expected 0 commissionectives with tag 'nonexistent', got %d", len(commissionectives))
	}
}

func TestManager_AddAndUpdateTask(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create a test commissionective
	commission := NewCommission("Test Commission", "This is a test commissionective")
	
	// Save the commissionective
	if err := manager.SaveCommission(ctx, commission); err != nil {
		t.Fatalf("Failed to save commissionective: %v", err)
	}
	
	// Create a test task
	task := NewCommissionTask("Test Task", "This is a test task", 0)
	
	// Add the task to the commissionective
	if err := manager.AddTask(ctx, commission.ID, task); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	
	// Get the updated commissionective
	updatedObj, err := manager.GetCommission(ctx, commission.ID)
	if err != nil {
		t.Fatalf("Failed to get updated commissionective: %v", err)
	}
	
	// Check task
	if len(updatedObj.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(updatedObj.Tasks))
	}
	
	if updatedObj.Tasks[0].Title != task.Title {
		t.Errorf("Expected task title '%s', got '%s'", task.Title, updatedObj.Tasks[0].Title)
	}
	
	// Update task status
	if err := manager.UpdateTaskStatus(ctx, commission.ID, task.ID, "done"); err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}
	
	// Get the updated commissionective again
	updatedObj, err = manager.GetCommission(ctx, commission.ID)
	if err != nil {
		t.Fatalf("Failed to get updated commissionective: %v", err)
	}
	
	// Check updated status
	if updatedObj.Tasks[0].Status != "done" {
		t.Errorf("Expected task status 'done', got '%s'", updatedObj.Tasks[0].Status)
	}
	
	// Check completion time
	if updatedObj.Tasks[0].CompletedAt == nil {
		t.Error("Expected completed time to be set, got nil")
	}
}

func TestManager_LoadCommissionFromFile(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create a test markdown file
	testContent := `# Test Commission

@priority: medium
@owner: tester
@tag:test @tag:example

This is a test commissionective loaded from a file.

## Context

This is the context section.

## Goals

- Goal 1
- Goal 2

## Implementation

- [ ] Task 1
- [ ] Task 2
- [x] Task 3
`
	
	testFile := filepath.Join(manager.fsBasePath, "test_commissionective.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Load the commissionective from file
	commission, err := manager.LoadCommissionFromFile(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to load commissionective from file: %v", err)
	}
	
	// Check properties
	if commission.Title != "Test Commission" {
		t.Errorf("Expected title 'Test Commission', got '%s'", commission.Title)
	}
	
	if commission.Priority != "medium" {
		t.Errorf("Expected priority 'medium', got '%s'", commission.Priority)
	}
	
	if commission.Owner != "tester" {
		t.Errorf("Expected owner 'tester', got '%s'", commission.Owner)
	}
	
	// Check tags
	expectedTags := []string{"test", "example"}
	if len(commission.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(commission.Tags))
	}
	
	// Check tasks
	if len(commission.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(commission.Tasks))
	}
	
	// Check completed task
	for _, task := range commission.Tasks {
		if task.Title == "Task 3" && task.Status != "done" {
			t.Errorf("Expected Task 3 to have status 'done', got '%s'", task.Status)
		}
	}
}