package bridges

import (
	"context"
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lancekrogers/guild/pkg/events"
	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/observability"
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

	// Metrics
	eventsReceived uint64
	eventsSent     uint64
	eventsFiltered uint64
	errors         uint64
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
	}
}

// NewUIEventBridge creates a new UI-event bridge
func NewUIEventBridge(eventBus events.EventBus, logger observability.Logger, config UIEventConfig) *UIEventBridge {
	return &UIEventBridge{
		eventBus:  eventBus,
		logger:    logger,
		config:    config,
		eventChan: make(chan tea.Msg, 100),
		stopCh:    make(chan struct{}),
	}
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

	// Event processing is handled by the subscription handler

	b.started = true
	b.logger.InfoContext(ctx, "UI-event bridge started",
		"ui_types", b.config.UIEventTypes,
		"system_types", b.config.SystemEventTypes)

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
	b.mu.Unlock()

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
			"errors", b.errors)
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
