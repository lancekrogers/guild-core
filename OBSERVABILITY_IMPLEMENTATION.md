# Production-Ready Observability Implementation

## Overview

Guild now has comprehensive production-ready error handling and observability infrastructure. This implementation provides structured logging, distributed tracing, metrics collection, and enhanced error handling to ensure Guild is ready for production deployment from day one.

## What Was Implemented

### 1. **Structured Error Handling (`pkg/gerror`)**
- Comprehensive error types with error codes (1xxx-6xxx series)
- Error categorization (System, Validation, Storage, Agent, Provider, Task/Orchestration)
- Automatic retryability detection
- User-safe error messages
- Stack trace capture
- Request/trace ID propagation
- Error wrapping with context preservation

**Key Features:**
- `GuildError` type with rich metadata
- Error codes for categorization and monitoring
- Retryable errors for transient failures
- User-safe errors for client display
- Stack traces for debugging
- Context propagation from requests

### 2. **Structured Logging (`pkg/observability/logger.go`)**
- JSON/text format logging with slog
- Context-aware logging with automatic field extraction
- Component and operation tracking
- Performance logging helpers
- Log level management
- Request correlation

**Key Features:**
- Automatic extraction of request/trace/span IDs
- Component-based logging
- Operation tracking
- Performance duration logging
- Error integration with gerror

### 3. **Distributed Tracing (`pkg/observability/tracing.go`)**
- OpenTelemetry integration
- OTLP exporter support
- Span creation and management
- Context propagation
- Error recording in traces
- Guild-specific span helpers

**Key Features:**
- Automatic trace context propagation
- Span attributes for detailed tracking
- Error recording with Guild error details
- Helper functions for common operations (agent, task, provider, storage spans)

### 4. **Metrics Collection (`pkg/observability/metrics.go`)**
- Prometheus metrics integration
- Comprehensive metric types:
  - Request metrics (duration, count, active)
  - Agent metrics (tasks, tokens, cost, utilization)
  - Task metrics (queue size, processed, duration, retries)
  - Storage metrics (operations, duration, errors)
  - Provider metrics (requests, tokens, cost, errors)

**Key Features:**
- HTTP metrics endpoint
- Histogram buckets for latency tracking
- Counter and gauge metrics
- Cost tracking for LLM usage
- Error categorization

### 5. **Context Management (`pkg/observability/context.go`)**
- Request ID generation and tracking
- Context value management
- Guild-specific context (agent, task, commission, campaign IDs)
- Context propagation helpers

**Key Features:**
- Automatic request ID generation
- Context value extraction
- Guild-specific context tracking
- Easy context propagation

## Configuration

### Environment Variables
```bash
# Logging
GUILD_ENV=production          # Environment (development, staging, production)
GUILD_SERVICE=guild-agent     # Service name
GUILD_VERSION=1.0.0          # Service version
GUILD_LOG_LEVEL=info         # Log level (debug, info, warn, error)

# Tracing
GUILD_TRACING_ENABLED=true   # Enable tracing
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317  # OpenTelemetry endpoint
OTEL_EXPORTER_OTLP_INSECURE=true           # Use insecure connection

# Metrics
GUILD_METRICS_ENABLED=true   # Enable metrics
```

## Usage Examples

### Error Handling
```go
// Create a structured error
err := gerror.New(gerror.ErrCodeAgentFailed, "agent execution failed", originalErr).
    WithComponent("agent-manager").
    WithOperation("execute_task").
    WithDetails("agent_id", agentID).
    WithDetails("task_id", taskID).
    FromContext(ctx)

// Check error types
if gerror.Is(err, gerror.ErrCodeTimeout) {
    // Handle timeout
}

// Check if retryable
if gerror.IsRetryable(err) {
    // Retry operation
}
```

### Logging
```go
// Get logger with context
logger := observability.GetLogger(ctx).
    WithComponent("task-processor").
    WithOperation("process")

// Log with context
logger.InfoContext(ctx, "Processing task",
    "task_id", taskID,
    "priority", priority,
)

// Log with error
logger.WithError(err).ErrorContext(ctx, "Task processing failed")
```

### Tracing
```go
// Start a span
ctx, span := observability.StartTaskSpan(ctx, taskID, "process")
defer span.End()

// Record error in trace
if err != nil {
    observability.RecordError(ctx, err)
}

// Set span attributes
observability.SetSpanAttributes(ctx, map[string]interface{}{
    "task.priority": priority,
    "task.type": taskType,
})
```

### Metrics
```go
metrics := observability.GetMetrics()

// Record operation
start := time.Now()
err := performOperation()
duration := time.Since(start)

if err != nil {
    metrics.RecordError(gerror.GetCode(err), "component", "operation")
} else {
    metrics.RecordRequest("POST", "/api/tasks", "200", duration)
}
```

## Benefits

1. **Production Visibility**: Full observability from day one
2. **Debugging**: Rich error context and stack traces
3. **Performance Monitoring**: Automatic latency and throughput tracking
4. **Cost Tracking**: Built-in LLM usage and cost metrics
5. **Reliability**: Error categorization enables smart retry logic
6. **User Experience**: User-safe errors for better client communication
7. **Operational Excellence**: Correlated logs, traces, and metrics

## Next Steps

1. **Component Migration**: Migrate all components to use new observability
2. **Dashboard Creation**: Set up Grafana dashboards for metrics
3. **Alerting**: Configure Prometheus alerts for critical errors
4. **Trace Analysis**: Set up Jaeger/Tempo for trace visualization
5. **Log Aggregation**: Configure log shipping to centralized system

## Migration Priority

High priority components to migrate:
1. Agent execution (`pkg/agent`)
2. Storage operations (`pkg/storage`)
3. Provider calls (`pkg/providers`)
4. Task orchestration (`pkg/orchestrator`)
5. API endpoints (`pkg/grpc`, `cmd/guild`)

The observability infrastructure is now ready for production use and provides comprehensive visibility into Guild's operations.