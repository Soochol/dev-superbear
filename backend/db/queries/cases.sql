-- name: ListCasesByUser :many
SELECT * FROM cases WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;
-- name: CountCasesByUser :one
SELECT count(*) FROM cases WHERE user_id = $1;
-- name: ListCasesByUserAndStatus :many
SELECT * FROM cases WHERE user_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3 OFFSET $4;
-- name: CountCasesByUserAndStatus :one
SELECT count(*) FROM cases WHERE user_id = $1 AND status = $2;
-- name: GetCase :one
SELECT * FROM cases WHERE id = $1 AND user_id = $2;
-- name: CreateCase :one
INSERT INTO cases (user_id, pipeline_id, symbol, event_date, event_snapshot, success_script, failure_script)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;
-- name: UpdateCaseStatus :one
UPDATE cases SET status = $2, closed_at = $3, closed_reason = $4 WHERE id = $1 RETURNING *;
-- name: DeleteCase :exec
DELETE FROM cases WHERE id = $1 AND user_id = $2;
