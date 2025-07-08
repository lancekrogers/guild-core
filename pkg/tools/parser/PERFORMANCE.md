# Performance Guide

This guide provides recommendations for optimizing parser performance in production environments.

## Benchmarks

Current performance on a 2.3 GHz Intel Core i7:

| Operation | Size | Time | Allocations |
|-----------|------|------|-------------|
| JSON Parse (small) | 200B | 2.3μs | 12 allocs |
| JSON Parse (medium) | 2KB | 8.2μs | 24 allocs |
| JSON Parse (large) | 20KB | 45μs | 156 allocs |
| XML Parse (small) | 250B | 4.1μs | 18 allocs |
| XML Parse (medium) | 2.5KB | 12.4μs | 42 allocs |
| Format Detection | Any | 1.5μs | 8 allocs |

## Optimization Strategies

### 1. Reuse Parser Instances

Parser instances are thread-safe and should be reused:

```go
// Good - create once, use many times
var globalParser = parser.NewResponseParser()

func handleRequest(response string) {
    calls, _ := globalParser.ExtractToolCalls(response)
    // Process calls
}

// Bad - creating parser for each request
func handleRequest(response string) {
    p := parser.NewResponseParser() // Unnecessary allocation
    calls, _ := p.ExtractToolCalls(response)
}
```

### 2. Use Context Timeouts

Always use context with timeouts for untrusted input:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

calls, err := parser.ExtractWithContext(ctx, response)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // Handle timeout
    }
}
```

### 3. Set Input Size Limits

Prevent memory exhaustion with size limits:

```go
parser := parser.NewResponseParser(
    parser.WithMaxInputSize(1 * 1024 * 1024), // 1MB max
)
```

### 4. Batch Processing

For multiple responses, process in batches:

```go
func processBatch(responses []string) [][]parser.ToolCall {
    results := make([][]parser.ToolCall, len(responses))
    
    // Use goroutines for parallel processing
    var wg sync.WaitGroup
    for i, response := range responses {
        wg.Add(1)
        go func(idx int, resp string) {
            defer wg.Done()
            results[idx], _ = globalParser.ExtractToolCalls(resp)
        }(i, response)
    }
    wg.Wait()
    
    return results
}
```

### 5. Skip Format Detection

If you know the format, skip detection:

```go
// If you know it's OpenAI format
jsonParser := json.NewParser()
calls, err := jsonParser.Parse(ctx, []byte(response))
```

### 6. Enable Caching

For repeated inputs, implement caching:

```go
type CachedParser struct {
    parser parser.ResponseParser
    cache  sync.Map // or use a proper cache like groupcache
}

func (c *CachedParser) ExtractToolCalls(response string) ([]parser.ToolCall, error) {
    // Check cache
    if cached, ok := c.cache.Load(response); ok {
        return cached.([]parser.ToolCall), nil
    }
    
    // Parse and cache
    calls, err := c.parser.ExtractToolCalls(response)
    if err == nil && len(calls) > 0 {
        c.cache.Store(response, calls)
    }
    
    return calls, err
}
```

## Memory Management

### Reduce Allocations

The parser minimizes allocations, but you can help:

```go
// Reuse slices when processing many responses
calls := make([]parser.ToolCall, 0, 10)
for _, response := range responses {
    calls = calls[:0] // Reset slice, keep capacity
    calls, _ = parser.ExtractToolCalls(response)
    // Process calls
}
```

### Large Input Handling

For very large inputs, consider streaming:

```go
// Process in chunks if needed
const chunkSize = 1024 * 1024 // 1MB chunks

if len(response) > chunkSize {
    // Process in parts or use streaming approach
}
```

## Monitoring Performance

### Use Metrics

Monitor parser performance in production:

```go
// Create instrumented parser
parser := parser.InstrumentParser(baseParser)

// Metrics are automatically collected:
// - guild_parser_parse_duration_seconds
// - guild_parser_parse_attempts_total
// - guild_parser_tool_calls_per_parse
```

### Set Up Alerts

Configure alerts for performance issues:

```go
am := parser.NewAlertManager()
am.AddCondition(parser.AlertCondition{
    Name:     "slow_parsing",
    Severity: parser.AlertSeverityWarning,
    Check: func(m parser.HealthMetrics) bool {
        return m.P95Latency > 100 // 100ms
    },
})
```

### Dashboard Monitoring

Use the built-in dashboard:

```go
parser, dashboard := parser.CreateFullyInstrumentedParser("v1.0.0")
go dashboard.Start() // Access at http://localhost:8080
```

## Common Performance Issues

### Issue: Slow Parsing

**Symptoms**: High latency in parse operations

**Solutions**:
1. Check input size - large inputs take longer
2. Verify format detection isn't timing out
3. Look for malformed JSON/XML causing retries
4. Enable metrics to identify bottlenecks

### Issue: High Memory Usage

**Symptoms**: Growing memory consumption

**Solutions**:
1. Set `WithMaxInputSize()` limit
2. Check for memory leaks in tool execution
3. Ensure parser instances are reused
4. Monitor with `runtime.MemStats`

### Issue: CPU Spikes

**Symptoms**: High CPU usage during parsing

**Solutions**:
1. Limit concurrent parsing operations
2. Use context timeouts to prevent runaway parsing
3. Check for complex nested structures
4. Profile with `pprof` to identify hot spots

## Profiling

Enable profiling to identify bottlenecks:

```go
import _ "net/http/pprof"

func main() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    // Your application
}

// Profile CPU usage
// go tool pprof http://localhost:6060/debug/pprof/profile

// Check memory allocations  
// go tool pprof http://localhost:6060/debug/pprof/heap
```

## Best Practices Summary

1. **Reuse parser instances** - they're thread-safe
2. **Set timeouts** - protect against hanging
3. **Limit input size** - prevent memory exhaustion
4. **Monitor metrics** - track performance trends
5. **Handle errors properly** - don't retry bad inputs
6. **Use appropriate formats** - JSON is faster than XML
7. **Batch when possible** - amortize overhead
8. **Profile in production** - identify real bottlenecks