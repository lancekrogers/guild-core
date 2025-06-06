-- name: CreateTask :exec
INSERT INTO tasks (id, commission_id, board_id, title, description, status, column, story_points, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasks :many
SELECT * FROM tasks ORDER BY created_at DESC;

-- name: ListTasksByStatus :many
SELECT * FROM tasks WHERE status = ? ORDER BY created_at DESC;

-- name: ListTasksByCommission :many
SELECT * FROM tasks WHERE commission_id = ? ORDER BY created_at DESC;

-- name: ListTasksByBoard :many
SELECT * FROM tasks WHERE board_id = ? ORDER BY created_at DESC;

-- name: ListTasksForKanban :many
SELECT 
    t.*,
    a.name as agent_name,
    a.type as agent_type
FROM tasks t
LEFT JOIN agents a ON t.assigned_agent_id = a.id
WHERE t.board_id = ?
ORDER BY t.created_at;

-- name: AssignTaskToAgent :exec
UPDATE tasks 
SET assigned_agent_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateTaskStatus :exec
UPDATE tasks 
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateTaskColumn :exec
UPDATE tasks 
SET column = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateTask :exec
UPDATE tasks 
SET title = ?, description = ?, status = ?, column = ?, story_points = ?, metadata = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = ?;

-- name: DeleteTaskEvents :exec  
DELETE FROM task_events WHERE task_id = ?;

-- name: GetAgentWorkload :many
SELECT 
    a.id,
    a.name,
    COUNT(t.id) as task_count,
    SUM(CASE WHEN t.status = 'in_progress' THEN 1 ELSE 0 END) as active_tasks
FROM agents a
LEFT JOIN tasks t ON a.id = t.assigned_agent_id
GROUP BY a.id, a.name;

-- name: RecordTaskEvent :exec
INSERT INTO task_events (task_id, agent_id, event_type, old_value, new_value, reason)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetTaskHistory :many
SELECT * FROM task_events 
WHERE task_id = ? 
ORDER BY created_at DESC;