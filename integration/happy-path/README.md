# Guild Framework Happy Path Integration Tests

This directory contains comprehensive integration tests that validate the Guild Framework's infrastructure reliability, communication systems, and provider integration capabilities under real-world conditions.

## Overview

The Agent 3 integration infrastructure tests implement three critical testing areas:

1. **gRPC/Daemon Resilience Testing** - Validates daemon lifecycle management, health monitoring, and streaming capabilities
2. **Provider Integration Resilience Testing** - Tests multi-provider failover, circuit breakers, and cost-aware selection
3. **Network Resilience and Authentication Testing** - Validates network failure handling, authentication recovery, and security

## Test Structure

```
integration/happy-path/
├── grpc/
│   ├── daemon_lifecycle_test.go        # Daemon management and health monitoring
│   └── streaming_backpressure_test.go  # Streaming performance under load
├── providers/
│   └── failover_test.go               # Provider failover and selection
├── network/
│   └── resilience_test.go             # Network conditions and auth challenges
├── framework_test.go                  # Common testing utilities
└── README.md                          # This file
```

## Running the Tests

### Prerequisites

1. Go 1.21+ installed
2. Guild Framework dependencies resolved (`go mod tidy`)
3. Access to test providers (optional, tests use mocks by default)

### Individual Test Suites

Run specific test suites:

```bash
# gRPC/Daemon resilience tests
cd guild-core
go test ./integration/happy-path/grpc -v

# Provider integration tests  
go test ./integration/happy-path/providers -v

# Network resilience tests
go test ./integration/happy-path/network -v

# Framework utilities tests
go test ./integration/happy-path -v
```

### All Happy Path Tests

Run all infrastructure tests:

```bash
cd guild-core
go test ./integration/happy-path/... -v
```

### With Coverage

Generate test coverage reports:

```bash
go test ./integration/happy-path/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## Test Scenarios

### gRPC/Daemon Resilience Testing (12 points)

#### Daemon Lifecycle Test (`daemon_lifecycle_test.go`)

**Scenarios Tested:**

- Clean daemon lifecycle with health monitoring
- Daemon crash recovery with automatic restart
- Network partition recovery with graceful degradation
- Resource exhaustion handling with circuit breakers

**Success Criteria:**

- ≥99.9% daemon uptime with automated recovery
- ≤3 seconds recovery time from process crashes
- ≤5 seconds recovery from network partitions  
- ≤10 seconds recovery from resource exhaustion
- Memory usage ≤500MB during peak load
- CPU usage ≤50% under normal operations

**Key Validations:**

- Connection pooling and retry logic
- Health check responsiveness (≤1 second detection)
- Concurrent client handling (up to 20 clients)
- Resource limit enforcement
- Circuit breaker effectiveness

#### Streaming Backpressure Test (`streaming_backpressure_test.go`)

**Scenarios Tested:**

- Low volume streaming (5 streams, 1KB messages)
- High volume with backpressure (20 streams, 4KB messages)
- Large message streaming (10 streams, 64KB messages)

**Success Criteria:**

- ≥5000 messages/second throughput for high volume
- ≤500ms average latency across all streams
- ≤10% backpressure event rate
- Memory usage ≤200MB during streaming
- CPU usage ≤80% under load

**Key Validations:**

- Flow control window management
- Backpressure detection and handling
- Message batching effectiveness
- Resource efficiency during streaming
- Graceful degradation under load

### Provider Integration Resilience Testing (10 points)

#### Multi-Provider Failover Test (`failover_test.go`)

**Scenarios Tested:**

- Simple failover between two providers (OpenAI → Anthropic)
- Complex multi-provider cascade (OpenAI → Anthropic → Local)
- Rate limiting and cost optimization
- Provider health monitoring and recovery

**Success Criteria:**

- ≤2 seconds failover time with zero request loss
- ≥95% overall success rate during failures
- Cost-aware provider selection (±10% optimal)
- Circuit breaker activation within 3 failed requests
- Automatic failback after recovery

**Key Validations:**

- Intelligent provider selection algorithms
- Circuit breaker state management
- Request routing during failures
- Cost tracking accuracy
- Provider health assessment

### Network Resilience and Authentication Testing (8 points)

#### Network Resilience Test (`resilience_test.go`)

**Scenarios Tested:**

- Intermittent connectivity (packet loss, high latency)
- Complete network partitions with auth token expiry
- TLS certificate rotation and validation
- Authentication server failures

**Success Criteria:**

- ≥95% communication success rate overall
- ≤60 seconds recovery from network partitions
- ≤5 authentication attempts for recovery
- Zero security violations during tests
- TLS 1.3 enforcement with mutual authentication

**Key Validations:**

- Exponential backoff implementation
- Authentication token renewal
- Certificate validation and rotation
- Secure communication maintenance
- Graceful degradation patterns

## Performance Benchmarks

### Expected Performance Targets

| Metric | Target | Tolerance |
|--------|--------|-----------|
| Daemon startup time | ≤2 seconds | ±20% |
| Provider failover time | ≤2 seconds | ±50% |
| Streaming throughput | ≥5000 msg/s | -20% |
| Memory usage (peak) | ≤500MB | +20% |
| CPU usage (avg) | ≤50% | +20% |
| Network recovery time | ≤60 seconds | +50% |
| Authentication recovery | ≤30 seconds | +100% |

### Resource Limits

- **Memory**: 500MB maximum per daemon instance
- **CPU**: 80% maximum sustained usage
- **Goroutines**: 1000 maximum concurrent
- **File descriptors**: 1024 maximum open
- **Network connections**: 100 concurrent maximum

## Debugging and Troubleshooting

### Verbose Logging

Enable detailed logging for debugging:

```bash
export GUILD_LOG_LEVEL=debug
go test ./integration/happy-path/... -v -args -test.v
```

### Common Issues

1. **Port conflicts**: Tests automatically allocate ports, but conflicts may occur
   - Solution: Use `GUILD_TEST_PORT_BASE` environment variable

2. **Timeout failures**: Network conditions may cause timeouts
   - Solution: Increase timeout values or check network connectivity

3. **Resource exhaustion**: Tests may consume significant resources
   - Solution: Run tests individually or increase system limits

4. **Provider authentication**: Real provider tests require valid credentials
   - Solution: Configure test credentials or use mock providers

### Test Data

Tests generate realistic project structures and commission content:

- Mock Go projects with proper module structure
- Realistic commission complexity levels (simple/medium/complex)
- Generated code samples and documentation
- Configurable test scenarios and parameters

## Integration with CI/CD

### GitHub Actions Configuration

```yaml
name: Happy Path Integration Tests
on: [push, pull_request]

jobs:
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run Happy Path Tests
        run: |
          cd guild-core
          go test ./integration/happy-path/... -v -timeout=30m
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### Performance Regression Detection

Tests include baseline performance validation:

- Automatic comparison against historical benchmarks
- Alerting when performance degrades >20%
- Trend analysis for gradual performance changes
- Resource usage monitoring and alerting

## Quality Gates

All tests must pass these quality gates:

### System Reliability

- [ ] ≥99.9% daemon uptime during tests
- [ ] ≤2 second average recovery time
- [ ] Zero data loss during failures
- [ ] Resource usage within limits

### Provider Integration

- [ ] ≤2 second failover time
- [ ] ≥95% request success rate
- [ ] Cost variance ≤10% from optimal
- [ ] Circuit breaker effectiveness

### Network Resilience  

- [ ] ≥95% communication success rate
- [ ] Zero security violations
- [ ] TLS integrity maintained
- [ ] Graceful auth recovery

### Test Quality

- [ ] ≥90% test coverage
- [ ] All scenarios pass consistently
- [ ] Performance targets met
- [ ] No memory leaks detected

## Contributing

When adding new integration tests:

1. Follow the established patterns in existing tests
2. Include comprehensive failure scenarios
3. Validate both happy path and edge cases
4. Add appropriate performance benchmarks
5. Update this documentation

### Test Naming Conventions

- Test functions: `TestComponentName_HappyPath`
- Scenarios: Descriptive names explaining the test case
- Mock types: `Mock` + component name (e.g., `MockProvider`)
- Metrics: Component + `Metrics` suffix

### Medieval Naming Support

The codebase supports medieval terminology where appropriate:

- Agents = Artisans
- Agent pools = Guilds  
- Tasks = Commissions
- Tools = Implements
- Memory = Archives

New tests should use this terminology when integrating with existing components.

## Results and Reporting

Tests generate comprehensive reports including:

- Performance metrics and trends
- Resource usage statistics
- Failure analysis and recovery times
- Success rates and error breakdowns
- Security compliance validation

Results are formatted for both human consumption and automated analysis, supporting continuous integration and performance monitoring workflows.
