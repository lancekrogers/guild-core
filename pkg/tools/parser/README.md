# Guild Tool Parser

A robust, production-grade parser for extracting tool calls from LLM responses. Supports multiple provider formats with automatic detection, comprehensive error handling, and built-in observability.

## Features

- **Multi-Provider Support**: Automatically detects and parses OpenAI and Anthropic formats
- **Format Detection**: Intelligent format detection with confidence scoring
- **Robust Parsing**: Handles mixed content, code blocks, and malformed inputs gracefully
- **Production Ready**: Comprehensive error handling, context support, and timeouts
- **Observable**: Built-in metrics, tracing, health checks, and alerting
- **Extensible**: Plugin architecture for adding new formats
- **High Performance**: Optimized for speed with streaming parsers
- **Thread Safe**: Safe for concurrent use

## Installation

```go
import "github.com/lancekrogers/guild/pkg/tools/parser"
```

## Quick Start

```go
// Create a parser
p := parser.NewResponseParser()

// Extract tool calls from an LLM response
response := `I'll help you search for that.
{"tool_calls": [{"id": "call_123", "type": "function", "function": {"name": "search", "arguments": "{\"query\": \"golang\"}"}}]}`

calls, err := p.ExtractToolCalls(response)
if err != nil {
    log.Fatal(err)
}

for _, call := range calls {
    fmt.Printf("Function: %s, Args: %s\n", call.Function.Name, call.Function.Arguments)
}
```

## Supported Formats

### OpenAI Format (JSON)

```json
{
  "tool_calls": [{
    "id": "call_abc123",
    "type": "function",
    "function": {
      "name": "get_weather",
      "arguments": "{\"location\": \"San Francisco\"}"
    }
  }]
}
```

### Anthropic Format (XML)

```xml
<function_calls>
  <invoke name="get_weather">
    <parameter name="location">San Francisco</parameter>
  </invoke>
</function_calls>
```

## Advanced Usage

### With Context and Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

calls, err := parser.ExtractWithContext(ctx, response)
```

### Format Detection

```go
format, confidence, err := parser.DetectFormat(response)
if err != nil {
    log.Printf("No tool calls detected")
    return
}

log.Printf("Detected format: %s (confidence: %.2f)", format, confidence)
```

### Custom Configuration

```go
parser := parser.NewResponseParser(
    parser.WithMaxInputSize(5 * 1024 * 1024),  // 5MB max
    parser.WithTimeout(10 * time.Second),       // 10s timeout
    parser.WithStrictValidation(true),          // Strict mode
)
```

### With Observability

```go
// Create a fully instrumented parser
parser, dashboard := parser.CreateFullyInstrumentedParser("v1.0.0")

// Start dashboard server
go dashboard.Start()

// Parser now emits metrics, traces, and health data
calls, err := parser.ExtractToolCalls(response)

// Check health
health := parser.GetHealth()
fmt.Printf("Parser health: %s\n", health.Status)

// Get active alerts
alerts := parser.GetAlerts()
for _, alert := range alerts {
    fmt.Printf("Alert: %s - %s\n", alert.Severity, alert.Description)
}

// Graceful shutdown
parser.Stop()
dashboard.Stop()
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      ResponseParser                         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐         ┌─────────────────┐          │
│  │ DetectorRegistry │────────▶│  FormatDetector │          │
│  └─────────────────┘         └─────────────────┘          │
│           │                                                 │
│           ▼                                                 │
│  ┌─────────────────┐         ┌─────────────────┐          │
│  │   JSON Detector  │         │   XML Detector   │          │
│  └─────────────────┘         └─────────────────┘          │
│                                                             │
│  ┌─────────────────┐         ┌─────────────────┐          │
│  │   FormatParser   │────────▶│     Parser      │          │
│  └─────────────────┘         └─────────────────┘          │
│           │                                                 │
│           ▼                                                 │
│  ┌─────────────────┐         ┌─────────────────┐          │
│  │   JSON Parser    │         │    XML Parser    │          │
│  └─────────────────┘         └─────────────────┘          │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Error Handling

The parser uses the `gerror` package for consistent error handling:

```go
calls, err := parser.ExtractToolCalls(response)
if err != nil {
    if gerr, ok := err.(*gerror.Error); ok {
        switch gerr.Code() {
        case gerror.ErrCodeValidation:
            // Input validation error
        case gerror.ErrCodeTimeout:
            // Operation timed out
        case gerror.ErrCodeNotFound:
            // No tool calls found
        default:
            // Other error
        }
    }
}
```

## Performance

Benchmarks on a 2.3 GHz Intel Core i7:

```
BenchmarkParser_JSON_Small-8         500000      2341 ns/op
BenchmarkParser_JSON_Medium-8        200000      8234 ns/op
BenchmarkParser_XML_Small-8          300000      4123 ns/op
BenchmarkParser_XML_Medium-8         100000     12456 ns/op
BenchmarkParser_MixedContent-8       100000     15234 ns/op
```

## Monitoring

### Metrics

The parser exposes Prometheus metrics:

- `guild_parser_parse_attempts_total` - Total parsing attempts
- `guild_parser_parse_successes_total` - Successful parses
- `guild_parser_parse_failures_total` - Failed parses
- `guild_parser_parse_duration_seconds` - Parse duration histogram
- `guild_parser_tool_calls_extracted_total` - Tool calls by format and function
- `guild_parser_format_distribution_total` - Distribution of detected formats

### Health Checks

```bash
# Liveness check
curl http://localhost:8080/health/live

# Readiness check
curl http://localhost:8080/health/ready

# Full health status
curl http://localhost:8080/health
```

### Dashboard

Access the monitoring dashboard at `http://localhost:8080/`

## Testing

```bash
# Run all tests
go test ./pkg/tools/parser/...

# Run with coverage
go test -cover ./pkg/tools/parser/...

# Run benchmarks
go test -bench=. ./pkg/tools/parser/...

# Run fuzz tests
go test -fuzz=FuzzParser ./pkg/tools/parser/...
```

## Contributing

1. Follow the existing code style and patterns
2. Add tests for new functionality
3. Update documentation as needed
4. Ensure all tests pass before submitting

## License

Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2
