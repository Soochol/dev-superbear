package casedomain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/dev-superbear/nexus-backend/internal/domain/casedomain"
)

func TestCalculateDayOffset(t *testing.T) {
	eventDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

	offset := casedomain.CalculateDayOffset(eventDate, now)
	assert.Equal(t, 127, offset)
}

func TestCalculateReturnPct(t *testing.T) {
	ret := casedomain.CalculateReturnPct(50000.0, 55000.0)
	assert.InDelta(t, 10.0, ret, 0.01)
}

func TestCalculateReturnPct_Negative(t *testing.T) {
	ret := casedomain.CalculateReturnPct(50000.0, 40000.0)
	assert.InDelta(t, -20.0, ret, 0.01)
}

func TestCalculateReturnPct_ZeroBase(t *testing.T) {
	ret := casedomain.CalculateReturnPct(0, 50000.0)
	assert.Equal(t, 0.0, ret)
}

func TestGetReturnTracking(t *testing.T) {
	eventDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	eventClose := 50000.0

	history := []casedomain.PricePoint{
		{Date: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC), Close: 51000},  // D+1
		{Date: time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC), Close: 54000},  // D+7
		{Date: time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC), Close: 58000},  // D+30
		{Date: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC), Close: 65000},  // Peak
		{Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Close: 48000},   // Current (latest)
	}

	result := casedomain.GetReturnTracking(eventClose, eventDate, history)

	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.Periods), 3) // At least D+1, D+7, D+30 or some subset
}
