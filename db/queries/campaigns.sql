-- name: CreateCampaign :exec
INSERT INTO campaigns (id, name, status)
VALUES (?, ?, ?);

-- name: GetCampaign :one
SELECT * FROM campaigns WHERE id = ?;

-- name: ListCampaigns :many
SELECT * FROM campaigns ORDER BY created_at DESC;

-- name: UpdateCampaignStatus :exec
UPDATE campaigns
SET status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteCampaign :exec
DELETE FROM campaigns WHERE id = ?;
