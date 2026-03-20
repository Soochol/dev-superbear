-- name: GetWatchlistByUser :many
SELECT id, user_id, symbol, name, created_at
FROM watchlist
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: AddToWatchlist :one
INSERT INTO watchlist (user_id, symbol, name)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, symbol) DO UPDATE SET name = EXCLUDED.name
RETURNING id, user_id, symbol, name, created_at;

-- name: RemoveFromWatchlist :exec
DELETE FROM watchlist
WHERE user_id = $1 AND symbol = $2;

-- name: IsInWatchlist :one
SELECT EXISTS(
    SELECT 1 FROM watchlist WHERE user_id = $1 AND symbol = $2
) AS is_in_watchlist;
