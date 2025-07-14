# Event Foundation Integration Guide

This document outlines the integration of the new Event Foundation system (Sprint 3) with Guild's existing infrastructure.

## Overview

Sprint 3 successfully delivered a comprehensive event-driven architecture with the following components:

### 1. Enhanced Event Bus (`pkg/eventbus/`)
- **Priority-based event processing** with high, normal, and low priority queues
- **Circuit breaker pattern** for reliability and fault tolerance  
- **Dead letter queue** for handling failed events with retry mechanisms
- **Backpressure handling** to prevent system overload
- **Event persistence** with SQLite storage and replay capabilities
- **Metrics collection** for monitoring and observability

### 2. Event Types & Schema System (`pkg/events/types/`)
- **Type-safe event builders** for different event categories:
  - Task events (task.created, task.completed, task.failed)
  - Agent events (agent.started, agent.stopped, agent.error)
  - System events (system.startup, system.shutdown, system.error)
  - Commission events (commission.created, commission.completed)
  - Memory events (memory.stored, memory.retrieved, memory.indexed)
  - UI events (ui.view_changed, ui.user_input, ui.notification)
- **Schema registry** with versioning and validation
- **Property validators** (email, URL, UUID, semver, percentage, etc.)
- **Multiple serialization formats** (JSON, binary, compressed)

### 3. Advanced Event Routing (`pkg/events/routing/`)
- **Rule-based routing** with support for:
  - Event type matching (exact and pattern)
  - Source filtering
  - Data field conditions (eq, ne, gt, lt, contains)
  - Metadata rules
  - Time window and frequency rules
- **Event transformation** pipeline
- **Priority-based route execution**
- **Comprehensive middleware**:
  - Rate limiting with token bucket algorithm
  - Retry with exponential backoff
  - Caching with TTL
  - Request deduplication
  - Timeout handling
  - Bulkhead pattern for isolation
  - Distributed tracing support

### 4. Event Debugging & Tools (`pkg/events/debug/`)
- **Real-time event inspector** with:
  - Configurable filters and hooks
  - Debug sessions with sampling
  - Event buffers for historical analysis
- **Event tracer** for tracking event flow through components
- **Performance analyzer** with percentile calculations
- **Event replayer** for testing and debugging
- **Event dumper** supporting JSON, CSV, and text formats

## Integration Points

### Current Guild Architecture Integration

#### 1. Orchestrator Integration (`pkg/orchestrator/`)
The event system integrates with the existing orchestrator:

```go
// Example: Orchestrator publishing task events
func (o *Orchestrator) executeTask(ctx context.Context, task *Task) error {
    // Publish task started event
    event := events.NewBaseEvent(
        task.ID,
        "task.started", 
        "orchestrator",
        map[string]interface{}{
            "task_id": task.ID,
            "agent_id": task.AgentID,
            "commission_id": task.CommissionID,
        },
    )
    o.eventBus.Publish(ctx, event)
    
    // Execute task...
    
    // Publish completion/failure event
    return nil
}
```

#### 2. Agent Integration (`pkg/agents/`)
Agents can publish lifecycle and status events:

```go
// Agent publishing status events
func (a *Agent) updateStatus(ctx context.Context, status string) {
    builder := types.NewAgentEventBuilder("agent.status_changed", a.registry)
    event := builder.
        WithAgentID(a.ID).
        WithStatus(status).
        WithCapabilities(a.capabilities).
        Build()
    a.eventBus.Publish(ctx, event)
}
```

#### 3. Memory System Integration (`pkg/memory/`)
Memory operations can be tracked via events:

```go
// Memory system publishing storage events
func (m *MemoryStore) Store(ctx context.Context, content *Content) error {
    // Store content...
    
    builder := types.NewMemoryEventBuilder("memory.stored", m.registry)
    event := builder.
        WithContentID(content.ID).
        WithContentType(content.Type).
        WithSize(len(content.Data)).
        Build()
    m.eventBus.Publish(ctx, event)
    return nil
}
```

### TUI Integration Strategy

#### Phase 1: Event Subscription Setup
```go
// In internal/ui/chat/app.go
type App struct {
    // ... existing fields ...
    
    // Event system
    eventBus    events.EventBus
    inspector   *debug.Inspector
    router      *routing.Router
}

func NewApp(config *Config) (*App, error) {
    // ... existing initialization ...
    
    // Initialize event system
    eventBus := eventbus.NewEnhancedBus(eventbus.DefaultConfig())
    inspector := debug.NewInspector(debug.DefaultInspectorConfig())
    router := routing.NewRouter(eventBus)
    
    app := &App{
        // ... existing fields ...
        eventBus:  eventBus,
        inspector: inspector,
        router:    router,
    }
    
    // Setup event subscriptions
    app.setupEventSubscriptions()
    
    return app, nil
}
```

#### Phase 2: Event Handlers for UI Updates
```go
func (a *App) setupEventSubscriptions() error {
    ctx := context.Background()
    
    // Subscribe to agent status events
    _, err := a.eventBus.Subscribe(ctx, "agent.*", a.handleAgentEvents)
    if err != nil {
        return err
    }
    
    // Subscribe to task events for progress updates
    _, err = a.eventBus.Subscribe(ctx, "task.*", a.handleTaskEvents)
    if err != nil {
        return err
    }
    
    // Subscribe to system events
    _, err = a.eventBus.Subscribe(ctx, "system.*", a.handleSystemEvents)
    if err != nil {
        return err
    }
    
    return nil
}

func (a *App) handleAgentEvents(ctx context.Context, event events.CoreEvent) error {
    switch event.GetType() {
    case "agent.started":
        // Update agent status in UI
        return a.updateAgentStatus(event)
    case "agent.error":
        // Show error notification
        return a.showNotification("Agent Error", event.GetData()["message"])
    }
    return nil
}
```

#### Phase 3: TUI Event Publishing
```go
func (a *App) publishUIEvent(eventType string, data map[string]interface{}) {
    builder := types.NewUIEventBuilder(eventType, a.registry)
    event := builder.
        WithSessionID(a.currentSession.ID).
        WithUserID(a.currentUser.ID).
        WithData(data).
        Build()
        
    a.eventBus.Publish(context.Background(), event)
}

// Example: User input events
func (a *App) handleUserInput(input string) {
    a.publishUIEvent("ui.user_input", map[string]interface{}{
        "input": input,
        "timestamp": time.Now(),
        "view": a.currentView,
    })
}
```

#### Phase 4: Real-time Debugging Integration
```go
func (a *App) enableEventDebugging() error {
    // Create debug session for UI events
    filters := []debug.EventFilter{
        &debug.EventTypeFilter{Types: []string{"ui.*", "agent.*", "task.*"}},
    }
    
    hooks := []debug.EventHook{
        &debug.LoggingHook{},
        &debug.MetricsHook{},
    }
    
    session, err := a.inspector.CreateSession("TUI Debug", filters, hooks)
    if err != nil {
        return err
    }
    
    // Start inspection
    return a.inspector.Start(context.Background())
}
```

## Performance Considerations

### Event Volume Management
- **Sampling**: Use sampling rates for high-frequency events (UI interactions)
- **Batching**: Batch low-priority events to reduce overhead
- **Filtering**: Apply filters early to reduce processing load

### Resource Usage
- **Circuit Breakers**: Protect against cascade failures
- **Bulkhead Pattern**: Isolate critical vs non-critical event processing
- **Rate Limiting**: Prevent event storms from overwhelming the system

### Memory Management
- **Event TTL**: Configure appropriate time-to-live for stored events
- **Buffer Limits**: Set reasonable limits for event buffers
- **Dead Letter Queue**: Monitor and clear old failed events

## Monitoring and Observability

### Key Metrics to Track
```go
type EventMetrics struct {
    EventsPublished   int64
    EventsProcessed   int64
    EventsFailed      int64
    ProcessingLatency time.Duration
    QueueDepth        int
    CircuitBreakerTrips int64
    DeadLetterEvents  int64
}
```

### Debugging Capabilities
- **Event Inspector**: Real-time event monitoring with filters
- **Event Tracer**: Track events through the entire system
- **Performance Analyzer**: Identify bottlenecks and optimization opportunities
- **Event Replayer**: Reproduce issues for debugging

## Testing Strategy

### Unit Tests
All packages have comprehensive unit tests covering:
- Happy path scenarios
- Error conditions
- Edge cases
- Performance characteristics

### Integration Tests
- Event flow between components
- Circuit breaker behavior under load
- Dead letter queue functionality
- UI event handling

### Performance Tests
- Event throughput under load
- Memory usage patterns
- Latency measurements
- Resource utilization

## Next Steps

### Immediate (Next Sprint)
1. **Complete TUI Integration**: Implement the phases outlined above
2. **Performance Tuning**: Optimize based on real-world usage patterns
3. **Monitoring Setup**: Implement comprehensive metrics collection

### Medium-term
1. **Event Store Optimization**: Consider more advanced storage options
2. **Distributed Events**: Support for multi-node deployments
3. **Advanced Analytics**: Real-time event analytics and insights

### Long-term
1. **Event Sourcing**: Full event sourcing implementation
2. **CQRS Integration**: Command Query Responsibility Segregation
3. **External Integrations**: Webhooks, external event systems

## Conclusion

Sprint 3 has successfully delivered a robust, scalable event foundation that provides:

- **Reliability**: Circuit breakers, dead letter queues, retry mechanisms
- **Performance**: Priority queues, batching, efficient routing
- **Observability**: Comprehensive debugging and monitoring tools
- **Flexibility**: Extensible middleware, configurable routing rules
- **Type Safety**: Schema validation, type-safe event builders

The event system is now ready for integration with the TUI and other Guild components, providing a solid foundation for building a responsive, observable, and maintainable system.