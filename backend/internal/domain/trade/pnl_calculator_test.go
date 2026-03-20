package trade_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/dev-superbear/nexus-backend/internal/domain/trade"
)

func TestCalculatePnL_BuySellScenario(t *testing.T) {
	// BUY 100 @ 50,000 → BUY 50 @ 48,000 → SELL 80 @ 55,000
	trades := []trade.TradeInput{
		{Type: "BUY", Price: 50000, Quantity: 100, Fee: 5000, TradedAt: time.Now().Add(-72 * time.Hour)},
		{Type: "BUY", Price: 48000, Quantity: 50, Fee: 2400, TradedAt: time.Now().Add(-48 * time.Hour)},
		{Type: "SELL", Price: 55000, Quantity: 80, Fee: 4400, TradedAt: time.Now().Add(-24 * time.Hour)},
	}

	result := trade.CalculatePnL(trades, 52000)

	assert.Equal(t, 150, result.TotalBuyQuantity)
	assert.Equal(t, 80, result.TotalSellQuantity)
	assert.Equal(t, 70, result.RemainingQuantity)
	assert.InDelta(t, 49333.33, result.AverageBuyPrice, 1.0)
	assert.Greater(t, result.RealizedPnL, 0.0)
	assert.NotZero(t, result.TotalFees)
}

func TestCalculatePnL_OnlyBuys(t *testing.T) {
	trades := []trade.TradeInput{
		{Type: "BUY", Price: 50000, Quantity: 100, Fee: 5000, TradedAt: time.Now()},
	}

	result := trade.CalculatePnL(trades, 55000)

	assert.Equal(t, 100, result.TotalBuyQuantity)
	assert.Equal(t, 0, result.TotalSellQuantity)
	assert.Equal(t, 100, result.RemainingQuantity)
	assert.Equal(t, 50000.0, result.AverageBuyPrice)
	assert.Equal(t, 0.0, result.RealizedPnL)
	assert.InDelta(t, 500000.0, result.UnrealizedPnL, 1.0)
}

func TestCalculatePnL_EmptyTrades(t *testing.T) {
	result := trade.CalculatePnL([]trade.TradeInput{}, 50000)

	assert.Equal(t, 0, result.RemainingQuantity)
	assert.Equal(t, 0.0, result.RealizedPnL)
	assert.Equal(t, 0.0, result.UnrealizedPnL)
}
