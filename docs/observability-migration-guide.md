# Observability Migration Guide

This guide helps migrate Guild components to use the new production-ready error handling and observability packages.

## Overview

The new observability system provides:
- **Structured error handling** with `pkg/gerror`
- **Structured logging** with `pkg/observability`
- **Distributed tracing** with OpenTelemetry
- **Metrics collection** with Prometheus
- **Request tracking** and correlation

## Migration Steps

### 1. Replace Error Handling

#### Before:
```go
return fmt.Errorf("failed to create task: %w", err)
```

#### After:
```go
import "github.com/guild-ventures/guild-core/pkg/gerror"

return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create task").
    WithComponent("kanban").
    WithOperation("create_task").
    WithDetails("board_id", boardID).
    FromContext(ctx)
```

### 2. Replace Logging

#### Before:
```go
import "log"
log.Printf("Processing task: %s", taskID)
```

#### After:
```go
import "github.com/guild-ventures/guild-core/pkg/observability"

logger := observability.GetLogger(ctx).
    WithComponent("task-processor").
    WithOperation("process")

logger.InfoContext(ctx, "Processing task",
    "task_id", taskID,
    "priority", priority,
)
```

### 3. Add Tracing

#### Before:
```go
func ProcessTask(ctx context.Context, taskID string) error {
    // Process task
    return nil
}
```

#### After:
```go
import "github.com/guild-ventures/guild-core/pkg/observability"

func ProcessTask(ctx context.Context, taskID string) error {
    ctx, span := observability.StartTaskSpan(ctx, taskID, "process")
    defer span.End()

    // Process task

    return nil
}
```

### 4. Add Metrics

#### Before:
```go
// No metrics
```

#### After:
```go
metrics := observability.GetMetrics()

start := time.Now()
err := processTask()
duration := time.Since(start)

if err != nil {
    metrics.RecordTaskProcessed("failed", campaignID)
    metrics.RecordError(gerror.GetCode(err), "task-processor", "process")
} else {
    metrics.RecordTaskProcessed("success", campaignID)
    metrics.RecordTaskDuration("process", duration)
}
```

## Component-Specific Examples

### Agent Package

```go
// pkg/agent/worker_agent.go

import (
    "github.com/guild-ventures/guild-core/pkg/gerror"
    "github.com/guild-ventures/guild-core/pkg/observability"
)

func (a *WorkerAgent) Execute(ctx context.Context, request AgentRequest) (AgentResponse, error) {
    // Ensure request context
    ctx = observability.EnsureRequestContext(ctx)

    // Create logger
    logger := observability.GetLogger(ctx).
        WithComponent("worker-agent").
        WithOperation("execute")

    // Start trace
    ctx, span := observability.StartAgentSpan(ctx, a.ID, "execute")
    defer span.End()

    // Get metrics
    metrics := observability.GetMetrics()

    logger.InfoContext(ctx, "Starting agent execution",
        "agent_id", a.ID,
        "request_type", request.Type,
    )

    start := time.Now()

    // Execute request
    response, err := a.provider.Complete(ctx, request)

    duration := time.Since(start)

    if err != nil {
        gerr := gerror.Wrap(err, gerror.ErrCodeAgentFailed, "agent execution failed").
            WithComponent("worker-agent").
            WithOperation("execute").
            WithDetails("agent_id", a.ID).
            FromContext(ctx)

        logger.WithError(gerr).ErrorContext(ctx, "Agent execution failed")
        observability.RecordError(ctx, gerr)

        metrics.RecordAgentTask(a.ID, a.Type, "failed")
        metrics.RecordError(string(gerr.Code), "agent", "execute")

        return AgentResponse{}, gerr
    }

    // Success logging
    logger.InfoContext(ctx, "Agent execution completed",
        "duration_ms", duration.Milliseconds(),
        "tokens_used", response.TokensUsed,
    )

    metrics.RecordAgentTask(a.ID, a.Type, "success")
    metrics.RecordAgentTaskDuration(a.ID, a.Type, duration)
    metrics.RecordAgentTokenUsage(a.ID, a.Type, "total", response.TokensUsed)

    return response, nil
}
```

### Storage Package

```go
// pkg/storage/database.go

import (
    "github.com/guild-ventures/guild-core/pkg/gerror"
    "github.com/guild-ventures/guild-core/pkg/observability"
)

func (db *Database) CreateTask(ctx context.Context, task *Task) error {
    logger := observability.GetLogger(ctx).
        WithComponent("storage").
        WithOperation("create_task")

    ctx, span := observability.StartStorageSpan(ctx, "create", "tasks")
    defer span.End()

    metrics := observability.GetMetrics()
    start := time.Now()

    logger.DebugContext(ctx, "Creating task",
        "task_id", task.ID,
        "commission_id", task.CommissionID,
    )

    err := db.conn.Create(ctx, task)

    duration := time.Since(start)

    if err != nil {
        gerr := gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create task").
            WithComponent("storage").
            WithOperation("create_task").
            WithDetails("task_id", task.ID).
            FromContext(ctx)

        logger.WithError(gerr).ErrorContext(ctx, "Failed to create task")
        observability.RecordError(ctx, gerr)

        metrics.RecordStorageOperation("create", "tasks", "failed")
        metrics.RecordStorageError("create", "tasks", "database_error")

        return gerr
    }

    logger.InfoContext(ctx, "Task created successfully",
        "task_id", task.ID,
        "duration_ms", duration.Milliseconds(),
    )

    metrics.RecordStorageOperation("create", "tasks", "success")
    metrics.RecordStorageDuration("create", "tasks", duration)

    return nil
}
```

### Provider Package

```go
// pkg/providers/openai/client.go

import (
    "github.com/guild-ventures/guild-core/pkg/gerror"
    "github.com/guild-ventures/guild-core/pkg/observability"
)

func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
    logger := observability.GetLogger(ctx).
        WithComponent("provider-openai").
        WithOperation("complete")

    ctx, span := observability.StartProviderSpan(ctx, "openai", "completion")
    defer span.End()

    observability.SetSpanAttributes(ctx, map[string]interface{}{
        "model": req.Model,
        "max_tokens": req.MaxTokens,
        "temperature": req.Temperature,
    })

    metrics := observability.GetMetrics()
    start := time.Now()

    logger.DebugContext(ctx, "Sending completion request",
        "model", req.Model,
        "prompt_length", len(req.Prompt),
    )

    resp, err := c.client.CreateCompletion(ctx, req)

    duration := time.Since(start)

    if err != nil {
        // Determine if retryable
        retryable := isRateLimitError(err) || isTimeoutError(err)

        code := gerror.ErrCodeProviderAPI
        if isRateLimitError(err) {
            code = gerror.ErrCodeRateLimit
        } else if isTimeoutError(err) {
            code = gerror.ErrCodeProviderTimeout
        }

        gerr := gerror.Wrap(err, code, "OpenAI API request failed").
            WithComponent("provider-openai").
            WithOperation("complete").
            WithDetails("model", req.Model).
            FromContext(ctx)

        if retryable {
            gerr.Retryable = true
        }

        logger.WithError(gerr).ErrorContext(ctx, "Provider request failed")
        observability.RecordError(ctx, gerr)

        metrics.RecordProviderRequest("openai", req.Model, "failed")
        metrics.RecordProviderError("openai", req.Model, string(code))

        return nil, gerr
    }

    cost := calculateCost(req.Model, resp.Usage)

    logger.InfoContext(ctx, "Completion request successful",
        "model", req.Model,
        "prompt_tokens", resp.Usage.PromptTokens,
        "completion_tokens", resp.Usage.CompletionTokens,
        "total_tokens", resp.Usage.TotalTokens,
        "cost", cost,
        "duration_ms", duration.Milliseconds(),
    )

    metrics.RecordProviderRequest("openai", req.Model, "success")
    metrics.RecordProviderDuration("openai", req.Model, duration)
    metrics.RecordProviderTokens("openai", req.Model, "prompt", resp.Usage.PromptTokens)
    metrics.RecordProviderTokens("openai", req.Model, "completion", resp.Usage.CompletionTokens)
    metrics.RecordProviderCost("openai", req.Model, cost)

    return resp, nil
}
```

## Initialization

Add to your main.go or service initialization:

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/guild-ventures/guild-core/pkg/observability"
)

func main() {
    ctx := context.Background()

    // Initialize observability
    logger := observability.NewLogger(&observability.Config{
        Level: observability.LevelInfo,
        Format: "json",
    })

    tracer, err := observability.InitTracing(ctx, &observability.TracingConfig{
        ServiceName: "guild-agent",
        Enabled: true,
    })
    if err != nil {
        log.Fatal("Failed to initialize tracing:", err)
    }
    defer tracer.Shutdown(ctx)

    metrics := observability.InitGlobalMetrics(&observability.MetricsConfig{
        ServiceName: "guild-agent",
        Enabled: true,
    })

    // Expose metrics endpoint
    http.Handle("/metrics", metrics.Handler())
    go http.ListenAndServe(":9090", nil)

    // Run your service
    logger.Info("Starting Guild service")
}
```

## Environment Variables

Configure observability via environment variables:

```bash
# Logging
export GUILD_ENV=production
export GUILD_SERVICE=guild-agent
export GUILD_VERSION=1.0.0
export GUILD_LOG_LEVEL=info

# Tracing
export GUILD_TRACING_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_INSECURE=true

# Metrics
export GUILD_METRICS_ENABLED=true

# File Logging (optional - off by default)
export GUILD_LOG_FILE=true        # Enable logging to .guild/logs/
```

## Testing

Update your tests to use the new packages:

```go
func TestAgentExecution(t *testing.T) {
    ctx := context.Background()
    ctx = observability.WithRequestID(ctx, "test-request-123")

    // Initialize test logger
    logger := observability.NewLogger(&observability.Config{
        Level: observability.LevelDebug,
        Format: "text",
    })
    ctx = observability.WithLogger(ctx, logger)

    // Test with proper error handling
    err := agent.Execute(ctx, request)

    var gerr *gerror.GuildError
    if errors.As(err, &gerr) {
        assert.Equal(t, gerror.ErrCodeAgentFailed, gerr.Code)
        assert.True(t, gerr.Retryable)
    }
}
```

## File Logging for Debugging

Guild can optionally log to files in `.guild/logs/` for easier debugging:

- **Daily log files**: `guild-2025-01-06.log`, `guild-2025-01-07.log`, etc.
- **Latest symlink**: `latest.log` always points to current day's log
- **User-friendly**: Framework users can easily find debugging information
- **Off by default**: No disk usage unless explicitly enabled

To enable file logging:
```bash
export GUILD_LOG_FILE=true
```

This creates log files in your project's `.guild/logs/` directory with both console and file output.

## Benefits

After migration, you'll have:
- **Better debugging**: Structured errors with stack traces and context
- **Production visibility**: Metrics dashboards and distributed traces
- **Improved reliability**: Proper error categorization and retry logic
- **Faster troubleshooting**: Correlated logs, traces, and metrics
- **Cost tracking**: Built-in metrics for LLM token usage and costs
