# Guild Logging Package

A production-ready structured logging system built on Go's `log/slog` with enhanced features for the Guild framework.

## Features

- **Structured Logging**: Built on Go's `log/slog` for consistent, machine-readable logs
- **Context Propagation**: Automatic extraction of request IDs, user IDs, and other contextual information
- **Multiple Output Formats**: JSON, Pretty (colored), and Text formatters
- **Log Rotation**: Size and age-based rotation with compression support
- **Sampling**: Performance-optimized sampling strategies to reduce log volume
- **Security**: PII scrubbing hooks to prevent sensitive data leakage
- **Middleware**: HTTP and gRPC middleware for automatic request logging
- **Hooks**: Extensible hook system for metrics emission and error tracking

## Quick Start

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/lancekrogers/guild/pkg/logging"
)

func main() {
    // Create logger with default configuration
    logger, err := logging.New(logging.DefaultConfig())
    if err != nil {
        panic(err)
    }

    // Basic logging
    logger.Info("Application started")
    
    // Structured logging with fields
    logger.Info("User action", 
        logging.String("user_id", "user-123"),
        logging.String("action", "login"),
        logging.Duration("duration", time.Millisecond*150),
    )
    
    // Context-aware logging
    ctx := logging.WithRequestID(context.Background(), "req-456")
    logger.WithContext(ctx).Info("Processing request")
}
```

## Configuration

### Default Configuration

```go
config := logging.DefaultConfig()
// Level: Info
// Format: JSON
// Output: stdout
// Sampling: Level-based (10% debug, 100% info+)
// Security: PII scrubbing enabled
```

### Development Configuration

```go
config := logging.DevelopmentConfig()
// Level: Debug
// Format: Pretty (colored)
// Output: stdout
// AddSource: true
// Sampling: disabled
// Security: PII scrubbing disabled
```

### Production Configuration

```go
config := logging.ProductionConfig("/var/log/guild/app.log")
// Level: Info
// Format: JSON
// Output: Multi (stdout + file)
// Rotation: 100MB files, 30 days retention, 10 backups
// Sampling: Adaptive (1000 logs/sec)
// Security: PII scrubbing enabled
```

### Custom Configuration

```go
config := logging.NewConfig(
    logging.WithLevel(slog.LevelWarn),
    logging.WithFormat("pretty"),
    logging.WithOutput("file"),
    logging.WithFile("/app/logs/app.log", 50*1024*1024, 7, 3),
    logging.WithSampling(true, "rate"),
    logging.WithDevelopment(false),
)
```

## Context Enrichment

The logging package provides comprehensive context propagation:

```go
ctx := context.Background()

// Enrich context with multiple IDs
enricher := logging.ContextEnricher{
    RequestID:    "req-123",
    UserID:       "user-456", 
    SessionID:    "session-789",
    CommissionID: "comm-001",
    AgentID:      "agent-002",
}
ctx = enricher.Apply(ctx)

// All subsequent logs will include these fields
logger.WithContext(ctx).Info("Processing commission")
```

### Individual Context Functions

```go
ctx = logging.WithRequestID(ctx, "req-123")
ctx = logging.WithUserID(ctx, "user-456")
ctx = logging.WithCommissionID(ctx, "comm-001")
ctx = logging.WithAgentID(ctx, "agent-elena")
```

## Field Constructors

### Common Fields

```go
logger.Info("Request completed",
    logging.String("method", "POST"),
    logging.Int("status", 200),
    logging.Duration("latency", 150*time.Millisecond),
    logging.Bool("cached", true),
)
```

### Guild-Specific Fields

```go
logger.Info("Commission started",
    logging.CommissionIDField("comm-123"),
    logging.AgentIDField("elena"),
    logging.TaskTypeField("code_generation"),
    logging.WorkspaceField("/workspace/project"),
)
```

### Performance Fields

```go
logger.Info("System metrics",
    logging.LatencyField(45*time.Millisecond),
    logging.CPUUsageField(0.75),
    logging.MemoryUsageField(1024*1024*512), // 512MB
    logging.GoroutinesField(42),
)
```

### Error Handling

```go
if err != nil {
    logger.Error("Operation failed",
        logging.ErrorField(err),
        logging.RetryCountField(3),
        logging.RetryableField(true),
    )
}
```

## Sampling

### Level-Based Sampling

```go
config.Sampling = logging.SamplingConfig{
    Enabled:   true,
    Type:      "level",
    DebugRate: 0.1,  // Sample 10% of debug logs
    InfoRate:  1.0,  // Keep all info+ logs
}
```

### Rate-Based Sampling

```go
config.Sampling = logging.SamplingConfig{
    Enabled: true,
    Type:    "rate", 
    Rate:    0.5,    // Sample 50% of logs
}
```

### Adaptive Sampling

```go
config.Sampling = logging.SamplingConfig{
    Enabled:    true,
    Type:       "adaptive",
    TargetRate: 1000,                // Max 1000 logs/sec
    Window:     time.Minute,         // 1-minute windows
}
```

## Security Features

### PII Scrubbing

Automatically detects and redacts sensitive information:

```go
// Automatically scrubs:
// - Credit card numbers
// - Email addresses  
// - Phone numbers
// - API keys/tokens
// - IP addresses
// - AWS keys
// - JWT tokens

logger.Info("Processing payment for user@example.com with card 4111-1111-1111-1111")
// Output: "Processing payment for [REDACTED-EMAIL] with card [REDACTED-CC]"
```

### Custom Redaction

```go
import "github.com/lancekrogers/guild/pkg/logging/hooks"

// Custom redactor showing type
hook := hooks.NewSensitiveDataHook()
hook.SetRedactor(hooks.TypedRedactor)

// Custom patterns
patterns := map[string]float64{
    "SECRET_": 0.0,  // Never log anything with SECRET_
}
patternSampler := hooks.NewMessagePatternSampler(patterns)
```

## Middleware

### HTTP Middleware

```go
import "github.com/lancekrogers/guild/pkg/logging"

mux := http.NewServeMux()
mux.HandleFunc("/api/health", healthHandler)

// Add logging middleware
handler := logging.HTTPMiddleware(logger)(mux)
http.ListenAndServe(":8080", handler)
```

### gRPC Middleware

```go
import (
    "google.golang.org/grpc"
    "github.com/lancekrogers/guild/pkg/logging"
)

server := grpc.NewServer(
    grpc.UnaryInterceptor(logging.GRPCUnaryServerInterceptor(logger)),
    grpc.StreamInterceptor(logging.GRPCStreamServerInterceptor(logger)),
)
```

## Log Rotation

```go
import "github.com/lancekrogers/guild/pkg/logging/writers"

// Create rotating writer
writer, err := writers.NewRotatingWriter(
    "/var/log/app/app.log",
    100*1024*1024,  // 100MB max size
    30,             // 30 days retention
    10,             // 10 backup files
    true,           // Enable compression
)
```

## Hooks

### Error Tracking Hook

```go
import "github.com/lancekrogers/guild/pkg/logging/hooks"

errorHook := hooks.NewErrorHook(
    true,  // Include stack traces
    true,  // Track error frequency
)

config.Hooks = []logging.Hook{errorHook}

// Get error statistics
topErrors := errorHook.GetTopErrors(5)
for _, err := range topErrors {
    fmt.Printf("Error: %s, Count: %d\n", err.Error, err.Count)
}
```

### Metrics Emission Hook

```go
metricsHook := hooks.NewMetricsHook(
    func(name string, value float64, tags map[string]string) {
        // Emit to your metrics system (Prometheus, etc.)
        prometheus.CounterVec.WithLabelValues(tags...).Add(value)
    },
    30*time.Second, // Emission interval
)

config.Hooks = []logging.Hook{metricsHook}
```

## Performance Considerations

- Zero-allocation logging for common field types
- Efficient context field extraction
- Sampling reduces overhead during high load
- Log rotation prevents disk space issues
- Structured format enables efficient parsing and querying

## Best Practices

1. **Use Context**: Always propagate context through your application
2. **Structured Fields**: Prefer structured fields over string formatting
3. **Consistent Naming**: Use the provided field constructors for consistency
4. **Error Wrapping**: Use gerror for rich error context
5. **Sampling**: Enable sampling in high-throughput applications
6. **Security**: Enable PII scrubbing in production

## Integration with Guild Framework

The logging package is designed to integrate seamlessly with Guild components:

```go
// In commission processing
logger.Info("Commission assigned",
    logging.CommissionIDField(commission.ID),
    logging.AgentIDField(agent.Name),
    logging.TaskTypeField("analysis"),
    logging.Duration("estimated_duration", commission.EstimatedDuration),
)

// In agent execution  
logger.WithContext(ctx).Info("Tool execution started",
    logging.String("tool", "file_reader"),
    logging.String("file_path", "/workspace/src/main.go"),
)

// In error scenarios
if err != nil {
    logger.Error("Commission failed",
        logging.ErrorField(err),
        logging.String("phase", "planning"),
        logging.Bool("recoverable", true),
    )
}
```

## Testing

The package includes comprehensive tests covering:

- Logger creation and configuration
- Context propagation and field extraction
- Sampling strategies and behavior
- Formatter output correctness
- Hook processing and error handling

Run tests with:
```bash
go test ./pkg/logging/...
```

## Contributing

When adding new features:

1. Follow Go conventions and existing patterns
2. Add comprehensive tests
3. Update documentation
4. Ensure zero allocations for hot paths
5. Maintain backward compatibility