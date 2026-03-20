-- name: ListPipelinesByUser :many
SELECT * FROM pipelines WHERE user_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3;

-- name: CountPipelinesByUser :one
SELECT count(*) FROM pipelines WHERE user_id = $1;

-- name: GetPipelineByID :one
SELECT * FROM pipelines WHERE id = $1 AND user_id = $2;

-- name: CreatePipeline :one
INSERT INTO pipelines (user_id, name, description, success_script, failure_script, is_public)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdatePipeline :one
UPDATE pipelines
SET name = $3, description = $4, success_script = $5, failure_script = $6, is_public = $7, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeletePipeline :exec
DELETE FROM pipelines WHERE id = $1 AND user_id = $2;

-- name: CreateStage :one
INSERT INTO stages (pipeline_id, section, order_index)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListStagesByPipeline :many
SELECT * FROM stages WHERE pipeline_id = $1 ORDER BY order_index;

-- name: DeleteStagesByPipeline :exec
DELETE FROM stages WHERE pipeline_id = $1;

-- name: CreatePipelineJob :one
INSERT INTO pipeline_jobs (pipeline_id, symbol, status)
VALUES ($1, $2, 'PENDING')
RETURNING *;

-- name: GetPipelineJob :one
SELECT * FROM pipeline_jobs WHERE id = $1;

-- name: UpdatePipelineJobStatus :one
UPDATE pipeline_jobs
SET status = $2, result = $3, error = $4, started_at = $5, completed_at = $6
WHERE id = $1
RETURNING *;

-- name: DeleteMonitorsByPipeline :exec
DELETE FROM monitor_blocks WHERE pipeline_id = $1;

-- name: DeletePriceAlertsByPipeline :exec
DELETE FROM price_alerts WHERE pipeline_id = $1;
