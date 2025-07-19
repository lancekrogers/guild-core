-- Sprint 6.5: Reasoning System Integration
-- Production hardening with indexes and state management

-- Add indexes for performance on reasoning_blocks
CREATE INDEX IF NOT EXISTS idx_reasoning_blocks_agent_timestamp 
ON reasoning_blocks(agent_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_reasoning_blocks_type_confidence 
ON reasoning_blocks(type, confidence);

CREATE INDEX IF NOT EXISTS idx_reasoning_blocks_session_id
ON reasoning_blocks(session_id);

-- Circuit breaker state table
CREATE TABLE IF NOT EXISTS reasoning_circuit_breaker (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL UNIQUE,
    state INTEGER NOT NULL DEFAULT 0, -- 0=closed, 1=open, 2=half-open
    failures INTEGER NOT NULL DEFAULT 0,
    successes INTEGER NOT NULL DEFAULT 0,
    last_failure_time TIMESTAMP,
    last_success_time TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_circuit_breaker_provider 
ON reasoning_circuit_breaker(provider);

-- Rate limiter state table
CREATE TABLE IF NOT EXISTS reasoning_rate_limits (
    agent_id TEXT PRIMARY KEY,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    window_start TIMESTAMP NOT NULL,
    last_request TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rate_limits_window_start 
ON reasoning_rate_limits(window_start);

-- Reasoning metrics table for historical analysis
CREATE TABLE IF NOT EXISTS reasoning_metrics (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    provider TEXT NOT NULL,
    extraction_count INTEGER NOT NULL DEFAULT 0,
    error_count INTEGER NOT NULL DEFAULT 0,
    avg_duration_ms REAL NOT NULL DEFAULT 0,
    p50_duration_ms REAL NOT NULL DEFAULT 0,
    p95_duration_ms REAL NOT NULL DEFAULT 0,
    p99_duration_ms REAL NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens INTEGER NOT NULL DEFAULT 0,
    content_tokens INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reasoning_metrics_timestamp 
ON reasoning_metrics(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_reasoning_metrics_provider_timestamp 
ON reasoning_metrics(provider, timestamp DESC);

-- Reasoning failures table for debugging
CREATE TABLE IF NOT EXISTS reasoning_failures (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    error_type TEXT NOT NULL,
    error_message TEXT,
    content_preview TEXT,
    stack_trace TEXT,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reasoning_failures_timestamp 
ON reasoning_failures(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_reasoning_failures_error_type 
ON reasoning_failures(error_type);

-- Add trigger to update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_circuit_breaker_updated_at
AFTER UPDATE ON reasoning_circuit_breaker
BEGIN
    UPDATE reasoning_circuit_breaker 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_rate_limits_updated_at
AFTER UPDATE ON reasoning_rate_limits
BEGIN
    UPDATE reasoning_rate_limits 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE agent_id = NEW.agent_id;
END;