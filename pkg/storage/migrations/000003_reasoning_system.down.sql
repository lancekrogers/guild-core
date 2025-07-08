-- Drop reasoning system tables

DROP TRIGGER IF EXISTS update_reasoning_patterns_timestamp;

DROP INDEX IF EXISTS idx_reasoning_analytics_time;
DROP INDEX IF EXISTS idx_reasoning_analytics_agent;

DROP INDEX IF EXISTS idx_reasoning_patterns_occurrences;
DROP INDEX IF EXISTS idx_reasoning_patterns_task_type;

DROP INDEX IF EXISTS idx_reasoning_chains_success;
DROP INDEX IF EXISTS idx_reasoning_chains_task_type;
DROP INDEX IF EXISTS idx_reasoning_chains_confidence;
DROP INDEX IF EXISTS idx_reasoning_chains_created;
DROP INDEX IF EXISTS idx_reasoning_chains_session;
DROP INDEX IF EXISTS idx_reasoning_chains_agent;

DROP TABLE IF EXISTS reasoning_analytics;
DROP TABLE IF EXISTS reasoning_patterns;
DROP TABLE IF EXISTS reasoning_chains;