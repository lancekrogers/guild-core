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

5. **Updated EventService** (`pkg/grpc/event_service_unified.go`)
   - Created unified version that works with pkg/events.EventBus
   - Server automatically uses unified version when available
   - Maintains backward compatibility with legacy event bus

6. **Updated Kanban Manager to use interface**
   - Changed from concrete TaskEventPublisher to TaskEventPublisherInterface
   - Allows injection of either legacy or unified publisher
   - Updated Board to also use the interface

7. **Created UnifiedCampaignManager** (`pkg/campaign/unified_manager.go`)
   - Full implementation using unified event system
   - Maintains backward compatibility with EventHandler interface
   - Server automatically uses unified version when available

8. **Created UnifiedCommissionTaskPlanner** (`pkg/orchestrator/commission_planner_unified.go`)
   - Migrated commission planner to use unified event system
   - Publishes task creation and assignment events to unified bus
   - CommissionIntegrationService automatically uses unified version when available

9. **Created UnifiedAccessController** (`pkg/security/access/controller_unified.go`)
   - Migrated security controller to use unified event system
   - Publishes access denied and security alert events to unified bus
   - Created UnifiedEventBusAdapter for backward compatibility

10. **Created UnifiedRetriever** (`pkg/corpus/retrieval/retriever_unified.go`)
    - Migrated corpus retriever to use unified event system
    - Publishes retrieval completion events with enhanced metadata
    - Created UnifiedEventBusAdapter for corpus EventBus interface

## Remaining Work

### Phase 1: Update Core Components (High Priority)

All Phase 1 high priority components have been migrated!

### Phase 2: Security and Integration Components

1. **Security Controller** (`pkg/security/access/controller.go`) - COMPLETED ✓
   - Created UnifiedAccessController that uses unified event system
   - Publishes audit and security events

2. **Corpus Retriever** (`pkg/corpus/retrieval/retriever.go`) - COMPLETED ✓
   - Created UnifiedRetriever that uses unified event system
   - Publishes enhanced retrieval events with context and metrics

3. **Update Integration Layer** (`pkg/integration/eventbus_integration.go`)
   - Complex adapter between multiple event systems
   - Requires significant refactoring due to different handler signatures
   - Should be done as a separate effort

### Phase 3: Cleanup (Medium Priority) - IN PROGRESS

1. **Enforce unified event bus usage** (COMPLETED ✓)
   - Updated server.go to panic if unified event bus not provided
   - Removed legacy event service (`pkg/grpc/event_service.go`)
   - Updated getCampaignManager to always use unified manager
   - Server now requires EventBusAdapter wrapping unified event bus

2. **Remove orchestrator.EventBus** (Not started - requires major refactoring)
   - Would need to update all EventHandler signatures to accept context
   - Would need to update all Publish calls to handle errors
   - Would need to update Subscribe method signatures
   - Affects: orchestrator factories, integration layer, tests

3. **Remove backward compatibility** (Partially complete)
   - Still need to remove `pkg/events/converters.go` (event type converters)
   - Still need to remove legacy event type aliases
   - EventBusAdapter still needed for interface compatibility

4. **Delete integration adapter**
   - Remove `pkg/integration/eventbus_integration.go`
   - Complex multi-adapter system that needs careful migration

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

## Migration Summary

### Completed Components ✓
1. **EventService** - Using unified event bus via adapter
2. **Kanban Manager** - Using UnifiedTaskEventPublisher
3. **Campaign Manager** - UnifiedCampaignManager implementation
4. **Commission Planner** - UnifiedCommissionTaskPlanner implementation
5. **Security Controller** - UnifiedAccessController implementation
6. **Corpus Retriever** - UnifiedRetriever implementation

### Active Components Using Dual Systems
- **serve command** - Creates unified event bus and wraps with adapter
- **gRPC server** - Automatically detects and uses unified components

### Remaining Legacy Usage
- **Integration Layer** - Complex multi-adapter system
- **Legacy Campaign Manager** - Still used when unified bus not available
- **Tests** - Many tests still use legacy event bus

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