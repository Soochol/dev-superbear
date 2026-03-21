package service

import (
	"context"
	"fmt"
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

// isIntraday returns true for minute/hour timeframes
func isIntraday(period string) bool {
	switch period {
	case "1m", "5m", "15m", "30m", "1H", "4H":
		return true
	default:
		return false
	}
}

// mapPeriodToKIS maps frontend timeframe to KIS daily API period code
func mapPeriodToKIS(period string) string {
	switch period {
	case "1D":
		return "D"
	case "1W":
		return "W"
	case "1M":
		return "M"
	default:
		return "D"
	}
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
		"period", period,
		"intraday", isIntraday(period),
	)

	if isIntraday(period) {
		return s.getIntradayCandles(ctx, symbol, period)
	}

	kisPeriod := mapPeriodToKIS(period)
	candles, err := s.kisClient.GetCandles(ctx, symbol, startDate, endDate, kisPeriod)
	if err != nil {
		slog.Error("failed to fetch daily candles", "symbol", symbol, "error", err)
		return nil, err
	}

	return candles, nil
}

func (s *CandleService) getIntradayCandles(ctx context.Context, symbol, period string) ([]kis.NormalizedCandle, error) {
	candles, err := s.kisClient.GetIntradayCandles(ctx, symbol)
	if err != nil {
		slog.Error("failed to fetch intraday candles", "symbol", symbol, "error", err)
		return nil, err
	}

	if period == "4H" {
		return aggregateToFourHour(candles), nil
	}

	// For other intraday periods, KIS returns minute-level data
	// TODO: filter/aggregate based on period (1m, 5m, 15m, 30m, 1H)
	// For now, return raw minute data
	return candles, nil
}

// aggregateToFourHour groups 1-minute candles into 4-hour bars
func aggregateToFourHour(candles []kis.NormalizedCandle) []kis.NormalizedCandle {
	if len(candles) == 0 {
		return candles
	}

	var result []kis.NormalizedCandle
	var current *kis.NormalizedCandle
	var currentSlot int64

	for _, c := range candles {
		// Parse the unix timestamp
		var ts int64
		fmt.Sscanf(c.Time, "%d", &ts)
		if ts == 0 {
			continue
		}

		// Calculate 4-hour slot (0, 4, 8, 12, 16, 20)
		t := time.Unix(ts, 0)
		slot := int64(t.Hour()/4) * 4
		slotKey := time.Date(t.Year(), t.Month(), t.Day(), int(slot), 0, 0, 0, t.Location()).Unix()

		if current == nil || slotKey != currentSlot {
			if current != nil {
				result = append(result, *current)
			}
			current = &kis.NormalizedCandle{
				Time:   fmt.Sprintf("%d", slotKey),
				Open:   c.Open,
				High:   c.High,
				Low:    c.Low,
				Close:  c.Close,
				Volume: c.Volume,
			}
			currentSlot = slotKey
		} else {
			if c.High > current.High {
				current.High = c.High
			}
			if c.Low < current.Low {
				current.Low = c.Low
			}
			current.Close = c.Close
			current.Volume += c.Volume
		}
	}

	if current != nil {
		result = append(result, *current)
	}

	return result
}
