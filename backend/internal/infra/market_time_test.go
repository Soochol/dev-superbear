package infra

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsMarketHours(t *testing.T) {
	kst := time.FixedZone("KST", 9*60*60)
	tests := []struct {
		name     string
		time     time.Time
		expected bool
	}{
		{"장중 10시", time.Date(2026, 3, 18, 10, 0, 0, 0, kst), true},
		{"장중 9시 정각", time.Date(2026, 3, 18, 9, 0, 0, 0, kst), true},
		{"장중 15시 30분", time.Date(2026, 3, 18, 15, 30, 0, 0, kst), true},
		{"장외 15시 31분", time.Date(2026, 3, 18, 15, 31, 0, 0, kst), false},
		{"장외 8시 59분", time.Date(2026, 3, 18, 8, 59, 0, 0, kst), false},
		{"장외 16시", time.Date(2026, 3, 18, 16, 0, 0, 0, kst), false},
		{"토요일 10시", time.Date(2026, 3, 21, 10, 0, 0, 0, kst), false},
		{"일요일 10시", time.Date(2026, 3, 22, 10, 0, 0, 0, kst), false},
		{"UTC→KST 변환", time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsMarketHours(tt.time))
		})
	}
}
