// Package repository provides database access for the portfolio domain.
// It wraps sqlc-generated queries and orchestrates DB operations with
// the pure domain logic from the portfolio package.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	domain "github.com/dev-superbear/nexus-backend/internal/domain/portfolio"
)

// ---------------------------------------------------------------------------
// PortfolioRepository
// ---------------------------------------------------------------------------

// PortfolioRepository handles all DB interactions for portfolio positions,
// FIFO lots, and realized PnL records.
type PortfolioRepository struct {
	db *sql.DB
}

// NewPortfolioRepository creates a new repository backed by the given DB.
func NewPortfolioRepository(db *sql.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// ---------------------------------------------------------------------------
// Position CRUD
// ---------------------------------------------------------------------------

// UpsertPosition creates or retrieves an existing portfolio position.
func (r *PortfolioRepository) UpsertPosition(ctx context.Context, input domain.BuyTradeInput) (*domain.PortfolioPosition, error) {
	query := `
		INSERT INTO portfolio_positions (user_id, symbol, symbol_name, market, quantity, avg_cost_price, total_cost, sector, sector_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, symbol) DO UPDATE SET updated_at = now()
		RETURNING id, user_id, symbol, symbol_name, market, quantity, avg_cost_price, total_cost, sector, sector_name, created_at, updated_at`

	pos := &domain.PortfolioPosition{}
	err := r.db.QueryRowContext(ctx, query,
		input.UserID, input.Symbol, input.SymbolName, string(input.Market),
		input.Quantity, input.Price, input.Price*float64(input.Quantity),
		input.Sector, input.SectorName,
	).Scan(
		&pos.ID, &pos.UserID, &pos.Symbol, &pos.SymbolName,
		&pos.Market, &pos.Quantity, &pos.AvgCostPrice, &pos.TotalCost,
		&pos.Sector, &pos.SectorName, &pos.CreatedAt, &pos.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert position: %w", err)
	}
	return pos, nil
}

// GetPositionByUserSymbol fetches a position by (user_id, symbol).
func (r *PortfolioRepository) GetPositionByUserSymbol(ctx context.Context, userID, symbol string) (*domain.PortfolioPosition, error) {
	query := `
		SELECT id, user_id, symbol, symbol_name, market, quantity, avg_cost_price, total_cost, sector, sector_name, created_at, updated_at
		FROM portfolio_positions
		WHERE user_id = $1 AND symbol = $2`

	pos := &domain.PortfolioPosition{}
	err := r.db.QueryRowContext(ctx, query, userID, symbol).Scan(
		&pos.ID, &pos.UserID, &pos.Symbol, &pos.SymbolName,
		&pos.Market, &pos.Quantity, &pos.AvgCostPrice, &pos.TotalCost,
		&pos.Sector, &pos.SectorName, &pos.CreatedAt, &pos.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get position by user/symbol: %w", err)
	}
	return pos, nil
}

// ListActivePositions returns all positions with quantity > 0 for a user.
func (r *PortfolioRepository) ListActivePositions(ctx context.Context, userID string) ([]domain.PortfolioPosition, error) {
	query := `
		SELECT id, user_id, symbol, symbol_name, market, quantity, avg_cost_price, total_cost, sector, sector_name, created_at, updated_at
		FROM portfolio_positions
		WHERE user_id = $1 AND quantity > 0
		ORDER BY symbol`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list active positions: %w", err)
	}
	defer rows.Close()

	var positions []domain.PortfolioPosition
	for rows.Next() {
		var pos domain.PortfolioPosition
		if err := rows.Scan(
			&pos.ID, &pos.UserID, &pos.Symbol, &pos.SymbolName,
			&pos.Market, &pos.Quantity, &pos.AvgCostPrice, &pos.TotalCost,
			&pos.Sector, &pos.SectorName, &pos.CreatedAt, &pos.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, pos)
	}
	return positions, rows.Err()
}

// UpdatePositionAggregates updates qty, avgCost, totalCost for a position.
func (r *PortfolioRepository) UpdatePositionAggregates(ctx context.Context, positionID string, agg domain.PositionAggregates) error {
	query := `
		UPDATE portfolio_positions
		SET quantity = $2, avg_cost_price = $3, total_cost = $4, updated_at = now()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, positionID, agg.Quantity, agg.AvgCostPrice, agg.TotalCost)
	if err != nil {
		return fmt.Errorf("update position aggregates: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// FIFO Lot operations
// ---------------------------------------------------------------------------

// CreateFifoLot inserts a new FIFO lot for a buy trade.
func (r *PortfolioRepository) CreateFifoLot(ctx context.Context, positionID, tradeID string, buyDate time.Time, buyPrice float64, qty int, fee float64) (*domain.FifoLot, error) {
	query := `
		INSERT INTO fifo_lots (position_id, trade_id, buy_date, buy_price, original_qty, remaining_qty, fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, trade_id, buy_date, buy_price, original_qty, remaining_qty, fee`

	lot := &domain.FifoLot{}
	err := r.db.QueryRowContext(ctx, query,
		positionID, tradeID, buyDate, buyPrice, qty, qty, fee,
	).Scan(&lot.ID, &lot.TradeID, &lot.BuyDate, &lot.BuyPrice, &lot.OriginalQty, &lot.RemainingQty, &lot.Fee)
	if err != nil {
		return nil, fmt.Errorf("create fifo lot: %w", err)
	}
	return lot, nil
}

// ListActiveLots returns FIFO lots with remaining_qty > 0 for a position,
// ordered oldest-first (buy_date ASC) for FIFO matching.
func (r *PortfolioRepository) ListActiveLots(ctx context.Context, positionID string) ([]domain.FifoLot, error) {
	query := `
		SELECT id, trade_id, buy_date, buy_price, original_qty, remaining_qty, fee
		FROM fifo_lots
		WHERE position_id = $1 AND remaining_qty > 0
		ORDER BY buy_date ASC`

	rows, err := r.db.QueryContext(ctx, query, positionID)
	if err != nil {
		return nil, fmt.Errorf("list active lots: %w", err)
	}
	defer rows.Close()

	var lots []domain.FifoLot
	for rows.Next() {
		var lot domain.FifoLot
		if err := rows.Scan(&lot.ID, &lot.TradeID, &lot.BuyDate, &lot.BuyPrice, &lot.OriginalQty, &lot.RemainingQty, &lot.Fee); err != nil {
			return nil, fmt.Errorf("scan lot: %w", err)
		}
		lots = append(lots, lot)
	}
	return lots, rows.Err()
}

// UpdateLotRemainingQty updates the remaining quantity of a single lot.
func (r *PortfolioRepository) UpdateLotRemainingQty(ctx context.Context, lotID string, remainingQty int) error {
	query := `UPDATE fifo_lots SET remaining_qty = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, lotID, remainingQty)
	if err != nil {
		return fmt.Errorf("update lot remaining qty: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Realized PnL operations
// ---------------------------------------------------------------------------

// CreateRealizedPnL inserts a realized PnL record.
func (r *PortfolioRepository) CreateRealizedPnL(ctx context.Context, positionID, sellTradeID, buyLotID string, qty int, buyPrice, sellPrice, grossPnL, fee, tax, netPnL float64, realizedAt time.Time) error {
	query := `
		INSERT INTO realized_pnls (position_id, sell_trade_id, buy_lot_id, quantity, buy_price, sell_price, gross_pnl, fee, tax, net_pnl, realized_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.db.ExecContext(ctx, query,
		positionID, sellTradeID, buyLotID,
		qty, buyPrice, sellPrice, grossPnL, fee, tax, netPnL, realizedAt,
	)
	if err != nil {
		return fmt.Errorf("create realized pnl: %w", err)
	}
	return nil
}

// SumRealizedPnLByUser returns the total net realized PnL for a user.
func (r *PortfolioRepository) SumRealizedPnLByUser(ctx context.Context, userID string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(rp.net_pnl), 0)
		FROM realized_pnls rp
		JOIN portfolio_positions pp ON rp.position_id = pp.id
		WHERE pp.user_id = $1`

	var total float64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum realized pnl: %w", err)
	}
	return total, nil
}

// ListRealizedPnLByPeriod returns realized PnL records for a user within a date range.
func (r *PortfolioRepository) ListRealizedPnLByPeriod(ctx context.Context, userID string, from, to time.Time) ([]domain.RealizedPnL, error) {
	query := `
		SELECT rp.id, rp.position_id, rp.sell_trade_id, rp.buy_lot_id,
		       rp.quantity, rp.buy_price, rp.sell_price,
		       rp.gross_pnl, rp.fee, rp.tax, rp.net_pnl, rp.realized_at
		FROM realized_pnls rp
		JOIN portfolio_positions pp ON rp.position_id = pp.id
		WHERE pp.user_id = $1 AND rp.realized_at >= $2 AND rp.realized_at <= $3
		ORDER BY rp.realized_at ASC`

	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list realized pnl by period: %w", err)
	}
	defer rows.Close()

	var records []domain.RealizedPnL
	for rows.Next() {
		var rp domain.RealizedPnL
		if err := rows.Scan(
			&rp.ID, &rp.PositionID, &rp.SellTradeID, &rp.BuyLotID,
			&rp.Quantity, &rp.BuyPrice, &rp.SellPrice,
			&rp.GrossPnL, &rp.Fee, &rp.Tax, &rp.NetPnL, &rp.RealizedAt,
		); err != nil {
			return nil, fmt.Errorf("scan realized pnl: %w", err)
		}
		records = append(records, rp)
	}
	return records, rows.Err()
}

// SumRealizedPnLByMarket returns aggregate PnL numbers for a user+market.
func (r *PortfolioRepository) SumRealizedPnLByMarket(ctx context.Context, userID string, market domain.Market) (totalNetPnL, totalGrossPnL, totalTax, totalSellAmount float64, err error) {
	query := `
		SELECT COALESCE(SUM(rp.net_pnl), 0),
		       COALESCE(SUM(rp.gross_pnl), 0),
		       COALESCE(SUM(rp.tax), 0),
		       COALESCE(SUM(rp.sell_price * rp.quantity), 0)
		FROM realized_pnls rp
		JOIN portfolio_positions pp ON rp.position_id = pp.id
		WHERE pp.user_id = $1 AND pp.market = $2`

	err = r.db.QueryRowContext(ctx, query, userID, string(market)).
		Scan(&totalNetPnL, &totalGrossPnL, &totalTax, &totalSellAmount)
	if err != nil {
		err = fmt.Errorf("sum realized pnl by market: %w", err)
	}
	return
}
