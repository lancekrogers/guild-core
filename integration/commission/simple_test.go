// +build integration

package commission_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/guild-ventures/guild-core/pkg/agent/manager"
	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/memory/boltdb"
	"github.com/guild-ventures/guild-core/pkg/prompts"
	"github.com/guild-ventures/guild-core/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreComponentInitialization tests basic component initialization
func TestCoreComponentInitialization(t *testing.T) {
	// Setup test directory
	tempDir, err := os.MkdirTemp("", "guild-simple-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Test 1: Registry
	reg := registry.NewComponentRegistry()
	require.NotNil(t, reg)
	fmt.Println("✓ Component registry created")

	// Test 2: BoltDB store
	dbPath := filepath.Join(tempDir, "test.db")
	store, err := boltdb.NewStore(dbPath)
	require.NoError(t, err)
	defer store.Close()
	fmt.Println("✓ BoltDB store created")

	// Test 3: Kanban manager
	kanbanManager, err := kanban.NewManager(store)
	require.NoError(t, err)
	require.NotNil(t, kanbanManager)
	fmt.Println("✓ Kanban manager created")

	// Test 4: Create a board
	board, err := kanbanManager.CreateBoard(ctx, "test-board", "Test Board")
	require.NoError(t, err)
	assert.NotEmpty(t, board.ID)
	fmt.Println("✓ Kanban board created")

	// Test 5: Response parser
	parser := manager.NewResponseParser()
	require.NotNil(t, parser)
	fmt.Println("✓ Response parser created")

	// Test 6: Prompt registry
	promptRegistry := prompts.NewMemoryRegistry()
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
	tempDir, err := os.MkdirTemp("", "guild-kanban-workflow-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Setup - include all kanban-related buckets
	dbPath := filepath.Join(tempDir, "kanban.db")
	store, err := boltdb.NewStore(dbPath, boltdb.WithCustomBuckets(
		"tasks_by_board_status",
		"board_events",
		"tasks_by_board",
	))
	require.NoError(t, err)
	defer store.Close()

	kanbanManager, err := kanban.NewManager(store)
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
	backlogTasks, err := kanbanManager.ListTasksByStatus(ctx, kanban.StatusBacklog)
	require.NoError(t, err)
	assert.Len(t, backlogTasks, 2) // Tasks 2 and 3 (still in backlog)

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
package models

type User struct {
    ID       string
    Email    string
    Username string
    Profile  UserProfile
}

## File: src/handlers/auth.go
package handlers

import "net/http"

func LoginHandler(w http.ResponseWriter, r *http.Request) {
    // Authentication logic here
}`,
	}

	structure, err := parser.ParseResponse(response)
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
