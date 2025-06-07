-- name: CreatePromptChain :exec
INSERT INTO prompt_chains (id, agent_id, task_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?);

-- name: GetPromptChain :one
SELECT id, agent_id, task_id, created_at, updated_at
FROM prompt_chains
WHERE id = ?;

-- name: GetPromptChainsByAgent :many
SELECT id, agent_id, task_id, created_at, updated_at
FROM prompt_chains
WHERE agent_id = ?
ORDER BY created_at DESC;

-- name: GetPromptChainsByTask :many
SELECT id, agent_id, task_id, created_at, updated_at
FROM prompt_chains
WHERE task_id = ?
ORDER BY created_at DESC;

-- name: DeletePromptChain :exec
DELETE FROM prompt_chains
WHERE id = ?;

-- name: AddPromptChainMessage :exec
INSERT INTO prompt_chain_messages (chain_id, role, content, name, timestamp, token_usage)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetPromptChainMessages :many
SELECT id, chain_id, role, content, name, timestamp, token_usage
FROM prompt_chain_messages
WHERE chain_id = ?
ORDER BY timestamp ASC;

-- name: GetPromptChainMessagesWithLimit :many
SELECT id, chain_id, role, content, name, timestamp, token_usage
FROM prompt_chain_messages
WHERE chain_id = ?
ORDER BY timestamp DESC
LIMIT ?;

-- name: DeletePromptChainMessages :exec
DELETE FROM prompt_chain_messages
WHERE chain_id = ?;
