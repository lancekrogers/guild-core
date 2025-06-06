package kanban_test

import (
	"context"
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/kanban"
	"github.com/guild-ventures/guild-core/pkg/kanban/mocks"
)

// setupTestBoard creates a board for testing
func setupTestBoard(t *testing.T) (*kanban.Board, *mocks.MockMemoryStore) {
	store := mocks.NewMockMemoryStore()
	board, err := kanban.NewBoard(store, "Test Board", "A board for testing")
	if err != nil {
		t.Fatalf("Failed to create test board: %v", err)
	}
	return board, store
}

// TestBoardCreation tests the creation of a new board
func TestBoardCreation(t *testing.T) {
	store := mocks.NewMockMemoryStore()
	name := "Test Board"
	description := "A board for testing"

	board, err := kanban.NewBoard(store, name, description)
	if err != nil {
		t.Fatalf("Failed to create board: %v", err)
	}

	if board.Name != name {
		t.Errorf("Expected board name %s, got %s", name, board.Name)
	}

	if board.Description != description {
		t.Errorf("Expected board description %s, got %s", description, board.Description)
	}

	if board.ID == "" {
		t.Error("Expected non-empty board ID")
	}

	if board.CreatedAt.IsZero() {
		t.Error("Expected non-zero creation time")
	}

	if board.UpdatedAt.IsZero() {
		t.Error("Expected non-zero update time")
	}

	// Test nil store
	_, err = kanban.NewBoard(nil, name, description)
	if err == nil {
		t.Error("Expected error with nil store, got nil")
	}
}

// TestLoadBoard tests loading a board from the store
func TestLoadBoard(t *testing.T) {
	// Create a board
	store := mocks.NewMockMemoryStore()
	originalBoard, err := kanban.NewBoard(store, "Test Board", "A board for testing")
	if err != nil {
		t.Fatalf("Failed to create board: %v", err)
	}

	// Load the board
	loadedBoard, err := kanban.LoadBoard(store, originalBoard.ID)
	if err != nil {
		t.Fatalf("Failed to load board: %v", err)
	}

	// Compare properties
	if loadedBoard.ID != originalBoard.ID {
		t.Errorf("Expected loaded board ID %s, got %s", originalBoard.ID, loadedBoard.ID)
	}

	if loadedBoard.Name != originalBoard.Name {
		t.Errorf("Expected loaded board name %s, got %s", originalBoard.Name, loadedBoard.Name)
	}

	if loadedBoard.Description != originalBoard.Description {
		t.Errorf("Expected loaded board description %s, got %s", originalBoard.Description, loadedBoard.Description)
	}

	// Test with nil store
	_, err = kanban.LoadBoard(nil, originalBoard.ID)
	if err == nil {
		t.Error("Expected error with nil store, got nil")
	}

	// Test with non-existent board ID
	_, err = kanban.LoadBoard(store, "non-existent")
	if err == nil {
		t.Error("Expected error with non-existent board ID, got nil")
	}
}

// TestListBoards tests listing all boards in the store
func TestListBoards(t *testing.T) {
	store := mocks.NewMockMemoryStore()

	// Create multiple boards
	board1, err := kanban.NewBoard(store, "Board 1", "First test board")
	if err != nil {
		t.Fatalf("Failed to create board 1: %v", err)
	}

	board2, err := kanban.NewBoard(store, "Board 2", "Second test board")
	if err != nil {
		t.Fatalf("Failed to create board 2: %v", err)
	}

	// List boards
	boards, err := kanban.ListBoards(store)
	if err != nil {
		t.Fatalf("Failed to list boards: %v", err)
	}

	if len(boards) != 2 {
		t.Errorf("Expected 2 boards, got %d", len(boards))
	}

	// Verify boards are present
	found1, found2 := false, false
	for _, board := range boards {
		if board.ID == board1.ID {
			found1 = true
		}
		if board.ID == board2.ID {
			found2 = true
		}
	}

	if !found1 {
		t.Error("Board 1 not found in list")
	}

	if !found2 {
		t.Error("Board 2 not found in list")
	}

	// Test with nil store
	_, err = kanban.ListBoards(nil)
	if err == nil {
		t.Error("Expected error with nil store, got nil")
	}
}

// TestBoardSave tests saving a board to the store
func TestBoardSave(t *testing.T) {
	board, store := setupTestBoard(t)

	// Modify the board
	board.Name = "Updated Board"
	prevUpdateTime := board.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	// Save the board
	ctx := context.Background()
	err := board.Save(ctx)
	if err != nil {
		t.Fatalf("Failed to save board: %v", err)
	}

	// Verify update time changed
	if !board.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected board update time to be updated")
	}

	// Load the board and verify changes
	loadedBoard, err := kanban.LoadBoard(store, board.ID)
	if err != nil {
		t.Fatalf("Failed to load board after save: %v", err)
	}

	if loadedBoard.Name != "Updated Board" {
		t.Errorf("Expected updated name 'Updated Board', got '%s'", loadedBoard.Name)
	}

	// Test saving after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.Save(cancelledCtx)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestBoardDelete tests deleting a board from the store
func TestBoardDelete(t *testing.T) {
	board, store := setupTestBoard(t)
	ctx := context.Background()

	// Create a task on the board
	_, err := board.CreateTask(ctx, "Test Task", "A task on the board")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Delete the board
	err = board.Delete(ctx)
	if err != nil {
		t.Fatalf("Failed to delete board: %v", err)
	}

	// Try to load the deleted board
	_, err = kanban.LoadBoard(store, board.ID)
	if err == nil {
		t.Error("Expected error loading deleted board, got nil")
	}

	// Test deleting after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.Delete(cancelledCtx)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestCreateTask tests creating a task on a board
func TestCreateTask(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	title := "Test Task"
	description := "A task description"

	// Record current update time
	prevUpdateTime := board.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	// Create a task
	task, err := board.CreateTask(ctx, title, description)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	if task.Title != title {
		t.Errorf("Expected task title %s, got %s", title, task.Title)
	}

	if task.Description != description {
		t.Errorf("Expected task description %s, got %s", description, task.Description)
	}

	if task.Status != kanban.StatusBacklog {
		t.Errorf("Expected task status %s, got %s", kanban.StatusBacklog, task.Status)
	}

	if task.ID == "" {
		t.Error("Expected non-empty task ID")
	}

	// Verify board ID in task metadata
	if task.Metadata["board_id"] != board.ID {
		t.Errorf("Expected board ID %s in task metadata, got %s", board.ID, task.Metadata["board_id"])
	}

	// Verify board update time changed
	if !board.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected board update time to be updated")
	}

	// Test creating task after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = board.CreateTask(cancelledCtx, title, description)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestGetTask tests retrieving a task from a board
func TestGetTask(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create a task
	title := "Test Task"
	description := "A task description"
	task, err := board.CreateTask(ctx, title, description)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Get the task
	retrievedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrievedTask.ID)
	}

	if retrievedTask.Title != title {
		t.Errorf("Expected task title %s, got %s", title, retrievedTask.Title)
	}

	// Test getting task after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with non-existent task ID
	board, _ = setupTestBoard(t)
	_, err = board.GetTask(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error with non-existent task ID, got nil")
	}

	// Test with task from another board
	otherBoard, _ := setupTestBoard(t)
	otherTask, err := otherBoard.CreateTask(ctx, "Other Task", "Task on another board")
	if err != nil {
		t.Fatalf("Failed to create task on other board: %v", err)
	}

	_, err = board.GetTask(ctx, otherTask.ID)
	if err == nil {
		t.Error("Expected error getting task from another board, got nil")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = board.GetTask(cancelledCtx, task.ID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestUpdateTask tests updating a task on a board
func TestUpdateTask(t *testing.T) {
	board, store := setupTestBoard(t)
	ctx := context.Background()

	// Create a task
	task, err := board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Record current update times
	prevTaskUpdateTime := task.UpdatedAt
	prevBoardUpdateTime := board.UpdatedAt

	// Wait a moment to ensure update times change
	time.Sleep(5 * time.Millisecond)

	// Modify the task
	task.Title = "Updated Task"
	task.Description = "Updated description"
	task.Status = kanban.StatusInProgress

	// Update the task
	err = board.UpdateTask(ctx, task)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Verify task was updated
	updatedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	if updatedTask.Title != "Updated Task" {
		t.Errorf("Expected updated title 'Updated Task', got '%s'", updatedTask.Title)
	}

	if updatedTask.Status != kanban.StatusInProgress {
		t.Errorf("Expected updated status %s, got %s", kanban.StatusInProgress, updatedTask.Status)
	}

	// Verify task update time changed
	if !updatedTask.UpdatedAt.After(prevTaskUpdateTime) {
		t.Error("Expected task update time to be updated")
	}

	// Verify board was updated
	updatedBoard, err := kanban.LoadBoard(store, board.ID)
	if err != nil {
		t.Fatalf("Failed to load updated board: %v", err)
	}

	if !updatedBoard.UpdatedAt.After(prevBoardUpdateTime) {
		t.Error("Expected board update time to be updated")
	}

	// Test updating task after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with task from another board
	board, _ = setupTestBoard(t)
	otherBoard, _ := setupTestBoard(t)
	otherTask, err := otherBoard.CreateTask(ctx, "Other Task", "Task on another board")
	if err != nil {
		t.Fatalf("Failed to create task on other board: %v", err)
	}

	err = board.UpdateTask(ctx, otherTask)
	if err == nil {
		t.Error("Expected error updating task from another board, got nil")
	}

	// Test with cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.UpdateTask(cancelledCtx, task)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestDeleteTask tests deleting a task from a board
func TestDeleteTask(t *testing.T) {
	board, store := setupTestBoard(t)
	ctx := context.Background()

	// Create a task
	task, err := board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Record current board update time
	prevBoardUpdateTime := board.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	// Delete the task
	err = board.DeleteTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}

	// Try to get the deleted task
	_, err = board.GetTask(ctx, task.ID)
	if err == nil {
		t.Error("Expected error getting deleted task, got nil")
	}

	// Verify board update time changed
	updatedBoard, err := kanban.LoadBoard(store, board.ID)
	if err != nil {
		t.Fatalf("Failed to load updated board: %v", err)
	}

	if !updatedBoard.UpdatedAt.After(prevBoardUpdateTime) {
		t.Error("Expected board update time to be updated")
	}

	// Test deleting task after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with non-existent task ID
	board, _ = setupTestBoard(t)
	err = board.DeleteTask(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error with non-existent task ID, got nil")
	}

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	task, err = board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.DeleteTask(cancelledCtx, task.ID)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestGetTasksByStatus tests getting tasks by status
func TestGetTasksByStatus(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create tasks with different statuses
	todoTask, err := board.CreateTask(ctx, "Todo Task", "A todo task")
	if err != nil {
		t.Fatalf("Failed to create todo task: %v", err)
	}
	todoTask.Status = kanban.StatusTodo
	err = board.UpdateTask(ctx, todoTask)
	if err != nil {
		t.Fatalf("Failed to update todo task: %v", err)
	}

	inProgressTask, err := board.CreateTask(ctx, "In Progress Task", "An in-progress task")
	if err != nil {
		t.Fatalf("Failed to create in-progress task: %v", err)
	}
	inProgressTask.Status = kanban.StatusInProgress
	err = board.UpdateTask(ctx, inProgressTask)
	if err != nil {
		t.Fatalf("Failed to update in-progress task: %v", err)
	}

	doneTask, err := board.CreateTask(ctx, "Done Task", "A done task")
	if err != nil {
		t.Fatalf("Failed to create done task: %v", err)
	}
	doneTask.Status = kanban.StatusDone
	err = board.UpdateTask(ctx, doneTask)
	if err != nil {
		t.Fatalf("Failed to update done task: %v", err)
	}

	// Get todo tasks
	todoTasks, err := board.GetTasksByStatus(ctx, kanban.StatusTodo)
	if err != nil {
		t.Fatalf("Failed to get todo tasks: %v", err)
	}

	if len(todoTasks) != 1 {
		t.Errorf("Expected 1 todo task, got %d", len(todoTasks))
	}

	if len(todoTasks) > 0 && todoTasks[0].ID != todoTask.ID {
		t.Errorf("Expected todo task ID %s, got %s", todoTask.ID, todoTasks[0].ID)
	}

	// Get in-progress tasks
	inProgressTasks, err := board.GetTasksByStatus(ctx, kanban.StatusInProgress)
	if err != nil {
		t.Fatalf("Failed to get in-progress tasks: %v", err)
	}

	if len(inProgressTasks) != 1 {
		t.Errorf("Expected 1 in-progress task, got %d", len(inProgressTasks))
	}

	if len(inProgressTasks) > 0 && inProgressTasks[0].ID != inProgressTask.ID {
		t.Errorf("Expected in-progress task ID %s, got %s", inProgressTask.ID, inProgressTasks[0].ID)
	}

	// Get done tasks
	doneTasks, err := board.GetTasksByStatus(ctx, kanban.StatusDone)
	if err != nil {
		t.Fatalf("Failed to get done tasks: %v", err)
	}

	if len(doneTasks) != 1 {
		t.Errorf("Expected 1 done task, got %d", len(doneTasks))
	}

	if len(doneTasks) > 0 && doneTasks[0].ID != doneTask.ID {
		t.Errorf("Expected done task ID %s, got %s", doneTask.ID, doneTasks[0].ID)
	}

	// Get tasks with a status that has no tasks
	blockedTasks, err := board.GetTasksByStatus(ctx, kanban.StatusBlocked)
	if err != nil {
		t.Fatalf("Failed to get blocked tasks: %v", err)
	}

	if len(blockedTasks) != 0 {
		t.Errorf("Expected 0 blocked tasks, got %d", len(blockedTasks))
	}

	// Test getting tasks by status after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = board.GetTasksByStatus(cancelledCtx, kanban.StatusTodo)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestGetAllTasks tests getting all tasks on a board
func TestGetAllTasks(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create tasks with different statuses
	_, err := board.CreateTask(ctx, "Task 1", "First task")
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}

	task2, err := board.CreateTask(ctx, "Task 2", "Second task")
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}
	task2.Status = kanban.StatusInProgress
	err = board.UpdateTask(ctx, task2)
	if err != nil {
		t.Fatalf("Failed to update task 2: %v", err)
	}

	task3, err := board.CreateTask(ctx, "Task 3", "Third task")
	if err != nil {
		t.Fatalf("Failed to create task 3: %v", err)
	}
	task3.Status = kanban.StatusDone
	err = board.UpdateTask(ctx, task3)
	if err != nil {
		t.Fatalf("Failed to update task 3: %v", err)
	}

	// Get all tasks
	allTasks, err := board.GetAllTasks(ctx)
	if err != nil {
		t.Fatalf("Failed to get all tasks: %v", err)
	}

	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}

	// Verify tasks are sorted by updated time (newest first)
	if len(allTasks) == 3 {
		// Last updated task should be first
		if allTasks[0].ID != task3.ID {
			t.Errorf("Expected first task to be task 3, got %s", allTasks[0].ID)
		}
	}

	// Test getting all tasks after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = board.GetAllTasks(cancelledCtx)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestFilterTasks tests filtering tasks based on different criteria
func TestFilterTasks(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create tasks with different properties
	task1, err := board.CreateTask(ctx, "Task 1", "First task")
	if err != nil {
		t.Fatalf("Failed to create task 1: %v", err)
	}
	task1.Status = kanban.StatusTodo
	task1.Priority = kanban.PriorityHigh
	task1.AssignedTo = "alice"
	task1.Tags = []string{"frontend", "bugfix"}
	err = board.UpdateTask(ctx, task1)
	if err != nil {
		t.Fatalf("Failed to update task 1: %v", err)
	}

	task2, err := board.CreateTask(ctx, "Task 2", "Second task")
	if err != nil {
		t.Fatalf("Failed to create task 2: %v", err)
	}
	task2.Status = kanban.StatusInProgress
	task2.Priority = kanban.PriorityMedium
	task2.AssignedTo = "bob"
	task2.Tags = []string{"backend", "feature"}
	err = board.UpdateTask(ctx, task2)
	if err != nil {
		t.Fatalf("Failed to update task 2: %v", err)
	}

	task3, err := board.CreateTask(ctx, "Task 3", "Third task")
	if err != nil {
		t.Fatalf("Failed to create task 3: %v", err)
	}
	task3.Status = kanban.StatusTodo
	task3.Priority = kanban.PriorityLow
	task3.AssignedTo = "alice"
	task3.Tags = []string{"documentation"}
	err = board.UpdateTask(ctx, task3)
	if err != nil {
		t.Fatalf("Failed to update task 3: %v", err)
	}

	// Filter by status
	todoTasks, err := board.FilterTasks(ctx, kanban.FilterByStatus(kanban.StatusTodo))
	if err != nil {
		t.Fatalf("Failed to filter tasks by status: %v", err)
	}

	if len(todoTasks) != 2 {
		t.Errorf("Expected 2 todo tasks, got %d", len(todoTasks))
	}

	// Filter by assignee
	aliceTasks, err := board.FilterTasks(ctx, kanban.FilterByAssignee("alice"))
	if err != nil {
		t.Fatalf("Failed to filter tasks by assignee: %v", err)
	}

	if len(aliceTasks) != 2 {
		t.Errorf("Expected 2 tasks assigned to Alice, got %d", len(aliceTasks))
	}

	// Filter by priority
	highPriorityTasks, err := board.FilterTasks(ctx, kanban.FilterByPriority(kanban.PriorityHigh))
	if err != nil {
		t.Fatalf("Failed to filter tasks by priority: %v", err)
	}

	if len(highPriorityTasks) != 1 {
		t.Errorf("Expected 1 high priority task, got %d", len(highPriorityTasks))
	}

	// Filter by tag
	frontendTasks, err := board.FilterTasks(ctx, kanban.FilterByTag("frontend"))
	if err != nil {
		t.Fatalf("Failed to filter tasks by tag: %v", err)
	}

	if len(frontendTasks) != 1 {
		t.Errorf("Expected 1 frontend task, got %d", len(frontendTasks))
	}

	// Combine filters
	todoAliceTasks, err := board.FilterTasks(ctx, kanban.CombineFilters(
		kanban.FilterByStatus(kanban.StatusTodo),
		kanban.FilterByAssignee("alice"),
	))
	if err != nil {
		t.Fatalf("Failed to filter tasks with combined filters: %v", err)
	}

	if len(todoAliceTasks) != 2 {
		t.Errorf("Expected 2 todo tasks assigned to Alice, got %d", len(todoAliceTasks))
	}

	// Add more specific filter
	todoAliceHighPriorityTasks, err := board.FilterTasks(ctx, kanban.CombineFilters(
		kanban.FilterByStatus(kanban.StatusTodo),
		kanban.FilterByAssignee("alice"),
		kanban.FilterByPriority(kanban.PriorityHigh),
	))
	if err != nil {
		t.Fatalf("Failed to filter tasks with three combined filters: %v", err)
	}

	if len(todoAliceHighPriorityTasks) != 1 {
		t.Errorf("Expected 1 high priority todo task assigned to Alice, got %d", len(todoAliceHighPriorityTasks))
	}

	// Test filtering tasks after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = board.FilterTasks(cancelledCtx, kanban.FilterByStatus(kanban.StatusTodo))
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestUpdateTaskStatus tests updating a task's status
func TestUpdateTaskStatus(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create a task
	task, err := board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Initial status should be backlog
	if task.Status != kanban.StatusBacklog {
		t.Errorf("Expected initial status %s, got %s", kanban.StatusBacklog, task.Status)
	}

	// Update the task status
	changedBy := "tester"
	comment := "Moving to in progress"
	newStatus := kanban.StatusInProgress

	err = board.UpdateTaskStatus(ctx, task.ID, newStatus, changedBy, comment)
	if err != nil {
		t.Fatalf("Failed to update task status: %v", err)
	}

	// Get the updated task
	updatedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	// Verify status changed
	if updatedTask.Status != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, updatedTask.Status)
	}

	// Verify history was recorded
	if len(updatedTask.History) < 1 {
		t.Fatalf("Expected at least 1 history entry, got %d", len(updatedTask.History))
	}

	// Test with invalid status
	err = board.UpdateTaskStatus(ctx, task.ID, "invalid_status", changedBy, comment)
	if err == nil {
		t.Error("Expected error with invalid status, got nil")
	}

	// Test updating task status after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with non-existent task ID
	board, _ = setupTestBoard(t)
	err = board.UpdateTaskStatus(ctx, "non-existent", kanban.StatusDone, changedBy, comment)
	if err == nil {
		t.Error("Expected error with non-existent task ID, got nil")
	}

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	task, err = board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.UpdateTaskStatus(cancelledCtx, task.ID, kanban.StatusDone, changedBy, comment)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestAssignTask tests assigning a task to a user
func TestAssignTask(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create a task
	task, err := board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Initial assignee should be empty
	if task.AssignedTo != "" {
		t.Errorf("Expected initial assignee to be empty, got %s", task.AssignedTo)
	}

	// Assign the task
	assignee := "alice"
	changedBy := "tester"
	comment := "Assigning to Alice"

	err = board.AssignTask(ctx, task.ID, assignee, changedBy, comment)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}

	// Get the updated task
	updatedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	// Verify assignee changed
	if updatedTask.AssignedTo != assignee {
		t.Errorf("Expected assignee %s, got %s", assignee, updatedTask.AssignedTo)
	}

	// Verify history was recorded
	if len(updatedTask.History) < 1 {
		t.Fatalf("Expected at least 1 history entry, got %d", len(updatedTask.History))
	}

	// Test assigning task after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with non-existent task ID
	board, _ = setupTestBoard(t)
	err = board.AssignTask(ctx, "non-existent", assignee, changedBy, comment)
	if err == nil {
		t.Error("Expected error with non-existent task ID, got nil")
	}

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	task, err = board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.AssignTask(cancelledCtx, task.ID, assignee, changedBy, comment)
	if err == nil {
		t.Error("Expected error with cancelled context, got nil")
	}
}

// TestAddRemoveTaskBlocker tests adding and removing task blockers
func TestAddRemoveTaskBlocker(t *testing.T) {
	board, _ := setupTestBoard(t)
	ctx := context.Background()

	// Create a task
	task, err := board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Initial blockers should be empty
	if len(task.Blockers) != 0 {
		t.Errorf("Expected initial blockers to be empty, got %v", task.Blockers)
	}

	// Add a blocker
	blockerID := "blocker-1"
	changedBy := "tester"
	comment := "Adding blocker"

	err = board.AddTaskBlocker(ctx, task.ID, blockerID, changedBy, comment)
	if err != nil {
		t.Fatalf("Failed to add task blocker: %v", err)
	}

	// Get the updated task
	updatedTask, err := board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task: %v", err)
	}

	// Verify blocker added
	if len(updatedTask.Blockers) != 1 {
		t.Fatalf("Expected 1 blocker, got %d", len(updatedTask.Blockers))
	}

	if updatedTask.Blockers[0] != blockerID {
		t.Errorf("Expected blocker %s, got %s", blockerID, updatedTask.Blockers[0])
	}

	// Verify status changed to blocked
	if updatedTask.Status != kanban.StatusBlocked {
		t.Errorf("Expected status to change to blocked, got %s", updatedTask.Status)
	}

	// Remove the blocker
	err = board.RemoveTaskBlocker(ctx, task.ID, blockerID, changedBy, "Removing blocker")
	if err != nil {
		t.Fatalf("Failed to remove task blocker: %v", err)
	}

	// Get the updated task again
	updatedTask, err = board.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Failed to get updated task after removing blocker: %v", err)
	}

	// Verify blocker removed
	if len(updatedTask.Blockers) != 0 {
		t.Fatalf("Expected 0 blockers after removal, got %d", len(updatedTask.Blockers))
	}

	// Verify status changed to todo
	if updatedTask.Status != kanban.StatusTodo {
		t.Errorf("Expected status to change to todo after removing blocker, got %s", updatedTask.Status)
	}

	// Test adding/removing blocker after board deletion by recreating a board with nil store handling
	// Since we can't directly access the store field, we'll skip this nil store test

	// Test with non-existent task ID
	board, _ = setupTestBoard(t)
	err = board.AddTaskBlocker(ctx, "non-existent", blockerID, changedBy, comment)
	if err == nil {
		t.Error("Expected error with non-existent task ID when adding blocker, got nil")
	}

	err = board.RemoveTaskBlocker(ctx, "non-existent", blockerID, changedBy, comment)
	if err == nil {
		t.Error("Expected error with non-existent task ID when removing blocker, got nil")
	}

	// Test with cancelled context
	board, _ = setupTestBoard(t)
	task, err = board.CreateTask(ctx, "Test Task", "A task description")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = board.AddTaskBlocker(cancelledCtx, task.ID, blockerID, changedBy, comment)
	if err == nil {
		t.Error("Expected error with cancelled context when adding blocker, got nil")
	}

	err = board.RemoveTaskBlocker(cancelledCtx, task.ID, blockerID, changedBy, comment)
	if err == nil {
		t.Error("Expected error with cancelled context when removing blocker, got nil")
	}
}