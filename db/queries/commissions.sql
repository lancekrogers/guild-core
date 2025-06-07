-- name: CreateCommission :exec
INSERT INTO commissions (
    id, campaign_id, title, description, domain, context, status
) VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetCommission :one
SELECT * FROM commissions WHERE id = ?;

-- name: UpdateCommission :exec
UPDATE commissions
SET title = ?, description = ?, domain = ?, context = ?, status = ?
WHERE id = ?;

-- name: UpdateCommissionStatus :exec
UPDATE commissions
SET status = ?
WHERE id = ?;

-- name: SetCommissionCompleted :exec
UPDATE commissions
SET status = 'completed'
WHERE id = ?;

-- name: DeleteCommission :exec
DELETE FROM commissions WHERE id = ?;

-- name: ListCommissions :many
SELECT * FROM commissions ORDER BY created_at DESC;

-- name: ListCommissionsByStatus :many
SELECT * FROM commissions WHERE status = ? ORDER BY created_at DESC;

-- name: ListCommissionsByDomain :many
SELECT * FROM commissions WHERE domain = ? ORDER BY created_at DESC;

-- name: ListCommissionsByCampaign :many
SELECT * FROM commissions WHERE campaign_id = ? ORDER BY created_at DESC;
