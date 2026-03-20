-- name: ListPresets :many
SELECT * FROM search_presets
WHERE user_id = $1 OR is_public = true
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPresets :one
SELECT COUNT(*) FROM search_presets
WHERE user_id = $1 OR is_public = true;

-- name: CreatePreset :one
INSERT INTO search_presets (user_id, name, dsl, nl_query, is_public)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPreset :one
SELECT * FROM search_presets WHERE id = $1;

-- name: DeletePreset :exec
DELETE FROM search_presets WHERE id = $1 AND user_id = $2;
