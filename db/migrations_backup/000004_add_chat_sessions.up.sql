-- Add chat sessions table to store persistent chat conversations
CREATE TABLE chat_sessions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    campaign_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id)
);

-- Add chat messages table to store individual messages
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

-- Add session bookmarks table for marking important messages
CREATE TABLE session_bookmarks (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE CASCADE
);

-- Create indexes for efficient lookups
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