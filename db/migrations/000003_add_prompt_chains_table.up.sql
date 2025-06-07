-- Add prompt chains table to store conversation history
CREATE TABLE prompt_chains (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    task_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add prompt chain messages table
CREATE TABLE prompt_chain_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chain_id TEXT NOT NULL REFERENCES prompt_chains(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT NOT NULL,
    name TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    token_usage INTEGER DEFAULT 0,
    FOREIGN KEY (chain_id) REFERENCES prompt_chains(id)
);

-- Create indexes for efficient lookups
CREATE INDEX idx_prompt_chains_agent ON prompt_chains(agent_id);
CREATE INDEX idx_prompt_chains_task ON prompt_chains(task_id);
CREATE INDEX idx_prompt_chain_messages_chain ON prompt_chain_messages(chain_id);
CREATE INDEX idx_prompt_chain_messages_timestamp ON prompt_chain_messages(timestamp);
