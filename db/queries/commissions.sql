-- name: CreateCommission :exec
INSERT INTO commissions (id, campaign_id, title, description, domain, context, status)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetCommission :one
SELECT * FROM commissions WHERE id = ?;

-- name: ListCommissionsByCampaign :many
SELECT * FROM commissions WHERE campaign_id = ? ORDER BY created_at DESC;

-- name: UpdateCommissionStatus :exec
UPDATE commissions 
SET status = ?
WHERE id = ?;

-- name: DeleteCommission :exec
DELETE FROM commissions WHERE id = ?;