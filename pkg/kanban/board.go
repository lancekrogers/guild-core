package kanban

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/memory"
	"github.com/google/uuid"
)

// Minimal interfaces to avoid import cycles
type ComponentRegistry interface {
	Storage() StorageRegistry
}

type StorageRegistry interface {
	GetCampaignRepository() CampaignRepository
	GetCommissionRepository() CommissionRepository
	GetTaskRepository() TaskRepository
	GetMemoryStore() MemoryStore
}

type MemoryStore interface {
	Get(ctx context.Context, bucket, key string) ([]byte, error)
	Put(ctx context.Context, bucket, key string, value []byte) error
	Delete(ctx context.Context, bucket, key string) error
	List(ctx context.Context, bucket string) ([]string, error)
}

type CampaignRepository interface {
	CreateCampaign(ctx context.Context, campaign interface{}) error
}

type CommissionRepository interface {
	CreateCommission(ctx context.Context, commission interface{}) error
	GetCommission(ctx context.Context, id string) (interface{}, error)
}

type TaskRepository interface {
	CreateTask(ctx context.Context, task interface{}) error
	RecordTaskEvent(ctx context.Context, event interface{}) error
}

// Board represents a kanban board that manages tasks
type Board struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	
	// Storage - use registry for SQLite access, fallback to memory store for compatibility
	registry     ComponentRegistry
	store        memory.Store
	eventManager *EventManager
}

// EventType represents the type of event that occurred
type EventType string

const (
	// EventTaskCreated is emitted when a task is created
	EventTaskCreated EventType = "task.created"

	// EventTaskMoved is emitted when a task is moved to a different column
	EventTaskMoved EventType = "task.moved"

	// EventTaskUpdated is emitted when a task is updated
	EventTaskUpdated EventType = "task.updated"

	// EventTaskDeleted is emitted when a task is deleted
	EventTaskDeleted EventType = "task.deleted"

	// EventTaskStatusChanged is emitted when a task's status changes
	EventTaskStatusChanged EventType = "task.status_changed"

	// EventTaskAssigned is emitted when a task is assigned
	EventTaskAssigned EventType = "task.assigned"

	// EventTaskBlocked is emitted when a task becomes blocked
	EventTaskBlocked EventType = "task.blocked"

	// EventTaskUnblocked is emitted when a task is no longer blocked
	EventTaskUnblocked EventType = "task.unblocked"
)

// BoardEvent represents an event that occurred on the board
type BoardEvent struct {
	EventType  EventType          `json:"event_type"`
	BoardID    string             `json:"board_id"`
	TaskID     string             `json:"task_id,omitempty"`
	Data       map[string]string  `json:"data,omitempty"`
	OccurredAt time.Time          `json:"occurred_at"`
}

// NewBoard creates a new kanban board
// NewBoard creates a new board using the legacy memory.Store interface (for backward compatibility)
func NewBoard(store memory.Store, name, description string) (*Board, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	board := &Board{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Metadata:    make(map[string]string),
		registry:    nil, // No registry for legacy mode
		store:       store,
		eventManager: nil, // Will be set by SetEventManager
	}

	// Save the board
	if err := board.Save(context.Background()); err != nil {
		return nil, err
	}

	return board, nil
}

// NewBoardWithRegistry creates a new board using SQLite storage via registry
func NewBoardWithRegistry(registry ComponentRegistry, name, description string) (*Board, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}

	board := &Board{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Metadata:    make(map[string]string),
		registry:    registry,
		store:       nil, // No legacy store for SQLite mode
		eventManager: nil, // Will be set by SetEventManager
	}

	// Save the board to SQLite (boards are stored as campaign records)
	if err := board.saveToSQLite(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to save board to SQLite: %w", err)
	}

	return board, nil
}

// saveToSQLite saves the board to SQLite as a campaign record
func (b *Board) saveToSQLite(ctx context.Context) error {
	if b.registry == nil {
		return fmt.Errorf("no registry available for SQLite operations")
	}

	storageReg := b.registry.Storage()
	if storageReg == nil {
		return fmt.Errorf("storage registry not available")
	}

	campaignRepo := storageReg.GetCampaignRepository()
	if campaignRepo == nil {
		return fmt.Errorf("campaign repository not available")
	}

	// Create campaign struct dynamically to avoid import cycle
	campaign := map[string]interface{}{
		"ID":        b.ID,
		"Name":      b.Name,
		"Status":    "active", // Boards are always active
		"CreatedAt": b.CreatedAt,
		"UpdatedAt": b.UpdatedAt,
	}

	// Create or update the campaign
	if err := campaignRepo.CreateCampaign(ctx, campaign); err != nil {
		return fmt.Errorf("failed to save board as campaign: %w", err)
	}

	return nil
}

// SetEventManager sets the event manager for this board
func (b *Board) SetEventManager(em *EventManager) {
	b.eventManager = em
}

// LoadBoard loads a board from the store
func LoadBoard(store memory.Store, boardID string) (*Board, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	
	data, err := store.Get(context.Background(), "boards", boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to load board: %w", err)
	}
	
	var board Board
	if err := json.Unmarshal(data, &board); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}
	
	// Set the store
	board.store = store
	
	return &board, nil
}

// ListBoards lists all boards in the store
func ListBoards(store memory.Store) ([]*Board, error) {
	if store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	
	keys, err := store.List(context.Background(), "boards")
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}
	
	var boards []*Board
	for _, key := range keys {
		board, err := LoadBoard(store, key)
		if err != nil {
			continue // Skip this one if there's an error
		}
		boards = append(boards, board)
	}
	
	return boards, nil
}

// Save saves the board to the store
func (b *Board) Save(ctx context.Context) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Update timestamp
	b.UpdatedAt = time.Now().UTC()
	
	// Marshal board to JSON
	data, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("failed to marshal board: %w", err)
	}
	
	// Save to store
	return b.store.Put(ctx, "boards", b.ID, data)
}

// Delete deletes the board from the store
func (b *Board) Delete(ctx context.Context) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// First, delete all tasks
	tasks, err := b.GetAllTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}
	
	for _, task := range tasks {
		if err := b.DeleteTask(ctx, task.ID); err != nil {
			return fmt.Errorf("failed to delete task %s: %w", task.ID, err)
		}
	}
	
	// Delete the board itself
	return b.store.Delete(ctx, "boards", b.ID)
}

// CreateTask creates a new task on the board
func (b *Board) CreateTask(ctx context.Context, title, description string) (*Task, error) {
	// Use SQLite if registry is available, otherwise fallback to legacy store
	if b.registry != nil {
		return b.createTaskSQLite(ctx, title, description)
	}
	
	if b.store == nil {
		return nil, fmt.Errorf("board not connected to any storage")
	}
	
	// Legacy BoltDB path
	task := NewTask(title, description)
	
	// Add board ID to metadata
	task.Metadata["board_id"] = b.ID
	
	// Save task
	taskData, err := json.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}
	
	if err := b.store.Put(ctx, "tasks", task.ID, taskData); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}
	
	// Index task by board and status
	indexKey := fmt.Sprintf("%s:%s", b.ID, task.Status)
	if err := b.store.Put(ctx, "tasks_by_board_status", indexKey+":"+task.ID, []byte(task.ID)); err != nil {
		return nil, fmt.Errorf("failed to index task: %w", err)
	}
	
	// Update the board's last updated time
	b.UpdatedAt = time.Now().UTC()
	if err := b.Save(ctx); err != nil {
		return nil, fmt.Errorf("failed to update board: %w", err)
	}
	
	// Emit task created event
	event := BoardEvent{
		EventType:  EventTaskCreated,
		BoardID:    b.ID,
		TaskID:     task.ID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]string{
			"title":       task.Title,
			"description": task.Description,
			"status":      string(task.Status),
		},
	}
	if err := b.emitEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to emit event: %v\n", err)
	}
	
	return task, nil
}

// createTaskSQLite creates a task using SQLite storage
func (b *Board) createTaskSQLite(ctx context.Context, title, description string) (*Task, error) {
	storageReg := b.registry.Storage()
	if storageReg == nil {
		return nil, fmt.Errorf("storage registry not available")
	}

	taskRepo := storageReg.GetTaskRepository()
	if taskRepo == nil {
		return nil, fmt.Errorf("task repository not available")
	}

	// Create a commission for this board/task if it doesn't exist
	commissionID := b.ID + "-default-commission"
	if err := b.ensureDefaultCommission(ctx, commissionID); err != nil {
		return nil, fmt.Errorf("failed to ensure default commission: %w", err)
	}

	// Create the task
	task := NewTask(title, description)
	
	// Convert kanban task metadata to interface{} map
	metadataInterface := make(map[string]interface{})
	for k, v := range task.Metadata {
		metadataInterface[k] = v
	}

	// Map kanban task to storage task format
	var assignedAgent *string
	if task.AssignedTo != "" {
		assignedAgent = &task.AssignedTo
	}

	// Convert to storage task format (using map to avoid import cycle)
	storageTask := map[string]interface{}{
		"ID":              task.ID,
		"CommissionID":    commissionID,
		"AssignedAgentID": assignedAgent,
		"Title":           task.Title,
		"Description":     &task.Description,
		"Status":          string(task.Status),
		"Column":          "todo", // Default column for new tasks
		"StoryPoints":     int32(1), // Default story points
		"Metadata":        metadataInterface,
		"CreatedAt":       task.CreatedAt,
		"UpdatedAt":       task.UpdatedAt,
	}

	// Save to SQLite
	if err := taskRepo.CreateTask(ctx, storageTask); err != nil {
		return nil, fmt.Errorf("failed to save task to SQLite: %w", err)
	}

	// Record task creation event
	if err := b.recordTaskEvent(ctx, task.ID, "created", "", string(task.Status), "Task created on board"); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to record task event: %v\n", err)
	}

	// Update the board's last updated time
	b.UpdatedAt = time.Now().UTC()
	if err := b.saveToSQLite(ctx); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to update board timestamp: %v\n", err)
	}

	return task, nil
}

// ensureDefaultCommission creates a default commission for the board if it doesn't exist
func (b *Board) ensureDefaultCommission(ctx context.Context, commissionID string) error {
	storageReg := b.registry.Storage()
	commissionRepo := storageReg.GetCommissionRepository()
	
	// Try to get existing commission
	_, err := commissionRepo.GetCommission(ctx, commissionID)
	if err == nil {
		return nil // Commission already exists
	}

	// Create default commission (using map to avoid import cycle)
	commission := map[string]interface{}{
		"ID":         commissionID,
		"CampaignID": b.ID, // Board ID is campaign ID
		"Title":      b.Name + " Tasks",
		"Status":     "active",
		"CreatedAt":  b.CreatedAt,
	}

	if err := commissionRepo.CreateCommission(ctx, commission); err != nil {
		return fmt.Errorf("failed to create default commission: %w", err)
	}

	return nil
}

// recordTaskEvent records a task event for audit trail
func (b *Board) recordTaskEvent(ctx context.Context, taskID, eventType, oldValue, newValue, reason string) error {
	storageReg := b.registry.Storage()
	taskRepo := storageReg.GetTaskRepository()

	// Create event using map to avoid import cycle
	event := map[string]interface{}{
		"TaskID":    taskID,
		"AgentID":   nil,
		"EventType": eventType,
		"OldValue":  &oldValue,
		"NewValue":  &newValue,
		"Reason":    &reason,
		"CreatedAt": time.Now().UTC(),
	}

	return taskRepo.RecordTaskEvent(ctx, event)
}

// GetTask retrieves a task by ID
func (b *Board) GetTask(ctx context.Context, taskID string) (*Task, error) {
	if b.store == nil {
		return nil, fmt.Errorf("board not connected to a store")
	}
	
	data, err := b.store.Get(ctx, "tasks", taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}
	
	// Verify the task belongs to this board
	if task.Metadata["board_id"] != b.ID {
		return nil, fmt.Errorf("task does not belong to this board")
	}
	
	return &task, nil
}

// UpdateTask updates a task on the board
func (b *Board) UpdateTask(ctx context.Context, task *Task) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Verify the task belongs to this board
	if task.Metadata["board_id"] != b.ID {
		return fmt.Errorf("task does not belong to this board")
	}
	
	// Get the current task to check if status has changed
	currentTask, err := b.GetTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get current task: %w", err)
	}
	
	oldStatus := currentTask.Status
	newStatus := task.Status
	
	// Update task
	task.UpdatedAt = time.Now().UTC()
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	
	if err := b.store.Put(ctx, "tasks", task.ID, taskData); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	
	// If status changed, update indices
	if oldStatus != newStatus {
		// Delete old status index
		oldIndexKey := fmt.Sprintf("%s:%s", b.ID, oldStatus)
		if err := b.store.Delete(ctx, "tasks_by_board_status", oldIndexKey+":"+task.ID); err != nil {
			return fmt.Errorf("failed to remove old status index: %w", err)
		}
		
		// Add new status index
		newIndexKey := fmt.Sprintf("%s:%s", b.ID, newStatus)
		if err := b.store.Put(ctx, "tasks_by_board_status", newIndexKey+":"+task.ID, []byte(task.ID)); err != nil {
			return fmt.Errorf("failed to add new status index: %w", err)
		}
		
		// Emit status changed event
		event := BoardEvent{
			EventType:  EventTaskStatusChanged,
			BoardID:    b.ID,
			TaskID:     task.ID,
			OccurredAt: time.Now().UTC(),
			Data: map[string]string{
				"old_status": string(oldStatus),
				"new_status": string(newStatus),
				"title":      task.Title,
			},
		}
		if err := b.emitEvent(ctx, event); err != nil {
			// Log but don't fail
			fmt.Printf("warning: failed to emit event: %v\n", err)
		}
	}
	
	// Update the board's last updated time
	b.UpdatedAt = time.Now().UTC()
	if err := b.Save(ctx); err != nil {
		return fmt.Errorf("failed to update board: %w", err)
	}
	
	// Emit task updated event
	event := BoardEvent{
		EventType:  EventTaskUpdated,
		BoardID:    b.ID,
		TaskID:     task.ID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]string{
			"title":  task.Title,
			"status": string(task.Status),
		},
	}
	if err := b.emitEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to emit event: %v\n", err)
	}
	
	return nil
}

// DeleteTask deletes a task from the board
func (b *Board) DeleteTask(ctx context.Context, taskID string) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Get the task to check if it belongs to this board
	task, err := b.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Delete the task
	if err := b.store.Delete(ctx, "tasks", taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	
	// Delete status index
	indexKey := fmt.Sprintf("%s:%s", b.ID, task.Status)
	if err := b.store.Delete(ctx, "tasks_by_board_status", indexKey+":"+taskID); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to delete status index: %v\n", err)
	}
	
	// Update the board's last updated time
	b.UpdatedAt = time.Now().UTC()
	if err := b.Save(ctx); err != nil {
		return fmt.Errorf("failed to update board: %w", err)
	}
	
	// Emit task deleted event
	event := BoardEvent{
		EventType:  EventTaskDeleted,
		BoardID:    b.ID,
		TaskID:     taskID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]string{
			"title":  task.Title,
			"status": string(task.Status),
		},
	}
	if err := b.emitEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to emit event: %v\n", err)
	}
	
	return nil
}

// GetTasksByStatus gets all tasks with a specific status
func (b *Board) GetTasksByStatus(ctx context.Context, status TaskStatus) ([]*Task, error) {
	if b.store == nil {
		return nil, fmt.Errorf("board not connected to a store")
	}
	
	// Get all tasks with the given status
	indexPrefix := fmt.Sprintf("%s:%s:", b.ID, status)
	taskIDs, err := b.store.ListKeys(ctx, "tasks_by_board_status", indexPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list task IDs: %w", err)
	}
	
	var tasks []*Task
	for _, key := range taskIDs {
		// Extract task ID from the key
		parts := strings.Split(key, ":")
		if len(parts) != 3 {
			continue // Invalid key format
		}
		
		taskID := parts[2]
		
		// Get the task
		task, err := b.GetTask(ctx, taskID)
		if err != nil {
			// Skip this one if there's an error
			continue
		}
		
		tasks = append(tasks, task)
	}
	
	// Sort tasks by UpdatedAt, newest first
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})
	
	return tasks, nil
}

// GetAllTasks gets all tasks on the board
func (b *Board) GetAllTasks(ctx context.Context) ([]*Task, error) {
	if b.store == nil {
		return nil, fmt.Errorf("board not connected to a store")
	}
	
	var allTasks []*Task
	
	// Get tasks for each status
	statuses := []TaskStatus{
		StatusBacklog,
		StatusTodo,
		StatusInProgress,
		StatusBlocked,
		StatusDone,
		StatusCancelled,
	}
	
	for _, status := range statuses {
		tasks, err := b.GetTasksByStatus(ctx, status)
		if err != nil {
			return nil, fmt.Errorf("failed to get tasks with status %s: %w", status, err)
		}
		
		allTasks = append(allTasks, tasks...)
	}
	
	// Sort tasks by UpdatedAt, newest first
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].UpdatedAt.After(allTasks[j].UpdatedAt)
	})
	
	return allTasks, nil
}

// FilterTasks filters tasks based on the provided filter
func (b *Board) FilterTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	if b.store == nil {
		return nil, fmt.Errorf("board not connected to a store")
	}
	
	// Get all tasks
	allTasks, err := b.GetAllTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tasks: %w", err)
	}
	
	// Apply filter
	var filteredTasks []*Task
	for _, task := range allTasks {
		if filter(task) {
			filteredTasks = append(filteredTasks, task)
		}
	}
	
	return filteredTasks, nil
}

// UpdateTaskStatus updates the status of a task
func (b *Board) UpdateTaskStatus(ctx context.Context, taskID string, newStatus TaskStatus, changedBy, comment string) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Get the task
	task, err := b.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Update the status
	if err := task.UpdateStatus(newStatus, changedBy, comment); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	
	// Save the task
	return b.UpdateTask(ctx, task)
}

// AssignTask assigns a task to a user
func (b *Board) AssignTask(ctx context.Context, taskID, assignee, changedBy, comment string) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Get the task
	task, err := b.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Update the assignee
	task.UpdateAssignee(assignee, changedBy, comment)
	
	// Save the task
	if err := b.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	
	// Emit task assigned event
	event := BoardEvent{
		EventType:  EventTaskAssigned,
		BoardID:    b.ID,
		TaskID:     taskID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]string{
			"assignee": assignee,
			"title":    task.Title,
		},
	}
	if err := b.emitEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to emit event: %v\n", err)
	}
	
	return nil
}

// AddTaskBlocker adds a blocker to a task
func (b *Board) AddTaskBlocker(ctx context.Context, taskID, blockerID, changedBy, comment string) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Get the task
	task, err := b.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Add the blocker
	task.AddBlocker(blockerID, changedBy, comment)
	
	// Save the task
	if err := b.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	
	// Emit task blocked event
	event := BoardEvent{
		EventType:  EventTaskBlocked,
		BoardID:    b.ID,
		TaskID:     taskID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]string{
			"blocker_id": blockerID,
			"title":      task.Title,
		},
	}
	if err := b.emitEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to emit event: %v\n", err)
	}
	
	return nil
}

// RemoveTaskBlocker removes a blocker from a task
func (b *Board) RemoveTaskBlocker(ctx context.Context, taskID, blockerID, changedBy, comment string) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}
	
	// Get the task
	task, err := b.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Remove the blocker
	task.RemoveBlocker(blockerID, changedBy, comment)
	
	// Save the task
	if err := b.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	
	// Emit task unblocked event
	event := BoardEvent{
		EventType:  EventTaskUnblocked,
		BoardID:    b.ID,
		TaskID:     taskID,
		OccurredAt: time.Now().UTC(),
		Data: map[string]string{
			"blocker_id": blockerID,
			"title":      task.Title,
		},
	}
	if err := b.emitEvent(ctx, event); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to emit event: %v\n", err)
	}
	
	return nil
}

// emitEvent saves and publishes an event
func (b *Board) emitEvent(ctx context.Context, event BoardEvent) error {
	if b.store == nil {
		return fmt.Errorf("board not connected to a store")
	}

	// Generate a unique ID for the event
	eventID := uuid.New().String()

	// Marshal event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Save event to local storage
	err = b.store.Put(ctx, "board_events", eventID, eventData)
	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	// If event manager is available, publish the event
	if b.eventManager != nil {
		if pubErr := b.eventManager.PublishEvent(&event); pubErr != nil {
			// Log but don't fail the operation if publishing fails
			fmt.Printf("warning: failed to publish event: %v\n", pubErr)
		}
	}

	return nil
}