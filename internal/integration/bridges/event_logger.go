// Package bridges provides integration bridges between Guild components
package bridges

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild-core/pkg/events"
	"github.com/lancekrogers/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-core/pkg/observability"
)

// EventLoggerBridge connects the event system to the logging infrastructure
type EventLoggerBridge struct {
	eventBus events.EventBus
	logger   observability.Logger

	// Configuration
	config EventLoggerConfig

	// State
	started        bool
	stopCh         chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
	subscriptionID events.SubscriptionID
	eventCh        chan events.CoreEvent

	// Metrics
	eventsLogged   uint64
	eventsFiltered uint64
	errors         uint64
}

// EventLoggerConfig configures the event-logger bridge
type EventLoggerConfig struct {
	// LogLevel determines minimum event priority to log
	LogLevel EventLogLevel

	// EventFilter allows custom filtering of events
	EventFilter EventFilterFunc

	// IncludeEventData includes full event data in logs
	IncludeEventData bool

	// BufferSize for event channel
	BufferSize int

	// FlushInterval for batched logging
	FlushInterval time.Duration

	// MaxBatchSize for batched logging
	MaxBatchSize int
}

// EventLogLevel represents logging levels for events
type EventLogLevel int

const (
	LogLevelDebug EventLogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// EventFilterFunc filters events for logging
type EventFilterFunc func(event events.CoreEvent) bool

// DefaultEventLoggerConfig returns default configuration
func DefaultEventLoggerConfig() EventLoggerConfig {
	return EventLoggerConfig{
		LogLevel:         LogLevelInfo,
		IncludeEventData: true,
		BufferSize:       1000,
		FlushInterval:    100 * time.Millisecond,
		MaxBatchSize:     100,
	}
}

// NewEventLoggerBridge creates a new event-logger bridge
func NewEventLoggerBridge(eventBus events.EventBus, logger observability.Logger, config EventLoggerConfig) *EventLoggerBridge {
	return &EventLoggerBridge{
		eventBus: eventBus,
		logger:   logger,
		config:   config,
		stopCh:   make(chan struct{}),
		eventCh:  make(chan events.CoreEvent, config.BufferSize),
	}
}

// Name returns the service name
func (b *EventLoggerBridge) Name() string {
	return "event-logger-bridge"
}

// Start initializes and starts the bridge
func (b *EventLoggerBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return gerror.New(gerror.ErrCodeAlreadyExists, "bridge already started", nil).
			WithComponent("event_logger_bridge")
	}

	// Create event handler
	eventHandler := func(ctx context.Context, event events.CoreEvent) error {
		return b.handleEvent(ctx, event)
	}

	// Subscribe to all events
	subscriptionID, err := b.eventBus.SubscribeAll(ctx, eventHandler)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to subscribe to events").
			WithComponent("event_logger_bridge")
	}
	b.subscriptionID = subscriptionID

	// Start event processor
	b.wg.Add(1)
	go b.processEvents(ctx)

	b.started = true
	b.logger.InfoContext(ctx, "Event-logger bridge started",
		"log_level", b.config.LogLevel,
		"buffer_size", b.config.BufferSize)

	return nil
}

// Stop gracefully shuts down the bridge
func (b *EventLoggerBridge) Stop(ctx context.Context) error {
	b.mu.Lock()
	if !b.started {
		b.mu.Unlock()
		return gerror.New(gerror.ErrCodeValidation, "bridge not started", nil).
			WithComponent("event_logger_bridge")
	}
	b.started = false
	b.mu.Unlock()

	// Signal shutdown
	close(b.stopCh)

	// Wait for processor to finish with timeout
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.logger.InfoContext(ctx, "Event-logger bridge stopped",
			"events_logged", b.eventsLogged,
			"events_filtered", b.eventsFiltered,
			"errors", b.errors)
		return nil
	case <-ctx.Done():
		return gerror.Wrap(ctx.Err(), gerror.ErrCodeTimeout, "timeout stopping bridge").
			WithComponent("event_logger_bridge")
	}
}

// Health checks if the bridge is healthy
func (b *EventLoggerBridge) Health(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.started {
		return gerror.New(gerror.ErrCodeInternal, "bridge not started", nil).
			WithComponent("event_logger_bridge")
	}

	return nil
}

// Ready checks if the bridge is ready to handle events
func (b *EventLoggerBridge) Ready(ctx context.Context) error {
	return b.Health(ctx)
}

// handleEvent handles a single event from the event bus
func (b *EventLoggerBridge) handleEvent(ctx context.Context, event events.CoreEvent) error {
	select {
	case b.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Channel full, increment dropped counter
		b.eventsFiltered++
		return nil
	}
}

// processEvents processes events from the event bus
func (b *EventLoggerBridge) processEvents(ctx context.Context) {
	defer b.wg.Done()

	batch := make([]events.CoreEvent, 0, b.config.MaxBatchSize)
	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			// Flush remaining events
			if len(batch) > 0 {
				b.logEventBatch(ctx, batch)
			}
			return

		case <-ticker.C:
			// Flush batch periodically
			if len(batch) > 0 {
				b.logEventBatch(ctx, batch)
				batch = batch[:0]
			}

		case event, ok := <-b.eventCh:
			if !ok {
				return
			}

			// Apply filter
			if b.config.EventFilter != nil && !b.config.EventFilter(event) {
				b.mu.Lock()
				b.eventsFiltered++
				b.mu.Unlock()
				continue
			}

			// Check log level
			if !b.shouldLog(event) {
				b.mu.Lock()
				b.eventsFiltered++
				b.mu.Unlock()
				continue
			}

			// Add to batch
			batch = append(batch, event)

			// Flush if batch is full
			if len(batch) >= b.config.MaxBatchSize {
				b.logEventBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// logEventBatch logs a batch of events
func (b *EventLoggerBridge) logEventBatch(ctx context.Context, events []events.CoreEvent) {
	for _, event := range events {
		b.logEvent(ctx, event)
	}
}

// logEvent logs a single event
func (b *EventLoggerBridge) logEvent(ctx context.Context, event events.CoreEvent) {
	fields := []interface{}{
		"event_id", event.GetID(),
		"event_type", event.GetType(),
		"timestamp", event.GetTimestamp(),
		"source", event.GetSource(),
	}

	// Add event data if configured
	if b.config.IncludeEventData {
		if data := event.GetData(); data != nil && len(data) > 0 {
			fields = append(fields, "data", data)
		}
	}

	// Add metadata
	if metadata := event.GetMetadata(); len(metadata) > 0 {
		fields = append(fields, "metadata", metadata)
	}

	// Determine log level and log
	switch b.getEventLogLevel(event) {
	case LogLevelDebug:
		b.logger.DebugContext(ctx, fmt.Sprintf("Event: %s", event.GetType()), fields...)
	case LogLevelInfo:
		b.logger.InfoContext(ctx, fmt.Sprintf("Event: %s", event.GetType()), fields...)
	case LogLevelWarn:
		b.logger.WarnContext(ctx, fmt.Sprintf("Event: %s", event.GetType()), fields...)
	case LogLevelError:
		b.logger.ErrorContext(ctx, fmt.Sprintf("Event: %s", event.GetType()), fields...)
	}

	b.mu.Lock()
	b.eventsLogged++
	b.mu.Unlock()
}

// shouldLog determines if an event should be logged based on priority
func (b *EventLoggerBridge) shouldLog(event events.CoreEvent) bool {
	eventLevel := b.getEventLogLevel(event)
	return eventLevel >= b.config.LogLevel
}

// getEventLogLevel determines the log level for an event
func (b *EventLoggerBridge) getEventLogLevel(event events.CoreEvent) EventLogLevel {
	// Check for error events
	if isErrorEvent(event) {
		return LogLevelError
	}

	// Check for warning events
	if isWarningEvent(event) {
		return LogLevelWarn
	}

	// Check for system events
	if isSystemEvent(event) {
		return LogLevelInfo
	}

	// Default to debug for other events
	return LogLevelDebug
}

// isErrorEvent checks if an event represents an error
func isErrorEvent(event events.CoreEvent) bool {
	eventType := event.GetType()
	return containsAny(eventType, []string{"error", "failed", "failure", "panic", "fatal"})
}

// isWarningEvent checks if an event represents a warning
func isWarningEvent(event events.CoreEvent) bool {
	eventType := event.GetType()
	return containsAny(eventType, []string{"warning", "warn", "deprecated", "timeout"})
}

// isSystemEvent checks if an event is a system-level event
func isSystemEvent(event events.CoreEvent) bool {
	eventType := event.GetType()
	return containsAny(eventType, []string{"start", "stop", "ready", "health", "config"})
}

// containsAny checks if a string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// contains is a simple case-insensitive substring check
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))
}

// findSubstring performs a simple substring search
func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetMetrics returns bridge metrics
func (b *EventLoggerBridge) GetMetrics() EventLoggerMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return EventLoggerMetrics{
		EventsLogged:   b.eventsLogged,
		EventsFiltered: b.eventsFiltered,
		Errors:         b.errors,
		Running:        b.started,
	}
}

// EventLoggerMetrics contains bridge metrics
type EventLoggerMetrics struct {
	EventsLogged   uint64
	EventsFiltered uint64
	Errors         uint64
	Running        bool
}
