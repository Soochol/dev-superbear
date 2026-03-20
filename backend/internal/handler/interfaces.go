package handler

import (
	"context"

	"github.com/dev-superbear/nexus-backend/internal/infra/dart"
	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

// CandleFetcher abstracts candle data retrieval for testability.
type CandleFetcher interface {
	GetCandles(ctx context.Context, symbol, startDate, endDate, period string) ([]kis.NormalizedCandle, error)
}

// FinancialsFetcher abstracts financial data retrieval for testability.
type FinancialsFetcher interface {
	GetFinancials(ctx context.Context, symbol string) (dart.NormalizedFinancials, error)
}

// WatchlistRepository abstracts watchlist persistence for testability.
type WatchlistRepository interface {
	GetByUser(ctx context.Context, userID int64) ([]repository.WatchlistItem, error)
	Add(ctx context.Context, userID int64, symbol, name string) (*repository.WatchlistItem, error)
	Remove(ctx context.Context, userID int64, symbol string) error
}
