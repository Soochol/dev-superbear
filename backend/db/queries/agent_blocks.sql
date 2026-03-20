-- name: ListAgentBlocksByUser :many
SELECT * FROM agent_blocks WHERE user_id = $1 OR is_public = true ORDER BY created_at DESC LIMIT $2 OFFSET $3;
-- name: CountAgentBlocksByUser :one
SELECT count(*) FROM agent_blocks WHERE user_id = $1 OR is_public = true;
-- name: GetAgentBlock :one
SELECT * FROM agent_blocks WHERE id = $1;
-- name: CreateAgentBlock :one
INSERT INTO agent_blocks (user_id, name, instruction, system_prompt, allowed_tools, output_schema, is_public)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;
-- name: UpdateAgentBlock :one
UPDATE agent_blocks SET name = $2, instruction = $3, system_prompt = $4, allowed_tools = $5, output_schema = $6, is_public = $7
WHERE id = $1 RETURNING *;
-- name: DeleteAgentBlock :exec
DELETE FROM agent_blocks WHERE id = $1 AND user_id = $2;
