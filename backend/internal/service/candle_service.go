package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
)

type CandleService struct {
	kisClient *kis.Client
}

func NewCandleService(kisClient *kis.Client) *CandleService {
	return &CandleService{kisClient: kisClient}
}

func (s *CandleService) GetCandles(ctx context.Context, symbol, startDate, endDate, period string) ([]kis.NormalizedCandle, error) {
	if endDate == "" {
		endDate = time.Now().Format("20060102")
	}
	if startDate == "" {
		startDate = time.Now().AddDate(-1, 0, 0).Format("20060102")
	}

	slog.Info("fetching candles",
		"symbol", symbol,
		"startDate", startDate,
		"endDate", endDate,
		"period", period,
	)

	candles, err := s.kisClient.GetCandles(ctx, symbol, startDate, endDate)
	if err != nil {
		slog.Error("failed to fetch candles",
			"symbol", symbol,
			"error", err,
		)
		return nil, err
	}

	return candles, nil
}
