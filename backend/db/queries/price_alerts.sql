-- name: ListPriceAlertsByCase :many
SELECT * FROM price_alerts WHERE case_id = $1 ORDER BY created_at DESC;
-- name: CreatePriceAlert :one
INSERT INTO price_alerts (case_id, pipeline_id, condition, label) VALUES ($1, $2, $3, $4) RETURNING *;
-- name: TriggerPriceAlert :one
UPDATE price_alerts SET triggered = true, triggered_at = $2 WHERE id = $1 RETURNING *;
