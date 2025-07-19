// Package services provides service wrappers for Guild components
package services

import (
	"context"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/kanban"
	"github.com/lancekrogers/guild/pkg/observability"
	"github.com/lancekrogers/guild/pkg/registry"
)

// KanbanService wraps the kanban board to integrate with the service framework
type KanbanService struct {
	board     *kanban.Board
	registry  registry.ComponentRegistry
	eventBus  events.EventBus
	logger    observability.Logger
	boardPath string

	// Service state
	started bool
	mu      sync.RWMutex

	// Metrics
	tasksCreated   uint64
	tasksCompleted uint64
	tasksDeleted   uint64
}

// KanbanServiceConfig configures the kanban service
type KanbanServiceConfig struct {
	BoardPath    string
	BoardName    string
	Description  string
	AutoSave     bool
	SaveInterval time.Duration
}

// NewKanbanService creates a new kanban service wrapper
func NewKanbanService(
	registry registry.ComponentRegistry,
	eventBus events.EventBus,
	logger observability.Logger,
	config KanbanServiceConfig,
) (*KanbanService, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("KanbanService")
	}
	if eventBus == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "event bus cannot be nil", nil).
			WithComponent("KanbanService")
	}
	if logger == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "logger cannot be nil", nil).
			WithComponent("KanbanService")
	}

	return &KanbanService{
		registry:  registry,
		eventBus:  eventBus,
		logger:    logger,
		boardPath: config.BoardPath,
	}, nil
}

// Name returns the service name
func (s *KanbanService) Name() string {
	return "kanban-service"
}

// Start initializes and starts the service
func (s *KanbanService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "service already started", nil).
			WithComponent("KanbanService")
	}

	// Create adapter to bridge registry interfaces
	kanbanRegistry := &kanbanRegistryAdapter{registry: s.registry}

	// Load or create board
	board, err := kanban.LoadBoard(ctx, kanbanRegistry, s.boardPath)
	if err != nil {
		// If board doesn't exist, create a new one
		s.logger.InfoContext(ctx, "Creating new kanban board", "path", s.boardPath)
		board, err = kanban.NewBoard(ctx, kanbanRegistry, "Guild Tasks", "Task management board for Guild operations")
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create board").
				WithComponent("KanbanService")
		}
	}

	// Create task event publisher that bridges to our event bus
	taskEventPublisher := NewKanbanEventBridge(s.eventBus, s.logger)
	board.SetTaskEventPublisher(taskEventPublisher)

	// Note: EventManager requires comms.PubSub which we don't have here
	// Instead, we'll use the TaskEventPublisher for all events
	// The board will emit events through the publisher

	s.board = board
	s.started = true

	// Emit service started event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"kanban-service-started",
		"service.started",
		"kanban",
		map[string]interface{}{
			"board_path": s.boardPath,
			"board_id":   board.ID,
			"task_count": len(s.getAllTasksUnsafe(ctx)),
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service started event", "error", err)
	}

	s.logger.InfoContext(ctx, "Kanban service started",
		"board_id", board.ID,
		"board_name", board.Name)

	return nil
}

// Stop gracefully shuts down the service
func (s *KanbanService) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("KanbanService")
	}

	// Save board state
	if s.board != nil {
		if err := s.board.Save(ctx); err != nil {
			s.logger.ErrorContext(ctx, "Failed to save board on shutdown", "error", err)
		}
	}

	// Emit service stopped event
	if err := s.eventBus.Publish(ctx, events.NewBaseEvent(
		"kanban-service-stopped",
		"service.stopped",
		"kanban",
		map[string]interface{}{
			"tasks_created":   s.tasksCreated,
			"tasks_completed": s.tasksCompleted,
			"tasks_deleted":   s.tasksDeleted,
		},
	)); err != nil {
		s.logger.WarnContext(ctx, "Failed to publish service stopped event", "error", err)
	}

	s.started = false
	s.board = nil

	s.logger.InfoContext(ctx, "Kanban service stopped",
		"tasks_created", s.tasksCreated,
		"tasks_completed", s.tasksCompleted)

	return nil
}

// Health checks if the service is healthy
func (s *KanbanService) Health(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "service not started", nil).
			WithComponent("KanbanService")
	}

	// Check board is loaded
	if s.board == nil {
		return gerror.New(gerror.ErrCodeInternal, "board not loaded", nil).
			WithComponent("KanbanService")
	}

	// Check registry is still available
	if s.registry == nil {
		return gerror.New(gerror.ErrCodeInternal, "registry not available", nil).
			WithComponent("KanbanService")
	}

	// Check we can access storage
	storageReg := s.registry.Storage()
	if storageReg == nil {
		return gerror.New(gerror.ErrCodeInternal, "storage registry not available", nil).
			WithComponent("KanbanService")
	}

	return nil
}

// Ready checks if the service is ready to handle requests
func (s *KanbanService) Ready(ctx context.Context) error {
	if err := s.Health(ctx); err != nil {
		return err
	}

	// Additional readiness checks could go here
	// For now, healthy == ready

	return nil
}

// CreateTask creates a new task on the board
func (s *KanbanService) CreateTask(ctx context.Context, title, description string) (*kanban.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil, gerror.New(gerror.ErrCodeValidation, "service not started", nil).
			WithComponent("KanbanService")
	}

	task, err := s.board.CreateTask(ctx, title, description)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create task").
			WithComponent("KanbanService")
	}

	s.tasksCreated++
	return task, nil
}

// GetBoard returns the underlying board (for compatibility)
func (s *KanbanService) GetBoard() *kanban.Board {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.board
}

// getAllTasksUnsafe gets all tasks without locking (caller must hold lock)
func (s *KanbanService) getAllTasksUnsafe(ctx context.Context) []*kanban.Task {
	tasks, err := s.board.GetAllTasks(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to get all tasks", "error", err)
		return []*kanban.Task{}
	}
	return tasks
}

// kanbanRegistryAdapter adapts registry.ComponentRegistry to kanban.ComponentRegistry
type kanbanRegistryAdapter struct {
	registry registry.ComponentRegistry
}

// Storage returns a storage registry adapter
func (a *kanbanRegistryAdapter) Storage() kanban.StorageRegistry {
	return &kanbanStorageAdapter{storage: a.registry.Storage()}
}

// kanbanStorageAdapter adapts registry.StorageRegistry to kanban.StorageRegistry
type kanbanStorageAdapter struct {
	storage registry.StorageRegistry
}

func (a *kanbanStorageAdapter) GetKanbanCampaignRepository() kanban.CampaignRepository {
	return a.storage.GetKanbanCampaignRepository()
}

func (a *kanbanStorageAdapter) GetKanbanCommissionRepository() kanban.CommissionRepository {
	return a.storage.GetKanbanCommissionRepository()
}

func (a *kanbanStorageAdapter) GetBoardRepository() kanban.BoardRepository {
	return a.storage.GetBoardRepository()
}

func (a *kanbanStorageAdapter) GetKanbanTaskRepository() kanban.TaskRepository {
	return a.storage.GetKanbanTaskRepository()
}

func (a *kanbanStorageAdapter) GetMemoryStore() kanban.MemoryStore {
	return a.storage.GetMemoryStore()
}

// KanbanEventBridge implements kanban.TaskEventPublisherInterface to bridge task events to the central event bus
type KanbanEventBridge struct {
	eventBus events.EventBus
	logger   observability.Logger
}

// NewKanbanEventBridge creates a new event bridge
func NewKanbanEventBridge(eventBus events.EventBus, logger observability.Logger) *KanbanEventBridge {
	return &KanbanEventBridge{
		eventBus: eventBus,
		logger:   logger,
	}
}

// PublishTaskCreated publishes a task created event
func (b *KanbanEventBridge) PublishTaskCreated(ctx context.Context, task *kanban.Task, boardID, userID string) error {
	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.created",
		"kanban",
		map[string]interface{}{
			"task_id":     task.ID,
			"board_id":    boardID,
			"title":       task.Title,
			"description": task.Description,
			"status":      string(task.Status),
			"user_id":     userID,
			"created_at":  task.CreatedAt,
		},
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskUpdated publishes a task updated event
func (b *KanbanEventBridge) PublishTaskUpdated(ctx context.Context, task *kanban.Task, boardID, userID string, changes map[string]interface{}) error {
	data := map[string]interface{}{
		"task_id":    task.ID,
		"board_id":   boardID,
		"user_id":    userID,
		"changes":    changes,
		"updated_at": task.UpdatedAt,
	}

	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.updated",
		"kanban",
		data,
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskMoved publishes a task moved event
func (b *KanbanEventBridge) PublishTaskMoved(ctx context.Context, task *kanban.Task, boardID, fromColumn, toColumn, userID, comment string) error {
	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.moved",
		"kanban",
		map[string]interface{}{
			"task_id":     task.ID,
			"board_id":    boardID,
			"from_column": fromColumn,
			"to_column":   toColumn,
			"user_id":     userID,
			"comment":     comment,
			"moved_at":    time.Now(),
		},
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskDeleted publishes a task deleted event
func (b *KanbanEventBridge) PublishTaskDeleted(ctx context.Context, taskID, boardID, userID, reason string) error {
	event := events.NewBaseEvent(
		taskID,
		"kanban.task.deleted",
		"kanban",
		map[string]interface{}{
			"task_id":    taskID,
			"board_id":   boardID,
			"user_id":    userID,
			"reason":     reason,
			"deleted_at": time.Now(),
		},
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskCompleted publishes a task completed event
func (b *KanbanEventBridge) PublishTaskCompleted(ctx context.Context, task *kanban.Task, boardID, userID, comment string) error {
	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.completed",
		"kanban",
		map[string]interface{}{
			"task_id":      task.ID,
			"board_id":     boardID,
			"title":        task.Title,
			"user_id":      userID,
			"comment":      comment,
			"completed_at": time.Now(),
		},
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskBlocked publishes a task blocked event
func (b *KanbanEventBridge) PublishTaskBlocked(ctx context.Context, task *kanban.Task, boardID, userID, reason string) error {
	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.blocked",
		"kanban",
		map[string]interface{}{
			"task_id":    task.ID,
			"board_id":   boardID,
			"user_id":    userID,
			"reason":     reason,
			"blocked_at": time.Now(),
		},
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskUnblocked publishes a task unblocked event
func (b *KanbanEventBridge) PublishTaskUnblocked(ctx context.Context, task *kanban.Task, boardID, userID, reason string) error {
	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.unblocked",
		"kanban",
		map[string]interface{}{
			"task_id":      task.ID,
			"board_id":     boardID,
			"user_id":      userID,
			"reason":       reason,
			"unblocked_at": time.Now(),
		},
	)
	return b.eventBus.Publish(ctx, event)
}

// PublishTaskAssigned publishes a task assigned event
func (b *KanbanEventBridge) PublishTaskAssigned(ctx context.Context, task *kanban.Task, boardID, assignedTo, assignedBy string) error {
	event := events.NewBaseEvent(
		task.ID,
		"kanban.task.assigned",
		"kanban",
		map[string]interface{}{
			"task_id":     task.ID,
			"board_id":    boardID,
			"assigned_to": assignedTo,
			"assigned_by": assignedBy,
			"assigned_at": time.Now(),
		},
	)
	return b.eventBus.Publish(ctx, event)
}
