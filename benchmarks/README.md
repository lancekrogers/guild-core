# Suggestion System Performance Benchmarks

This directory contains comprehensive performance benchmarks for the Guild Framework's suggestion system, designed to validate production enhancement performance targets.

## production enhancement Performance Targets

- **Latency**: <100ms average response time
- **Token Reduction**: 15-25% context optimization
- **Cache Hit Rate**: ≥80% for repeated queries
- **Memory Usage**: <1MB per service instance
- **Concurrent Performance**: Handle 50+ simultaneous users

## Quick Start

### Run All Benchmarks
```bash
make benchmark
```

### Run Suggestion-Specific Benchmarks
```bash
make benchmark-suggestions
```

### Run Individual Benchmark Suites
```bash
# Basic suggestion latency
go test -bench=BenchmarkSuggestionLatency -benchmem ./benchmarks

# Token optimization
go test -bench=BenchmarkTokenOptimization -benchmem ./benchmarks

# Concurrent access
go test -bench=BenchmarkConcurrentAccess -benchmem ./benchmarks

# Cache effectiveness
go test -bench=BenchmarkCacheEffectiveness -benchmem ./benchmarks

# Memory usage
go test -bench=BenchmarkMemoryUsage -benchmem ./benchmarks

# Load testing
go test -bench=BenchmarkLoadTest -benchmem -timeout=10m ./benchmarks
```

### Run Demo Benchmarks (Quick Validation)
```bash
go test -bench=BenchmarkSimple -benchmem ./benchmarks
go test -bench=BenchmarkCacheDemo -benchmem ./benchmarks
go test -bench=BenchmarkTokenOptimizationDemo -benchmem ./benchmarks
```

## Benchmark Suites

### 1. Suggestion Latency (`BenchmarkSuggestionLatency`)
Tests response time for different query types:
- Simple queries
- Complex multi-part queries  
- Follow-up queries with context

**Target**: <100ms average latency

### 2. Token Optimization (`BenchmarkTokenOptimization`)  
Validates context compression effectiveness:
- Various context sizes (1KB to 20KB)
- Token reduction percentage measurement
- Optimization algorithm performance

**Target**: 15-25% token reduction

### 3. Concurrent Access (`BenchmarkConcurrentAccess`)
Tests performance under concurrent load:
- 1, 5, 10, 20, 50 concurrent users
- Request latency under load
- Error rate measurement
- System stability validation

**Target**: Maintain <100ms P95 latency with 50 users

### 4. Cache Effectiveness (`BenchmarkCacheEffectiveness`)
Evaluates caching system performance:
- Cache hit rate measurement
- Cached vs uncached response times
- Cache speedup factor calculation
- Memory usage tracking

**Target**: ≥80% cache hit rate, 5x+ speedup

### 5. Memory Usage (`BenchmarkMemoryUsage`)
Monitors memory consumption:
- Service instance footprint
- Cache memory growth
- Memory leak detection
- Garbage collection impact

**Target**: <1MB per service instance

### 6. Provider Chain (`BenchmarkProviderChain`)
Tests suggestion provider efficiency:
- Single vs multiple provider performance
- Provider coordination overhead
- Parallel query execution

### 7. Integration Flow (`BenchmarkIntegrationFlow`)
Full suggestion system workflow:
- Complete chat conversation simulation
- End-to-end latency measurement
- Real-world usage patterns

### 8. Load Testing (`BenchmarkLoadTest`)
Comprehensive load testing:
- Sustained load over time (30s-5min)
- Ramp-up scenarios
- Stress testing to failure point
- Memory pressure testing

## Performance Report Generation

The benchmark suite generates comprehensive performance reports:

### JSON Report
```bash
go run benchmarks/run_benchmarks.go
# Generates: benchmarks/reports/performance_report_YYYY-MM-DD_HH-MM-SS.json
```

### Markdown Report
Human-readable performance summary with:
- Overall pass/fail status
- Detailed benchmark results
- Identified bottlenecks
- Optimization recommendations

## Understanding Results

### Benchmark Output Metrics
- `ns/op`: Nanoseconds per operation
- `avg_ms`: Average latency in milliseconds  
- `p95_ms`: 95th percentile latency
- `p99_ms`: 99th percentile latency
- `reduction_%`: Token reduction percentage
- `cache_hit_%`: Cache hit rate percentage
- `KB/service`: Memory usage per service

### Pass/Fail Criteria
Benchmarks automatically validate against production enhancement targets:
- ✅ **PASS**: Meets all performance targets
- ❌ **FAIL**: One or more targets not met

### Example Output
```
BenchmarkSuggestionLatency/SimpleQuery-8         1000    98523 ns/op    avg_ms:98.52    p95_ms:125.1
BenchmarkTokenOptimization/ContextSize_5000-8     500   245892 ns/op   reduction_%:18.5
BenchmarkCacheEffectiveness/CacheHitRate-8       2000    15234 ns/op   cache_hit_%:85.2
```

## Bottleneck Identification

The benchmark suite automatically identifies performance bottlenecks:

### Common Issues
1. **High Latency**: Suggestion retrieval >100ms
2. **Poor Token Reduction**: <15% context optimization
3. **Low Cache Hit Rate**: <80% cache effectiveness
4. **Memory Leaks**: Growing memory usage
5. **Concurrency Issues**: Performance degradation under load

### Optimization Recommendations
- Request batching and parallel queries
- Semantic compression for contexts
- LRU cache eviction policies
- Connection pooling
- Memory recycling with sync.Pool

## Performance Monitoring

### Continuous Integration
Add to CI/CD pipeline:
```yaml
- name: Performance Benchmarks
  run: make benchmark
  env:
    BENCHMARK_TIMEOUT: 10m
```

### Regular Monitoring
Run benchmarks regularly to detect performance regressions:
```bash
# Daily performance check
make benchmark-suggestions >> daily_perf.log

# Weekly comprehensive check  
make benchmark >> weekly_perf.log
```

### Performance Alerts
Set up alerts for:
- Average latency >100ms
- Cache hit rate <80%
- Error rate >1%
- Memory usage >1MB/service

## Troubleshooting

### Common Issues

#### Benchmark Failures
```bash
# Check system resources
htop
free -h

# Run with verbose output
go test -bench=Benchmark -benchmem -v ./benchmarks

# Run single benchmark for debugging
go test -bench=BenchmarkSuggestionLatency/SimpleQuery -benchmem -v ./benchmarks
```

#### Memory Issues
```bash
# Run with memory profiling
go test -bench=BenchmarkMemoryUsage -benchmem -memprofile=mem.prof ./benchmarks
go tool pprof mem.prof
```

#### Performance Regression
```bash
# Compare with baseline
go test -bench=. -benchmem -count=10 ./benchmarks > new.txt
benchcmp old.txt new.txt
```

### Environment Requirements
- Go 1.21+
- Minimum 4GB RAM
- SSD storage recommended
- Stable network connection for consistent results

## Advanced Usage

### Custom Benchmark Configuration
```go
config := LoadTestConfig{
    Duration:      60 * time.Second,
    Concurrency:   25,
    TargetLatency: 150 * time.Millisecond,
    TargetTPS:     100,
}
```

### Benchmark Profiling
```bash
# CPU profiling
go test -bench=BenchmarkSuggestionLatency -cpuprofile=cpu.prof ./benchmarks

# Memory profiling  
go test -bench=BenchmarkMemoryUsage -memprofile=mem.prof ./benchmarks

# Block profiling
go test -bench=BenchmarkConcurrentAccess -blockprofile=block.prof ./benchmarks
```

### Custom Metrics
Add custom metrics to benchmarks:
```go
b.ReportMetric(customValue, "custom_metric")
```

## Contributing

When adding new benchmarks:
1. Follow existing naming conventions
2. Include Sprint target validation
3. Add appropriate documentation
4. Test with various load levels
5. Update this README

### Benchmark Naming Convention
- `BenchmarkSuggestion*`: Core suggestion system tests
- `BenchmarkLoadTest*`: Load and stress tests  
- `BenchmarkMemory*`: Memory-related tests
- `BenchmarkCache*`: Cache performance tests
- `Benchmark*Demo`: Simple demonstration benchmarks