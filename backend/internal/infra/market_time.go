package infra

import "time"

const (
	marketOpenHour    = 9
	marketCloseHour   = 15
	marketCloseMinute = 30
)

var kstLocation = time.FixedZone("KST", 9*60*60)

// IsMarketHours checks whether the given time falls within
// Korean stock market hours (09:00 ~ 15:30 KST).
// Does NOT account for holidays or half-day schedules.
func IsMarketHours(now time.Time) bool {
	kst := now.In(kstLocation)
	hour := kst.Hour()
	minute := kst.Minute()

	if hour < marketOpenHour {
		return false
	}
	if hour > marketCloseHour {
		return false
	}
	if hour == marketCloseHour && minute > marketCloseMinute {
		return false
	}
	return true
}
