package kanban_test

import (
	"testing"
	"time"

	"github.com/guild-ventures/guild-core/pkg/kanban"
)

// TestNewTask tests the creation of a new task
func TestNewTask(t *testing.T) {
	title := "Test Task"
	description := "This is a test task"

	task := kanban.NewTask(title, description)

	if task.Title != title {
		t.Errorf("Expected task title %s, got %s", title, task.Title)
	}

	if task.Description != description {
		t.Errorf("Expected task description %s, got %s", description, task.Description)
	}

	if task.Status != kanban.StatusBacklog {
		t.Errorf("Expected task status %s, got %s", kanban.StatusBacklog, task.Status)
	}

	if task.Priority != kanban.PriorityMedium {
		t.Errorf("Expected task priority %s, got %s", kanban.PriorityMedium, task.Priority)
	}

	if task.Progress != 0 {
		t.Errorf("Expected task progress 0, got %d", task.Progress)
	}

	if task.ID == "" {
		t.Error("Expected non-empty task ID")
	}

	if task.CreatedAt.IsZero() {
		t.Error("Expected non-zero creation time")
	}

	if task.UpdatedAt.IsZero() {
		t.Error("Expected non-zero update time")
	}

	if len(task.Tags) != 0 {
		t.Errorf("Expected empty tags, got %v", task.Tags)
	}

	if len(task.Dependencies) != 0 {
		t.Errorf("Expected empty dependencies, got %v", task.Dependencies)
	}

	if len(task.Blockers) != 0 {
		t.Errorf("Expected empty blockers, got %v", task.Blockers)
	}

	if len(task.Notes) != 0 {
		t.Errorf("Expected empty notes, got %v", task.Notes)
	}

	if len(task.History) != 0 {
		t.Errorf("Expected empty history, got %v", task.History)
	}
}

// TestAddNote tests adding a note to a task
func TestAddNote(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	noteContent := "This is a test note"
	createdBy := "tester"

	// Record current update time
	prevUpdateTime := task.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	task.AddNote(noteContent, createdBy)

	if len(task.Notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(task.Notes))
	}

	note := task.Notes[0]

	if note.Content != noteContent {
		t.Errorf("Expected note content %s, got %s", noteContent, note.Content)
	}

	if note.CreatedBy != createdBy {
		t.Errorf("Expected note created by %s, got %s", createdBy, note.CreatedBy)
	}

	if note.CreatedAt.IsZero() {
		t.Error("Expected non-zero note creation time")
	}

	if note.ID == "" {
		t.Error("Expected non-empty note ID")
	}

	// Verify task update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated")
	}
}

// TestAddHistory tests adding a history entry to a task
func TestAddHistory(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	changedBy := "tester"
	comment := "Test comment"
	changes := map[string]string{
		"field1": "old value -> new value",
		"field2": "added",
	}

	task.AddHistory(changedBy, comment, changes)

	if len(task.History) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(task.History))
	}

	history := task.History[0]

	if history.ChangedBy != changedBy {
		t.Errorf("Expected history changed by %s, got %s", changedBy, history.ChangedBy)
	}

	if history.Comment != comment {
		t.Errorf("Expected history comment %s, got %s", comment, history.Comment)
	}

	if history.Timestamp.IsZero() {
		t.Error("Expected non-zero history timestamp")
	}

	if len(history.Changes) != len(changes) {
		t.Errorf("Expected %d changes, got %d", len(changes), len(history.Changes))
	}

	for k, v := range changes {
		if history.Changes[k] != v {
			t.Errorf("Expected change %s -> %s, got %s", k, v, history.Changes[k])
		}
	}
}

// TestUpdateStatus tests updating a task's status
func TestUpdateStatus(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	// Initial status should be backlog
	if task.Status != kanban.StatusBacklog {
		t.Errorf("Expected initial status %s, got %s", kanban.StatusBacklog, task.Status)
	}

	changedBy := "tester"
	comment := "Moving to in progress"
	newStatus := kanban.StatusInProgress

	// Record current update time
	prevUpdateTime := task.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	err := task.UpdateStatus(newStatus, changedBy, comment)
	if err != nil {
		t.Fatalf("Unexpected error updating status: %v", err)
	}

	// Verify status changed
	if task.Status != newStatus {
		t.Errorf("Expected status %s, got %s", newStatus, task.Status)
	}

	// Verify history was recorded
	if len(task.History) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(task.History))
	}

	history := task.History[0]

	if history.FromStatus != kanban.StatusBacklog {
		t.Errorf("Expected from status %s, got %s", kanban.StatusBacklog, history.FromStatus)
	}

	if history.ToStatus != newStatus {
		t.Errorf("Expected to status %s, got %s", newStatus, history.ToStatus)
	}

	if history.ChangedBy != changedBy {
		t.Errorf("Expected history changed by %s, got %s", changedBy, history.ChangedBy)
	}

	if history.Comment != comment {
		t.Errorf("Expected history comment %s, got %s", comment, history.Comment)
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated")
	}

	// Test invalid status
	err = task.UpdateStatus("invalid_status", changedBy, comment)
	if err == nil {
		t.Error("Expected error with invalid status, got nil")
	}

	// Test status-specific behavior

	// Done status should set progress to 100%
	err = task.UpdateStatus(kanban.StatusDone, changedBy, comment)
	if err != nil {
		t.Fatalf("Unexpected error updating status to done: %v", err)
	}

	if task.Progress != 100 {
		t.Errorf("Expected progress 100 for done status, got %d", task.Progress)
	}

	// Backlog status should set progress to 0%
	err = task.UpdateStatus(kanban.StatusBacklog, changedBy, comment)
	if err != nil {
		t.Fatalf("Unexpected error updating status to backlog: %v", err)
	}

	if task.Progress != 0 {
		t.Errorf("Expected progress 0 for backlog status, got %d", task.Progress)
	}
}

// TestUpdateAssignee tests updating a task's assignee
func TestUpdateAssignee(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	// Initial assignee should be empty
	if task.AssignedTo != "" {
		t.Errorf("Expected initial assignee to be empty, got %s", task.AssignedTo)
	}

	changedBy := "tester"
	comment := "Assigning to Alice"
	newAssignee := "alice"

	// Record current update time
	prevUpdateTime := task.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	task.UpdateAssignee(newAssignee, changedBy, comment)

	// Verify assignee changed
	if task.AssignedTo != newAssignee {
		t.Errorf("Expected assignee %s, got %s", newAssignee, task.AssignedTo)
	}

	// Verify history was recorded
	if len(task.History) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(task.History))
	}

	history := task.History[0]

	if history.FromAssignee != "" {
		t.Errorf("Expected from assignee '', got %s", history.FromAssignee)
	}

	if history.ToAssignee != newAssignee {
		t.Errorf("Expected to assignee %s, got %s", newAssignee, history.ToAssignee)
	}

	if history.ChangedBy != changedBy {
		t.Errorf("Expected history changed by %s, got %s", changedBy, history.ChangedBy)
	}

	if history.Comment != comment {
		t.Errorf("Expected history comment %s, got %s", comment, history.Comment)
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated")
	}

	// Change assignee again
	newAssignee2 := "bob"
	comment2 := "Reassigning to Bob"

	task.UpdateAssignee(newAssignee2, changedBy, comment2)

	// Verify assignee changed
	if task.AssignedTo != newAssignee2 {
		t.Errorf("Expected assignee %s, got %s", newAssignee2, task.AssignedTo)
	}

	// Verify history was recorded
	if len(task.History) != 2 {
		t.Fatalf("Expected 2 history entries, got %d", len(task.History))
	}

	history = task.History[1]

	if history.FromAssignee != newAssignee {
		t.Errorf("Expected from assignee %s, got %s", newAssignee, history.FromAssignee)
	}

	if history.ToAssignee != newAssignee2 {
		t.Errorf("Expected to assignee %s, got %s", newAssignee2, history.ToAssignee)
	}

	// Remove assignee
	task.UpdateAssignee("", changedBy, "Unassigning")

	// Verify assignee is empty
	if task.AssignedTo != "" {
		t.Errorf("Expected empty assignee, got %s", task.AssignedTo)
	}
}

// TestUpdateProgress tests updating a task's progress
func TestUpdateProgress(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	// Initial progress should be 0
	if task.Progress != 0 {
		t.Errorf("Expected initial progress 0, got %d", task.Progress)
	}

	changedBy := "tester"
	comment := "Updating progress"
	newProgress := 50

	// Record current update time
	prevUpdateTime := task.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	err := task.UpdateProgress(newProgress, changedBy, comment)
	if err != nil {
		t.Fatalf("Unexpected error updating progress: %v", err)
	}

	// Verify progress changed
	if task.Progress != newProgress {
		t.Errorf("Expected progress %d, got %d", newProgress, task.Progress)
	}

	// Verify history was recorded
	if len(task.History) != 1 {
		t.Fatalf("Expected 1 history entry, got %d", len(task.History))
	}

	history := task.History[0]

	if history.ChangedBy != changedBy {
		t.Errorf("Expected history changed by %s, got %s", changedBy, history.ChangedBy)
	}

	if history.Comment != comment {
		t.Errorf("Expected history comment %s, got %s", comment, history.Comment)
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated")
	}

	// Test invalid progress values
	err = task.UpdateProgress(-1, changedBy, comment)
	if err == nil {
		t.Error("Expected error with negative progress, got nil")
	}

	err = task.UpdateProgress(101, changedBy, comment)
	if err == nil {
		t.Error("Expected error with progress > 100, got nil")
	}

	// Progress should remain unchanged after errors
	if task.Progress != newProgress {
		t.Errorf("Expected progress to remain %d after errors, got %d", newProgress, task.Progress)
	}
}

// TestAddRemoveDependency tests adding and removing dependencies
func TestAddRemoveDependency(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	// Initial dependencies should be empty
	if len(task.Dependencies) != 0 {
		t.Errorf("Expected initial dependencies to be empty, got %v", task.Dependencies)
	}

	dependencyID := "dep-1"

	// Record current update time
	prevUpdateTime := task.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	// Add dependency
	task.AddDependency(dependencyID)

	// Verify dependency added
	if len(task.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(task.Dependencies))
	}

	if task.Dependencies[0] != dependencyID {
		t.Errorf("Expected dependency %s, got %s", dependencyID, task.Dependencies[0])
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated")
	}

	// Add the same dependency again (should not duplicate)
	task.AddDependency(dependencyID)

	// Verify still only one dependency
	if len(task.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency after duplicate add, got %d", len(task.Dependencies))
	}

	// Add another dependency
	dependencyID2 := "dep-2"
	task.AddDependency(dependencyID2)

	// Verify both dependencies exist
	if len(task.Dependencies) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(task.Dependencies))
	}

	// Remove a dependency
	prevUpdateTime = task.UpdatedAt
	time.Sleep(5 * time.Millisecond)

	task.RemoveDependency(dependencyID)

	// Verify dependency was removed
	if len(task.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency after removal, got %d", len(task.Dependencies))
	}

	if task.Dependencies[0] != dependencyID2 {
		t.Errorf("Expected remaining dependency %s, got %s", dependencyID2, task.Dependencies[0])
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated after removal")
	}

	// Remove non-existent dependency (should not error)
	task.RemoveDependency("non-existent")

	// Verify dependencies unchanged
	if len(task.Dependencies) != 1 {
		t.Fatalf("Expected 1 dependency after removing non-existent, got %d", len(task.Dependencies))
	}
}

// TestAddRemoveBlocker tests adding and removing blockers
func TestAddRemoveBlocker(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	// Initial blockers should be empty
	if len(task.Blockers) != 0 {
		t.Errorf("Expected initial blockers to be empty, got %v", task.Blockers)
	}

	blockerID := "blocker-1"
	changedBy := "tester"
	comment := "Adding blocker"

	// Record current update time
	prevUpdateTime := task.UpdatedAt

	// Wait a moment to ensure update time changes
	time.Sleep(5 * time.Millisecond)

	// Add blocker
	task.AddBlocker(blockerID, changedBy, comment)

	// Verify blocker added
	if len(task.Blockers) != 1 {
		t.Fatalf("Expected 1 blocker, got %d", len(task.Blockers))
	}

	if task.Blockers[0] != blockerID {
		t.Errorf("Expected blocker %s, got %s", blockerID, task.Blockers[0])
	}

	// Verify status was changed to blocked
	if task.Status != kanban.StatusBlocked {
		t.Errorf("Expected status to change to blocked, got %s", task.Status)
	}

	// Verify history was recorded
	if len(task.History) < 2 {
		t.Fatalf("Expected at least 2 history entries, got %d", len(task.History))
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated")
	}

	// Add the same blocker again (should not duplicate)
	task.AddBlocker(blockerID, changedBy, comment)

	// Verify still only one blocker
	if len(task.Blockers) != 1 {
		t.Fatalf("Expected 1 blocker after duplicate add, got %d", len(task.Blockers))
	}

	// Add another blocker
	blockerID2 := "blocker-2"
	task.AddBlocker(blockerID2, changedBy, comment)

	// Verify both blockers exist
	if len(task.Blockers) != 2 {
		t.Fatalf("Expected 2 blockers, got %d", len(task.Blockers))
	}

	// Remove a blocker
	prevUpdateTime = task.UpdatedAt
	time.Sleep(5 * time.Millisecond)

	task.RemoveBlocker(blockerID, changedBy, "Removing blocker")

	// Verify blocker was removed
	if len(task.Blockers) != 1 {
		t.Fatalf("Expected 1 blocker after removal, got %d", len(task.Blockers))
	}

	if task.Blockers[0] != blockerID2 {
		t.Errorf("Expected remaining blocker %s, got %s", blockerID2, task.Blockers[0])
	}

	// Verify status still blocked (because one blocker remains)
	if task.Status != kanban.StatusBlocked {
		t.Errorf("Expected status to remain blocked, got %s", task.Status)
	}

	// Verify update time changed
	if !task.UpdatedAt.After(prevUpdateTime) {
		t.Error("Expected task update time to be updated after removal")
	}

	// Remove last blocker
	task.RemoveBlocker(blockerID2, changedBy, "Removing last blocker")

	// Verify blockers empty
	if len(task.Blockers) != 0 {
		t.Fatalf("Expected 0 blockers after removing all, got %d", len(task.Blockers))
	}

	// Verify status changed to todo
	if task.Status != kanban.StatusTodo {
		t.Errorf("Expected status to change to todo after removing all blockers, got %s", task.Status)
	}

	// Remove non-existent blocker (should not error)
	task.RemoveBlocker("non-existent", changedBy, comment)

	// Verify blockers unchanged
	if len(task.Blockers) != 0 {
		t.Fatalf("Expected 0 blockers after removing non-existent, got %d", len(task.Blockers))
	}
}

// TestIsBlocked tests the IsBlocked method
func TestIsBlocked(t *testing.T) {
	task := kanban.NewTask("Test Task", "This is a test task")

	// Initially not blocked
	if task.IsBlocked() {
		t.Error("Expected new task not to be blocked")
	}

	// Add a blocker
	task.AddBlocker("blocker-1", "tester", "Adding blocker")

	// Should be blocked
	if !task.IsBlocked() {
		t.Error("Expected task to be blocked after adding blocker")
	}

	// Remove the blocker
	task.RemoveBlocker("blocker-1", "tester", "Removing blocker")

	// Should not be blocked
	if task.IsBlocked() {
		t.Error("Expected task not to be blocked after removing blocker")
	}
}

// TestValidationFunctions tests the validation functions
func TestValidationFunctions(t *testing.T) {
	// Test IsValidStatus
	validStatuses := []kanban.TaskStatus{
		kanban.StatusBacklog,
		kanban.StatusTodo,
		kanban.StatusInProgress,
		kanban.StatusBlocked,
		kanban.StatusDone,
		kanban.StatusCancelled,
	}

	for _, status := range validStatuses {
		if !kanban.IsValidStatus(status) {
			t.Errorf("Expected status %s to be valid", status)
		}
	}

	if kanban.IsValidStatus("invalid_status") {
		t.Error("Expected 'invalid_status' to be invalid")
	}

	// Test IsValidPriority
	validPriorities := []kanban.TaskPriority{
		kanban.PriorityHigh,
		kanban.PriorityMedium,
		kanban.PriorityLow,
	}

	for _, priority := range validPriorities {
		if !kanban.IsValidPriority(priority) {
			t.Errorf("Expected priority %s to be valid", priority)
		}
	}

	if kanban.IsValidPriority("invalid_priority") {
		t.Error("Expected 'invalid_priority' to be invalid")
	}
}

// TestTaskFilters tests the task filter functions
func TestTaskFilters(t *testing.T) {
	// Create test tasks
	task1 := kanban.NewTask("Task 1", "Description 1")
	task1.Status = kanban.StatusTodo
	task1.Priority = kanban.PriorityHigh
	task1.AssignedTo = "alice"
	task1.Tags = []string{"frontend", "bugfix"}

	task2 := kanban.NewTask("Task 2", "Description 2")
	task2.Status = kanban.StatusInProgress
	task2.Priority = kanban.PriorityMedium
	task2.AssignedTo = "bob"
	task2.Tags = []string{"backend", "feature"}

	task3 := kanban.NewTask("Task 3", "Description 3")
	task3.Status = kanban.StatusTodo
	task3.Priority = kanban.PriorityLow
	task3.AssignedTo = "alice"
	task3.Tags = []string{"documentation"}

	// Test FilterByStatus
	statusFilter := kanban.FilterByStatus(kanban.StatusTodo)

	if !statusFilter(task1) {
		t.Error("Expected task1 to match StatusTodo filter")
	}

	if statusFilter(task2) {
		t.Error("Expected task2 not to match StatusTodo filter")
	}

	if !statusFilter(task3) {
		t.Error("Expected task3 to match StatusTodo filter")
	}

	// Test FilterByAssignee
	assigneeFilter := kanban.FilterByAssignee("alice")

	if !assigneeFilter(task1) {
		t.Error("Expected task1 to match Alice assignee filter")
	}

	if assigneeFilter(task2) {
		t.Error("Expected task2 not to match Alice assignee filter")
	}

	if !assigneeFilter(task3) {
		t.Error("Expected task3 to match Alice assignee filter")
	}

	// Test FilterByPriority
	priorityFilter := kanban.FilterByPriority(kanban.PriorityHigh)

	if !priorityFilter(task1) {
		t.Error("Expected task1 to match High priority filter")
	}

	if priorityFilter(task2) {
		t.Error("Expected task2 not to match High priority filter")
	}

	if priorityFilter(task3) {
		t.Error("Expected task3 not to match High priority filter")
	}

	// Test FilterByTag
	tagFilter := kanban.FilterByTag("frontend")

	if !tagFilter(task1) {
		t.Error("Expected task1 to match frontend tag filter")
	}

	if tagFilter(task2) {
		t.Error("Expected task2 not to match frontend tag filter")
	}

	if tagFilter(task3) {
		t.Error("Expected task3 not to match frontend tag filter")
	}

	// Test CombineFilters
	combinedFilter := kanban.CombineFilters(
		kanban.FilterByStatus(kanban.StatusTodo),
		kanban.FilterByAssignee("alice"),
	)

	if !combinedFilter(task1) {
		t.Error("Expected task1 to match combined filter")
	}

	if combinedFilter(task2) {
		t.Error("Expected task2 not to match combined filter")
	}

	if !combinedFilter(task3) {
		t.Error("Expected task3 to match combined filter")
	}

	// Add priority to combined filter
	combinedFilter = kanban.CombineFilters(
		kanban.FilterByStatus(kanban.StatusTodo),
		kanban.FilterByAssignee("alice"),
		kanban.FilterByPriority(kanban.PriorityHigh),
	)

	if !combinedFilter(task1) {
		t.Error("Expected task1 to match enhanced combined filter")
	}

	if combinedFilter(task2) {
		t.Error("Expected task2 not to match enhanced combined filter")
	}

	if combinedFilter(task3) {
		t.Error("Expected task3 not to match enhanced combined filter")
	}
}

