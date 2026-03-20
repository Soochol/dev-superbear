package casedomain

import "time"

type ReturnPeriod struct {
	Label     string  `json:"label"`
	ReturnPct float64 `json:"return_pct"`
	VsKospi   float64 `json:"vs_kospi"`
	VsSector  float64 `json:"vs_sector"`
	DayOffset int     `json:"day_offset"`
}

type ReturnTrackingData struct {
	Periods []ReturnPeriod `json:"periods"`
}

func CalculateDayOffset(eventDate, now time.Time) int {
	eventDay := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, time.UTC)
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return int(nowDay.Sub(eventDay).Hours() / 24)
}

func CalculateReturnPct(basePrice, currentPrice float64) float64 {
	if basePrice == 0 {
		return 0
	}
	return (currentPrice - basePrice) / basePrice * 100
}

type PricePoint struct {
	Date  time.Time
	Close float64
}

func GetReturnTracking(eventClose float64, eventDate time.Time, priceHistory []PricePoint) *ReturnTrackingData {
	periods := make([]ReturnPeriod, 0, 5)

	targetOffsets := []struct {
		label  string
		offset int
	}{
		{"D+1", 1},
		{"D+7", 7},
		{"D+30", 30},
	}

	for _, target := range targetOffsets {
		targetDate := eventDate.AddDate(0, 0, target.offset)
		price := findClosestPrice(priceHistory, targetDate)
		if price > 0 {
			periods = append(periods, ReturnPeriod{
				Label:     target.label,
				ReturnPct: CalculateReturnPct(eventClose, price),
				DayOffset: target.offset,
			})
		}
	}

	if peakPrice, peakOffset := findPeakPrice(priceHistory, eventDate); peakPrice > 0 {
		periods = append(periods, ReturnPeriod{
			Label:     "Peak",
			ReturnPct: CalculateReturnPct(eventClose, peakPrice),
			DayOffset: peakOffset,
		})
	}

	if len(priceHistory) > 0 {
		latest := priceHistory[len(priceHistory)-1]
		periods = append(periods, ReturnPeriod{
			Label:     "Current",
			ReturnPct: CalculateReturnPct(eventClose, latest.Close),
			DayOffset: CalculateDayOffset(eventDate, latest.Date),
		})
	}

	return &ReturnTrackingData{Periods: periods}
}

func findClosestPrice(history []PricePoint, targetDate time.Time) float64 {
	target := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
	bestDiff := time.Duration(1<<63 - 1)
	bestPrice := 0.0

	for _, p := range history {
		d := time.Date(p.Date.Year(), p.Date.Month(), p.Date.Day(), 0, 0, 0, 0, time.UTC)
		diff := d.Sub(target)
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff && diff <= 3*24*time.Hour {
			bestDiff = diff
			bestPrice = p.Close
		}
	}
	return bestPrice
}

func findPeakPrice(history []PricePoint, eventDate time.Time) (float64, int) {
	peakPrice := 0.0
	peakOffset := 0

	for _, p := range history {
		if p.Date.After(eventDate) && p.Close > peakPrice {
			peakPrice = p.Close
			peakOffset = CalculateDayOffset(eventDate, p.Date)
		}
	}
	return peakPrice, peakOffset
}
