package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/dev-superbear/nexus-backend/internal/infra/dart"
)

type FinancialsService struct {
	dartClient *dart.Client
}

func NewFinancialsService(dartClient *dart.Client) *FinancialsService {
	return &FinancialsService{dartClient: dartClient}
}

func (s *FinancialsService) GetFinancials(ctx context.Context, symbol string) (dart.NormalizedFinancials, error) {
	corpCode := symbol
	year := strconv.Itoa(time.Now().Year() - 1)

	slog.Info("fetching financials",
		"symbol", symbol,
		"year", year,
	)

	return s.dartClient.FetchFinancialStatements(ctx, corpCode, year, "11011")
}
