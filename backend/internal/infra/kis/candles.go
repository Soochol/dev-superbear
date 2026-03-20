package kis

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"time"
)

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
		open, err := strconv.ParseFloat(c.StckOprc, 64)
		if err != nil {
			continue
		}
		high, err := strconv.ParseFloat(c.StckHgpr, 64)
		if err != nil {
			continue
		}
		low, err := strconv.ParseFloat(c.StckLwpr, 64)
		if err != nil {
			continue
		}
		closeVal, err := strconv.ParseFloat(c.StckClpr, 64)
		if err != nil {
			continue
		}
		volume, err := strconv.ParseInt(c.AcmlVol, 10, 64)
		if err != nil {
			continue
		}

		ts := formatIntradayTime(c.StckBsopDate)
		if ts == 0 {
			continue
		}

		result = append(result, NormalizedCandle{
			Time:   fmt.Sprintf("%d", ts),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closeVal,
			Volume: volume,
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
		open, err := strconv.ParseFloat(c.StckOprc, 64)
		if err != nil {
			slog.Warn("skipping candle: failed to parse open", "value", c.StckOprc, "error", err)
			continue
		}
		high, err := strconv.ParseFloat(c.StckHgpr, 64)
		if err != nil {
			slog.Warn("skipping candle: failed to parse high", "value", c.StckHgpr, "error", err)
			continue
		}
		low, err := strconv.ParseFloat(c.StckLwpr, 64)
		if err != nil {
			slog.Warn("skipping candle: failed to parse low", "value", c.StckLwpr, "error", err)
			continue
		}
		closeVal, err := strconv.ParseFloat(c.StckClpr, 64)
		if err != nil {
			slog.Warn("skipping candle: failed to parse close", "value", c.StckClpr, "error", err)
			continue
		}
		volume, err := strconv.ParseInt(c.AcmlVol, 10, 64)
		if err != nil {
			slog.Warn("skipping candle: failed to parse volume", "value", c.AcmlVol, "error", err)
			continue
		}

		result = append(result, NormalizedCandle{
			Time:   formatDate(c.StckBsopDate),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closeVal,
			Volume: volume,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})

	return result
}
