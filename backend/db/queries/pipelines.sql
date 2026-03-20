-- name: ListPipelinesByUser :many
SELECT * FROM pipelines WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3;
-- name: CountPipelinesByUser :one
SELECT count(*) FROM pipelines WHERE user_id = $1;
-- name: GetPipeline :one
SELECT * FROM pipelines WHERE id = $1 AND user_id = $2;
-- name: CreatePipeline :one
INSERT INTO pipelines (user_id, name, description, analysis_stages, monitors, success_script, failure_script)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;
-- name: UpdatePipeline :one
UPDATE pipelines SET name = $2, description = $3, analysis_stages = $4, monitors = $5, success_script = $6, failure_script = $7
WHERE id = $1 RETURNING *;
-- name: DeletePipeline :exec
DELETE FROM pipelines WHERE id = $1 AND user_id = $2;
