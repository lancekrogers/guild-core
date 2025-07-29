-- Reasoning system tables for Guild Framework
-- Stores reasoning chains, patterns, and analytics

-- Reasoning chains from agent executions
CREATE TABLE reasoning_chains (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    session_id TEXT,
    request_id TEXT,
    content TEXT NOT NULL,
    reasoning TEXT NOT NULL,
    confidence REAL NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    task_type TEXT,
    success BOOLEAN NOT NULL DEFAULT true,
    tokens_used INTEGER DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON,
    FOREIGN KEY (session_id) REFERENCES chat_sessions(id) ON DELETE SET NULL
);

-- Learned reasoning patterns
CREATE TABLE reasoning_patterns (
    id TEXT PRIMARY KEY,
    pattern TEXT NOT NULL,
    task_type TEXT,
    occurrences INTEGER NOT NULL DEFAULT 1,
    avg_success REAL DEFAULT 0 CHECK (avg_success >= 0 AND avg_success <= 1),
    examples JSON, -- Array of chain IDs
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSON
);

-- Reasoning analytics cache
CREATE TABLE reasoning_analytics (
    id TEXT PRIMARY KEY,
    agent_id TEXT,
    time_range TEXT NOT NULL, -- e.g., "2024-01-15_daily", "2024-W03_weekly"
    total_chains INTEGER NOT NULL DEFAULT 0,
    avg_confidence REAL DEFAULT 0,
    avg_duration_ms INTEGER DEFAULT 0,
    success_rate REAL DEFAULT 0,
    confidence_distribution JSON,
    task_type_distribution JSON,
    insights JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
);

-- Reasoning insights for agent optimization
CREATE TABLE reasoning_insights (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    pattern_id TEXT NOT NULL REFERENCES reasoning_patterns(id),
    insight TEXT NOT NULL,
    confidence REAL NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    validation_status TEXT DEFAULT 'pending' CHECK (validation_status IN ('pending', 'validated', 'rejected')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    validated_at TIMESTAMP,
    metadata JSON
);

-- Indexes for performance
CREATE INDEX idx_reasoning_chains_agent_id ON reasoning_chains(agent_id);
CREATE INDEX idx_reasoning_chains_session_id ON reasoning_chains(session_id);
CREATE INDEX idx_reasoning_chains_task_type ON reasoning_chains(task_type);
CREATE INDEX idx_reasoning_chains_created_at ON reasoning_chains(created_at);
CREATE INDEX idx_reasoning_chains_success ON reasoning_chains(success);

CREATE INDEX idx_reasoning_patterns_task_type ON reasoning_patterns(task_type);
CREATE INDEX idx_reasoning_patterns_occurrences ON reasoning_patterns(occurrences);
CREATE INDEX idx_reasoning_patterns_avg_success ON reasoning_patterns(avg_success);

CREATE INDEX idx_reasoning_insights_agent_id ON reasoning_insights(agent_id);
CREATE INDEX idx_reasoning_insights_pattern_id ON reasoning_insights(pattern_id);
CREATE INDEX idx_reasoning_insights_validation_status ON reasoning_insights(validation_status);
CREATE INDEX idx_reasoning_insights_created_at ON reasoning_insights(created_at);

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_reasoning_patterns_timestamp 
    AFTER UPDATE ON reasoning_patterns 
    FOR EACH ROW
    BEGIN
        UPDATE reasoning_patterns 
        SET updated_at = CURRENT_TIMESTAMP 
        WHERE id = NEW.id;
    END;