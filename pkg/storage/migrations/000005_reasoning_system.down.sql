-- Drop reasoning system tables

-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS reasoning_insights;
DROP TABLE IF EXISTS reasoning_analytics;
DROP TABLE IF EXISTS reasoning_patterns;
DROP TABLE IF EXISTS reasoning_chains;