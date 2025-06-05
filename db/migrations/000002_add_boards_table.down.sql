-- Remove board-related changes
DROP INDEX IF EXISTS idx_tasks_board;
DROP INDEX IF EXISTS idx_boards_commission;

-- Remove board_id column from tasks
ALTER TABLE tasks DROP COLUMN board_id;

-- Drop boards table
DROP TABLE boards;