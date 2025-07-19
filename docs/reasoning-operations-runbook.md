# Guild Reasoning & Intelligence Layer - Operations Runbook

## Overview

This runbook provides operational guidance for maintaining, monitoring, and troubleshooting the Guild Reasoning & Intelligence Layer in production environments.

## Table of Contents

1. [System Architecture](#system-architecture)
2. [Deployment](#deployment)
3. [Monitoring](#monitoring)
4. [Common Issues & Troubleshooting](#common-issues--troubleshooting)
5. [Performance Tuning](#performance-tuning)
6. [Emergency Procedures](#emergency-procedures)
7. [Maintenance Tasks](#maintenance-tasks)

## System Architecture

### Components

```
┌─────────────────────────────────────────────────────────┐
│                    Client Applications                   │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│                  Reasoning Registry                      │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │Rate Limiter │→ │    Retryer   │→ │Circuit Breaker│   │
│  └─────────────┘  └──────────────┘  └──────────────┘   │
│                         │                                │
│                         ▼                                │
│                  ┌──────────────┐                       │
│                  │  Extractor   │                       │
│                  └──────────────┘                       │
│                         │                                │
│           ┌─────────────┴─────────────┐                 │
│           ▼                           ▼                 │
│    ┌─────────────┐            ┌──────────────┐         │
│    │ Event Bus   │            │Dead Letter Q │         │
│    └─────────────┘            └──────────────┘         │
└─────────────────────────────────────────────────────────┘
```

### Key Features

- **Rate Limiting**: Global and per-agent rate limits
- **Circuit Breaker**: Prevents cascade failures
- **Retry Logic**: Exponential backoff with jitter
- **Dead Letter Queue**: Captures failed extractions
- **Health Monitoring**: Real-time component health
- **Distributed Tracing**: Full request tracing
- **Metrics Collection**: Comprehensive performance metrics

## Deployment

### Prerequisites

- Go 1.21+
- PostgreSQL 14+ (for dead letter queue)
- Prometheus (for metrics)
- Grafana (for dashboards)
- OpenTelemetry Collector (for tracing)

### Configuration

```yaml
# config/reasoning.yaml
reasoning:
  circuit_breaker:
    failure_threshold: 10
    success_threshold: 5
    timeout: 30s
    max_half_open_calls: 10
    observation_window: 60s
    
  rate_limiter:
    global_rps: 1000
    per_agent_rps: 50
    burst_size: 10
    max_agents: 1000
    cleanup_interval: 5m
    
  retry:
    max_attempts: 3
    initial_delay: 100ms
    max_delay: 5s
    multiplier: 2.0
    jitter: 0.1
    
  dead_letter:
    max_retries: 5
    retention_period: 168h # 7 days
    cleanup_interval: 1h
```

### Database Migration

```bash
# Run migrations
guild migrate up --component reasoning

# Verify migration
guild migrate status --component reasoning
```

### Starting the Service

```bash
# Start with default config
guild start --enable-reasoning

# Start with custom config
guild start --enable-reasoning --config /path/to/config.yaml

# Verify service is running
guild health --component reasoning
```

## Monitoring

### Key Metrics

#### Request Metrics
- `reasoning_extraction_total` - Total extractions by status
- `reasoning_extraction_duration_seconds` - Extraction latency
- `reasoning_extraction_errors_total` - Error counts by type

#### Circuit Breaker Metrics
- `reasoning_circuit_breaker_state` - Current state (0=closed, 1=open, 2=half-open)
- `reasoning_circuit_breaker_trips_total` - State transitions

#### Rate Limiter Metrics
- `reasoning_rate_limiter_usage_ratio` - Usage as ratio of limit
- `reasoning_rate_limit_hits_total` - Rate limit rejections

#### System Health
- `reasoning_active_extractions` - Currently processing
- `reasoning_dead_letter_queue_size` - Unprocessed failures

### Grafana Dashboard

Import the dashboard from `deployment/dashboards/reasoning.json`:

```bash
# Using Grafana API
curl -X POST http://grafana:3000/api/dashboards/db \
  -H "Authorization: Bearer $GRAFANA_API_KEY" \
  -H "Content-Type: application/json" \
  -d @deployment/dashboards/reasoning.json
```

### Alert Rules

```yaml
# alerts/reasoning.yaml
groups:
  - name: reasoning
    rules:
      - alert: HighErrorRate
        expr: rate(reasoning_extraction_errors_total[5m]) > 0.05
        for: 5m
        annotations:
          summary: "High error rate in reasoning extraction"
          
      - alert: CircuitBreakerOpen
        expr: reasoning_circuit_breaker_state == 1
        for: 1m
        annotations:
          summary: "Circuit breaker is open"
          
      - alert: DeadLetterQueueGrowing
        expr: rate(reasoning_dead_letter_queue_size[5m]) > 0
        for: 10m
        annotations:
          summary: "Dead letter queue is growing"
```

## Common Issues & Troubleshooting

### Issue: High Latency

**Symptoms:**
- P99 latency > 1s
- Increasing queue sizes

**Diagnosis:**
```bash
# Check active extractions
guild metrics get reasoning_active_extractions

# Check circuit breaker state
guild metrics get reasoning_circuit_breaker_state

# View recent errors
guild logs --component reasoning --level error --since 10m
```

**Resolution:**
1. Check provider API status
2. Verify network connectivity
3. Review extraction patterns for complexity
4. Consider increasing rate limits temporarily

### Issue: Circuit Breaker Open

**Symptoms:**
- All requests failing with "circuit breaker open"
- `reasoning_circuit_breaker_state` = 1

**Diagnosis:**
```bash
# Check failure history
guild reasoning circuit-breaker status

# View error logs
guild logs --component reasoning --grep "circuit breaker" --since 30m
```

**Resolution:**
1. Identify root cause of failures
2. Fix underlying issue
3. Reset circuit breaker if needed:
   ```bash
   guild reasoning circuit-breaker reset
   ```

### Issue: Rate Limiting

**Symptoms:**
- 429 errors
- `reasoning_rate_limit_hits_total` increasing

**Diagnosis:**
```bash
# Check current usage
guild reasoning rate-limiter status

# Identify heavy users
guild reasoning rate-limiter top-agents --limit 10
```

**Resolution:**
1. Review agent configurations
2. Implement client-side rate limiting
3. Adjust limits if legitimate:
   ```bash
   guild config set reasoning.rate_limiter.per_agent_rps=100
   ```

### Issue: Dead Letter Queue Growing

**Symptoms:**
- `reasoning_dead_letter_queue_size` increasing
- Failed extractions not being processed

**Diagnosis:**
```bash
# View dead letter entries
guild reasoning dead-letter list --limit 10

# Check specific failure
guild reasoning dead-letter inspect <entry-id>
```

**Resolution:**
1. Identify common failure patterns
2. Fix root causes
3. Reprocess entries:
   ```bash
   guild reasoning dead-letter reprocess --batch-size 100
   ```

## Performance Tuning

### Rate Limiter Tuning

```bash
# Monitor current usage
guild metrics query 'reasoning_rate_limiter_usage_ratio'

# Adjust based on usage patterns
guild config set reasoning.rate_limiter.global_rps=2000
guild config set reasoning.rate_limiter.burst_size=20
```

### Circuit Breaker Tuning

```bash
# Check trip frequency
guild metrics query 'rate(reasoning_circuit_breaker_trips_total[1h])'

# Adjust thresholds
guild config set reasoning.circuit_breaker.failure_threshold=15
guild config set reasoning.circuit_breaker.observation_window=120s
```

### Database Optimization

```sql
-- Analyze query performance
EXPLAIN ANALYZE
SELECT * FROM reasoning_dead_letter
WHERE status = 'unprocessed'
ORDER BY created_at DESC
LIMIT 100;

-- Update statistics
ANALYZE reasoning_dead_letter;
ANALYZE reasoning_metrics;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan;
```

## Emergency Procedures

### Complete System Failure

1. **Immediate Actions:**
   ```bash
   # Check system health
   guild health --all
   
   # Review recent errors
   guild logs --level error --since 5m
   
   # Check database connectivity
   guild db ping
   ```

2. **Failover:**
   ```bash
   # Switch to backup instance
   guild failover activate --component reasoning
   
   # Verify failover
   guild health --component reasoning
   ```

3. **Recovery:**
   ```bash
   # Replay missed events
   guild events replay --component reasoning --since <timestamp>
   
   # Process dead letter queue
   guild reasoning dead-letter reprocess --all
   ```

### Memory Leak

1. **Identify:**
   ```bash
   # Get memory profile
   guild debug pprof --type heap
   
   # Monitor growth
   watch -n 5 'guild metrics get process_resident_memory_bytes'
   ```

2. **Mitigate:**
   ```bash
   # Restart with memory limit
   guild restart --component reasoning --memory-limit 2G
   
   # Enable aggressive GC
   guild config set runtime.gc_percent=50
   ```

### Provider Outage

1. **Detection:**
   ```bash
   # Check provider status
   guild providers health
   
   # View provider-specific errors
   guild logs --grep "provider.*error" --since 10m
   ```

2. **Mitigation:**
   ```bash
   # Switch to backup provider
   guild config set reasoning.default_provider=anthropic
   
   # Enable provider rotation
   guild config set reasoning.provider_rotation.enabled=true
   ```

## Maintenance Tasks

### Daily

1. **Health Check:**
   ```bash
   guild health --component reasoning --verbose
   ```

2. **Metrics Review:**
   - Check error rates
   - Review latency trends
   - Monitor queue sizes

### Weekly

1. **Dead Letter Processing:**
   ```bash
   # Review old entries
   guild reasoning dead-letter list --older-than 3d
   
   # Clean up resolved entries
   guild reasoning dead-letter cleanup --status processed
   ```

2. **Performance Analysis:**
   ```bash
   # Generate performance report
   guild report generate --component reasoning --period 7d
   ```

### Monthly

1. **Database Maintenance:**
   ```sql
   -- Vacuum and analyze
   VACUUM ANALYZE reasoning_dead_letter;
   VACUUM ANALYZE reasoning_metrics;
   
   -- Check table sizes
   SELECT 
     schemaname,
     tablename,
     pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename))
   FROM pg_tables
   WHERE schemaname = 'public'
   ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
   ```

2. **Configuration Review:**
   ```bash
   # Backup current config
   guild config backup --component reasoning
   
   # Review and optimize settings
   guild config audit --component reasoning
   ```

3. **Capacity Planning:**
   ```bash
   # Generate growth report
   guild report capacity --component reasoning --forecast 3m
   ```

## Appendix

### Useful Commands

```bash
# Live tail of reasoning logs
guild logs --component reasoning --follow

# Export metrics for analysis
guild metrics export --component reasoning --format csv

# Test extraction
guild reasoning test --content "test content" --agent-id test-agent

# Benchmark performance
guild benchmark --component reasoning --duration 60s
```

### Support Contacts

- **On-Call**: Use PagerDuty
- **Engineering Team**: #guild-reasoning (Slack)
- **Escalation**: reasoning-oncall@guild.ai

### References

- [API Documentation](./api/reasoning.md)
- [Architecture Design](./architecture/reasoning-layer.md)
- [Provider Integration Guide](./guides/provider-integration.md)