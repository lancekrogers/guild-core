package kanban

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/guild-ventures/guild-core/pkg/storage"
	"github.com/google/uuid"
)

// Minimal interfaces to avoid import cycles
type ComponentRegistry interface {
	Storage() StorageRegistry
}

type StorageRegistry interface {
	GetKanbanCampaignRepository() CampaignRepository
	GetKanbanCommissionRepository() CommissionRepository
	GetBoardRepository() BoardRepository
	GetKanbanTaskRepository() TaskRepository
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

type BoardRepository interface {
	CreateBoard(ctx context.Context, board interface{}) error
	GetBoard(ctx context.Context, id string) (interface{}, error)
	UpdateBoard(ctx context.Context, board interface{}) error
	DeleteBoard(ctx context.Context, id string) error
	ListBoards(ctx context.Context) ([]interface{}, error)
}

type TaskRepository interface {
	CreateTask(ctx context.Context, task interface{}) error
	UpdateTask(ctx context.Context, task interface{}) error
	DeleteTask(ctx context.Context, id string) error
	ListTasksByBoard(ctx context.Context, boardID string) ([]interface{}, error)
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
	
	// Storage - SQLite only via registry
	registry     ComponentRegistry
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

// NewBoard creates a new kanban board using SQLite
func NewBoard(registry ComponentRegistry, name, description string) (*Board, error) {
	return NewBoardWithRegistry(registry, name, description)
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
		eventManager: nil, // Will be set by SetEventManager
	}

	// Save the board to SQLite (boards are stored as campaign records)
	if err := board.saveToSQLite(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to save board to SQLite: %w", err)
	}

	return board, nil
}

// saveToSQLite saves the board to SQLite with proper campaign relationship
func (b *Board) saveToSQLite(ctx context.Context) error {
	if b.registry == nil {
		return fmt.Errorf("no registry available for SQLite operations")
	}

	storageReg := b.registry.Storage()
	if storageReg == nil {
		return fmt.Errorf("storage registry not available")
	}

	campaignRepo := storageReg.GetKanbanCampaignRepository()
	if campaignRepo == nil {
		return fmt.Errorf("campaign repository not available")
	}

	// Get or create the campaign that this board belongs to
	campaignID := b.getCampaignID()
	
	// Ensure the campaign exists
	if err := b.ensureCampaignExists(ctx, campaignRepo, campaignID); err != nil {
		return fmt.Errorf("failed to ensure campaign exists: %w", err)
	}

	// Store the board's campaign association in metadata
	if b.Metadata == nil {
		b.Metadata = make(map[string]string)
	}
	b.Metadata["campaign_id"] = campaignID

	// Note: The board itself doesn't need to be stored as a separate entity in SQLite
	// It exists as a logical grouping of tasks within a campaign
	// The tasks will reference both the commission and indirectly the campaign
	
	return nil
}

// getCampaignID determines which campaign this board belongs to
func (b *Board) getCampaignID() string {
	// Check if campaign ID is already set in metadata
	if b.Metadata != nil {
		if campaignID, exists := b.Metadata["campaign_id"]; exists && campaignID != "" {
			return campaignID
		}
	}
	
	// Default: use board name to generate a campaign ID
	// This allows multiple boards for the same type of work to share a campaign
	if strings.Contains(strings.ToLower(b.Name), "commission") {
		return "commission-campaign"
	}
	
	// Fallback: create a campaign based on board name
	return fmt.Sprintf("%s-campaign", strings.ToLower(strings.ReplaceAll(b.Name, " ", "-")))
}

// ensureCampaignExists creates the campaign if it doesn't exist
func (b *Board) ensureCampaignExists(ctx context.Context, campaignRepo CampaignRepository, campaignID string) error {
	// Create campaign struct
	campaign := map[string]interface{}{
		"ID":        campaignID,
		"Name":      b.getCampaignName(campaignID),
		"Status":    "active",
		"CreatedAt": b.CreatedAt,
		"UpdatedAt": b.UpdatedAt,
	}

	// Try to create the campaign (idempotent operation)
	if err := campaignRepo.CreateCampaign(ctx, campaign); err != nil {
		// Ignore UNIQUE constraint errors - campaign already exists
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("failed to create campaign %s: %w", campaignID, err)
		}
	}

	return nil
}

// getCampaignName generates a human-readable campaign name
func (b *Board) getCampaignName(campaignID string) string {
	switch campaignID {
	case "commission-campaign":
		return "Commission Processing Campaign"
	default:
		// Convert ID to title case
		parts := strings.Split(campaignID, "-")
		for i, part := range parts {
			if len(part) > 0 {
				parts[i] = strings.ToUpper(part[:1]) + part[1:]
			}
		}
		return strings.Join(parts, " ")
	}
}

// SetEventManager sets the event manager for this board
func (b *Board) SetEventManager(em *EventManager) {
	b.eventManager = em
}

// LoadBoard loads a board from SQLite using the board ID
func LoadBoard(registry ComponentRegistry, boardID string) (*Board, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}
	
	storageReg := registry.Storage()
	if storageReg == nil {
		return nil, fmt.Errorf("storage registry not available")
	}
	
	boardRepo := storageReg.GetBoardRepository()
	if boardRepo == nil {
		return nil, fmt.Errorf("board repository not available")
	}
	
	// Get board from SQLite
	boardInterface, err := boardRepo.GetBoard(context.Background(), boardID)
	if err != nil {
		return nil, fmt.Errorf("failed to load board: %w", err)
	}
	
	// Cast to storage.Board
	storageBoard, ok := boardInterface.(*storage.Board)
	if !ok {
		return nil, fmt.Errorf("failed to cast board to storage.Board")
	}
	
	// Convert storage board to kanban board
	board := &Board{
		ID:          storageBoard.ID,
		Name:        storageBoard.Name,
		Description: *storageBoard.Description,
		CreatedAt:   storageBoard.CreatedAt,
		UpdatedAt:   storageBoard.UpdatedAt,
		Metadata:    make(map[string]string),
		registry:    registry,
	}
	
	return board, nil
}

// ListBoards lists all boards from SQLite
func ListBoards(registry ComponentRegistry) ([]*Board, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}
	
	storageReg := registry.Storage()
	if storageReg == nil {
		return nil, fmt.Errorf("storage registry not available")
	}
	
	boardRepo := storageReg.GetBoardRepository()
	if boardRepo == nil {
		return nil, fmt.Errorf("board repository not available")
	}
	
	// Get all boards from SQLite
	boardInterfaces, err := boardRepo.ListBoards(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
	}
	
	var boards []*Board
	for _, boardInterface := range boardInterfaces {
		// Cast to storage.Board
		storageBoard, ok := boardInterface.(*storage.Board)
		if !ok {
			continue // Skip invalid boards
		}
		
		board := &Board{
			ID:          storageBoard.ID,
			Name:        storageBoard.Name,
			Description: *storageBoard.Description,
			CreatedAt:   storageBoard.CreatedAt,
			UpdatedAt:   storageBoard.UpdatedAt,
			Metadata:    make(map[string]string),
			registry:    registry,
		}
		boards = append(boards, board)
	}
	
	return boards, nil
}

// Save saves the board using SQLite
func (b *Board) Save(ctx context.Context) error {
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
	}
	
	// Update timestamp
	b.UpdatedAt = time.Now().UTC()
	
	// Get board repository
	storageReg := b.registry.Storage()
	if storageReg == nil {
		return fmt.Errorf("storage registry not available")
	}
	
	boardRepo := storageReg.GetBoardRepository()
	if boardRepo == nil {
		return fmt.Errorf("board repository not available")
	}
	
	// Ensure commission exists first
	commissionID := b.ID + "-commission" // Generate commission ID based on board ID
	campaignID := b.getCampaignID()
	if err := b.ensureCommissionExists(ctx, commissionID, campaignID); err != nil {
		return fmt.Errorf("failed to ensure commission exists: %w", err)
	}
	
	// Convert kanban.Board to storage.Board
	description := b.Description
	storageBoard := &storage.Board{
		ID:           b.ID,
		CommissionID: commissionID,
		Name:         b.Name,
		Description:  &description,
		Status:       "active",
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
	
	// Try update first, if fails then create
	if err := boardRepo.UpdateBoard(ctx, storageBoard); err != nil {
		return boardRepo.CreateBoard(ctx, storageBoard)
	}
	
	return nil
}

// Delete deletes the board from the store
func (b *Board) Delete(ctx context.Context) error {
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
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
	
	// Delete the board itself from SQLite
	storageReg := b.registry.Storage()
	boardRepo := storageReg.GetBoardRepository()
	if boardRepo == nil {
		return fmt.Errorf("board repository not available")
	}
	
	return boardRepo.DeleteBoard(ctx, b.ID)
}

// CreateTask creates a new task on the board
func (b *Board) CreateTask(ctx context.Context, title, description string) (*Task, error) {
	// Use SQLite if registry is available, otherwise fallback to legacy store
	if b.registry != nil {
		return b.createTaskSQLite(ctx, title, description)
	}
	
	// No legacy store - all operations use SQLite via registry
	return nil, fmt.Errorf("board not connected to registry - cannot create tasks without SQLite backend")
}

// createTaskSQLite creates a task using SQLite storage
func (b *Board) createTaskSQLite(ctx context.Context, title, description string) (*Task, error) {
	storageReg := b.registry.Storage()
	if storageReg == nil {
		return nil, fmt.Errorf("storage registry not available")
	}

	taskRepo := storageReg.GetKanbanTaskRepository()
	if taskRepo == nil {
		return nil, fmt.Errorf("task repository not available")
	}

	// Ensure board exists in SQLite before creating tasks
	if err := b.ensureBoardExists(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure board exists: %w", err)
	}

	// Create the task
	task := NewTask(title, description)
	
	// Add board ID to metadata for ownership validation
	task.Metadata["board_id"] = b.ID
	
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

	// Map kanban status to SQLite-compatible status
	storageStatus := b.mapKanbanStatusToStorageStatus(task.Status)
	
	// Convert to storage task format with both BoardID and CommissionID for compatibility
	commissionID := b.ID + "-commission" // Generate commission ID based on board ID
	storageTask := map[string]interface{}{
		"ID":              task.ID,
		"BoardID":         b.ID,           // Use board ID
		"CommissionID":    commissionID,   // Also set commission ID for compatibility
		"AssignedAgentID": assignedAgent,
		"Title":           task.Title,
		"Description":     &task.Description,
		"Status":          storageStatus,
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

// ensureBoardExists creates the board in SQLite if it doesn't exist
func (b *Board) ensureBoardExists(ctx context.Context) error {
	storageReg := b.registry.Storage()
	if storageReg == nil {
		return fmt.Errorf("storage registry not available")
	}

	boardRepo := storageReg.GetBoardRepository()
	if boardRepo == nil {
		return fmt.Errorf("board repository not available")
	}

	// Try to get existing board
	_, err := boardRepo.GetBoard(ctx, b.ID)
	if err == nil {
		return nil // Board already exists
	}

	// Ensure campaign and commission exist first
	campaignID := b.getCampaignID()
	if err := b.ensureCampaignExists(ctx, storageReg.GetKanbanCampaignRepository(), campaignID); err != nil {
		return fmt.Errorf("failed to ensure campaign exists: %w", err)
	}

	commissionID := b.ID + "-commission"
	if err := b.ensureCommissionExists(ctx, commissionID, campaignID); err != nil {
		return fmt.Errorf("failed to ensure commission exists: %w", err)
	}

	// Create board using storage model
	storageBoard := map[string]interface{}{
		"ID":           b.ID,
		"CommissionID": commissionID,
		"Name":         b.Name,
		"Description":  &b.Description,
		"Status":       "active",
		"CreatedAt":    b.CreatedAt,
		"UpdatedAt":    b.UpdatedAt,
	}

	if err := boardRepo.CreateBoard(ctx, storageBoard); err != nil {
		return fmt.Errorf("failed to create board: %w", err)
	}

	return nil
}

// ensureCommissionExists creates a commission for the board if it doesn't exist
func (b *Board) ensureCommissionExists(ctx context.Context, commissionID, campaignID string) error {
	storageReg := b.registry.Storage()
	commissionRepo := storageReg.GetKanbanCommissionRepository()
	
	// Try to get existing commission
	_, err := commissionRepo.GetCommission(ctx, commissionID)
	if err == nil {
		return nil // Commission already exists
	}

	// Create commission (using map to avoid import cycle)
	commission := map[string]interface{}{
		"ID":         commissionID,
		"CampaignID": campaignID,
		"Title":      b.Name + " Tasks",
		"Status":     "active",
		"CreatedAt":  b.CreatedAt,
	}

	if err := commissionRepo.CreateCommission(ctx, commission); err != nil {
		return fmt.Errorf("failed to create commission: %w", err)
	}

	return nil
}

// recordTaskEvent records a task event for audit trail
func (b *Board) recordTaskEvent(ctx context.Context, taskID, eventType, oldValue, newValue, reason string) error {
	storageReg := b.registry.Storage()
	taskRepo := storageReg.GetKanbanTaskRepository()

	// Create event using proper TaskEvent struct
	event := &storage.TaskEvent{
		TaskID:    taskID,
		AgentID:   nil,
		EventType: eventType,
		OldValue:  &oldValue,
		NewValue:  &newValue,
		Reason:    &reason,
		CreatedAt: time.Now().UTC(),
	}

	return taskRepo.RecordTaskEvent(ctx, event)
}

// GetTask retrieves a task by ID
func (b *Board) GetTask(ctx context.Context, taskID string) (*Task, error) {
	// Use SQLite if registry is available
	if b.registry != nil {
		return b.getTaskSQLite(ctx, taskID)
	}
	
	// No legacy store - all operations use SQLite via registry
	return nil, fmt.Errorf("board not connected to registry - cannot get tasks without SQLite backend")
}

// getTaskSQLite retrieves a task from SQLite storage
func (b *Board) getTaskSQLite(ctx context.Context, taskID string) (*Task, error) {
	// Get the storage registry to access the underlying task repository
	storageReg := b.registry.Storage()
	if storageReg == nil {
		return nil, fmt.Errorf("storage registry not available")
	}
	
	// Get all tasks for this board and find the matching one
	taskRepo := storageReg.GetKanbanTaskRepository()
	if taskRepo == nil {
		return nil, fmt.Errorf("task repository not available")
	}
	
	// Get all tasks for this board from SQLite
	taskInterfaces, err := taskRepo.ListTasksByBoard(ctx, b.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for board: %w", err)
	}
	
	// Find the specific task
	for _, taskInterface := range taskInterfaces {
		if storageTask, ok := taskInterface.(*storage.Task); ok && storageTask.ID == taskID {
			// Convert storage.Task back to kanban.Task
			task := &Task{
				ID:          storageTask.ID,
				Title:       storageTask.Title,
				Description: *storageTask.Description,
				Status:      b.mapStorageStatusToKanbanStatus(storageTask.Status),
				CreatedAt:   storageTask.CreatedAt,
				UpdatedAt:   storageTask.UpdatedAt,
				Metadata:    make(map[string]string),
			}
			
			// Set assigned agent if available
			if storageTask.AssignedAgentID != nil {
				task.AssignedTo = *storageTask.AssignedAgentID
			}
			
			// Add board ID to metadata for compatibility
			task.Metadata["board_id"] = b.ID
			
			return task, nil
		}
	}
	
	return nil, fmt.Errorf("task not found: %s", taskID)
}

// mapStorageStatusToKanbanStatus maps database status values back to kanban status values
func (b *Board) mapStorageStatusToKanbanStatus(storageStatus string) TaskStatus {
	switch storageStatus {
	case "pending_review":
		return StatusReadyForReview
	case "todo", "in_progress", "blocked", "done":
		return TaskStatus(storageStatus)
	default:
		return StatusTodo // Default fallback
	}
}

// mapKanbanStatusToStorageStatus maps kanban status values to database-compatible status values
func (b *Board) mapKanbanStatusToStorageStatus(kanbanStatus TaskStatus) string {
	switch kanbanStatus {
	case StatusBacklog:
		return "todo" // Map backlog to todo for SQLite compatibility
	case StatusReadyForReview:
		return "pending_review" // Map ready_for_review to pending_review
	case StatusCancelled:
		return "done" // Map cancelled to done (with metadata indicating cancellation)
	case StatusTodo, StatusInProgress, StatusBlocked, StatusDone:
		return string(kanbanStatus) // These map directly
	default:
		return "todo" // Default fallback
	}
}

// UpdateTask updates a task on the board
func (b *Board) UpdateTask(ctx context.Context, task *Task) error {
	// Use SQLite via registry for all task operations
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
	}
	
	return b.updateTaskSQLite(ctx, task)
}

// updateTaskSQLite updates a task using SQLite storage
func (b *Board) updateTaskSQLite(ctx context.Context, task *Task) error {
	storageReg := b.registry.Storage()
	if storageReg == nil {
		return fmt.Errorf("storage registry not available")
	}

	taskRepo := storageReg.GetKanbanTaskRepository()
	if taskRepo == nil {
		return fmt.Errorf("task repository not available")
	}

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

	// Map kanban status to SQLite-compatible status
	storageStatus := b.mapKanbanStatusToStorageStatus(task.Status)
	
	// Convert to storage task format with both BoardID and CommissionID for compatibility
	commissionID := b.ID + "-commission" // Generate commission ID based on board ID
	storageTask := map[string]interface{}{
		"ID":              task.ID,
		"BoardID":         b.ID,           // Use board ID
		"CommissionID":    commissionID,   // Also set commission ID for compatibility
		"AssignedAgentID": assignedAgent,
		"Title":           task.Title,
		"Description":     &task.Description,
		"Status":          storageStatus,
		"StoryPoints":     int32(1), // Default story points
		"Metadata":        metadataInterface,
		"CreatedAt":       task.CreatedAt,
		"UpdatedAt":       time.Now().UTC(),
	}

	// Update in SQLite using the kanban task repository adapter
	if err := taskRepo.UpdateTask(ctx, storageTask); err != nil {
		return fmt.Errorf("failed to update task in SQLite: %w", err)
	}

	// Record task update event
	if err := b.recordTaskEvent(ctx, task.ID, "updated", "", string(task.Status), "Task updated via kanban board"); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to record task event: %v\n", err)
	}

	// Update the board's last updated time
	b.UpdatedAt = time.Now().UTC()
	if err := b.saveToSQLite(ctx); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to update board timestamp: %v\n", err)
	}

	return nil
}

// DeleteTask deletes a task from the board
func (b *Board) DeleteTask(ctx context.Context, taskID string) error {
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
	}
	
	// Get the task to check if it belongs to this board
	task, err := b.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Delete the task from SQLite
	storageReg := b.registry.Storage()
	taskRepo := storageReg.GetKanbanTaskRepository()
	if taskRepo == nil {
		return fmt.Errorf("task repository not available")
	}
	
	if err := taskRepo.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	
	// Record task deletion event
	if err := b.recordTaskEvent(ctx, taskID, "deleted", string(task.Status), "", "Task deleted from kanban board"); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to record task event: %v\n", err)
	}
	
	// Update the board's last updated time
	b.UpdatedAt = time.Now().UTC()
	if err := b.saveToSQLite(ctx); err != nil {
		// Log but don't fail
		fmt.Printf("warning: failed to update board timestamp: %v\n", err)
	}
	
	return nil
}

// GetTasksByStatus gets all tasks with a specific status from SQLite
func (b *Board) GetTasksByStatus(ctx context.Context, status TaskStatus) ([]*Task, error) {
	if b.registry == nil {
		return nil, fmt.Errorf("board not connected to registry")
	}
	
	storageReg := b.registry.Storage()
	taskRepo := storageReg.GetKanbanTaskRepository()
	if taskRepo == nil {
		return nil, fmt.Errorf("task repository not available")
	}
	
	// Get all tasks for this board from SQLite, then filter by status
	taskInterfaces, err := taskRepo.ListTasksByBoard(ctx, b.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for board: %w", err)
	}
	
	// Convert storage tasks to kanban tasks and filter by status
	var tasks []*Task
	for _, taskInterface := range taskInterfaces {
		// Cast to storage.Task
		storageTask, ok := taskInterface.(*storage.Task)
		if !ok {
			continue // Skip invalid tasks
		}
		
		// Filter by status
		if storageTask.Status != string(status) {
			continue
		}
		
		task := &Task{
			ID:          storageTask.ID,
			Title:       storageTask.Title,
			Description: *storageTask.Description,
			Status:      TaskStatus(storageTask.Status),
			CreatedAt:   storageTask.CreatedAt,
			UpdatedAt:   storageTask.UpdatedAt,
			Metadata:    make(map[string]string),
		}
		
		// Set assigned agent if available
		if storageTask.AssignedAgentID != nil {
			task.AssignedTo = *storageTask.AssignedAgentID
		}
		
		// Add board ID to metadata for compatibility
		task.Metadata["board_id"] = b.ID
		
		tasks = append(tasks, task)
	}
	
	// Sort tasks by UpdatedAt, newest first
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.After(tasks[j].UpdatedAt)
	})
	
	return tasks, nil
}

// GetAllTasks gets all tasks on the board from SQLite
func (b *Board) GetAllTasks(ctx context.Context) ([]*Task, error) {
	if b.registry == nil {
		return nil, fmt.Errorf("board not connected to registry")
	}
	
	storageReg := b.registry.Storage()
	taskRepo := storageReg.GetKanbanTaskRepository()
	if taskRepo == nil {
		return nil, fmt.Errorf("task repository not available")
	}
	
	// Get all tasks for this board from SQLite
	taskInterfaces, err := taskRepo.ListTasksByBoard(ctx, b.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tasks: %w", err)
	}
	
	// Convert storage tasks to kanban tasks
	var allTasks []*Task
	for _, taskInterface := range taskInterfaces {
		// Cast to storage.Task
		storageTask, ok := taskInterface.(*storage.Task)
		if !ok {
			continue // Skip invalid tasks
		}
		
		task := &Task{
			ID:          storageTask.ID,
			Title:       storageTask.Title,
			Description: *storageTask.Description,
			Status:      TaskStatus(storageTask.Status),
			CreatedAt:   storageTask.CreatedAt,
			UpdatedAt:   storageTask.UpdatedAt,
			Metadata:    make(map[string]string),
		}
		
		// Set assigned agent if available
		if storageTask.AssignedAgentID != nil {
			task.AssignedTo = *storageTask.AssignedAgentID
		}
		
		// Add board ID to metadata for compatibility
		task.Metadata["board_id"] = b.ID
		
		allTasks = append(allTasks, task)
	}
	
	// Sort tasks by UpdatedAt, newest first
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].UpdatedAt.After(allTasks[j].UpdatedAt)
	})
	
	return allTasks, nil
}

// FilterTasks filters tasks based on the provided filter using SQLite
func (b *Board) FilterTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	if b.registry == nil {
		return nil, fmt.Errorf("board not connected to registry")
	}
	
	// Get all tasks from SQLite
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
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
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
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
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
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
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
	if b.registry == nil {
		return fmt.Errorf("board not connected to registry")
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

// emitEvent publishes an event (no storage - SQLite handles persistence)
func (b *Board) emitEvent(ctx context.Context, event BoardEvent) error {
	// If event manager is available, publish the event
	if b.eventManager != nil {
		if pubErr := b.eventManager.PublishEvent(&event); pubErr != nil {
			// Log but don't fail the operation if publishing fails
			fmt.Printf("warning: failed to publish event: %v\n", pubErr)
		}
	}

	// Event tracking is now handled by SQLite task operations
	return nil
}