package kis

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"
)

// ohlcv holds the parsed numeric values from a KIS candle.
type ohlcv struct {
	open, high, low, close float64
	volume                 int64
}

// parseOHLCV extracts numeric OHLCV values from a raw KIS candle.
// Returns false if any field fails to parse.
func parseOHLCV(c KISCandle) (ohlcv, bool) {
	open, err := strconv.ParseFloat(c.StckOprc, 64)
	if err != nil {
		return ohlcv{}, false
	}
	high, err := strconv.ParseFloat(c.StckHgpr, 64)
	if err != nil {
		return ohlcv{}, false
	}
	low, err := strconv.ParseFloat(c.StckLwpr, 64)
	if err != nil {
		return ohlcv{}, false
	}
	closeVal, err := strconv.ParseFloat(c.StckClpr, 64)
	if err != nil {
		return ohlcv{}, false
	}
	volume, err := strconv.ParseInt(c.AcmlVol, 10, 64)
	if err != nil {
		return ohlcv{}, false
	}
	return ohlcv{open: open, high: high, low: low, close: closeVal, volume: volume}, true
}

func formatIntradayTime(dateTimeStr string) int64 {
	// KIS returns "YYYYMMDDHHMMSS" for intraday
	if len(dateTimeStr) < 12 {
		return 0
	}
	t, err := time.Parse("20060102150405", dateTimeStr)
	if err != nil {
		slog.Warn("failed to parse intraday time", "value", dateTimeStr, "error", err)
		return 0
	}
	return t.Unix()
}

func NormalizeKISIntradayCandles(raw []KISCandle) []NormalizedCandle {
	result := make([]NormalizedCandle, 0, len(raw))

	for _, c := range raw {
		v, ok := parseOHLCV(c)
		if !ok {
			continue
		}

		ts := formatIntradayTime(c.StckBsopDate)
		if ts == 0 {
			continue
		}

		result = append(result, NormalizedCandle{
			Time:   fmt.Sprintf("%d", ts),
			Open:   v.open,
			High:   v.high,
			Low:    v.low,
			Close:  v.close,
			Volume: v.volume,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})

	return result
}

func formatDate(yyyymmdd string) string {
	if len(yyyymmdd) != 8 {
		return yyyymmdd
	}
	return fmt.Sprintf("%s-%s-%s", yyyymmdd[:4], yyyymmdd[4:6], yyyymmdd[6:8])
}

func NormalizeKISCandles(raw []KISCandle) []NormalizedCandle {
	result := make([]NormalizedCandle, 0, len(raw))

	for _, c := range raw {
		v, ok := parseOHLCV(c)
		if !ok {
			slog.Warn("skipping candle: failed to parse OHLCV", "date", c.StckBsopDate)
			continue
		}

		result = append(result, NormalizedCandle{
			Time:   formatDate(c.StckBsopDate),
			Open:   v.open,
			High:   v.high,
			Low:    v.low,
			Close:  v.close,
			Volume: v.volume,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})

	return result
}
