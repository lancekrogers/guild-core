-- Drop triggers
DROP TRIGGER IF EXISTS update_rate_limits_updated_at;
DROP TRIGGER IF EXISTS update_circuit_breaker_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_reasoning_failures_error_type;
DROP INDEX IF EXISTS idx_reasoning_failures_timestamp;
DROP INDEX IF EXISTS idx_reasoning_metrics_provider_timestamp;
DROP INDEX IF EXISTS idx_reasoning_metrics_timestamp;
DROP INDEX IF EXISTS idx_rate_limits_window_start;
DROP INDEX IF EXISTS idx_circuit_breaker_provider;
DROP INDEX IF EXISTS idx_reasoning_blocks_session_id;
DROP INDEX IF EXISTS idx_reasoning_blocks_type_confidence;
DROP INDEX IF EXISTS idx_reasoning_blocks_agent_timestamp;

-- Drop tables
DROP TABLE IF EXISTS reasoning_failures;
DROP TABLE IF EXISTS reasoning_metrics;
DROP TABLE IF EXISTS reasoning_rate_limits;
DROP TABLE IF EXISTS reasoning_circuit_breaker;
DROP TABLE IF EXISTS reasoning_blocks;