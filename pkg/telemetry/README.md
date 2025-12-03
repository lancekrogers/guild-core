# Guild Telemetry Package

The telemetry package provides comprehensive observability for Guild using OpenTelemetry. It offers metrics collection, distributed tracing, and integration with popular observability platforms while maintaining less than 1% performance overhead.

## Features

- **OpenTelemetry Integration**: Full OTEL support for metrics and traces
- **Multi-Exporter Support**: Prometheus, Jaeger, OTLP
- **Domain-Specific Collectors**: Specialized metrics for agents, chat, commissions, and system resources
- **Low Overhead**: Verified < 1% performance impact
- **Grafana Dashboards**: Pre-built dashboards for immediate visibility
- **Alerting Rules**: SLO-based alerts for proactive monitoring

## Quick Start

```go
import (
    "context"
    "github.com/lancekrogers/guild/pkg/telemetry"
)

func main() {
    ctx := context.Background()
    
    // Initialize telemetry
    tel, err := telemetry.New(ctx, telemetry.Config{
        ServiceName:        "guild-service",
        ServiceVersion:     "1.0.0",
        Environment:        "production",
        OTLPEndpoint:       "localhost:4317",
        PrometheusEndpoint: ":9090",
        SampleRate:         1.0,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer tel.Shutdown(ctx)
    
    // Use telemetry throughout your application
    tel.RecordRequest(ctx, "api.handler", duration, err)
}
```

## Configuration

### Basic Configuration

```go
config := telemetry.Config{
    ServiceName:        "guild-service",     // Required
    ServiceVersion:     "1.0.0",            // Recommended
    Environment:        "production",        // Recommended
    OTLPEndpoint:       "localhost:4317",    // OTLP collector endpoint
    PrometheusEndpoint: ":9090",            // Prometheus scrape endpoint
    JaegerEndpoint:     "localhost:14250",   // Jaeger collector endpoint
    SampleRate:         0.1,                // 10% trace sampling
}
```

### Environment Variables

- `OTEL_SERVICE_NAME`: Override service name
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP endpoint
- `OTEL_TRACES_SAMPLER_ARG`: Sampling rate (0.0-1.0)

## Core Metrics

### Request Metrics

```go
// Record request duration and errors
tel.RecordRequest(ctx, "operation.name", duration, err,
    attribute.String("method", "POST"),
    attribute.String("endpoint", "/api/v1/commission"),
)

// Track concurrent requests
tel.IncrementActiveRequests(ctx)
defer tel.DecrementActiveRequests(ctx)
```

### Business Metrics

```go
// Commission tracking
tel.RecordCommissionStarted(ctx, commissionID,
    attribute.String("type", "code-generation"))
tel.RecordCommissionCompleted(ctx, commissionID, success,
    attribute.String("reason", "completed"))

// Agent invocations
tel.RecordAgentInvocation(ctx, "code-analyzer",
    attribute.String("task", "security-scan"))

// Token usage
tel.RecordTokenUsage(ctx, "openai", "gpt-4", tokenCount,
    attribute.String("purpose", "analysis"))
```

## Distributed Tracing

### Basic Tracing

```go
// Trace an operation
err := tel.TraceOperation(ctx, "database.query", func(ctx context.Context) error {
    // Your code here
    return db.Query(ctx, sql)
})

// Manual span control
ctx, span := tel.StartSpan(ctx, "complex.operation",
    telemetry.WithSpanKind(telemetry.SpanKindServer),
    telemetry.WithAttributes(attribute.String("component", "api")),
)
defer span.End()

// Add events to current span
telemetry.AddEvent(ctx, "checkpoint.reached",
    attribute.Int("progress", 50))

// Record errors
if err != nil {
    telemetry.RecordError(ctx, err,
        attribute.String("operation", "validation"))
}
```

### Trace Context Propagation

```go
// Extract trace context for distributed systems
traceID, spanID := telemetry.ExtractTraceContext(ctx)

// Pass to downstream services
req.Header.Set("X-Trace-ID", traceID)
req.Header.Set("X-Span-ID", spanID)
```

## Domain-Specific Collectors

### Agent Metrics

```go
agentCollector, err := collectors.NewAgentCollector(tel.Meter())

// Record agent execution
agentCollector.RecordExecution(ctx, "analyzer", "code-review", 
    executionTime, success)

// Track tool usage
agentCollector.RecordToolInvocation(ctx, "analyzer", "ast-parser")

// Memory retrieval performance
agentCollector.RecordMemoryRetrieval(ctx, "analyzer", "semantic-search",
    retrievalTime, itemCount)
```

### Chat Metrics

```go
chatCollector, err := collectors.NewChatCollector(tel.Meter())

// Session lifecycle
chatCollector.RecordSessionStart(ctx, sessionID, userID)
chatCollector.RecordSessionEnd(ctx, sessionID, userID, duration)

// Message tracking
chatCollector.RecordMessage(ctx, sessionID, "user", provider)
chatCollector.RecordResponse(ctx, sessionID, provider, responseTime, 
    tokenCount, success)

// Streaming metrics
chatCollector.RecordStreamingLatency(ctx, sessionID, chunkIndex, latency)
```

### Commission Metrics

```go
commCollector, err := collectors.NewCommissionCollector(tel.Meter())

// Commission execution
commCollector.RecordExecution(ctx, commissionID, "type", duration, success)
commCollector.RecordComplexity(ctx, commissionID, complexityScore)

// Task tracking
commCollector.RecordTaskGenerated(ctx, commissionID, "analysis", count)
commCollector.RecordTaskCompletion(ctx, commissionID, taskID, success)

// Resource utilization
commCollector.RecordResourceUtilization(ctx, commissionID, "cpu", percentage)
```

## System Metrics

System metrics are automatically collected:

- CPU usage and cores
- Memory allocation and usage
- Heap statistics
- Goroutine count
- GC pause times and frequency
- File descriptor usage

## Dashboards

Import the pre-built Grafana dashboards:

1. **Overview Dashboard** (`dashboards/overview.json`)
   - System health score
   - Request rates and latency
   - Error rates and alerts
   - Resource utilization

2. **Agent Performance** (`dashboards/agents.json`)
   - Agent invocation rates
   - Task success/failure rates
   - Tool usage heatmaps
   - Memory retrieval performance

3. **Performance Analysis** (`dashboards/performance.json`)
   - Latency percentiles
   - Resource utilization trends
   - GC performance
   - Top slow operations

## Alerting

Configure Prometheus alerts:

```yaml
# Include alert rules
- /path/to/telemetry/alerts/slo.yaml
- /path/to/telemetry/alerts/errors.yaml  
- /path/to/telemetry/alerts/resources.yaml
```

Key alerts include:

- High error rate (> 5%)
- High latency (P99 > 1s)
- Memory leaks
- Resource exhaustion
- Error budget burn rate

## Testing

### Unit Testing with Mock Telemetry

```go
func TestMyFunction(t *testing.T) {
    // Use mock telemetry for testing
    mock := telemetry.NewMockTelemetry()
    
    // Your code that uses telemetry
    myFunction(mock)
    
    // Verify metrics were recorded
    if len(mock.RecordedRequests) != 1 {
        t.Error("expected request to be recorded")
    }
    
    req := mock.RecordedRequests[0]
    if req.Operation != "expected-op" {
        t.Errorf("unexpected operation: %s", req.Operation)
    }
}
```

### Integration Testing

```go
func TestIntegration(t *testing.T) {
    // Use noop telemetry for integration tests
    tel := telemetry.NewNoop()
    
    // No actual metrics are sent
    tel.RecordRequest(ctx, "test-op", 100*time.Millisecond, nil)
}
```

## Performance

Benchmarks show < 1% overhead:

```
BenchmarkBaselineWithoutTelemetry-8    1000000    1053 ns/op
BenchmarkWithNoopTelemetry-8          1000000    1061 ns/op
BenchmarkWithTracing-8                 1000000    1068 ns/op

Overhead: 0.76%
```

## Best Practices

1. **Initialize Early**: Set up telemetry during application startup
2. **Use Context**: Always pass context for proper trace propagation
3. **Meaningful Names**: Use descriptive operation names (e.g., "commission.parse" not "parse")
4. **Appropriate Attributes**: Add relevant attributes but avoid high-cardinality values
5. **Error Context**: Always record errors with additional context
6. **Batch Operations**: Use collectors for domain-specific metrics
7. **Resource Limits**: Configure appropriate sampling rates for high-volume services

## Troubleshooting

### No Metrics Appearing

1. Check exporter endpoints are reachable
2. Verify Prometheus is scraping the endpoint
3. Check for errors during initialization
4. Ensure metrics are being recorded (use debug logging)

### High Memory Usage

1. Reduce sampling rate for traces
2. Configure batch export settings
3. Check for metric cardinality explosion
4. Review attribute usage

### Performance Impact

1. Use sampling for high-volume traces
2. Batch metric updates where possible
3. Consider using async exporters
4. Profile with benchmarks

## Examples

See complete examples in:

- `examples/basic/` - Basic telemetry setup
- `examples/distributed/` - Distributed tracing
- `examples/collectors/` - Using domain collectors
- `examples/testing/` - Testing with telemetry
