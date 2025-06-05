-- name: CreateObjective :exec
INSERT INTO objectives (
    id, title, description, status, goal, priority, owner, 
    completion, iteration, campaign_id, source, tags, metadata, 
    context, assignees, requirements, related, ai_docs, specs
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetObjective :one
SELECT * FROM objectives WHERE id = ?;

-- name: UpdateObjective :exec
UPDATE objectives 
SET title = ?, description = ?, status = ?, goal = ?, priority = ?, 
    owner = ?, completion = ?, iteration = ?, campaign_id = ?, 
    source = ?, tags = ?, metadata = ?, context = ?, assignees = ?, 
    requirements = ?, related = ?, ai_docs = ?, specs = ?, 
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateObjectiveStatus :exec
UPDATE objectives 
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: SetObjectiveCompleted :exec
UPDATE objectives 
SET status = 'completed', completed_at = CURRENT_TIMESTAMP, 
    completion = 1.0, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteObjective :exec
DELETE FROM objectives WHERE id = ?;

-- name: ListObjectives :many
SELECT * FROM objectives ORDER BY created_at DESC;

-- name: ListObjectivesByStatus :many
SELECT * FROM objectives WHERE status = ? ORDER BY created_at DESC;

-- name: ListObjectivesByOwner :many
SELECT * FROM objectives WHERE owner = ? ORDER BY created_at DESC;

-- name: ListObjectivesByCampaign :many
SELECT * FROM objectives WHERE campaign_id = ? ORDER BY created_at DESC;

-- name: IncrementObjectiveIteration :exec
UPDATE objectives 
SET iteration = iteration + 1, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateObjectiveCompletion :exec
UPDATE objectives 
SET completion = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- Objective Parts queries
-- name: CreateObjectivePart :exec
INSERT INTO objective_parts (id, objective_id, title, content, type, sort_order, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetObjectivePart :one
SELECT * FROM objective_parts WHERE id = ?;

-- name: ListObjectiveParts :many
SELECT * FROM objective_parts WHERE objective_id = ? ORDER BY sort_order;

-- name: UpdateObjectivePart :exec
UPDATE objective_parts 
SET title = ?, content = ?, type = ?, sort_order = ?, metadata = ?
WHERE id = ?;

-- name: DeleteObjectivePart :exec
DELETE FROM objective_parts WHERE id = ?;

-- name: DeleteObjectiveParts :exec
DELETE FROM objective_parts WHERE objective_id = ?;

-- Objective Tasks queries
-- name: CreateObjectiveTask :exec
INSERT INTO objective_tasks (
    id, objective_id, title, description, status, assignee, 
    parent_id, sort_order, dependencies, metadata
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetObjectiveTask :one
SELECT * FROM objective_tasks WHERE id = ?;

-- name: ListObjectiveTasks :many
SELECT * FROM objective_tasks WHERE objective_id = ? ORDER BY sort_order;

-- name: ListObjectiveTasksByStatus :many
SELECT * FROM objective_tasks 
WHERE objective_id = ? AND status = ? 
ORDER BY sort_order;

-- name: UpdateObjectiveTask :exec
UPDATE objective_tasks 
SET title = ?, description = ?, status = ?, assignee = ?, 
    parent_id = ?, sort_order = ?, dependencies = ?, metadata = ?, 
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateObjectiveTaskStatus :exec
UPDATE objective_tasks 
SET status = ?, updated_at = CURRENT_TIMESTAMP,
    completed_at = CASE WHEN ? = 'done' THEN CURRENT_TIMESTAMP ELSE NULL END
WHERE id = ?;

-- name: DeleteObjectiveTask :exec
DELETE FROM objective_tasks WHERE id = ?;

-- name: DeleteObjectiveTasks :exec
DELETE FROM objective_tasks WHERE objective_id = ?;

-- name: CountObjectiveTasksByStatus :one
SELECT 
    COUNT(*) as total,
    SUM(CASE WHEN status = 'done' THEN 1 ELSE 0 END) as completed
FROM objective_tasks 
WHERE objective_id = ?;