package kanban

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/guild-ventures/guild-core/pkg/comms"
	"github.com/guild-ventures/guild-core/pkg/comms/channel"
	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/memory"
)

// Manager manages multiple kanban boards
type Manager struct {
	store        memory.Store
	registry     ComponentRegistry // Optional registry for new storage backend
	boards       map[string]*Board
	eventStream  chan BoardEvent
	eventManager *EventManager
	pubsub       comms.PubSub
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// ManagerEventHandler is a function that handles board events for the manager
type ManagerEventHandler func(event BoardEvent)

// NewManager creates a new kanban manager
func NewManager(ctx context.Context, store memory.Store) (*Manager, error) {
	return NewManagerWithConfig(ctx, store, map[string]interface{}{
		"pub_endpoint": "tcp://127.0.0.1:5556",
		"sub_endpoint": "tcp://127.0.0.1:5556",
		"identity":     "kanban-manager",
	})
}

// NewManagerWithConfig creates a new kanban manager with custom channel config
func NewManagerWithConfig(ctx context.Context, store memory.Store, channelConfig map[string]interface{}) (*Manager, error) {
	if store == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "store cannot be nil", nil).
			WithComponent("KanbanManager").
			WithOperation("NewManagerWithConfig")
	}

	// Initialize channel-based messaging
	transport := channel.NewTransport()
	pubsub, err := transport.NewPubSub(ctx, channelConfig)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create channel pubsub").
			WithComponent("KanbanManager").
			WithOperation("NewManagerWithConfig")
	}

	// Create event manager
	eventManager := NewEventManager(ctx, pubsub, "kanban.")

	// Create a cancellable context for the manager
	mgrCtx, cancel := context.WithCancel(ctx)

	// Create manager
	manager := &Manager{
		store:        store,
		registry:     nil, // Will be set by NewManagerWithRegistry if needed
		boards:       make(map[string]*Board),
		eventStream:  make(chan BoardEvent, 100), // Buffer up to 100 events
		eventManager: eventManager,
		pubsub:       pubsub,
		mu:           sync.RWMutex{},
		ctx:          mgrCtx,
		cancel:       cancel,
	}

	// Subscribe to important events for internal processing
	eventManager.SubscribeAll(func(event *BoardEvent) error {
		// Forward to channel for backward compatibility
		select {
		case manager.eventStream <- *event:
		default:
			// Channel full, just drop the event
		}
		return nil
	})

	// Start event processor for backward compatibility
	go manager.processEvents()

	return manager, nil
}

// NewManagerWithRegistry creates a new kanban manager using the component registry
// This allows the manager to work with either SQLite or BoltDB storage backends
func NewManagerWithRegistry(ctx context.Context, registry ComponentRegistry) (*Manager, error) {
	if registry == nil {
		return nil, gerror.New(gerror.ErrCodeValidation, "registry cannot be nil", nil).
			WithComponent("KanbanManager").
			WithOperation("NewManagerWithRegistry")
	}

	// Get memory store from storage registry
	storageReg := registry.Storage()
	if storageReg == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "storage registry not initialized", nil).
			WithComponent("KanbanManager").
			WithOperation("NewManagerWithRegistry")
	}

	memoryStore := storageReg.GetMemoryStore()
	if memoryStore == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "memory store not available from storage registry", nil).
			WithComponent("KanbanManager").
			WithOperation("NewManagerWithRegistry")
	}

	// Convert local interface to memory.Store for compatibility
	memStoreCompat, ok := memoryStore.(memory.Store)
	if !ok {
		return nil, gerror.New(gerror.ErrCodeInternal, "memory store does not implement required interface", nil).
			WithComponent("KanbanManager").
			WithOperation("NewManagerWithRegistry")
	}

	// Create manager with registry support
	manager, err := NewManagerWithConfig(ctx, memStoreCompat, map[string]interface{}{
		"pub_endpoint": "tcp://127.0.0.1:5556",
		"sub_endpoint": "tcp://127.0.0.1:5556",
		"identity":     "kanban-manager",
	})
	if err != nil {
		return nil, err
	}

	// Set registry for advanced operations
	manager.registry = registry

	return manager, nil
}

// CreateBoard creates a new board
func (m *Manager) CreateBoard(ctx context.Context, name, description string) (*Board, error) {
	var board *Board
	var err error

	// Use SQLite if registry is available, otherwise use legacy store
	if m.registry != nil {
		board, err = NewBoardWithRegistry(ctx, m.registry, name, description)
	} else {
		return nil, gerror.New(gerror.ErrCodeInternal, "board creation requires registry for SQLite backend", nil).
			WithComponent("KanbanManager").
			WithOperation("CreateBoard")
	}
	
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create board").
			WithComponent("KanbanManager").
			WithOperation("CreateBoard")
	}

	// Set the event manager on the new board
	if m.eventManager != nil {
		board.SetEventManager(m.eventManager)
	}

	m.mu.Lock()
	m.boards[board.ID] = board
	m.mu.Unlock()

	return board, nil
}

// GetBoard gets a board by ID, loading it from the store if necessary
func (m *Manager) GetBoard(ctx context.Context, boardID string) (*Board, error) {
	m.mu.RLock()
	board, exists := m.boards[boardID]
	m.mu.RUnlock()

	if exists {
		return board, nil
	}

	// Load from SQLite
	if m.registry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "board loading requires registry for SQLite backend", nil).
			WithComponent("KanbanManager").
			WithOperation("GetBoard")
	}
	board, err := LoadBoard(ctx, m.registry, boardID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load board").
			WithComponent("KanbanManager").
			WithOperation("GetBoard")
	}

	// Set the event manager on the loaded board
	if m.eventManager != nil {
		board.SetEventManager(m.eventManager)
	}

	m.mu.Lock()
	m.boards[board.ID] = board
	m.mu.Unlock()

	return board, nil
}

// ListBoards lists all boards
func (m *Manager) ListBoards(ctx context.Context) ([]*Board, error) {
	if m.registry == nil {
		return nil, gerror.New(gerror.ErrCodeInternal, "board listing requires registry for SQLite backend", nil).
			WithComponent("KanbanManager").
			WithOperation("ListBoards")
	}
	return ListBoards(ctx, m.registry)
}

// DeleteBoard deletes a board
func (m *Manager) DeleteBoard(ctx context.Context, boardID string) error {
	m.mu.Lock()
	board, exists := m.boards[boardID]
	if exists {
		delete(m.boards, boardID)
	}
	m.mu.Unlock()

	if !exists {
		// Try to load it first
		var err error
		if m.registry == nil {
			return gerror.New(gerror.ErrCodeInternal, "board loading requires registry for SQLite backend", nil).
				WithComponent("KanbanManager").
				WithOperation("DeleteBoard")
		}
		board, err = LoadBoard(ctx, m.registry, boardID)
		if err != nil {
			return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to load board").
				WithComponent("KanbanManager").
				WithOperation("DeleteBoard")
		}
	}

	return board.Delete(ctx)
}

// GetTask gets a task by ID, searching across all boards
func (m *Manager) GetTask(ctx context.Context, taskID string) (*Task, error) {
	// Try to get the task directly from the store
	data, err := m.store.Get(ctx, "tasks", taskID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get task").
			WithComponent("KanbanManager").
			WithOperation("GetTask")
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal task").
			WithComponent("KanbanManager").
			WithOperation("GetTask")
	}

	// Get the board ID from the task's metadata
	boardID, ok := task.Metadata["board_id"]
	if !ok {
		return nil, gerror.New(gerror.ErrCodeValidation, "task has no board_id in metadata", nil).
			WithComponent("KanbanManager").
			WithOperation("GetTask")
	}

	// Get the board
	board, err := m.GetBoard(ctx, boardID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get board").
			WithComponent("KanbanManager").
			WithOperation("GetTask")
	}

	// Verify the task belongs to the board
	return board.GetTask(ctx, taskID)
}

// CreateTask creates a task on the specified board
func (m *Manager) CreateTask(ctx context.Context, boardID, title, description string) (*Task, error) {
	board, err := m.GetBoard(ctx, boardID)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get board").
			WithComponent("KanbanManager").
			WithOperation("CreateTask")
	}

	return board.CreateTask(ctx, title, description)
}

// UpdateTaskStatus updates a task's status
func (m *Manager) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, changedBy, comment string) error {
	// Get the task to find its board
	task, err := m.GetTask(ctx, taskID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get task").
			WithComponent("KanbanManager").
			WithOperation("UpdateTaskStatus")
	}

	// Get the board
	boardID, ok := task.Metadata["board_id"]
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "task has no board_id in metadata", nil).
			WithComponent("KanbanManager").
			WithOperation("UpdateTaskStatus")
	}

	board, err := m.GetBoard(ctx, boardID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get board").
			WithComponent("KanbanManager").
			WithOperation("UpdateTaskStatus")
	}

	return board.UpdateTaskStatus(ctx, taskID, status, changedBy, comment)
}

// AssignTask assigns a task to a user
func (m *Manager) AssignTask(ctx context.Context, taskID, assignee, changedBy, comment string) error {
	// Get the task to find its board
	task, err := m.GetTask(ctx, taskID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get task").
			WithComponent("KanbanManager").
			WithOperation("AssignTask")
	}

	// Get the board
	boardID, ok := task.Metadata["board_id"]
	if !ok {
		return gerror.New(gerror.ErrCodeValidation, "task has no board_id in metadata", nil).
			WithComponent("KanbanManager").
			WithOperation("AssignTask")
	}

	board, err := m.GetBoard(ctx, boardID)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get board").
			WithComponent("KanbanManager").
			WithOperation("AssignTask")
	}

	return board.AssignTask(ctx, taskID, assignee, changedBy, comment)
}

// ListTasksByStatus gets all tasks with the given status across all boards
func (m *Manager) ListTasksByStatus(ctx context.Context, status TaskStatus) ([]*Task, error) {
	var allTasks []*Task

	// List all boards
	boards, err := m.ListBoards(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list boards").
			WithComponent("KanbanManager").
			WithOperation("ListTasksByStatus")
	}

	// Get tasks from each board
	for _, board := range boards {
		tasks, err := board.GetTasksByStatus(ctx, status)
		if err != nil {
			// Log error but continue
			fmt.Printf("warning: failed to get tasks for board %s: %v\n", board.ID, err)
			continue
		}

		allTasks = append(allTasks, tasks...)
	}

	return allTasks, nil
}

// ListTasksByAgent gets all tasks assigned to a specific agent
func (m *Manager) ListTasksByAgent(ctx context.Context, agentID string) ([]*Task, error) {
	var agentTasks []*Task

	// List all boards
	boards, err := m.ListBoards(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to list boards").
			WithComponent("KanbanManager").
			WithOperation("ListTasksByAgent")
	}

	// Get tasks from each board
	for _, board := range boards {
		tasks, err := board.FilterTasks(ctx, FilterByAssignee(agentID))
		if err != nil {
			// Log error but continue
			fmt.Printf("warning: failed to filter tasks for board %s: %v\n", board.ID, err)
			continue
		}

		agentTasks = append(agentTasks, tasks...)
	}

	return agentTasks, nil
}

// AddEventListener adds an event listener for board events
func (m *Manager) AddEventListener(handler ManagerEventHandler) chan<- bool {
	stopCh := make(chan bool)
	go func() {
		events := m.GetEventChannel()
		for {
			select {
			case event := <-events:
				handler(event)
			case <-stopCh:
				return
			}
		}
	}()
	return stopCh
}

// GetEventChannel returns a read-only channel for board events
func (m *Manager) GetEventChannel() <-chan BoardEvent {
	return m.eventStream
}

// processEvents processes events from the store
func (m *Manager) processEvents() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastProcessedTime time.Time

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Load recent events from the store
			events, err := m.loadEventsAfter(m.ctx, lastProcessedTime)
			if err != nil {
				fmt.Printf("error loading events: %v\n", err)
				continue
			}

			if len(events) > 0 {
			lastProcessedTime = events[len(events)-1].OccurredAt

			// Send events to the event stream
			for _, event := range events {
				select {
				case m.eventStream <- event:
					// Event sent successfully
				default:
					// Channel is full, log and continue
					fmt.Printf("warning: event channel is full, dropping event\n")
				}
			}
		}
	}
}

// loadEventsAfter loads events that occurred after the given time
func (m *Manager) loadEventsAfter(ctx context.Context, after time.Time) ([]BoardEvent, error) {
	// Event streaming simplified - SQLite handles task state persistence
	// For now, return empty slice. Real-time events can be implemented later via pub/sub
	return []BoardEvent{}, nil
}

// sortEvents sorts events by occurrence time
func sortEvents(events []BoardEvent) {
	// Simple bubble sort for now
	n := len(events)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if events[j].OccurredAt.After(events[j+1].OccurredAt) {
				events[j], events[j+1] = events[j+1], events[j]
			}
		}
	}
}

// Close closes the manager and releases resources
func (m *Manager) Close() error {
	// Close the event manager
	if m.eventManager != nil {
		m.eventManager.Close()
	}

	// Close the pubsub
	if m.pubsub != nil {
		m.pubsub.Close()
	}

	// Close the event stream
	close(m.eventStream)

	return nil
}

