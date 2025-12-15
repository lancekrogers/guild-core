package bridges

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// UIEventBridge connects the UI to the event system
type UIEventBridge struct {
	eventBus events.EventBus
	logger   observability.Logger

	// Configuration
	config UIEventConfig

	// State
	started      bool
	program      *tea.Program
	eventChan    chan tea.Msg
	subscription events.SubscriptionID
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex

	// UI State Management
	uiState   *UIState
	stateFile string
	stateMu   sync.RWMutex

	// State sync
	syncInterval time.Duration
	syncTicker   *time.Ticker

	// Event replay for recovery
	eventHistory []events.CoreEvent
	maxHistory   int

	// Metrics
	eventsReceived uint64
	eventsSent     uint64
	eventsFiltered uint64
	errors         uint64
	stateUpdates   uint64
}

// UIEventConfig configures the UI-event bridge
type UIEventConfig struct {
	// EventFilter filters which events to forward to UI
	EventFilter UIEventFilterFunc

	// BatchEvents enables event batching
	BatchEvents bool

	// BatchInterval for batched events
	BatchInterval time.Duration

	// MaxBatchSize for batched events
	MaxBatchSize int

	// UIEventTypes to subscribe to
	UIEventTypes []string

	// SystemEventTypes to subscribe to for UI updates
	SystemEventTypes []string

	// State persistence
	EnableStatePersistence bool
	StateFile              string
	StateSyncInterval      time.Duration

	// Event history for recovery
	EnableEventHistory bool
	MaxEventHistory    int
}

// UIState represents the current UI state
type UIState struct {
	// Active view
	ActiveView string `json:"active_view"`

	// Session information
	SessionID     string `json:"session_id"`
	CampaignID    string `json:"campaign_id"`
	SelectedGuild string `json:"selected_guild"`

	// Agent states
	AgentStates map[string]AgentUIState `json:"agent_states"`

	// UI component states
	ComponentStates map[string]interface{} `json:"component_states"`

	// Notifications
	Notifications []UINotification `json:"notifications"`

	// Last update
	LastUpdate time.Time `json:"last_update"`
}

// AgentUIState represents an agent's UI state
type AgentUIState struct {
	AgentID     string    `json:"agent_id"`
	Status      string    `json:"status"`
	LastMessage string    `json:"last_message"`
	IsTyping    bool      `json:"is_typing"`
	LastUpdate  time.Time `json:"last_update"`
}

// UINotification represents a UI notification
type UINotification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}

// UIEventFilterFunc filters events for the UI
type UIEventFilterFunc func(event events.CoreEvent) bool

// DefaultUIEventConfig returns default configuration
func DefaultUIEventConfig() UIEventConfig {
	return UIEventConfig{
		BatchEvents:   true,
		BatchInterval: 50 * time.Millisecond,
		MaxBatchSize:  10,
		UIEventTypes: []string{
			"ui.*",
			"user.*",
			"session.*",
		},
		SystemEventTypes: []string{
			"agent.status.*",
			"task.status.*",
			"commission.status.*",
			"system.notification.*",
		},
		EnableStatePersistence: true,
		StateFile:              ".guild/ui-state.json",
		StateSyncInterval:      5 * time.Second,
		EnableEventHistory:     true,
		MaxEventHistory:        100,
	}
}

// NewUIEventBridge creates a new UI-event bridge
func NewUIEventBridge(eventBus events.EventBus, logger observability.Logger, config UIEventConfig) *UIEventBridge {
	bridge := &UIEventBridge{
		eventBus:     eventBus,
		logger:       logger,
		config:       config,
		eventChan:    make(chan tea.Msg, 100),
		stopCh:       make(chan struct{}),
		syncInterval: config.StateSyncInterval,
		stateFile:    config.StateFile,
		maxHistory:   config.MaxEventHistory,
	}

	// Initialize UI state
	bridge.uiState = &UIState{
		AgentStates:     make(map[string]AgentUIState),
		ComponentStates: make(map[string]interface{}),
		Notifications:   []UINotification{},
		LastUpdate:      time.Now(),
	}

	// Initialize event history if enabled
	if config.EnableEventHistory {
		bridge.eventHistory = make([]events.CoreEvent, 0, config.MaxEventHistory)
	}

	return bridge
}

// Name returns the service name
func (b *UIEventBridge) Name() string {
	return "ui-event-bridge"
}

// Start initializes and starts the bridge
func (b *UIEventBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "bridge already started", nil).
			WithComponent("ui_event_bridge")
	}

	// Load persisted state if enabled
	if b.config.EnableStatePersistence {
		if err := b.loadState(ctx); err != nil {
			b.logger.WarnContext(ctx, "Failed to load UI state, starting fresh",
				"error", err,
				"state_file", b.stateFile)
		}
	}

	// Subscribe to all events and filter in handler
	handler := func(ctx context.Context, event events.CoreEvent) error {
		// Process event asynchronously
		select {
		case <-b.stopCh:
			return nil
		default:
			b.processEvent(event)
		}
		return nil
	}

	subID, err := b.eventBus.SubscribeAll(ctx, handler)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to events").
			WithComponent("ui_event_bridge")
	}

	b.subscription = subID

	// Start state sync if enabled
	if b.config.EnableStatePersistence {
		b.syncTicker = time.NewTicker(b.syncInterval)
		b.wg.Add(1)
		go b.stateSyncLoop(ctx)
	}

	// Event processing is handled by the subscription handler

	b.started = true
	b.logger.InfoContext(ctx, "UI-event bridge started",
		"ui_types", b.config.UIEventTypes,
		"system_types", b.config.SystemEventTypes,
		"state_persistence", b.config.EnableStatePersistence,
		"event_history", b.config.EnableEventHistory)

	return nil
}

// Stop gracefully shuts down the bridge
func (b *UIEventBridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return gerror.New(gerror.ErrCodeValidation, "bridge not started", nil).
			WithComponent("ui_event_bridge")
	}
	b.started = false
	subscription := b.subscription
	ticker := b.syncTicker
	b.mu.Unlock()

	// Stop state sync ticker
	if ticker != nil {
		ticker.Stop()
	}

	// Save final state if enabled
	if b.config.EnableStatePersistence {
		if err := b.saveState(ctx); err != nil {
			b.logger.ErrorContext(ctx, "Failed to save final UI state",
				"error", err,
				"state_file", b.stateFile)
		}
	}

	// Unsubscribe from events
	if err := b.eventBus.Unsubscribe(ctx, subscription); err != nil {
		b.logger.ErrorContext(ctx, "Failed to unsubscribe from events",
			"error", err,
			"subscription", subscription)
	}

	// Signal shutdown
	close(b.stopCh)

	// Wait for processor to finish
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(b.eventChan)
		b.logger.InfoContext(ctx, "UI-event bridge stopped",
			"events_received", b.eventsReceived,
			"events_sent", b.eventsSent,
			"events_filtered", b.eventsFiltered,
			"errors", b.errors,
			"state_updates", b.stateUpdates)
		return nil
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout stopping bridge").
			WithComponent("ui_event_bridge")
	}
}

// Health checks if the bridge is healthy
func (b *UIEventBridge) Health(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.started {
		return gerror.New(gerror.ErrCodeResourceExhausted, "bridge not started", nil).
			WithComponent("ui_event_bridge")
	}

	return nil
}

// Ready checks if the bridge is ready
func (b *UIEventBridge) Ready(ctx context.Context) error {
	return b.Health(ctx)
}

// SetProgram sets the Bubble Tea program for UI updates
func (b *UIEventBridge) SetProgram(p *tea.Program) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.program = p
}

// EventChannel returns the channel for UI events
func (b *UIEventBridge) EventChannel() <-chan tea.Msg {
	return b.eventChan
}

// PublishUIEvent publishes a UI event to the event system
func (b *UIEventBridge) PublishUIEvent(ctx context.Context, eventType string, data interface{}) error {
	// Generate a unique event ID
	eventID := fmt.Sprintf("ui-%d-%s", time.Now().UnixNano(), eventType)

	// Convert data to map[string]interface{}
	dataMap := make(map[string]interface{})
	if data != nil {
		dataMap["payload"] = data
	}

	event := events.NewBaseEvent(eventID, eventType, "ui", dataMap).
		WithMetadata("bridge", "ui-event")

	if err := b.eventBus.Publish(ctx, event); err != nil {
		b.mu.Lock()
		b.errors++
		b.mu.Unlock()

		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to publish UI event").
			WithComponent("ui_event_bridge").
			WithDetails("event_type", eventType)
	}

	b.mu.Lock()
	b.eventsSent++
	b.mu.Unlock()

	return nil
}

// processEvent processes a single event from the event bus
func (b *UIEventBridge) processEvent(event events.CoreEvent) {
	b.mu.Lock()
	b.eventsReceived++
	b.mu.Unlock()

	// Add to history if enabled
	b.AddToHistory(event)

	// Check if event type matches our subscription
	eventType := event.GetType()
	matches := false
	for _, pattern := range append(b.config.UIEventTypes, b.config.SystemEventTypes...) {
		if matchesPattern(eventType, pattern) {
			matches = true
			break
		}
	}

	if !matches {
		return
	}

	// Apply filter if configured
	if b.config.EventFilter != nil && !b.config.EventFilter(event) {
		b.mu.Lock()
		b.eventsFiltered++
		b.mu.Unlock()
		return
	}

	// Convert to UI message
	uiMsg := b.eventToUIMessage(event)

	// Send to UI channel
	select {
	case b.eventChan <- uiMsg:
		// Update program if set
		b.mu.RLock()
		program := b.program
		b.mu.RUnlock()

		if program != nil {
			program.Send(uiMsg)
		}
	case <-b.stopCh:
		return
	default:
		// Channel full, drop event
		b.logger.Warn("UI event channel full, dropping event",
			"event_type", event.GetType())
	}
}

// matchesPattern checks if an event type matches a pattern
func matchesPattern(eventType, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(eventType) >= len(prefix) && eventType[:len(prefix)] == prefix
	}
	return eventType == pattern
}

// eventToUIMessage converts an event to a UI message
func (b *UIEventBridge) eventToUIMessage(event events.CoreEvent) tea.Msg {
	// Convert based on event type
	eventType := event.GetType()

	switch {
	case isUIEvent(eventType):
		return UIEventMsg{
			Type:      eventType,
			Data:      event.GetData(),
			Timestamp: event.GetTimestamp(),
			Metadata:  event.GetMetadata(),
		}

	case isSystemNotification(eventType):
		return SystemNotificationMsg{
			Type:      eventType,
			Data:      event.GetData(),
			Timestamp: event.GetTimestamp(),
		}

	case isStatusUpdate(eventType):
		return StatusUpdateMsg{
			Type:      eventType,
			Data:      event.GetData(),
			Timestamp: event.GetTimestamp(),
		}

	default:
		return GenericEventMsg{
			Event: event,
		}
	}
}

// Helper functions

func isUIEvent(eventType string) bool {
	return len(eventType) >= 3 && eventType[:3] == "ui."
}

func isSystemNotification(eventType string) bool {
	return containsPrefix(eventType, "system.notification")
}

func isStatusUpdate(eventType string) bool {
	return containsPrefix(eventType, "status") ||
		containsPrefix(eventType, "agent.status") ||
		containsPrefix(eventType, "task.status")
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// GetMetrics returns bridge metrics
func (b *UIEventBridge) GetMetrics() UIEventMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return UIEventMetrics{
		EventsReceived: b.eventsReceived,
		EventsSent:     b.eventsSent,
		EventsFiltered: b.eventsFiltered,
		Errors:         b.errors,
		Running:        b.started,
	}
}

// UIEventMetrics contains bridge metrics
type UIEventMetrics struct {
	EventsReceived uint64
	EventsSent     uint64
	EventsFiltered uint64
	Errors         uint64
	Running        bool
}

// UI Message types for Bubble Tea

// UIEventMsg represents a UI event
type UIEventMsg struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// SystemNotificationMsg represents a system notification
type SystemNotificationMsg struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
}

// StatusUpdateMsg represents a status update
type StatusUpdateMsg struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
}

// GenericEventMsg wraps any event
type GenericEventMsg struct {
	Event events.CoreEvent
}

// State Management Methods

// UpdateUIState updates the UI state
func (b *UIEventBridge) UpdateUIState(update func(*UIState)) {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()

	update(b.uiState)
	b.uiState.LastUpdate = time.Now()
	b.stateUpdates++

	// Emit state update event
	go func() {
		ctx := context.Background()
		if err := b.PublishUIEvent(ctx, "ui.state.updated", b.uiState); err != nil {
			b.logger.Warn("Failed to publish UI state update", "error", err)
		}
	}()
}

// GetUIState returns a copy of the current UI state
func (b *UIEventBridge) GetUIState() UIState {
	b.stateMu.RLock()
	defer b.stateMu.RUnlock()

	// Return a deep copy to prevent external modifications
	state := *b.uiState

	// Deep copy agent states
	state.AgentStates = make(map[string]AgentUIState)
	for k, v := range b.uiState.AgentStates {
		state.AgentStates[k] = v
	}

	// Deep copy notifications
	state.Notifications = make([]UINotification, len(b.uiState.Notifications))
	copy(state.Notifications, b.uiState.Notifications)

	return state
}

// UpdateAgentState updates a specific agent's UI state
func (b *UIEventBridge) UpdateAgentState(agentID string, update func(*AgentUIState)) {
	b.UpdateUIState(func(state *UIState) {
		agentState, exists := state.AgentStates[agentID]
		if !exists {
			agentState = AgentUIState{
				AgentID: agentID,
			}
		}
		update(&agentState)
		agentState.LastUpdate = time.Now()
		state.AgentStates[agentID] = agentState
	})
}

// AddNotification adds a notification to the UI state
func (b *UIEventBridge) AddNotification(notificationType, message string) {
	notification := UINotification{
		ID:        fmt.Sprintf("notif-%d", time.Now().UnixNano()),
		Type:      notificationType,
		Message:   message,
		Timestamp: time.Now(),
		Read:      false,
	}

	b.UpdateUIState(func(state *UIState) {
		state.Notifications = append(state.Notifications, notification)

		// Keep only last 50 notifications
		if len(state.Notifications) > 50 {
			state.Notifications = state.Notifications[len(state.Notifications)-50:]
		}
	})
}

// RegisterComponent registers a UI component state
func (b *UIEventBridge) RegisterComponent(componentID string, initialState interface{}) {
	b.UpdateUIState(func(state *UIState) {
		if state.ComponentStates == nil {
			state.ComponentStates = make(map[string]interface{})
		}
		state.ComponentStates[componentID] = initialState
	})
}

// UpdateComponentState updates a UI component's state
func (b *UIEventBridge) UpdateComponentState(componentID string, newState interface{}) {
	b.UpdateUIState(func(state *UIState) {
		if state.ComponentStates == nil {
			state.ComponentStates = make(map[string]interface{})
		}
		state.ComponentStates[componentID] = newState
	})
}

// State Persistence Methods

// loadState loads UI state from disk
func (b *UIEventBridge) loadState(ctx context.Context) error {
	data, err := os.ReadFile(b.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No state file yet, not an error
			return nil
		}
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read state file").
			WithComponent("ui_event_bridge")
	}

	var state UIState
	if err := json.Unmarshal(data, &state); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to unmarshal state").
			WithComponent("ui_event_bridge")
	}

	b.stateMu.Lock()
	b.uiState = &state
	b.stateMu.Unlock()

	b.logger.InfoContext(ctx, "Loaded UI state from disk",
		"state_file", b.stateFile,
		"active_view", state.ActiveView,
		"agent_count", len(state.AgentStates))

	return nil
}

// saveState saves UI state to disk
func (b *UIEventBridge) saveState(ctx context.Context) error {
	b.stateMu.RLock()
	data, err := json.MarshalIndent(b.uiState, "", "  ")
	b.stateMu.RUnlock()

	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to marshal state").
			WithComponent("ui_event_bridge")
	}

	// Ensure directory exists
	dir := filepath.Dir(b.stateFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create state directory").
			WithComponent("ui_event_bridge")
	}

	// Write atomically
	tmpFile := b.stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to write state file").
			WithComponent("ui_event_bridge")
	}

	if err := os.Rename(tmpFile, b.stateFile); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to rename state file").
			WithComponent("ui_event_bridge")
	}

	return nil
}

// stateSyncLoop periodically saves state to disk
func (b *UIEventBridge) stateSyncLoop(ctx context.Context) {
	defer b.wg.Done()

	for {
		select {
		case <-b.stopCh:
			return
		case <-b.syncTicker.C:
			if err := b.saveState(ctx); err != nil {
				b.logger.ErrorContext(ctx, "Failed to sync UI state",
					"error", err)
				b.mu.Lock()
				b.errors++
				b.mu.Unlock()
			}
		}
	}
}

// Event History Methods

// AddToHistory adds an event to the history
func (b *UIEventBridge) AddToHistory(event events.CoreEvent) {
	if !b.config.EnableEventHistory {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.eventHistory = append(b.eventHistory, event)

	// Trim to max size
	if len(b.eventHistory) > b.maxHistory {
		b.eventHistory = b.eventHistory[len(b.eventHistory)-b.maxHistory:]
	}
}

// GetEventHistory returns the event history
func (b *UIEventBridge) GetEventHistory() []events.CoreEvent {
	b.mu.RLock()
	defer b.mu.RUnlock()

	history := make([]events.CoreEvent, len(b.eventHistory))
	copy(history, b.eventHistory)
	return history
}

// ReplayEvents replays events from history
func (b *UIEventBridge) ReplayEvents(ctx context.Context, filter func(events.CoreEvent) bool) error {
	history := b.GetEventHistory()

	for _, event := range history {
		if filter != nil && !filter(event) {
			continue
		}

		// Process event through normal pipeline
		b.processEvent(event)
	}

	b.logger.InfoContext(ctx, "Replayed events from history",
		"total_events", len(history))

	return nil
}
