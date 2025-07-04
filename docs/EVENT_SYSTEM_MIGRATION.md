# Event System Migration Guide

## Overview

This document tracks the migration from the legacy orchestrator.EventBus to the unified pkg/events system.

## Current State (Completed)

1. **Created unified event system** (`pkg/events/`)
   - Context-aware operations
   - Proper error handling
   - Metrics and observability built-in
   - Type-safe event interfaces

2. **Created EventBusAdapter** (`pkg/grpc/eventbus_adapter.go`)
   - Bridges the simple grpc.EventBus interface to pkg/events.EventBus
   - Allows legacy components to publish events that flow to the unified system

3. **Updated serve command** (`cmd/guild/serve.go`)
   - Creates unified MemoryEventBus
   - Wraps it with adapter for grpc compatibility
   - Properly shuts down event bus on exit

4. **Created UnifiedTaskEventPublisher** (`pkg/kanban/task_event_publisher_unified.go`)
   - Publishes kanban events only to the unified event system
   - Eliminates triple-publishing to different event systems

## Remaining Work

### Phase 1: Update Core Components (High Priority)

1. **Update EventService** (`pkg/grpc/event_service.go`)
   - Currently uses orchestrator.EventBus directly
   - Should subscribe to unified event bus and convert events for gRPC streaming

2. **Update Kanban Manager initialization**
   - Modify components that create kanban.Manager to inject UnifiedTaskEventPublisher
   - Update Board to use TaskEventPublisherInterface instead of concrete type

3. **Update Campaign Manager** (`pkg/campaign/manager.go`)
   - Currently uses orchestrator.EventBus
   - Should use unified events.EventBus

4. **Update Commission Planner** (`pkg/orchestrator/commission_planner.go`)
   - Publishes events to orchestrator.EventBus
   - Should use unified system

### Phase 2: Security and Integration Components

1. **Update Security Controller** (`pkg/security/access/controller.go`)
   - Uses EventBus for audit events
   - Critical for security logging

2. **Update Corpus Retriever** (`pkg/corpus/retrieval/retriever.go`)
   - Optional event publishing
   - Low priority but should be consistent

3. **Update Integration Layer** (`pkg/integration/eventbus_integration.go`)
   - Complex adapter between multiple event systems
   - Can be simplified once everything uses unified system

### Phase 3: Cleanup (Medium Priority)

1. **Remove orchestrator.EventBus**
   - Delete `pkg/orchestrator/eventbus.go`
   - Remove EventBus interface from `pkg/orchestrator/interfaces.go`
   - Update all imports

2. **Remove backward compatibility**
   - Delete `pkg/events/converters.go` (event type converters)
   - Remove legacy event type aliases

3. **Delete integration adapter**
   - Remove `pkg/integration/eventbus_integration.go`
   - No longer needed with unified system

## Migration Strategy

### For Each Component:

1. **Add unified EventBus dependency**
   ```go
   type Component struct {
       eventBus events.EventBus // Add this
       // ... existing fields
   }
   ```

2. **Update constructor**
   ```go
   func NewComponent(eventBus events.EventBus, ...) *Component {
       return &Component{
           eventBus: eventBus,
           // ... other fields
       }
   }
   ```

3. **Convert event publishing**
   ```go
   // Old:
   event := interfaces.Event{...}
   c.eventBus.Publish(event)
   
   // New:
   event := events.NewBaseEvent(uuid.New().String(), eventType, source, data)
   err := c.eventBus.Publish(ctx, event)
   ```

4. **Convert event subscribing**
   ```go
   // Old:
   c.eventBus.Subscribe(eventType, handler)
   
   // New:
   subID, err := c.eventBus.Subscribe(ctx, eventType, func(ctx context.Context, event events.CoreEvent) error {
       // Handle event
       return nil
   })
   ```

## Benefits of Migration

1. **Context propagation** - Proper cancellation and tracing
2. **Error handling** - All operations return errors
3. **Type safety** - CoreEvent interface with specialized types
4. **Metrics** - Built-in observability
5. **Simplification** - One event system instead of three
6. **Performance** - Better buffering and concurrent processing

## Testing Strategy

1. **Unit tests** - Mock the events.EventBus interface
2. **Integration tests** - Use MemoryEventBus for testing
3. **Event flow tests** - Verify events flow correctly between components
4. **Performance tests** - Ensure no regression in throughput

## Notes

- The grpc.EventBus interface is intentionally simple for backward compatibility
- The EventBusAdapter allows gradual migration
- Components can be migrated independently
- No breaking changes to external APIs