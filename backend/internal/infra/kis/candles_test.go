package kis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeKISCandles(t *testing.T) {
	rawCandles := []KISCandle{
		{
			StckBsopDate: "20260318",
			StckOprc:     "77000",
			StckHgpr:     "79000",
			StckLwpr:     "76500",
			StckClpr:     "78400",
			AcmlVol:      "15234000",
			AcmlTrPbmn:   "1189000000000",
		},
		{
			StckBsopDate: "20260317",
			StckOprc:     "76000",
			StckHgpr:     "77500",
			StckLwpr:     "75800",
			StckClpr:     "77000",
			AcmlVol:      "12100000",
			AcmlTrPbmn:   "932000000000",
		},
	}

	t.Run("converts KIS format to normalized candle data", func(t *testing.T) {
		result := NormalizeKISCandles(rawCandles)
		require.Len(t, result, 2)
		assert.Equal(t, "2026-03-18", result[1].Time)
		assert.Equal(t, float64(77000), result[1].Open)
		assert.Equal(t, float64(79000), result[1].High)
		assert.Equal(t, float64(76500), result[1].Low)
		assert.Equal(t, float64(78400), result[1].Close)
		assert.Equal(t, int64(15234000), result[1].Volume)
	})

	t.Run("sorts candles by date ascending", func(t *testing.T) {
		result := NormalizeKISCandles(rawCandles)
		assert.Equal(t, "2026-03-17", result[0].Time)
		assert.Equal(t, "2026-03-18", result[1].Time)
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := NormalizeKISCandles([]KISCandle{})
		assert.Empty(t, result)
	})

	t.Run("converts date format YYYYMMDD to YYYY-MM-DD", func(t *testing.T) {
		result := NormalizeKISCandles(rawCandles[:1])
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, result[0].Time)
	})
}
