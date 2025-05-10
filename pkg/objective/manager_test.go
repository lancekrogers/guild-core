package objective

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
	
	"github.com/blockhead-consulting/Guild/pkg/memory/boltdb"
)

func setupTestManager(t *testing.T) (*Manager, func()) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "objective-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Create a temporary database file
	dbPath := filepath.Join(tempDir, "test.db")
	
	// Create a BoltDB store
	store, err := boltdb.NewStore(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create BoltDB store: %v", err)
	}
	
	// Create an objective manager
	manager, err := NewManager(store, tempDir)
	if err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create objective manager: %v", err)
	}
	
	// Initialize the manager
	if err := manager.Init(context.Background()); err != nil {
		store.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize manager: %v", err)
	}
	
	// Return the manager and a cleanup function
	cleanup := func() {
		store.Close()
		os.RemoveAll(tempDir)
	}
	
	return manager, cleanup
}

func TestManager_SaveAndGetObjective(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	// Create a test objective
	obj := NewObjective("Test Objective", "This is a test objective")
	obj.Tags = []string{"test", "example"}
	obj.Priority = "high"
	
	// Save the objective
	ctx := context.Background()
	if err := manager.SaveObjective(ctx, obj); err != nil {
		t.Fatalf("Failed to save objective: %v", err)
	}
	
	// Get the objective
	retrievedObj, err := manager.GetObjective(ctx, obj.ID)
	if err != nil {
		t.Fatalf("Failed to get objective: %v", err)
	}
	
	// Check properties
	if retrievedObj.Title != obj.Title {
		t.Errorf("Expected title '%s', got '%s'", obj.Title, retrievedObj.Title)
	}
	
	if retrievedObj.Priority != obj.Priority {
		t.Errorf("Expected priority '%s', got '%s'", obj.Priority, retrievedObj.Priority)
	}
	
	if len(retrievedObj.Tags) != len(obj.Tags) {
		t.Errorf("Expected %d tags, got %d", len(obj.Tags), len(retrievedObj.Tags))
	}
}

func TestManager_DeleteObjective(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	// Create a test objective
	obj := NewObjective("Test Objective", "This is a test objective")
	
	// Save the objective
	ctx := context.Background()
	if err := manager.SaveObjective(ctx, obj); err != nil {
		t.Fatalf("Failed to save objective: %v", err)
	}
	
	// Delete the objective
	if err := manager.DeleteObjective(ctx, obj.ID); err != nil {
		t.Fatalf("Failed to delete objective: %v", err)
	}
	
	// Try to get the deleted objective
	_, err := manager.GetObjective(ctx, obj.ID)
	if err == nil {
		t.Error("Expected error when getting deleted objective, got nil")
	}
}

func TestManager_ListObjectives(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test objectives
	obj1 := NewObjective("Test Objective 1", "First test objective")
	obj2 := NewObjective("Test Objective 2", "Second test objective")
	obj3 := NewObjective("Test Objective 3", "Third test objective")
	
	// Save objectives
	if err := manager.SaveObjective(ctx, obj1); err != nil {
		t.Fatalf("Failed to save objective 1: %v", err)
	}
	if err := manager.SaveObjective(ctx, obj2); err != nil {
		t.Fatalf("Failed to save objective 2: %v", err)
	}
	if err := manager.SaveObjective(ctx, obj3); err != nil {
		t.Fatalf("Failed to save objective 3: %v", err)
	}
	
	// List objectives
	objectives, err := manager.ListObjectives(ctx)
	if err != nil {
		t.Fatalf("Failed to list objectives: %v", err)
	}
	
	// Check count
	if len(objectives) != 3 {
		t.Errorf("Expected 3 objectives, got %d", len(objectives))
	}
}

func TestManager_FindObjectivesByTags(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create test objectives with tags
	obj1 := NewObjective("Test Objective 1", "First test objective")
	obj1.Tags = []string{"test", "important"}
	
	obj2 := NewObjective("Test Objective 2", "Second test objective")
	obj2.Tags = []string{"test", "example"}
	
	obj3 := NewObjective("Test Objective 3", "Third test objective")
	obj3.Tags = []string{"example", "low-priority"}
	
	// Save objectives
	if err := manager.SaveObjective(ctx, obj1); err != nil {
		t.Fatalf("Failed to save objective 1: %v", err)
	}
	if err := manager.SaveObjective(ctx, obj2); err != nil {
		t.Fatalf("Failed to save objective 2: %v", err)
	}
	if err := manager.SaveObjective(ctx, obj3); err != nil {
		t.Fatalf("Failed to save objective 3: %v", err)
	}
	
	// Find objectives with a single tag
	objectives, err := manager.FindObjectivesByTags(ctx, []string{"test"})
	if err != nil {
		t.Fatalf("Failed to find objectives by tag: %v", err)
	}
	
	if len(objectives) != 2 {
		t.Errorf("Expected 2 objectives with tag 'test', got %d", len(objectives))
	}
	
	// Find objectives with multiple tags
	objectives, err = manager.FindObjectivesByTags(ctx, []string{"test", "example"})
	if err != nil {
		t.Fatalf("Failed to find objectives by tags: %v", err)
	}
	
	if len(objectives) != 1 {
		t.Errorf("Expected 1 objective with tags 'test' and 'example', got %d", len(objectives))
	}
	
	// Find objectives with a non-existent tag
	objectives, err = manager.FindObjectivesByTags(ctx, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Failed to find objectives by tag: %v", err)
	}
	
	if len(objectives) != 0 {
		t.Errorf("Expected 0 objectives with tag 'nonexistent', got %d", len(objectives))
	}
}

func TestManager_AddAndUpdateTask(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create a test objective
	obj := NewObjective("Test Objective", "This is a test objective")
	
	// Save the objective
	if err := manager.SaveObjective(ctx, obj); err != nil {
		t.Fatalf("Failed to save objective: %v", err)
	}
	
	// Create a test task
	task := NewObjectiveTask("Test Task", "This is a test task", 0)
	
	// Add the task to the objective
	if err := manager.AddTask(ctx, obj.ID, task); err != nil {
		t.Fatalf("Failed to add task: %v", err)
	}
	
	// Get the updated objective
	updatedObj, err := manager.GetObjective(ctx, obj.ID)
	if err != nil {
		t.Fatalf("Failed to get updated objective: %v", err)
	}
	
	// Check task
	if len(updatedObj.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(updatedObj.Tasks))
	}
	
	if updatedObj.Tasks[0].Title != task.Title {
		t.Errorf("Expected task title '%s', got '%s'", task.Title, updatedObj.Tasks[0].Title)
	}
	
	// Update task status
	if err := manager.UpdateTaskStatus(ctx, obj.ID, task.ID, "done"); err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}
	
	// Get the updated objective again
	updatedObj, err = manager.GetObjective(ctx, obj.ID)
	if err != nil {
		t.Fatalf("Failed to get updated objective: %v", err)
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

func TestManager_LoadObjectiveFromFile(t *testing.T) {
	manager, cleanup := setupTestManager(t)
	defer cleanup()
	
	ctx := context.Background()
	
	// Create a test markdown file
	testContent := `# Test Objective

@priority: medium
@owner: tester
@tag:test @tag:example

This is a test objective loaded from a file.

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
	
	testFile := filepath.Join(manager.fsBasePath, "test_objective.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Load the objective from file
	obj, err := manager.LoadObjectiveFromFile(ctx, testFile)
	if err != nil {
		t.Fatalf("Failed to load objective from file: %v", err)
	}
	
	// Check properties
	if obj.Title != "Test Objective" {
		t.Errorf("Expected title 'Test Objective', got '%s'", obj.Title)
	}
	
	if obj.Priority != "medium" {
		t.Errorf("Expected priority 'medium', got '%s'", obj.Priority)
	}
	
	if obj.Owner != "tester" {
		t.Errorf("Expected owner 'tester', got '%s'", obj.Owner)
	}
	
	// Check tags
	expectedTags := []string{"test", "example"}
	if len(obj.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(obj.Tags))
	}
	
	// Check tasks
	if len(obj.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(obj.Tasks))
	}
	
	// Check completed task
	for _, task := range obj.Tasks {
		if task.Title == "Task 3" && task.Status != "done" {
			t.Errorf("Expected Task 3 to have status 'done', got '%s'", task.Status)
		}
	}
}