package dart

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeFinancialStatements(t *testing.T) {
	t.Run("normalizes financial statement response", func(t *testing.T) {
		raw := RawFinancials{
			Revenue:         "67890000000000",
			OperatingProfit: "12345000000000",
			NetIncome:       "9876000000000",
		}

		result := NormalizeFinancialStatements(raw)
		assert.NotNil(t, result.Revenue)
		assert.InDelta(t, 678900.0, *result.Revenue, 1.0)
		assert.NotNil(t, result.OperatingProfit)
		assert.InDelta(t, 123450.0, *result.OperatingProfit, 1.0)
	})

	t.Run("handles missing values gracefully", func(t *testing.T) {
		result := NormalizeFinancialStatements(RawFinancials{})
		assert.Nil(t, result.Revenue)
		assert.Nil(t, result.OperatingProfit)
	})
}
