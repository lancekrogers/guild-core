DROP INDEX IF EXISTS idx_commissions_campaign;
DROP INDEX IF EXISTS idx_task_events_task;
DROP INDEX IF EXISTS idx_tasks_agent;
DROP INDEX IF EXISTS idx_tasks_commission;
DROP INDEX IF EXISTS idx_tasks_status;

DROP TABLE IF EXISTS task_events;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS commissions;
DROP TABLE IF EXISTS campaigns;
