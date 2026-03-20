// Package portfolio provides pure domain logic for FIFO-based portfolio
// position tracking, realized PnL computation, and tax calculation.
package portfolio

import (
	"errors"
	"time"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// FifoLot represents a single buy-lot tracked for FIFO matching.
type FifoLot struct {
	ID           string
	TradeID      string
	BuyDate      time.Time
	BuyPrice     float64
	OriginalQty  int
	RemainingQty int
	Fee          float64
}

// LotSellResult describes the outcome of selling against a single FIFO lot.
type LotSellResult struct {
	LotID           string
	SellQty         int
	GrossPnL        float64 // (sellPrice - buyPrice) * sellQty
	LotFee          float64 // prorated buy-side fee
	SellFee         float64 // prorated sell-side fee
	Tax             float64
	NetPnL          float64 // grossPnL - lotFee - sellFee - tax
	NewRemainingQty int
}

// FifoSellResult aggregates lot-level results for a single sell order.
type FifoSellResult struct {
	LotResults  []LotSellResult
	TotalPnL    float64
	TotalQty    int
	AvgCostUsed float64
}

// PositionAggregates holds recomputed position-level numbers.
type PositionAggregates struct {
	Quantity     int
	AvgCostPrice float64
	TotalCost    float64
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrInsufficientQty = errors.New("portfolio: sell quantity exceeds available lots")
	ErrInvalidQty      = errors.New("portfolio: quantity must be > 0")
	ErrInvalidPrice    = errors.New("portfolio: price must be > 0")
)

// ---------------------------------------------------------------------------
// ComputeFifoSell — pure FIFO sell computation
// ---------------------------------------------------------------------------

// ComputeFifoSell matches a sell order against lots sorted oldest-first (FIFO)
// and returns per-lot sell results with PnL, fees and tax.
// This function is pure — it performs no DB calls.
func ComputeFifoSell(lots []FifoLot, sellPrice float64, sellQty int, sellFee float64, market Market) ([]LotSellResult, error) {
	if sellQty <= 0 {
		return nil, ErrInvalidQty
	}
	if sellPrice <= 0 {
		return nil, ErrInvalidPrice
	}

	// Check total available quantity
	available := 0
	for _, lot := range lots {
		available += lot.RemainingQty
	}
	if sellQty > available {
		return nil, ErrInsufficientQty
	}

	results := make([]LotSellResult, 0, len(lots))
	remaining := sellQty

	for _, lot := range lots {
		if remaining <= 0 {
			break
		}

		qty := min(lot.RemainingQty, remaining)
		grossPnL := (sellPrice - lot.BuyPrice) * float64(qty)
		tax := CalculateTax(market, sellPrice, qty, grossPnL)

		// Prorate buy-side fee proportionally to original qty
		lotFee := 0.0
		if lot.OriginalQty > 0 {
			lotFee = (lot.Fee / float64(lot.OriginalQty)) * float64(qty)
		}

		// Prorate sell-side fee proportionally to total sell qty
		sellFeeAlloc := 0.0
		if sellQty > 0 {
			sellFeeAlloc = (sellFee / float64(sellQty)) * float64(qty)
		}

		results = append(results, LotSellResult{
			LotID:           lot.ID,
			SellQty:         qty,
			GrossPnL:        grossPnL,
			LotFee:          lotFee,
			SellFee:         sellFeeAlloc,
			Tax:             tax,
			NetPnL:          grossPnL - lotFee - sellFeeAlloc - tax,
			NewRemainingQty: lot.RemainingQty - qty,
		})

		remaining -= qty
	}

	return results, nil
}

// ---------------------------------------------------------------------------
// RecalculatePositionFromLots — recompute position aggregates
// ---------------------------------------------------------------------------

// RecalculatePositionFromLots recomputes quantity, average cost price and total
// cost from the list of active FIFO lots. Pure function, no DB dependency.
func RecalculatePositionFromLots(lots []FifoLot) PositionAggregates {
	totalQty := 0
	totalCost := 0.0

	for _, lot := range lots {
		if lot.RemainingQty > 0 {
			totalQty += lot.RemainingQty
			totalCost += lot.BuyPrice * float64(lot.RemainingQty)
		}
	}

	avgCost := 0.0
	if totalQty > 0 {
		avgCost = totalCost / float64(totalQty)
	}

	return PositionAggregates{
		Quantity:     totalQty,
		AvgCostPrice: avgCost,
		TotalCost:    totalCost,
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
