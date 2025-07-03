// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

/*
Package events provides a unified event system for Guild Framework components.

This package resolves the EventBus conflicts that existed across multiple Guild components
by providing a single, consistent event interface and implementation.

# Core Concepts

The events package defines several key interfaces and types:

- CoreEvent: The base interface that all Guild events must implement
- EventBus: The unified interface for publishing and subscribing to events
- BaseEvent: The standard implementation of CoreEvent

# Specialized Event Types

The package provides specialized event types for different Guild components:

- TaskEvent: For kanban and task management operations
- AgentEvent: For agent lifecycle and coordination events  
- SystemEvent: For system-level infrastructure events
- CommissionEvent: For commission workflow events
- UIEvent: For user interface interactions
- PerformanceEvent: For performance monitoring and alerting
- MemoryEvent: For memory/corpus operations

# Event Bus Implementation

The package includes MemoryEventBus, a production-ready in-memory implementation
with the following features:

- Concurrent event processing with configurable buffer sizes
- Subscription filtering and event routing
- Metrics collection and performance monitoring
- Graceful shutdown and error handling
- Context-aware operations for cancellation and tracing

# Backward Compatibility

The package provides conversion utilities to maintain compatibility with
existing event types from:

- pkg/orchestrator/interfaces (orchestrator events)
- pkg/kanban (kanban board events)
- pkg/corpus/retrieval (corpus events)
- pkg/integration (integration events)
- gRPC protobuf events

# Usage Examples

Basic event publishing and subscription:

	bus := NewMemoryEventBusWithDefaults()
	defer bus.Close(context.Background())
	
	// Subscribe to task events
	subID, err := bus.Subscribe(ctx, EventTypeTaskCreated, func(ctx context.Context, event CoreEvent) error {
		taskEvent := event.(*TaskEvent)
		fmt.Printf("Task created: %s\n", taskEvent.TaskID)
		return nil
	})
	
	// Publish a task created event
	event := NewTaskEvent(EventTypeTaskCreated, "task-123", map[string]interface{}{
		"title": "Implement feature X",
		"priority": "high",
	})
	
	err = bus.Publish(ctx, event)

Creating custom events with the builder pattern:

	event := NewEventBuilder(EventTypeSystemError, "my-component").
		WithTarget("error-handler").
		WithData("error_code", "E001").
		WithData("message", "Something went wrong").
		WithMetadata("severity", "high").
		Build()

Converting legacy events:

	// From kanban BoardEvent
	taskEvent := FromKanbanEvent("board-1", "task-123", "task.created", map[string]string{
		"title": "New task",
	})
	
	// From JSON
	event, err := FromJSON(`{"type": "system.startup", "source": "api", ...}`)

# Performance Considerations

The MemoryEventBus is optimized for high-throughput scenarios:

- Events are processed asynchronously in dedicated goroutines
- Subscription matching uses efficient map lookups
- Event delivery includes panic recovery to prevent cascading failures
- Configurable buffer sizes prevent memory exhaustion
- Metrics collection has minimal overhead

# Thread Safety

All operations in this package are thread-safe. The MemoryEventBus uses
appropriate synchronization primitives to ensure concurrent access is safe.

# Error Handling

The package uses the Guild's gerror framework for consistent error handling
and provides specific error types for different failure scenarios.
*/
package events