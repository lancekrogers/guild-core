-- Create preferences table for hierarchical preference management
-- Supports System -> User -> Campaign -> Guild -> Agent hierarchy
CREATE TABLE IF NOT EXISTS preferences (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL CHECK (scope IN ('system', 'user', 'campaign', 'guild', 'agent')),
    scope_id TEXT, -- NULL for system scope, otherwise the ID of the scoped entity
    key TEXT NOT NULL,
    value JSON NOT NULL,
    version INTEGER DEFAULT 1,
    metadata JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_preference UNIQUE(scope, scope_id, key)
);

-- Indexes for efficient queries
CREATE INDEX idx_preferences_scope ON preferences(scope);
CREATE INDEX idx_preferences_scope_id ON preferences(scope_id);
CREATE INDEX idx_preferences_key ON preferences(key);
CREATE INDEX idx_preferences_updated_at ON preferences(updated_at);

-- Trigger to update updated_at timestamp
CREATE TRIGGER update_preferences_timestamp
AFTER UPDATE ON preferences
BEGIN
    UPDATE preferences SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- Table for preference inheritance relationships
-- This allows us to define custom inheritance chains
CREATE TABLE IF NOT EXISTS preference_inheritance (
    id TEXT PRIMARY KEY,
    child_scope TEXT NOT NULL,
    child_scope_id TEXT,
    parent_scope TEXT NOT NULL,
    parent_scope_id TEXT,
    priority INTEGER DEFAULT 0, -- Higher priority overrides lower
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_inheritance UNIQUE(child_scope, child_scope_id, parent_scope, parent_scope_id)
);

CREATE INDEX idx_preference_inheritance_child ON preference_inheritance(child_scope, child_scope_id);
CREATE INDEX idx_preference_inheritance_parent ON preference_inheritance(parent_scope, parent_scope_id);