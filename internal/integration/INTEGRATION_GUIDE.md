# Guild Integration Layer - Developer Guide

## Overview

The Guild integration layer provides a unified framework for connecting all system components through service lifecycle management and event-driven communication. This guide covers architecture, usage patterns, and operational procedures.

## Architecture

### Core Components

```
┌─────────────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Service Registry  │────▶│ Event System │────▶│     Bridges      │
└─────────────────┘         └──────────────┘     └─────────────────┘
         │                          │                      │
         ▼                          ▼                      ▼
┌─────────────────┐         ┌──────────────┐     ┌─────────────────┐
│ Lifecycle Hooks  │         │   Event Bus  │     │  UI ↔ Events    │
└─────────────────┘         └──────────────┘     └─────────────────┘
```

### Key Interfaces

1. **Service Interface** (`services/interfaces.go`)
   - Defines lifecycle: Start, Stop, Health, Ready
   - All components implement this interface

2. **ServiceRegistry** (`services/registry.go`)
   - Manages service lifecycle and dependencies
   - Provides health monitoring and graceful shutdown

3. **Event Bridges** (`bridges/`)
   - EventLoggerBridge: Routes events to logging
   - PersistenceEventBridge: Emits database events
   - UIEventBridge: Connects UI to event system

## Integration Patterns

### 1. Creating a New Service

```go
type MyService struct {
    config MyConfig
    logger observability.Logger
    started bool
    mu sync.RWMutex
}

func (s *MyService) Name() string {
    return "my-service"
}

func (s *MyService) Start(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.started {
        return gerror.New(gerror.ErrCodeAlreadyExists, "already started", nil)
    }
    
    // Initialize service
    // ...
    
    s.started = true
    s.logger.InfoContext(ctx, "Service started")
    return nil
}

func (s *MyService) Stop(ctx context.Context) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if !s.started {
        return gerror.New(gerror.ErrCodeValidation, "not started", nil)
    }
    
    // Cleanup
    // ...
    
    s.started = false
    return nil
}

func (s *MyService) Health(ctx context.Context) error {
    if !s.started {
        return gerror.New(gerror.ErrCodeResourceExhausted, "not started", nil)
    }
    // Check health
    return nil
}

func (s *MyService) Ready(ctx context.Context) error {
    return s.Health(ctx)
}
```

### 2. Registering Services with Dependencies

```go
// Create registry
registry := services.NewServiceRegistry(ctx)

// Register services
registry.Register(databaseService)
registry.Register(cacheService)
registry.Register(apiService)

// Set dependencies
registry.SetDependency("api-service", "database-service")
registry.SetDependency("api-service", "cache-service")

// Start all services in dependency order
if err := registry.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### 3. Event-Driven Integration

```go
// Publish events
event := events.NewBaseEvent(
    "user-123",
    "user.created",
    "user-service",
    map[string]interface{}{
        "email": "user@example.com",
        "role": "member",
    },
)
eventBus.Publish(ctx, event)

// Subscribe to events
handler := func(ctx context.Context, event events.CoreEvent) error {
    if event.GetType() == "user.created" {
        // Handle user creation
    }
    return nil
}
eventBus.SubscribeAll(ctx, handler)
```

### 4. Using Event Bridges

```go
// Create bridges
eventLogger := bridges.NewEventLoggerBridge(eventBus, logger, config)
uiBridge := bridges.NewUIEventBridge(eventBus, logger, uiConfig)

// Register as services
registry.Register(eventLogger)
registry.Register(uiBridge)

// Events will now flow through bridges automatically
```

## Configuration

### Service Registry Configuration

```go
type ServiceOptions struct {
    StartTimeout           time.Duration // Default: 30s
    StopTimeout            time.Duration // Default: 30s
    HealthCheckInterval    time.Duration // Default: 10s
    ReadinessCheckInterval time.Duration // Default: 5s
    MaxRetries             int           // Default: 3
    RetryDelay             time.Duration // Default: 1s
}
```

### Bridge Configurations

#### EventLoggerBridge

```go
type EventLoggerConfig struct {
    LogLevel         EventLogLevel // Minimum event priority to log
    IncludeEventData bool          // Include full event data
    BufferSize       int           // Event channel buffer
    FlushInterval    time.Duration // Batch flush interval
    MaxBatchSize     int           // Maximum batch size
}
```

#### UIEventBridge

```go
type UIEventConfig struct {
    BatchEvents      bool          // Enable event batching
    BatchInterval    time.Duration // Batch interval
    MaxBatchSize     int           // Max events per batch
    UIEventTypes     []string      // Event types to forward to UI
    SystemEventTypes []string      // System events for UI
}
```

## Operational Procedures

### Application Startup

```go
func main() {
    // Create application
    app, err := bootstrap.NewApplication(bootstrap.Options{
        ConfigPath: "guild.yaml",
        LogLevel:   "info",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Run application (blocks until shutdown)
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Health Monitoring

```go
// Check all service health
health := registry.Health(ctx)
for service, err := range health {
    if err != nil {
        log.Printf("Service %s unhealthy: %v", service, err)
    }
}

// Check specific service
service, err := registry.Get("my-service")
if err == nil {
    if err := service.Health(ctx); err != nil {
        log.Printf("Service unhealthy: %v", err)
    }
}
```

### Graceful Shutdown

The application handles shutdown signals automatically:

1. Receives SIGINT/SIGTERM
2. Stops accepting new requests
3. Waits for in-flight operations
4. Stops services in reverse dependency order
5. Persists state if configured
6. Exits cleanly

### Performance Monitoring

```go
// Get bridge metrics
metrics := eventLogger.GetMetrics()
log.Printf("Events logged: %d, Filtered: %d, Errors: %d",
    metrics.EventsLogged,
    metrics.EventsFiltered,
    metrics.Errors)

// Get service info
services := registry.List()
for _, svc := range services {
    log.Printf("Service: %s, State: %s, Healthy: %v",
        svc.Name, svc.State, svc.Healthy)
}
```

## Troubleshooting

### Common Issues

1. **Service fails to start**
   - Check logs for specific error
   - Verify dependencies are satisfied
   - Ensure resources are available

2. **Events not being received**
   - Verify subscription is active
   - Check event type filtering
   - Ensure event bus is started

3. **High memory usage**
   - Check event buffer sizes
   - Monitor goroutine count
   - Review batch sizes

### Debug Mode

Enable debug logging:

```go
logger := observability.NewLogger(&observability.Config{
    Level: observability.LevelDebug,
})
```

### Health Check Failures

Common causes:

- Service not started
- Resource exhaustion
- Dependency failures
- Network issues

## Performance Tuning

### Event Processing

- Increase buffer sizes for high throughput
- Enable batching for better efficiency
- Adjust flush intervals based on latency requirements

### Service Startup

- Parallel startup where possible (no dependencies)
- Increase timeouts for slow services
- Use readiness checks to avoid premature traffic

### Memory Optimization

- Limit concurrent operations
- Use appropriate buffer sizes
- Enable garbage collection tuning

## Testing

### Unit Tests

```bash
go test ./internal/integration/...
```

### Integration Tests

```bash
go test -tags=integration ./internal/integration/tests/...
```

### Benchmarks

```bash
go test -bench=. ./internal/integration/tests/...
```

### Performance Baselines

- Service startup: < 100ms
- Event publish: < 1ms
- Event delivery: < 10ms p99
- Throughput: > 10,000 events/sec

## Best Practices

1. **Always use context** - Pass context through all operations
2. **Handle errors with gerror** - Consistent error handling
3. **Log important events** - Use structured logging
4. **Monitor health** - Implement meaningful health checks
5. **Test shutdown** - Ensure clean shutdown
6. **Document dependencies** - Make integration points clear
7. **Version events** - Plan for schema evolution

## Migration Guide

### From Direct Integration to Service Registry

Before:

```go
// Direct initialization
db := database.New(config)
cache := cache.New(config)
api := api.New(db, cache)
```

After:

```go
// Service registry
registry := services.NewServiceRegistry(ctx)
registry.Register(dbService)
registry.Register(cacheService)
registry.Register(apiService)
registry.SetDependency("api", "db")
registry.SetDependency("api", "cache")
registry.Start(ctx)
```

### From Direct Events to Bridges

Before:

```go
// Manual event handling
event := CreateEvent(...)
logger.Info("Event:", event)
ui.Update(event)
```

After:

```go
// Automatic via bridges
eventBus.Publish(ctx, event)
// Bridges handle logging and UI updates
```

## Further Reading

- [Service Lifecycle Design](services/interfaces.go)
- [Event System Architecture](../../pkg/events/README.md)
- [Bootstrap Process](bootstrap/app.go)
- [Performance Benchmarks](tests/benchmark_test.go)
