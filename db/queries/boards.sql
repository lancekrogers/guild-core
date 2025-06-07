-- name: CreateBoard :exec
INSERT INTO boards (
    id, commission_id, name, description, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

-- name: GetBoard :one
SELECT id, commission_id, name, description, status, created_at, updated_at FROM boards
WHERE id = ? LIMIT 1;

-- name: GetBoardByCommission :one
SELECT id, commission_id, name, description, status, created_at, updated_at FROM boards
WHERE commission_id = ? LIMIT 1;

-- name: UpdateBoard :exec
UPDATE boards
SET name = ?, description = ?, status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteBoard :exec
DELETE FROM boards WHERE id = ?;

-- name: ListBoards :many
SELECT id, commission_id, name, description, status, created_at, updated_at FROM boards
ORDER BY created_at DESC;
