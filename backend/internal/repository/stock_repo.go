package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StockSearchResult struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type StockRepository struct {
	pool *pgxpool.Pool
}

func NewStockRepository(pool *pgxpool.Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

func (r *StockRepository) Search(ctx context.Context, query string, limit int) ([]StockSearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx,
		`SELECT symbol, name FROM stocks
		 WHERE name ILIKE '%' || $1 || '%' OR symbol ILIKE '%' || $1 || '%'
		 ORDER BY
		   CASE WHEN symbol = UPPER($1) THEN 0
		        WHEN symbol LIKE UPPER($1) || '%' THEN 1
		        WHEN name LIKE $1 || '%' THEN 2
		        ELSE 3
		   END,
		   name
		 LIMIT $2`,
		query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StockSearchResult
	for rows.Next() {
		var s StockSearchResult
		if err := rows.Scan(&s.Symbol, &s.Name); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}
