-- Complete schema for Guild Framework MVP
-- This is the single source of truth for the database schema

-- Core campaign management
CREATE TABLE campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Commissions (objectives) tied to campaigns
CREATE TABLE commissions (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL REFERENCES campaigns(id),
    title TEXT NOT NULL,
    description TEXT,
    domain TEXT,
    context JSON,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Boards for organizing tasks within commissions
CREATE TABLE boards (
    id TEXT PRIMARY KEY,
    commission_id TEXT NOT NULL REFERENCES commissions(id),
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(commission_id) -- Ensures one board per commission
);

-- AI agents configuration
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- manager, worker, specialist
    provider TEXT,
    model TEXT,
    capabilities JSON,
    tools JSON,
    cost_magnitude INTEGER DEFAULT 2,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks assigned to agents
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    commission_id TEXT NOT NULL REFERENCES commissions(id),
    board_id TEXT REFERENCES boards(id),
    assigned_agent_id TEXT REFERENCES agents(id),
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'blocked', 'pending_review', 'done')),
    column TEXT NOT NULL DEFAULT 'backlog',
    story_points INTEGER DEFAULT 1,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Task history tracking
CREATE TABLE task_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    agent_id TEXT REFERENCES agents(id),
    event_type TEXT NOT NULL, -- created, assigned, started, completed, blocked
    old_value TEXT,
    new_value TEXT,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Prompt chains for agent conversations
CREATE TABLE prompt_chains (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    task_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Messages within prompt chains
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

-- Chat sessions for persistent conversations
CREATE TABLE chat_sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    campaign_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
);

-- Individual messages in chat sessions
CREATE TABLE chat_messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'tool')),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tool_calls JSON,
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE
);

-- Bookmarks for important messages
CREATE TABLE session_bookmarks (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE
);

-- Memory store for general key-value storage
CREATE TABLE memory_store (
    bucket TEXT NOT NULL,
    key TEXT NOT NULL,
    value BLOB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (bucket, key)
);

-- Performance indexes
CREATE INDEX idx_commissions_campaign ON commissions(campaign_id);
CREATE INDEX idx_boards_commission ON boards(commission_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_commission ON tasks(commission_id);
CREATE INDEX idx_tasks_board ON tasks(board_id);
CREATE INDEX idx_tasks_agent ON tasks(assigned_agent_id);
CREATE INDEX idx_task_events_task ON task_events(task_id);
CREATE INDEX idx_prompt_chains_agent ON prompt_chains(agent_id);
CREATE INDEX idx_prompt_chains_task ON prompt_chains(task_id);
CREATE INDEX idx_prompt_chain_messages_chain ON prompt_chain_messages(chain_id);
CREATE INDEX idx_prompt_chain_messages_timestamp ON prompt_chain_messages(timestamp);
CREATE INDEX idx_chat_sessions_campaign ON chat_sessions(campaign_id);
CREATE INDEX idx_chat_sessions_updated ON chat_sessions(updated_at);
CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_created ON chat_messages(created_at);
CREATE INDEX idx_chat_messages_role ON chat_messages(role);
CREATE INDEX idx_session_bookmarks_session ON session_bookmarks(session_id);
CREATE INDEX idx_session_bookmarks_message ON session_bookmarks(message_id);
CREATE INDEX idx_session_bookmarks_created ON session_bookmarks(created_at);

-- Trigger to update chat_sessions.updated_at on new messages
CREATE TRIGGER update_session_timestamp
AFTER INSERT ON chat_messages
BEGIN
    UPDATE chat_sessions 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.session_id;
END;