-- Drop agent performance tracking tables

DROP TRIGGER IF EXISTS update_agent_availability_timestamp;
DROP TRIGGER IF EXISTS update_agent_capabilities_timestamp;
DROP TRIGGER IF EXISTS update_agent_specialties_timestamp;
DROP TRIGGER IF EXISTS update_agent_performance_timestamp;

DROP INDEX IF EXISTS idx_task_assignments_completed;
DROP INDEX IF EXISTS idx_task_assignments_agent;
DROP INDEX IF EXISTS idx_task_assignments_task;
DROP INDEX IF EXISTS idx_agent_availability_status;
DROP INDEX IF EXISTS idx_agent_availability_agent;
DROP INDEX IF EXISTS idx_agent_capabilities_capability;
DROP INDEX IF EXISTS idx_agent_capabilities_agent;
DROP INDEX IF EXISTS idx_agent_specialties_specialty;
DROP INDEX IF EXISTS idx_agent_specialties_agent;
DROP INDEX IF EXISTS idx_agent_performance_agent;

DROP TABLE IF EXISTS task_assignments;
DROP TABLE IF EXISTS agent_availability;
DROP TABLE IF EXISTS agent_capabilities;
DROP TABLE IF EXISTS agent_specialties;
DROP TABLE IF EXISTS agent_performance;