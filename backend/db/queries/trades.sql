-- name: ListTradesByCase :many
SELECT * FROM trades WHERE case_id = $1 ORDER BY date DESC;
-- name: CreateTrade :one
INSERT INTO trades (case_id, user_id, type, price, quantity, fee, date, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *;
