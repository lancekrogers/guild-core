-- Add boards table to support Commission -> Board -> Task hierarchy
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

-- Add new board_id column to tasks table
ALTER TABLE tasks ADD COLUMN board_id TEXT REFERENCES boards(id);

-- Create index for board relationships
CREATE INDEX idx_boards_commission ON boards(commission_id);
CREATE INDEX idx_tasks_board ON tasks(board_id);

-- Note: We keep commission_id in tasks for backward compatibility during migration
-- In a future migration, we can remove commission_id after data migration is complete