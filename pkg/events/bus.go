// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package events

import (
	"context"
)

// SubscriptionID uniquely identifies an event subscription
type SubscriptionID string

// EventHandler processes events with context for cancellation and tracing
type EventHandler func(ctx context.Context, event CoreEvent) error

// EventBus provides a unified interface for event publishing and subscription
type EventBus interface {
	// Core operations with context support
	Publish(ctx context.Context, event CoreEvent) error
	Subscribe(ctx context.Context, eventType string, handler EventHandler) (SubscriptionID, error)
	Unsubscribe(ctx context.Context, subscriptionID SubscriptionID) error
	
	// Convenience methods
	SubscribeAll(ctx context.Context, handler EventHandler) (SubscriptionID, error)
	PublishJSON(ctx context.Context, jsonEvent string) error
	
	// Lifecycle management
	Close(ctx context.Context) error
	
	// Status and monitoring
	IsRunning() bool
	GetSubscriptionCount() int
}

// EventFilter allows filtering events based on criteria
type EventFilter func(event CoreEvent) bool

// FilteredSubscription represents a subscription with an optional filter
type FilteredSubscription struct {
	ID          SubscriptionID
	EventType   string // empty string means subscribe to all events
	Handler     EventHandler
	Filter      EventFilter // optional filter function
}

// EventBusConfig configures the event bus behavior
type EventBusConfig struct {
	// Buffer size for event channels
	BufferSize int
	
	// Maximum number of subscriptions
	MaxSubscriptions int
	
	// Maximum event payload size in bytes
	MaxEventSize int
	
	// Whether to log all events for debugging
	LogEvents bool
	
	// Whether to enable metrics collection
	EnableMetrics bool
}

// DefaultEventBusConfig returns sensible defaults
func DefaultEventBusConfig() EventBusConfig {
	return EventBusConfig{
		BufferSize:       1000,
		MaxSubscriptions: 100,
		MaxEventSize:     1024 * 1024, // 1MB
		LogEvents:        false,
		EnableMetrics:    true,
	}
}

// EventBusStats provides statistics about event bus operations
type EventBusStats struct {
	EventsPublished   int64
	EventsDelivered   int64
	EventsDropped     int64
	ActiveSubscriptions int
	AverageDeliveryTime float64 // milliseconds
}

// EventBusMetrics provides enhanced metrics for monitoring
type EventBusMetrics interface {
	// Record event publishing
	RecordEventPublished(eventType string)
	RecordEventDelivered(eventType string, deliveryTime float64)
	RecordEventDropped(eventType string, reason string)
	
	// Record subscription operations
	RecordSubscription(eventType string)
	RecordUnsubscription(eventType string)
	
	// Get current statistics
	GetStats() EventBusStats
}

// CommonEventTypes defines standard event types used across Guild components
const (
	// System events
	EventTypeSystemStartup    = "system.startup"
	EventTypeSystemShutdown   = "system.shutdown"
	EventTypeSystemError      = "system.error"
	EventTypeSystemHealthCheck = "system.health_check"
	
	// Task events
	EventTypeTaskCreated    = "task.created"
	EventTypeTaskUpdated    = "task.updated"
	EventTypeTaskStarted    = "task.started"
	EventTypeTaskCompleted  = "task.completed"
	EventTypeTaskFailed     = "task.failed"
	EventTypeTaskCancelled  = "task.cancelled"
	
	// Agent events
	EventTypeAgentStarted   = "agent.started"
	EventTypeAgentStopped   = "agent.stopped"
	EventTypeAgentError     = "agent.error"
	EventTypeAgentIdle      = "agent.idle"
	EventTypeAgentBusy      = "agent.busy"
	
	// Commission events
	EventTypeCommissionCreated   = "commission.created"
	EventTypeCommissionStarted   = "commission.started"
	EventTypeCommissionCompleted = "commission.completed"
	EventTypeCommissionFailed    = "commission.failed"
	
	// UI events
	EventTypeUICommand      = "ui.command"
	EventTypeUIRefresh      = "ui.refresh"
	EventTypeUIError        = "ui.error"
	EventTypeUIThemeChanged = "ui.theme_changed"
	
	// Kanban events
	EventTypeKanbanBoardCreated = "kanban.board_created"
	EventTypeKanbanTaskMoved    = "kanban.task_moved"
	EventTypeKanbanTaskUpdated  = "kanban.task_updated"
	
	// Memory/Corpus events
	EventTypeMemoryStored     = "memory.stored"
	EventTypeMemoryRetrieved  = "memory.retrieved"
	EventTypeCorpusUpdated    = "corpus.updated"
	
	// Performance events
	EventTypePerformanceMetric = "performance.metric"
	EventTypePerformanceAlert  = "performance.alert"
)