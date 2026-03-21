package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WatchlistItem struct {
	ID        int64     `json:"id"`
	UserID    string     `json:"userId"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type WatchlistRepo struct {
	pool *pgxpool.Pool
}

func NewWatchlistRepo(pool *pgxpool.Pool) *WatchlistRepo {
	return &WatchlistRepo{pool: pool}
}

func (r *WatchlistRepo) GetByUser(ctx context.Context, userID string) ([]WatchlistItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, symbol, name, created_at
		 FROM watchlist WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var item WatchlistItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.Symbol, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *WatchlistRepo) Add(ctx context.Context, userID string, symbol, name string) (*WatchlistItem, error) {
	var item WatchlistItem
	err := r.pool.QueryRow(ctx,
		`INSERT INTO watchlist (user_id, symbol, name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, symbol) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id, user_id, symbol, name, created_at`,
		userID, symbol, name).
		Scan(&item.ID, &item.UserID, &item.Symbol, &item.Name, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *WatchlistRepo) Remove(ctx context.Context, userID string, symbol string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM watchlist WHERE user_id = $1 AND symbol = $2`, userID, symbol)
	return err
}
