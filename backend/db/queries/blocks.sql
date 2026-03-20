-- name: ListBlocksByUser :many
SELECT * FROM agent_blocks
WHERE user_id = $1 AND stage_id IS NULL
ORDER BY created_at DESC;

-- name: ListTemplates :many
SELECT * FROM agent_blocks
WHERE (user_id = $1 OR is_public = true) AND is_template = true
ORDER BY name;

-- name: GetBlockByID :one
SELECT * FROM agent_blocks WHERE id = $1;

-- name: CreateBlock :one
INSERT INTO agent_blocks (
  user_id, stage_id, name, objective, input_desc, tools, output_format,
  constraints, examples, instruction, system_prompt, allowed_tools,
  output_schema, is_public, is_template, template_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateBlock :one
UPDATE agent_blocks
SET name = $2, objective = $3, input_desc = $4, tools = $5, output_format = $6,
    constraints = $7, examples = $8, instruction = $9, system_prompt = $10,
    allowed_tools = $11, output_schema = $12, is_public = $13, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteBlock :exec
DELETE FROM agent_blocks WHERE id = $1 AND user_id = $2;

-- name: ListBlocksByStage :many
SELECT * FROM agent_blocks WHERE stage_id = $1 ORDER BY created_at;

-- name: CreateMonitorBlock :one
INSERT INTO monitor_blocks (pipeline_id, block_id, cron, enabled)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMonitorsByPipeline :many
SELECT * FROM monitor_blocks WHERE pipeline_id = $1;

-- name: UpdateMonitorBlock :one
UPDATE monitor_blocks SET cron = $2, enabled = $3, updated_at = now() WHERE id = $1
RETURNING *;

-- name: DeleteMonitorBlock :exec
DELETE FROM monitor_blocks WHERE id = $1;

