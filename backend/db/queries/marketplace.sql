-- =============================================================================
-- Marketplace Item CRUD
-- =============================================================================

-- name: CreateMarketplaceItem :one
INSERT INTO marketplace_items (
  user_id, type, title, description, tags,
  pipeline_id, agent_block_id, search_preset_id, judgment_script_id,
  forked_from_id
) VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8, $9,
  $10
) RETURNING *;

-- name: GetMarketplaceItemByID :one
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.id = $1;

-- name: GetMarketplaceItemWithForkOrigin :one
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image,
       fo.id AS fork_origin_id, fo.title AS fork_origin_title,
       fu.name AS fork_origin_author_name
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
LEFT JOIN marketplace_items fo ON mi.forked_from_id = fo.id
LEFT JOIN users fu ON fo.user_id = fu.id
WHERE mi.id = $1;

-- name: UpdateMarketplaceItem :one
UPDATE marketplace_items
SET title = $2,
    description = $3,
    tags = $4
WHERE id = $1 AND user_id = $5
RETURNING *;

-- name: SetMarketplaceItemStatus :exec
UPDATE marketplace_items
SET status = $2
WHERE id = $1 AND user_id = $3;

-- name: DeleteMarketplaceItem :exec
UPDATE marketplace_items
SET status = 'REMOVED'
WHERE id = $1 AND user_id = $2;

-- =============================================================================
-- Listing & Filtering (no full-text search)
-- =============================================================================

-- name: ListMarketplaceItemsByRecent :many
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.status = 'ACTIVE'
  AND ($1::marketplace_item_type IS NULL OR mi.type = $1)
  AND ($2::boolean IS NULL OR mi.verified = $2)
  AND ($3::text[] IS NULL OR mi.tags && $3)
ORDER BY mi.published_at DESC
LIMIT $4 OFFSET $5;

-- name: ListMarketplaceItemsByPopular :many
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.status = 'ACTIVE'
  AND ($1::marketplace_item_type IS NULL OR mi.type = $1)
  AND ($2::boolean IS NULL OR mi.verified = $2)
  AND ($3::text[] IS NULL OR mi.tags && $3)
ORDER BY mi.usage_count DESC
LIMIT $4 OFFSET $5;

-- name: ListMarketplaceItemsByPerformance :many
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.status = 'ACTIVE'
  AND ($1::marketplace_item_type IS NULL OR mi.type = $1)
  AND ($2::boolean IS NULL OR mi.verified = $2)
  AND ($3::text[] IS NULL OR mi.tags && $3)
ORDER BY mi.backtest_win_rate DESC NULLS LAST
LIMIT $4 OFFSET $5;

-- name: ListMarketplaceItemsByMostForked :many
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.status = 'ACTIVE'
  AND ($1::marketplace_item_type IS NULL OR mi.type = $1)
  AND ($2::boolean IS NULL OR mi.verified = $2)
  AND ($3::text[] IS NULL OR mi.tags && $3)
ORDER BY mi.fork_count DESC
LIMIT $4 OFFSET $5;

-- name: CountMarketplaceItems :one
SELECT COUNT(*) FROM marketplace_items
WHERE status = 'ACTIVE'
  AND ($1::marketplace_item_type IS NULL OR type = $1)
  AND ($2::boolean IS NULL OR verified = $2)
  AND ($3::text[] IS NULL OR tags && $3);

-- =============================================================================
-- Full-Text Search (PostgreSQL tsvector)
-- =============================================================================

-- name: SearchMarketplaceItems :many
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image,
       ts_rank(mi.search_vector, plainto_tsquery('simple', $1)) AS rank
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.search_vector @@ plainto_tsquery('simple', $1)
  AND mi.status = 'ACTIVE'
  AND ($2::marketplace_item_type IS NULL OR mi.type = $2)
  AND ($3::boolean IS NULL OR mi.verified = $3)
  AND ($4::text[] IS NULL OR mi.tags && $4)
ORDER BY rank DESC
LIMIT $5 OFFSET $6;

-- name: CountSearchMarketplaceItems :one
SELECT COUNT(*) FROM marketplace_items
WHERE search_vector @@ plainto_tsquery('simple', $1)
  AND status = 'ACTIVE'
  AND ($2::marketplace_item_type IS NULL OR type = $2)
  AND ($3::boolean IS NULL OR verified = $3)
  AND ($4::text[] IS NULL OR tags && $4);

-- =============================================================================
-- Statistics Updates
-- =============================================================================

-- name: IncrementViewCount :exec
UPDATE marketplace_items
SET view_count = view_count + 1
WHERE id = $1;

-- name: IncrementForkCount :exec
UPDATE marketplace_items
SET fork_count = fork_count + 1
WHERE id = $1;

-- name: IncrementUsageCount :exec
UPDATE marketplace_items
SET usage_count = usage_count + 1
WHERE id = $1;

-- name: IncrementLikeCount :exec
UPDATE marketplace_items
SET like_count = like_count + 1
WHERE id = $1;

-- name: DecrementLikeCount :exec
UPDATE marketplace_items
SET like_count = GREATEST(like_count - 1, 0)
WHERE id = $1;

-- name: SetVerificationStats :exec
UPDATE marketplace_items
SET verified = true,
    backtest_job_id = $2,
    backtest_win_rate = $3,
    backtest_avg_return = $4,
    backtest_total_events = $5
WHERE id = $1;

-- name: ClearVerification :exec
UPDATE marketplace_items
SET verified = false,
    backtest_job_id = NULL,
    backtest_win_rate = NULL,
    backtest_avg_return = NULL,
    backtest_total_events = NULL
WHERE id = $1;

-- =============================================================================
-- Batch Stats Refresh (for asynq worker)
-- =============================================================================

-- name: RefreshItemForkCount :exec
UPDATE marketplace_items mi
SET fork_count = (
  SELECT COUNT(*) FROM marketplace_items WHERE forked_from_id = mi.id
)
WHERE mi.id = $1;

-- name: RefreshItemLikeCount :exec
UPDATE marketplace_items mi
SET like_count = (
  SELECT COUNT(*) FROM marketplace_likes WHERE item_id = mi.id
)
WHERE mi.id = $1;

-- name: RefreshItemUsageCount :exec
UPDATE marketplace_items mi
SET usage_count = (
  SELECT COUNT(*) FROM marketplace_usage_logs WHERE item_id = mi.id AND action = 'EXECUTE'
)
WHERE mi.id = $1;

-- name: RefreshItemViewCount :exec
UPDATE marketplace_items mi
SET view_count = (
  SELECT COUNT(*) FROM marketplace_usage_logs WHERE item_id = mi.id AND action = 'VIEW'
)
WHERE mi.id = $1;

-- name: ListActiveItemIDs :many
SELECT id FROM marketplace_items WHERE status = 'ACTIVE';

-- =============================================================================
-- Likes
-- =============================================================================

-- name: CreateLike :one
INSERT INTO marketplace_likes (user_id, item_id)
VALUES ($1, $2)
ON CONFLICT (user_id, item_id) DO NOTHING
RETURNING *;

-- name: DeleteLike :exec
DELETE FROM marketplace_likes
WHERE user_id = $1 AND item_id = $2;

-- name: GetLikeByUserAndItem :one
SELECT * FROM marketplace_likes
WHERE user_id = $1 AND item_id = $2;

-- name: IsItemLikedByUser :one
SELECT EXISTS(
  SELECT 1 FROM marketplace_likes WHERE user_id = $1 AND item_id = $2
) AS liked;

-- =============================================================================
-- Usage Logs
-- =============================================================================

-- name: CreateUsageLog :one
INSERT INTO marketplace_usage_logs (user_id, item_id, action)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRecentViewLog :one
SELECT * FROM marketplace_usage_logs
WHERE item_id = $1 AND user_id = $2 AND action = 'VIEW'
  AND created_at >= $3
ORDER BY created_at DESC
LIMIT 1;

-- name: FindItemByPipelineID :one
SELECT id FROM marketplace_items
WHERE pipeline_id = $1 AND status = 'ACTIVE'
LIMIT 1;

-- name: FindItemByAgentBlockID :one
SELECT id FROM marketplace_items
WHERE agent_block_id = $1 AND status = 'ACTIVE'
LIMIT 1;

-- name: FindItemBySearchPresetID :one
SELECT id FROM marketplace_items
WHERE search_preset_id = $1 AND status = 'ACTIVE'
LIMIT 1;

-- name: FindItemByJudgmentScriptID :one
SELECT id FROM marketplace_items
WHERE judgment_script_id = $1 AND status = 'ACTIVE'
LIMIT 1;

-- =============================================================================
-- User's own items
-- =============================================================================

-- name: ListMyMarketplaceItems :many
SELECT mi.*,
       u.id AS author_id, u.name AS author_name, u.image AS author_image
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.user_id = $1 AND mi.status != 'REMOVED'
ORDER BY mi.published_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMyMarketplaceItems :one
SELECT COUNT(*) FROM marketplace_items
WHERE user_id = $1 AND status != 'REMOVED';
