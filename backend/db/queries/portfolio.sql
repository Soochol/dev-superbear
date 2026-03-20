-- name: UpsertPortfolioPosition :one
INSERT INTO portfolio_positions (user_id, symbol, symbol_name, market, quantity, avg_cost_price, total_cost, sector, sector_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, symbol) DO UPDATE SET
  quantity = portfolio_positions.quantity,
  avg_cost_price = portfolio_positions.avg_cost_price,
  total_cost = portfolio_positions.total_cost,
  updated_at = now()
RETURNING *;

-- name: GetPositionByUserSymbol :one
SELECT * FROM portfolio_positions
WHERE user_id = $1 AND symbol = $2;

-- name: GetPositionByID :one
SELECT * FROM portfolio_positions
WHERE id = $1;

-- name: ListPositionsByUser :many
SELECT * FROM portfolio_positions
WHERE user_id = $1 AND quantity > 0
ORDER BY symbol;

-- name: ListAllPositionsByUser :many
SELECT * FROM portfolio_positions
WHERE user_id = $1
ORDER BY symbol;

-- name: UpdatePositionAggregates :exec
UPDATE portfolio_positions
SET quantity = $2, avg_cost_price = $3, total_cost = $4, updated_at = now()
WHERE id = $1;

-- name: CreateFifoLot :one
INSERT INTO fifo_lots (position_id, trade_id, buy_date, buy_price, original_qty, remaining_qty, fee)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListActiveLotsByPosition :many
SELECT * FROM fifo_lots
WHERE position_id = $1 AND remaining_qty > 0
ORDER BY buy_date ASC;

-- name: UpdateLotRemainingQty :exec
UPDATE fifo_lots
SET remaining_qty = $2
WHERE id = $1;

-- name: CreateRealizedPnL :one
INSERT INTO realized_pnls (position_id, sell_trade_id, buy_lot_id, quantity, buy_price, sell_price, gross_pnl, fee, tax, net_pnl, realized_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: SumRealizedPnLByUser :one
SELECT COALESCE(SUM(rp.net_pnl), 0)::NUMERIC(12,2) AS total_net_pnl
FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1;

-- name: ListRealizedPnLByPosition :many
SELECT * FROM realized_pnls
WHERE position_id = $1
ORDER BY realized_at DESC;

-- name: ListRealizedPnLByUserPeriod :many
SELECT rp.*
FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1
  AND rp.realized_at >= $2
  AND rp.realized_at <= $3
ORDER BY rp.realized_at ASC;

-- name: SumRealizedPnLByUserAndMarket :one
SELECT COALESCE(SUM(rp.net_pnl), 0)::NUMERIC(12,2) AS total_net_pnl,
       COALESCE(SUM(rp.gross_pnl), 0)::NUMERIC(12,2) AS total_gross_pnl,
       COALESCE(SUM(rp.tax), 0)::NUMERIC(12,2) AS total_tax,
       COALESCE(SUM(rp.sell_price * rp.quantity), 0)::NUMERIC(12,2) AS total_sell_amount
FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1 AND pp.market = $2;
