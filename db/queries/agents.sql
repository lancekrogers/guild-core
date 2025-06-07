-- name: CreateAgent :exec
INSERT INTO agents (id, name, type, provider, model, capabilities, tools, cost_magnitude)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetAgent :one
SELECT * FROM agents WHERE id = ?;

-- name: ListAgents :many
SELECT * FROM agents ORDER BY created_at DESC;

-- name: ListAgentsByType :many
SELECT * FROM agents WHERE type = ? ORDER BY created_at DESC;

-- name: UpdateAgent :exec
UPDATE agents
SET name = ?, type = ?, provider = ?, model = ?, capabilities = ?, tools = ?, cost_magnitude = ?
WHERE id = ?;

-- name: DeleteAgent :exec
DELETE FROM agents WHERE id = ?;
