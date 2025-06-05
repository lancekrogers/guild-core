-- Drop prompt chains tables
DROP INDEX IF EXISTS idx_prompt_chain_messages_timestamp;
DROP INDEX IF EXISTS idx_prompt_chain_messages_chain;
DROP INDEX IF EXISTS idx_prompt_chains_task;
DROP INDEX IF EXISTS idx_prompt_chains_agent;

DROP TABLE IF EXISTS prompt_chain_messages;
DROP TABLE IF EXISTS prompt_chains;