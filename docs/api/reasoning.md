# Reasoning & Intelligence Layer API Documentation

## Overview

The Reasoning & Intelligence Layer provides sophisticated extraction and analysis of reasoning patterns from LLM responses. It includes built-in resilience patterns, monitoring, and provider-agnostic interfaces.

## Table of Contents

1. [Core Types](#core-types)
2. [Registry API](#registry-api)
3. [Circuit Breaker API](#circuit-breaker-api)
4. [Rate Limiter API](#rate-limiter-api)
5. [Dead Letter Queue API](#dead-letter-queue-api)
6. [Events API](#events-api)
7. [Metrics API](#metrics-api)
8. [Error Handling](#error-handling)

## Core Types

### ReasoningBlock

Represents a unit of extracted reasoning.

```go
type ReasoningBlock struct {
    ID         string                 `json:"id"`
    Type       string                 `json:"type"`       // e.g., "thinking", "planning"
    Content    string                 `json:"content"`
    Timestamp  time.Time              `json:"timestamp"`
    Duration   time.Duration          `json:"duration"`
    TokenCount int                    `json:"token_count"`
    Depth      int                    `json:"depth"`      // Nesting level
    ParentID   string                 `json:"parent_id,omitempty"`
    Children   []string               `json:"children,omitempty"`
    Confidence float64                `json:"confidence,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
```

### HealthStatus

```go
type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
)
```

### HealthReport

```go
type HealthReport struct {
    Status     HealthStatus              `json:"status"`
    Components map[string]ComponentHealth `json:"components"`
    Timestamp  time.Time                 `json:"timestamp"`
}

type ComponentHealth struct {
    Status  HealthStatus           `json:"status"`
    Message string                 `json:"message,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

## Registry API

The main entry point for reasoning extraction.

### Creating a Registry

```go
import (
    "github.com/guild-framework/guild-core/pkg/reasoning"
    "github.com/guild-framework/guild-core/pkg/orchestrator"
    "log/slog"
)

// Create components
extractor := reasoning.NewDefaultExtractor()
circuitBreaker := reasoning.NewCircuitBreaker(reasoning.CircuitBreakerConfig{
    FailureThreshold:   10,
    SuccessThreshold:   5,
    Timeout:            30 * time.Second,
    MaxHalfOpenCalls:   10,
    ObservationWindow:  60 * time.Second,
})

rateLimiter := reasoning.NewRateLimiter(reasoning.RateLimiterConfig{
    GlobalRPS:       1000,
    PerAgentRPS:     50,
    BurstSize:       10,
    MaxAgents:       1000,
    CleanupInterval: 5 * time.Minute,
})

retryer := reasoning.NewRetryer(reasoning.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 100 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,
    Jitter:       0.1,
})

deadLetter := reasoning.NewDeadLetterQueue(db, logger, reasoning.DeadLetterConfig{
    MaxRetries:      5,
    RetentionPeriod: 7 * 24 * time.Hour,
    CleanupInterval: 1 * time.Hour,
})

healthChecker := reasoning.NewHealthChecker(logger)

// Create registry
registry := reasoning.NewRegistry(
    extractor, circuitBreaker, rateLimiter, retryer,
    deadLetter, healthChecker, logger, eventBus,
)
```

### Starting and Stopping

```go
// Start the registry
ctx := context.Background()
if err := registry.Start(ctx); err != nil {
    log.Fatal("Failed to start registry:", err)
}

// Stop the registry
defer func() {
    if err := registry.Stop(ctx); err != nil {
        log.Error("Failed to stop registry:", err)
    }
}()
```

### Extracting Reasoning

```go
// Extract reasoning from content
blocks, err := registry.Extract(ctx, "agent-123", "<thinking>...</thinking>")
if err != nil {
    if reasoning.IsCircuitBreakerOpen(err) {
        // Circuit breaker is open, back off
    } else if reasoning.IsRateLimitExceeded(err) {
        // Rate limit exceeded, retry later
    } else {
        // Other error
    }
}

// Process blocks
for _, block := range blocks {
    fmt.Printf("Type: %s, Tokens: %d\n", block.Type, block.TokenCount)
}
```

### Health Monitoring

```go
// Get health status
health := registry.Health(ctx)
if health.Status != reasoning.HealthStatusHealthy {
    // System is degraded or unhealthy
    for component, status := range health.Components {
        if status.Status != reasoning.HealthStatusHealthy {
            log.Warn("Component unhealthy", 
                "component", component,
                "status", status.Status,
                "message", status.Message)
        }
    }
}
```

## Circuit Breaker API

### Configuration

```go
type CircuitBreakerConfig struct {
    FailureThreshold   int           // Failures to open circuit
    SuccessThreshold   int           // Successes to close circuit
    Timeout            time.Duration // Time before half-open
    MaxHalfOpenCalls   int           // Calls allowed in half-open
    ObservationWindow  time.Duration // Window for tracking failures
    OnStateChange      func(from, to CircuitState) // Optional callback
}
```

### States

```go
type CircuitState int

const (
    StateClosed   CircuitState = iota // Normal operation
    StateOpen                         // Failing, rejecting calls
    StateHalfOpen                     // Testing if recovered
)
```

### Usage

```go
cb := reasoning.NewCircuitBreaker(config)

// Execute with circuit breaker protection
err := cb.Execute(ctx, func() error {
    return someRiskyOperation()
})

// Check state
state := cb.State()
if state == reasoning.StateOpen {
    // Circuit is open, operations will fail fast
}

// Get statistics
stats := cb.Stats()
fmt.Printf("Failures: %d, Successes: %d, State: %s\n", 
    stats.Failures, stats.Successes, stats.State)
```

## Rate Limiter API

### Configuration

```go
type RateLimiterConfig struct {
    GlobalRPS       float64       // Global requests per second
    PerAgentRPS     float64       // Per-agent requests per second
    BurstSize       int           // Token bucket burst size
    MaxAgents       int           // Maximum tracked agents
    CleanupInterval time.Duration // Cleanup inactive agents
}
```

### Usage

```go
rl := reasoning.NewRateLimiter(config)

// Check if allowed (non-blocking)
if err := rl.Allow(ctx, "agent-123"); err != nil {
    // Rate limit exceeded
}

// Wait for permission (blocking)
if err := rl.Wait(ctx, "agent-123"); err != nil {
    // Context cancelled or rate limit exceeded
}

// Get current usage
usage := rl.Usage()
for agentID, ratio := range usage {
    fmt.Printf("Agent %s: %.2f%% of limit\n", agentID, ratio*100)
}
```

## Dead Letter Queue API

### Configuration

```go
type DeadLetterConfig struct {
    MaxRetries      int           // Max reprocessing attempts
    RetentionPeriod time.Duration // How long to keep entries
    CleanupInterval time.Duration // Cleanup frequency
}
```

### Adding Entries

```go
dlq := reasoning.NewDeadLetterQueue(db, logger, config)

// Add failed extraction
err := dlq.Add(ctx, "agent-123", content, extractionError, 3, map[string]interface{}{
    "provider": "openai",
    "model":    "gpt-4",
})
```

### Listing Entries

```go
// List unprocessed entries
entries, err := dlq.List(ctx, reasoning.DeadLetterFilter{
    Status:   "unprocessed",
    AgentID:  "agent-123",
    Limit:    100,
    OlderThan: 24 * time.Hour,
})

for _, entry := range entries {
    fmt.Printf("ID: %s, Agent: %s, Error: %s\n", 
        entry.ID, entry.AgentID, entry.ErrorMessage)
}
```

### Reprocessing

```go
// Reprocess specific entry
result, err := dlq.Reprocess(ctx, entryID)
if err != nil {
    // Reprocessing failed
} else if result.Success {
    // Successfully reprocessed
    fmt.Printf("Extracted %d blocks\n", len(result.Blocks))
}

// Reprocess batch
results, err := dlq.ReprocessBatch(ctx, reasoning.ReprocessOptions{
    MaxItems:    100,
    MaxAge:      24 * time.Hour,
    ErrorTypes:  []string{"timeout", "rate_limit"},
})
```

## Events API

### Event Types

```go
// Reasoning extracted successfully
type ReasoningExtractedEvent struct {
    orchestrator.BaseEvent
    AgentID         string
    Provider        string
    Blocks          []ReasoningBlock
    TokensExtracted int
    Duration        time.Duration
    Timestamp       time.Time
}

// Extraction failed
type ReasoningFailedEvent struct {
    orchestrator.BaseEvent
    AgentID   string
    Provider  string
    Error     string
    ErrorCode string
    Attempts  int
    Timestamp time.Time
}

// Circuit breaker state changed
type CircuitBreakerStateChangeEvent struct {
    orchestrator.BaseEvent
    Component string
    FromState string
    ToState   string
    Timestamp time.Time
}

// Rate limit exceeded
type RateLimitExceededEvent struct {
    orchestrator.BaseEvent
    AgentID   string
    LimitType string // "global" or "agent"
    Limit     float64
    Current   float64
    Timestamp time.Time
}
```

### Subscribing to Events

```go
// Subscribe to extraction events
eventBus.Subscribe("ReasoningExtractedEvent", func(event interface{}) {
    if e, ok := event.(*reasoning.ReasoningExtractedEvent); ok {
        fmt.Printf("Extracted %d blocks for agent %s\n", 
            len(e.Blocks), e.AgentID)
    }
})

// Subscribe to failures
eventBus.Subscribe("ReasoningFailedEvent", func(event interface{}) {
    if e, ok := event.(*reasoning.ReasoningFailedEvent); ok {
        log.Error("Extraction failed", 
            "agent", e.AgentID,
            "error", e.Error,
            "attempts", e.Attempts)
    }
})
```

## Metrics API

### Available Metrics

```go
// Create metrics collector
collector, err := reasoning.NewMetricsCollector(metricsRegistry)

// Record extraction
collector.RecordExtraction(ctx, "openai", "agent-123", 
    500*time.Millisecond, blocks, nil)

// Record circuit breaker state change
collector.RecordCircuitBreakerStateChange(
    reasoning.StateClosed, reasoning.StateOpen)

// Update gauges
collector.UpdateRateLimiterUsage(map[string]float64{
    "global":   0.75,
    "agent-123": 0.90,
})
```

### Metric Names

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `reasoning_extraction_total` | Counter | Total extractions | provider, agent_id, status |
| `reasoning_extraction_duration_seconds` | Histogram | Extraction latency | provider, agent_id |
| `reasoning_extraction_errors_total` | Counter | Error counts | provider, agent_id, error_type |
| `reasoning_tokens_processed_total` | Counter | Tokens processed | provider, agent_id, block_type |
| `reasoning_circuit_breaker_state` | Gauge | Current state (0/1/2) | provider |
| `reasoning_circuit_breaker_trips_total` | Counter | State transitions | from_state, to_state |
| `reasoning_rate_limiter_usage_ratio` | Gauge | Usage ratio | agent_id, type |
| `reasoning_rate_limit_hits_total` | Counter | Rate limit hits | agent_id, limit_type |
| `reasoning_dead_letter_queue_size` | Gauge | Queue size | status |
| `reasoning_active_extractions` | Gauge | Active extractions | provider |

## Error Handling

### Error Types

```go
// Common errors
var (
    ErrCircuitBreakerOpen = gerror.New("circuit breaker is open").
        WithCode(gerror.ErrCodeResourceExhausted)
    
    ErrRateLimitExceeded = gerror.New("rate limit exceeded").
        WithCode(gerror.ErrCodeResourceExhausted)
    
    ErrRegistryNotStarted = gerror.New("reasoning registry not started").
        WithCode(gerror.ErrCodeFailedPrecondition)
)
```

### Error Checking

```go
// Check specific error types
if reasoning.IsCircuitBreakerOpen(err) {
    // Wait for circuit to close
}

if reasoning.IsRateLimitExceeded(err) {
    // Implement backoff
}

// Check error codes
if gerr, ok := err.(*gerror.Error); ok {
    switch gerr.Code {
    case gerror.ErrCodeResourceExhausted:
        // Resource limits hit
    case gerror.ErrCodeUnavailable:
        // Temporary failure, retry
    case gerror.ErrCodeInternal:
        // Internal error, check logs
    }
}
```

### Retry Logic

```go
// Determine if error is retryable
if reasoning.IsRetryable(err) {
    // Safe to retry
    backoff := time.Duration(attempt) * 100 * time.Millisecond
    time.Sleep(backoff)
}
```

## Best Practices

### 1. Always Handle Context Cancellation

```go
blocks, err := registry.Extract(ctx, agentID, content)
if err != nil {
    if ctx.Err() != nil {
        // Context was cancelled
        return ctx.Err()
    }
    // Handle other errors
}
```

### 2. Monitor Health Regularly

```go
// Set up health monitoring
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

go func() {
    for range ticker.C {
        health := registry.Health(ctx)
        if health.Status != reasoning.HealthStatusHealthy {
            alerting.NotifyDegraded(health)
        }
    }
}()
```

### 3. Configure Timeouts Appropriately

```go
// Use context with timeout for extractions
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

blocks, err := registry.Extract(ctx, agentID, content)
```

### 4. Handle Rate Limits Gracefully

```go
// Implement exponential backoff
for attempt := 0; attempt < maxAttempts; attempt++ {
    blocks, err := registry.Extract(ctx, agentID, content)
    if err == nil {
        return blocks, nil
    }
    
    if !reasoning.IsRateLimitExceeded(err) {
        return nil, err
    }
    
    backoff := time.Duration(1<<uint(attempt)) * time.Second
    select {
    case <-time.After(backoff):
        continue
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 5. Process Dead Letter Queue Regularly

```go
// Set up periodic reprocessing
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        results, err := dlq.ReprocessBatch(ctx, reasoning.ReprocessOptions{
            MaxItems: 100,
            MaxAge:   24 * time.Hour,
        })
        
        if err != nil {
            log.Error("Failed to reprocess dead letters", "error", err)
        } else {
            log.Info("Reprocessed dead letters", 
                "total", results.Total,
                "successful", results.Successful)
        }
    }
}()
```
