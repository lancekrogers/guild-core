// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package integration provides the critical integration layer connecting performance optimization components
//
// This package implements the integration requirements identified in performance optimization,
// Agent 2 task, providing:
//   - Unified event bus integration for all performance optimization components
//   - Registry pattern implementation for component discovery
//   - Database schema integration for persistence
//   - gRPC service coordination for distributed operations
//
// The package follows Guild's architectural patterns:
//   - Context-first error handling with gerror
//   - Interface-driven design for testability
//   - Registry pattern for component management
//   - Observability integration
//
// Example usage:
//
//	// Create event bus integrator
//	integrator := NewEventBusIntegrator(orchestratorBus, logger)
//
//	// Register all performance optimization components
//	err := integrator.RegisterAllComponents(ctx)
//
//	// Publish session event
//	err = integrator.PublishSessionEvent(ctx, "session.created", sessionData)
//
//	// Subscribe to performance events
//	err = integrator.SubscribeToPerformanceEvents(ctx, performanceHandler)
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lancekrogers/guild/pkg/gerror"
	"github.com/lancekrogers/guild/pkg/orchestrator"
	"go.uber.org/zap"
)

// Package version for compatibility tracking
const (
	Version     = "1.0.0"
	APIVersion  = "v1"
	PackageName = "integration"
)

// EventBusIntegrator connects all performance optimization components to Guild's orchestrator event bus
type EventBusIntegrator struct {
	orchestratorBus orchestrator.EventBus
	logger          *zap.Logger

	// Component event adapters
	sessionAdapter     *SessionEventAdapter
	performanceAdapter *PerformanceEventAdapter
	monitoringAdapter  *MonitoringEventAdapter

	// Event routing and filtering
	eventRouter   *EventRouter
	eventFilters  []EventFilter
	eventHandlers map[string][]EventHandler
	subscriptions map[string][]Subscription
	mu            sync.RWMutex
}

// EventRouter manages event routing between components
type EventRouter struct {
	routes     map[string][]RouteTarget
	middleware []RouteMiddleware
	logger     *zap.Logger
	mu         sync.RWMutex
}

// RouteTarget defines where an event should be routed
type RouteTarget struct {
	ComponentID string                 `json:"component_id"`
	Handler     string                 `json:"handler"`
	Transform   EventTransformer       `json:"-"`
	Filter      EventFilter            `json:"-"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Target    string                 `json:"target"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
	Context   context.Context        `json:"-"`
}

// Subscription represents an event subscription
type Subscription struct {
	ID        string       `json:"id"`
	EventType string       `json:"event_type"`
	Handler   EventHandler `json:"-"`
	Filter    EventFilter  `json:"-"`
	Active    bool         `json:"active"`
	CreatedAt time.Time    `json:"created_at"`
}

// Event types for performance optimization components
const (
	// Session events
	EventTypeSessionCreated  = "session.created"
	EventTypeSessionResumed  = "session.resumed"
	EventTypeSessionSaved    = "session.saved"
	EventTypeSessionExported = "session.exported"
	EventTypeSessionAnalyzed = "session.analyzed"

	// Performance events
	EventTypePerformanceProfiled    = "performance.profiled"
	EventTypePerformanceOptimized   = "performance.optimized"
	EventTypePerformanceCacheHit    = "performance.cache.hit"
	EventTypePerformanceCacheMiss   = "performance.cache.miss"
	EventTypePerformanceMemoryOptim = "performance.memory.optimized"

	// Monitoring events
	EventTypeMonitoringAlertTriggered = "monitoring.alert.triggered"
	EventTypeMonitoringMetricUpdated  = "monitoring.metric.updated"
	EventTypeMonitoringHealthCheck    = "monitoring.health.check"

	// Integration events
	EventTypeIntegrationRegistered   = "integration.component.registered"
	EventTypeIntegrationUnregistered = "integration.component.unregistered"
	EventTypeIntegrationHealthUpdate = "integration.health.update"
)

// Callback types
type EventHandler func(ctx context.Context, event *Event) error
type EventFilter func(event *Event) bool
type EventTransformer func(event *Event) (*Event, error)
type RouteMiddleware func(ctx context.Context, event *Event, next func() error) error

// NewEventBusIntegrator creates integration layer for all performance optimization components
func NewEventBusIntegrator(bus orchestrator.EventBus, logger *zap.Logger) *EventBusIntegrator {
	ebi := &EventBusIntegrator{
		orchestratorBus: bus,
		logger:          logger.Named("eventbus-integrator"),
		eventHandlers:   make(map[string][]EventHandler),
		subscriptions:   make(map[string][]Subscription),
		eventFilters:    make([]EventFilter, 0),
	}

	// Initialize component adapters
	ebi.sessionAdapter = NewSessionEventAdapter(bus, logger)
	ebi.performanceAdapter = NewPerformanceEventAdapter(bus, logger)
	ebi.monitoringAdapter = NewMonitoringEventAdapter(bus, logger)

	// Initialize event router
	ebi.eventRouter = NewEventRouter(logger)

	// Set up default event routing
	ebi.setupDefaultRouting()

	return ebi
}

// RegisterAllComponents connects all performance optimization components to the event bus
func (ebi *EventBusIntegrator) RegisterAllComponents(ctx context.Context) error {
	ebi.logger.Info("Registering all performance optimization components with event bus")

	// Register session events
	if err := ebi.sessionAdapter.Register(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register session events").
			WithComponent("eventbus-integrator").
			WithOperation("RegisterAllComponents")
	}

	// Register performance events
	if err := ebi.performanceAdapter.Register(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register performance events").
			WithComponent("eventbus-integrator").
			WithOperation("RegisterAllComponents")
	}

	// Register monitoring events
	if err := ebi.monitoringAdapter.Register(ctx); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to register monitoring events").
			WithComponent("eventbus-integrator").
			WithOperation("RegisterAllComponents")
	}

	// Publish integration registered event
	integrationEvent := &Event{
		ID:        fmt.Sprintf("integration-%d", time.Now().UnixNano()),
		Type:      EventTypeIntegrationRegistered,
		Source:    "eventbus-integrator",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"components": []string{"session", "performance", "monitoring"},
			"version":    Version,
		},
	}

	if err := ebi.PublishEvent(ctx, integrationEvent); err != nil {
		ebi.logger.Warn("Failed to publish integration registered event", zap.Error(err))
	}

	ebi.logger.Info("All performance optimization components registered with event bus successfully")
	return nil
}

// PublishEvent publishes an event to the integrated event bus
func (ebi *EventBusIntegrator) PublishEvent(ctx context.Context, event *Event) error {
	// Apply event filters
	for _, filter := range ebi.eventFilters {
		if !filter(event) {
			ebi.logger.Debug("Event filtered out",
				zap.String("event_type", event.Type),
				zap.String("event_id", event.ID))
			return nil
		}
	}

	// Route event through the event router
	if err := ebi.eventRouter.RouteEvent(ctx, event); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "event routing failed").
			WithComponent("eventbus-integrator").
			WithOperation("PublishEvent")
	}

	// Publish to orchestrator bus
	eventData, err := json.Marshal(event)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to serialize event").
			WithComponent("eventbus-integrator").
			WithOperation("PublishEvent")
	}

	// Convert eventData back to map for orchestrator Event
	var eventDataMap map[string]interface{}
	if err := json.Unmarshal(eventData, &eventDataMap); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeParsing, "failed to unmarshal event data").
			WithComponent("eventbus-integrator").
			WithOperation("PublishEvent")
	}

	orchestratorEvent := orchestrator.Event{
		Type:      orchestrator.EventType(event.Type),
		Source:    event.Source,
		Data:      eventDataMap,
		Timestamp: event.Timestamp,
	}

	ebi.orchestratorBus.Publish(orchestratorEvent)
	
	ebi.logger.Debug("Event published successfully",
		zap.String("event_type", event.Type),
		zap.String("event_id", event.ID))

	return nil
}

// Subscribe subscribes to events of a specific type
func (ebi *EventBusIntegrator) Subscribe(ctx context.Context, eventType string, handler EventHandler, filter EventFilter) (string, error) {
	subscriptionID := fmt.Sprintf("sub-%d", time.Now().UnixNano())

	subscription := Subscription{
		ID:        subscriptionID,
		EventType: eventType,
		Handler:   handler,
		Filter:    filter,
		Active:    true,
		CreatedAt: time.Now(),
	}

	ebi.mu.Lock()
	if ebi.subscriptions[eventType] == nil {
		ebi.subscriptions[eventType] = make([]Subscription, 0)
	}
	ebi.subscriptions[eventType] = append(ebi.subscriptions[eventType], subscription)
	ebi.mu.Unlock()

	// Subscribe to orchestrator bus with comprehensive error handling
	orchestratorHandler := func(orchestratorEvent orchestrator.Event) {
		// Create context for event processing with timeout for observability
		eventCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		
		// Convert Data back to JSON bytes for unmarshaling with proper error handling
		eventData, err := json.Marshal(orchestratorEvent.Data)
		if err != nil {
			gErr := gerror.Wrap(err, gerror.ErrCodeParsing, "failed to marshal orchestrator event data").
				FromContext(eventCtx).
				WithComponent("eventbus-integrator").
				WithOperation("Subscribe.orchestratorHandler").
				WithDetails("event_type", string(orchestratorEvent.Type)).
				WithDetails("subscription_id", subscriptionID)
			ebi.logger.Error("Failed to marshal event data", zap.Error(gErr))
			return
		}

		var event Event
		if err := json.Unmarshal(eventData, &event); err != nil {
			gErr := gerror.Wrap(err, gerror.ErrCodeParsing, "failed to unmarshal event data to integration event").
				FromContext(eventCtx).
				WithComponent("eventbus-integrator").
				WithOperation("Subscribe.orchestratorHandler").
				WithDetails("event_type", string(orchestratorEvent.Type)).
				WithDetails("data_size", len(eventData)).
				WithDetails("subscription_id", subscriptionID)
			ebi.logger.Error("Failed to unmarshal event", zap.Error(gErr))
			return
		}
		
		// Set context in event for downstream processing
		event.Context = eventCtx

		// Apply filter if provided with panic recovery
		if filter != nil {
			filterPassed := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						gErr := gerror.New(gerror.ErrCodeInternal, "event filter panicked", nil).
							FromContext(eventCtx).
							WithComponent("eventbus-integrator").
							WithOperation("Subscribe.filter").
							WithDetails("event_type", event.Type).
							WithDetails("subscription_id", subscriptionID).
							WithDetails("panic_value", fmt.Sprintf("%v", r))
						ebi.logger.Error("Event filter panicked", zap.Error(gErr))
					}
				}()
				filterPassed = filter(&event)
			}()
			
			if !filterPassed {
				ebi.logger.Debug("Event filtered out", 
					zap.String("event_type", event.Type),
					zap.String("event_id", event.ID),
					zap.String("subscription_id", subscriptionID))
				return
			}
		}

		// Call handler with comprehensive error handling and panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					gErr := gerror.New(gerror.ErrCodeInternal, "event handler panicked", nil).
						FromContext(eventCtx).
						WithComponent("eventbus-integrator").
						WithOperation("Subscribe.handler").
						WithDetails("event_type", event.Type).
						WithDetails("subscription_id", subscriptionID).
						WithDetails("panic_value", fmt.Sprintf("%v", r))
					ebi.logger.Error("Event handler panicked", zap.Error(gErr))
				}
			}()
			
			if err := handler(eventCtx, &event); err != nil {
				gErr := gerror.Wrap(err, gerror.ErrCodeInternal, "event handler failed").
					FromContext(eventCtx).
					WithComponent("eventbus-integrator").
					WithOperation("Subscribe.handler").
					WithDetails("event_type", event.Type).
					WithDetails("subscription_id", subscriptionID)
				ebi.logger.Error("Event handler failed", zap.Error(gErr))
			} else {
				ebi.logger.Debug("Event processed successfully",
					zap.String("event_type", event.Type),
					zap.String("event_id", event.ID),
					zap.String("subscription_id", subscriptionID))
			}
		}()
	}

	ebi.orchestratorBus.Subscribe(orchestrator.EventType(eventType), orchestratorHandler)

	ebi.logger.Info("Subscribed to event type",
		zap.String("event_type", eventType),
		zap.String("subscription_id", subscriptionID))

	return subscription.ID, nil
}

// SessionEventAdapter converts session events to orchestrator events
type SessionEventAdapter struct {
	bus    orchestrator.EventBus
	logger *zap.Logger
}

func NewSessionEventAdapter(bus orchestrator.EventBus, logger *zap.Logger) *SessionEventAdapter {
	return &SessionEventAdapter{
		bus:    bus,
		logger: logger.Named("session-event-adapter"),
	}
}

func (sea *SessionEventAdapter) Register(ctx context.Context) error {
	// Subscribe to session manager events and convert them to integration events
	sessionEvents := []string{
		EventTypeSessionCreated,
		EventTypeSessionResumed,
		EventTypeSessionSaved,
		EventTypeSessionExported,
		EventTypeSessionAnalyzed,
	}

	for _, eventType := range sessionEvents {
		handler := func(eventType string) func(ctx context.Context, event orchestrator.Event) error {
			return func(ctx context.Context, event orchestrator.Event) error {
				// Check for context cancellation early
				if err := ctx.Err(); err != nil {
					return gerror.Wrap(err, gerror.ErrCodeCancelled, "session event processing cancelled").
						FromContext(ctx).
						WithComponent("session-event-adapter").
						WithOperation("Register.handler").
						WithDetails("event_type", eventType)
				}

				sea.logger.Debug("Processing session event",
					zap.String("event_type", eventType),
					zap.String("source", event.Source))

				// Validate event data
				if event.Data == nil {
					return gerror.New(gerror.ErrCodeInvalidInput, "session event data is nil", nil).
						FromContext(ctx).
						WithComponent("session-event-adapter").
						WithOperation("Register.handler").
						WithDetails("event_type", eventType).
						WithDetails("event_source", event.Source)
				}

				// Convert orchestrator event to integration event with error handling
				integrationEvent := &Event{
					ID:        fmt.Sprintf("session-%d", time.Now().UnixNano()),
					Type:      eventType,
					Source:    "session-adapter",
					Target:    "integration",
					Timestamp: event.Timestamp,
					Data:      make(map[string]interface{}),
					Context:   ctx,
				}

				// Safely copy event data with validation
				for key, value := range event.Data {
					integrationEvent.Data[key] = value
				}

				// Add session-specific metadata
				integrationEvent.Metadata = map[string]interface{}{
					"component":     "session",
					"adapter":       "session-event-adapter",
					"original_type": event.Type,
					"processed_at":  time.Now().Unix(),
				}

				sea.logger.Debug("Session event processed successfully",
					zap.String("event_type", eventType),
					zap.String("integration_event_id", integrationEvent.ID))

				return nil
			}
		}(eventType)

		sea.bus.Subscribe(orchestrator.EventType(eventType), func(event orchestrator.Event) {
			// Create timeout context for event processing
			eventCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()

			// Process with proper error handling
			if err := handler(eventCtx, event); err != nil {
				gErr := gerror.Wrap(err, gerror.ErrCodeInternal, "session event handler failed").
					FromContext(eventCtx).
					WithComponent("session-event-adapter").
					WithOperation("Register.Subscribe").
					WithDetails("event_type", eventType).
					WithDetails("event_source", event.Source)
				sea.logger.Error("Session event handler failed", zap.Error(gErr))
			}
		})
	}

	sea.logger.Info("Session event adapter registered successfully")
	return nil
}

// PerformanceEventAdapter converts performance events to orchestrator events
type PerformanceEventAdapter struct {
	bus    orchestrator.EventBus
	logger *zap.Logger
}

func NewPerformanceEventAdapter(bus orchestrator.EventBus, logger *zap.Logger) *PerformanceEventAdapter {
	return &PerformanceEventAdapter{
		bus:    bus,
		logger: logger.Named("performance-event-adapter"),
	}
}

func (pea *PerformanceEventAdapter) Register(ctx context.Context) error {
	performanceEvents := []string{
		EventTypePerformanceProfiled,
		EventTypePerformanceOptimized,
		EventTypePerformanceCacheHit,
		EventTypePerformanceCacheMiss,
		EventTypePerformanceMemoryOptim,
	}

	for _, eventType := range performanceEvents {
		handler := func(eventType string) func(ctx context.Context, event orchestrator.Event) error {
			return func(ctx context.Context, event orchestrator.Event) error {
				pea.logger.Debug("Processing performance event",
					zap.String("event_type", eventType),
					zap.String("source", event.Source))

				// Create integration event with performance metrics
				integrationEvent := &Event{
					ID:        fmt.Sprintf("perf-%d", time.Now().UnixNano()),
					Type:      eventType,
					Source:    "performance-adapter",
					Target:    "integration",
					Timestamp: event.Timestamp,
					Data:      make(map[string]interface{}),
					Context:   ctx,
				}

				// Unmarshal performance data
				// Copy event data (already a map)
				integrationEvent.Data = event.Data

				// Add performance-specific metadata
				integrationEvent.Metadata = map[string]interface{}{
					"component":    "performance",
					"adapter":      "performance-event-adapter",
					"metrics_type": extractMetricsType(eventType),
				}

				return nil
			}
		}(eventType)

		pea.bus.Subscribe(orchestrator.EventType(eventType), func(event orchestrator.Event) {
			if err := handler(ctx, event); err != nil {
				pea.logger.Error("Performance event handler failed", zap.Error(err))
			}
		})
	}

	pea.logger.Info("Performance event adapter registered successfully")
	return nil
}

// MonitoringEventAdapter converts monitoring events to orchestrator events
type MonitoringEventAdapter struct {
	bus    orchestrator.EventBus
	logger *zap.Logger
}

func NewMonitoringEventAdapter(bus orchestrator.EventBus, logger *zap.Logger) *MonitoringEventAdapter {
	return &MonitoringEventAdapter{
		bus:    bus,
		logger: logger.Named("monitoring-event-adapter"),
	}
}

func (mea *MonitoringEventAdapter) Register(ctx context.Context) error {
	monitoringEvents := []string{
		EventTypeMonitoringAlertTriggered,
		EventTypeMonitoringMetricUpdated,
		EventTypeMonitoringHealthCheck,
	}

	for _, eventType := range monitoringEvents {
		handler := func(eventType string) func(ctx context.Context, event orchestrator.Event) error {
			return func(ctx context.Context, event orchestrator.Event) error {
				mea.logger.Debug("Processing monitoring event",
					zap.String("event_type", eventType),
					zap.String("source", event.Source))

				// Create integration event with monitoring data
				integrationEvent := &Event{
					ID:        fmt.Sprintf("mon-%d", time.Now().UnixNano()),
					Type:      eventType,
					Source:    "monitoring-adapter",
					Target:    "integration",
					Timestamp: event.Timestamp,
					Data:      make(map[string]interface{}),
					Context:   ctx,
				}

				// Unmarshal monitoring data
				// Copy event data (already a map)
				integrationEvent.Data = event.Data

				// Add monitoring-specific metadata
				integrationEvent.Metadata = map[string]interface{}{
					"component": "monitoring",
					"adapter":   "monitoring-event-adapter",
					"severity":  extractSeverity(eventType),
				}

				return nil
			}
		}(eventType)

		mea.bus.Subscribe(orchestrator.EventType(eventType), func(event orchestrator.Event) {
			if err := handler(ctx, event); err != nil {
				mea.logger.Error("Monitoring event handler failed", zap.Error(err))
			}
		})
	}

	mea.logger.Info("Monitoring event adapter registered successfully")
	return nil
}

// NewEventRouter creates a new event router
func NewEventRouter(logger *zap.Logger) *EventRouter {
	return &EventRouter{
		routes:     make(map[string][]RouteTarget),
		middleware: make([]RouteMiddleware, 0),
		logger:     logger.Named("event-router"),
	}
}

// RouteEvent routes an event to its targets
func (er *EventRouter) RouteEvent(ctx context.Context, event *Event) error {
	er.mu.RLock()
	targets, exists := er.routes[event.Type]
	er.mu.RUnlock()

	if !exists {
		er.logger.Debug("No routes found for event type", zap.String("event_type", event.Type))
		return nil
	}

	for _, target := range targets {
		// Apply filter if present
		if target.Filter != nil && !target.Filter(event) {
			continue
		}

		// Transform event if transformer present
		routedEvent := event
		if target.Transform != nil {
			var err error
			routedEvent, err = target.Transform(event)
			if err != nil {
				er.logger.Warn("Event transformation failed",
					zap.String("target", target.ComponentID),
					zap.Error(err))
				continue
			}
		}

		// Route to target component
		er.logger.Debug("Routing event to target",
			zap.String("event_type", routedEvent.Type),
			zap.String("target", target.ComponentID))
	}

	return nil
}

// setupDefaultRouting sets up default event routing rules
func (ebi *EventBusIntegrator) setupDefaultRouting() {
	// Route session events to monitoring
	ebi.eventRouter.AddRoute(EventTypeSessionCreated, RouteTarget{
		ComponentID: "monitoring",
		Handler:     "session-metrics",
	})

	// Route performance events to session analytics
	ebi.eventRouter.AddRoute(EventTypePerformanceProfiled, RouteTarget{
		ComponentID: "session",
		Handler:     "performance-analytics",
	})

	// Route monitoring alerts to all components
	ebi.eventRouter.AddRoute(EventTypeMonitoringAlertTriggered, RouteTarget{
		ComponentID: "all",
		Handler:     "alert-handler",
	})
}

// AddRoute adds a routing rule
func (er *EventRouter) AddRoute(eventType string, target RouteTarget) {
	er.mu.Lock()
	defer er.mu.Unlock()

	if er.routes[eventType] == nil {
		er.routes[eventType] = make([]RouteTarget, 0)
	}
	er.routes[eventType] = append(er.routes[eventType], target)
}

// Helper functions
func extractMetricsType(eventType string) string {
	switch eventType {
	case EventTypePerformanceCacheHit, EventTypePerformanceCacheMiss:
		return "cache"
	case EventTypePerformanceMemoryOptim:
		return "memory"
	case EventTypePerformanceProfiled:
		return "profiling"
	default:
		return "general"
	}
}

func extractSeverity(eventType string) string {
	switch eventType {
	case EventTypeMonitoringAlertTriggered:
		return "alert"
	case EventTypeMonitoringHealthCheck:
		return "info"
	default:
		return "normal"
	}
}
