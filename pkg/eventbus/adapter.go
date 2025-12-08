// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package eventbus

import (
	"github.com/guild-framework/guild-core/pkg/events"
)

// Event is an alias for the existing CoreEvent interface
type Event = events.CoreEvent

// EventHandler is an alias for the existing EventHandler type
type EventHandler = events.EventHandler

// EventBus is an alias for the existing EventBus interface
type EventBus = events.EventBus

// Subscription represents an event subscription
type Subscription = events.SubscriptionID

// Metrics is an alias for EventBusMetrics
type Metrics = events.EventBusMetrics

// EventAdapter wraps a CoreEvent to provide additional methods needed by eventbus
type EventAdapter struct {
	events.CoreEvent
	correlationID string
	parentID      string
}

// NewEventAdapter creates a new event adapter
func NewEventAdapter(event events.CoreEvent) *EventAdapter {
	adapter := &EventAdapter{
		CoreEvent: event,
	}

	// Extract correlation and parent IDs from metadata if available
	if metadata := event.GetMetadata(); metadata != nil {
		if corrID, ok := metadata["correlation_id"].(string); ok {
			adapter.correlationID = corrID
		}
		if parID, ok := metadata["parent_id"].(string); ok {
			adapter.parentID = parID
		}
	}

	return adapter
}

// GetCorrelationID returns the correlation ID
func (e *EventAdapter) GetCorrelationID() string {
	return e.correlationID
}

// GetParentID returns the parent event ID
func (e *EventAdapter) GetParentID() string {
	return e.parentID
}

// Clone creates a copy of the event
func (e *EventAdapter) Clone() events.CoreEvent {
	// Create a new base event with copied data
	data := make(map[string]interface{})
	for k, v := range e.GetData() {
		data[k] = v
	}

	metadata := make(map[string]interface{})
	for k, v := range e.GetMetadata() {
		metadata[k] = v
	}

	baseEvent := events.NewBaseEvent(e.GetID(), e.GetType(), e.GetSource(), data)
	baseEvent.WithTarget(e.GetTarget())
	for k, v := range metadata {
		baseEvent.WithMetadata(k, v)
	}

	return baseEvent
}
