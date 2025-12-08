// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/guild-framework/guild-core/pkg/events"
	"github.com/guild-framework/guild-core/pkg/observability"
)

// EventBusAdapter adapts the pkg/events.EventBus to the simple grpc.EventBus interface
type EventBusAdapter struct {
	eventBus events.EventBus
	handlers map[string][]func(event interface{})
	mu       sync.RWMutex
}

// NewEventBusAdapter creates a new adapter
func NewEventBusAdapter(eventBus events.EventBus) EventBus {
	adapter := &EventBusAdapter{
		eventBus: eventBus,
		handlers: make(map[string][]func(event interface{})),
	}

	// Subscribe to all events from the real event bus
	ctx := context.Background()
	adapter.eventBus.SubscribeAll(ctx, func(ctx context.Context, event events.CoreEvent) error {
		// Convert CoreEvent to the simple interface{} format expected by grpc handlers
		adapter.deliverToHandlers(event.GetType(), event)
		return nil
	})

	return adapter
}

// UnifiedEventBus returns the underlying unified event bus
func (a *EventBusAdapter) UnifiedEventBus() events.EventBus {
	return a.eventBus
}

// Publish implements the simple EventBus interface
func (a *EventBusAdapter) Publish(event interface{}) {
	ctx := context.Background()

	// Try to convert the event to a CoreEvent
	var coreEvent events.CoreEvent

	switch e := event.(type) {
	case events.CoreEvent:
		coreEvent = e
	case map[string]interface{}:
		// Create a generic event from map data
		eventType, _ := e["type"].(string)
		if eventType == "" {
			eventType = "generic.event"
		}
		coreEvent = events.NewBaseEvent(uuid.New().String(), eventType, "grpc", e)
	case string:
		// Try to parse as JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(e), &data); err == nil {
			eventType, _ := data["type"].(string)
			if eventType == "" {
				eventType = "generic.event"
			}
			coreEvent = events.NewBaseEvent(uuid.New().String(), eventType, "grpc", data)
		} else {
			// Create a simple string event
			coreEvent = events.NewBaseEvent(uuid.New().String(), "generic.message", "grpc", map[string]interface{}{
				"message": e,
			})
		}
	default:
		// Wrap any other type
		coreEvent = events.NewBaseEvent(uuid.New().String(), fmt.Sprintf("generic.%T", event), "grpc", map[string]interface{}{
			"data": event,
		})
	}

	// Publish to the real event bus
	if err := a.eventBus.Publish(ctx, coreEvent); err != nil {
		// Log error but don't panic - maintain compatibility with simple interface
		logger := observability.GetLogger(ctx)
		logger.ErrorContext(ctx, "failed to publish event to unified event bus", "error", err)
	}
}

// Subscribe implements the simple EventBus interface
func (a *EventBusAdapter) Subscribe(eventType string, handler func(event interface{})) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.handlers[eventType] = append(a.handlers[eventType], handler)

	// Also subscribe to the real event bus for this specific type
	ctx := context.Background()
	a.eventBus.Subscribe(ctx, eventType, func(ctx context.Context, event events.CoreEvent) error {
		// Deliver to the simple handler
		handler(event)
		return nil
	})
}

// deliverToHandlers delivers events to subscribed handlers
func (a *EventBusAdapter) deliverToHandlers(eventType string, event interface{}) {
	a.mu.RLock()
	handlers := append([]func(event interface{}){}, a.handlers[eventType]...)
	allHandlers := append([]func(event interface{}){}, a.handlers["*"]...)
	a.mu.RUnlock()

	// Deliver to specific type handlers
	for _, handler := range handlers {
		handler(event)
	}

	// Deliver to wildcard handlers
	for _, handler := range allHandlers {
		handler(event)
	}
}
