# Integration Layer Architecture

## Overview

The Guild integration layer implements a **bridge pattern** to connect disparate system components through a unified service lifecycle and event-driven communication model.

## Design Principles

1. **Separation of Concerns**: Each bridge handles one integration aspect
2. **Loose Coupling**: Components communicate through events, not direct calls
3. **Observable**: All operations emit metrics and support health monitoring
4. **Resilient**: Graceful degradation when components fail
5. **Testable**: Clean interfaces enable comprehensive testing

## Component Architecture

### Service Registry

The central coordinator for all system components.

```
┌─────────────────────────────────────────┐
│            Service Registry             │
├─────────────────────────────────────────┤
│ - Service Registration                  │
│ - Dependency Graph Management           │
│ - Lifecycle Orchestration               │
│ - Health Monitoring                     │
│ - Graceful Shutdown                     │
└─────────────────────────────────────────┘
            │
            ├── Dependency Resolution (Topological Sort)
            ├── Circular Dependency Detection
            ├── Parallel Start/Stop (where possible)
            └── Health Check Coordination
```

**Key Features:**

- Dependency graph with cycle detection
- Ordered startup/shutdown based on dependencies
- Background health monitoring
- Lifecycle hooks for cross-cutting concerns

### Event Bridges

Bridges translate between component-specific protocols and the unified event system.

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Logger    │     │  Database   │     │     UI      │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       ▼                   ▼                   ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│Event Logger │     │ Persistence │     │     UI      │
│   Bridge    │     │Event Bridge │     │Event Bridge │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                           ▼
                   ┌─────────────┐
                   │  Event Bus  │
                   └─────────────┘
```

**Bridge Responsibilities:**

- Protocol translation
- Event filtering and routing
- Batching and buffering
- Metric collection
- Error handling

### Session Management

Provides cross-component session lifecycle management.

```
┌─────────────────────────────────────────┐
│           Session Service               │
├─────────────────────────────────────────┤
│ - Session Creation/Destruction          │
│ - Session Persistence                   │
│ - Automatic Cleanup                     │
│ - Capacity Management                   │
│ - Recovery After Restart                │
└─────────────────────────────────────────┘
            │
            ├── Integrates with Session Manager
            ├── Persists Active Sessions
            ├── Restores on Startup
            └── Enforces Limits
```

## Event Flow Architecture

### Event Publishing

```
Component → Event Creation → Event Bus → Bridges → Consumers
    │            │              │           │          │
    └── Context  └── Metadata   └── Pub/Sub └── Filter └── Process
```

### Event Types and Routing

1. **System Events**: Service lifecycle, health changes
2. **Business Events**: User actions, data changes
3. **UI Events**: User interactions, display updates
4. **Infrastructure Events**: Performance, errors

## Integration Patterns

### 1. Service Integration Pattern

```go
// 1. Define service implementing lifecycle interface
type MyService struct {
    // ... fields
}

func (s *MyService) Start(ctx context.Context) error
func (s *MyService) Stop(ctx context.Context) error
func (s *MyService) Health(ctx context.Context) error
func (s *MyService) Ready(ctx context.Context) error

// 2. Register with service registry
registry.Register(myService)

// 3. Define dependencies
registry.SetDependency("my-service", "required-service")

// 4. Registry handles lifecycle
registry.Start(ctx) // Starts in dependency order
```

### 2. Event Bridge Pattern

```go
// 1. Bridge subscribes to relevant events
bridge.eventBus.SubscribeAll(ctx, bridge.handleEvent)

// 2. Bridge translates events
func (b *Bridge) handleEvent(ctx context.Context, event CoreEvent) error {
    // Transform event to target format
    translated := b.translateEvent(event)
    
    // Forward to target system
    return b.target.Process(translated)
}

// 3. Bridge emits metrics
b.metrics.EventsProcessed++
```

### 3. Health Monitoring Pattern

```go
// 1. Service implements health check
func (s *Service) Health(ctx context.Context) error {
    // Check critical dependencies
    if !s.database.Ping() {
        return gerror.New(gerror.ErrCodeResourceExhausted, "database unavailable", nil)
    }
    return nil
}

// 2. Registry monitors health
go registry.monitorHealth(ctx, service, interval)

// 3. Hooks notified on health changes
for _, hook := range registry.hooks {
    hook.OnHealthCheck(ctx, service, healthy, err)
}
```

## Concurrency Model

### Thread Safety

- All public methods are thread-safe
- Internal state protected by mutexes
- Read-heavy operations use RWMutex

### Goroutine Management

```
Main
 ├── Service Registry
 │    ├── Health Monitor (per service)
 │    └── Readiness Monitor (per service)
 │
 ├── Event Logger Bridge
 │    └── Event Processor
 │
 ├── UI Event Bridge
 │    └── Event Converter
 │
 └── Session Service
      └── Cleanup Timer
```

## Error Handling Strategy

### Error Propagation

```
Component Error
    ↓
gerror.Wrap() with context
    ↓
Bridge/Service Error Handler
    ↓
Emit Error Event
    ↓
Log with Structured Fields
    ↓
Update Health Status
    ↓
Notify Monitoring
```

### Error Recovery

1. **Transient Errors**: Retry with backoff
2. **Resource Errors**: Circuit breaker pattern
3. **Fatal Errors**: Graceful degradation

## Performance Considerations

### Optimization Points

1. **Event Batching**: Reduces system calls
2. **Buffered Channels**: Prevents blocking
3. **Lazy Initialization**: On-demand resource creation
4. **Connection Pooling**: Reuse expensive resources

### Benchmarked Operations

- Service startup: < 100ms
- Event routing: < 1ms
- Health check: < 10ms
- Shutdown: < 30s

## Security Model

### Access Control

- Services run with least privilege
- Event filtering prevents information leakage
- Sensitive data sanitized in events

### Audit Trail

- All service lifecycle events logged
- Event flow tracked with correlation IDs
- Health status changes recorded

## Extensibility

### Adding New Services

1. Implement Service interface
2. Register with ServiceRegistry
3. Define dependencies
4. Add health checks

### Adding New Bridges

1. Implement bridge logic
2. Implement Service interface
3. Define event subscriptions
4. Add metrics collection

### Custom Hooks

```go
type CustomHook struct{}

func (h *CustomHook) OnStart(ctx context.Context, service Service) error
func (h *CustomHook) OnStop(ctx context.Context, service Service) error
func (h *CustomHook) OnError(ctx context.Context, service Service, err error)

registry.AddHook(customHook)
```

## Future Enhancements

1. **Distributed Tracing**: OpenTelemetry integration
2. **Service Mesh**: Multi-node support
3. **Dynamic Configuration**: Hot reload
4. **Advanced Routing**: Content-based routing
5. **Event Sourcing**: Full event persistence
