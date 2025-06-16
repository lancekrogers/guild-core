-- Drop all tables in reverse order of dependencies

DROP TRIGGER IF EXISTS update_session_timestamp;

DROP TABLE IF EXISTS session_bookmarks;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;
DROP TABLE IF EXISTS memory_store;
DROP TABLE IF EXISTS prompt_chain_messages;
DROP TABLE IF EXISTS prompt_chains;
DROP TABLE IF EXISTS task_events;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS boards;
DROP TABLE IF EXISTS commissions;
DROP TABLE IF EXISTS campaigns;