package kis

import (
	"fmt"
	"sort"
	"strconv"
)

func formatDate(yyyymmdd string) string {
	if len(yyyymmdd) != 8 {
		return yyyymmdd
	}
	return fmt.Sprintf("%s-%s-%s", yyyymmdd[:4], yyyymmdd[4:6], yyyymmdd[6:8])
}

func NormalizeKISCandles(raw []KISCandle) []NormalizedCandle {
	result := make([]NormalizedCandle, 0, len(raw))

	for _, c := range raw {
		open, _ := strconv.ParseFloat(c.StckOprc, 64)
		high, _ := strconv.ParseFloat(c.StckHgpr, 64)
		low, _ := strconv.ParseFloat(c.StckLwpr, 64)
		closeVal, _ := strconv.ParseFloat(c.StckClpr, 64)
		volume, _ := strconv.ParseInt(c.AcmlVol, 10, 64)

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
