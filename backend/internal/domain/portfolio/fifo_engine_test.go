package portfolio

import (
	"math"
	"testing"
	"time"
)

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

// ---------------------------------------------------------------------------
// Test: basic FIFO sell from a single lot
// ---------------------------------------------------------------------------

func TestComputeFifoSell_SingleLot(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now(), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 100, Fee: 1000},
	}

	results, err := ComputeFifoSell(lots, 15000, 80, 800, MarketKR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.LotID != "lot-1" {
		t.Errorf("expected lot-1, got %s", r.LotID)
	}
	if r.SellQty != 80 {
		t.Errorf("expected sell qty 80, got %d", r.SellQty)
	}
	// grossPnL = (15000-10000)*80 = 400,000
	if !almostEqual(r.GrossPnL, 400000, 0.01) {
		t.Errorf("expected grossPnL 400000, got %f", r.GrossPnL)
	}
	if r.NewRemainingQty != 20 {
		t.Errorf("expected remaining 20, got %d", r.NewRemainingQty)
	}
}

// ---------------------------------------------------------------------------
// Test: FIFO sell spanning two lots
// ---------------------------------------------------------------------------

func TestComputeFifoSell_TwoLots(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now().Add(-24 * time.Hour), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 20, Fee: 1000},
		{ID: "lot-2", BuyDate: time.Now(), BuyPrice: 12000, OriginalQty: 50, RemainingQty: 50, Fee: 500},
	}

	results, err := ComputeFifoSell(lots, 15000, 70, 700, MarketKR)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First lot: sell 20 (all remaining)
	if results[0].SellQty != 20 {
		t.Errorf("lot-1 sell qty: expected 20, got %d", results[0].SellQty)
	}
	if results[0].NewRemainingQty != 0 {
		t.Errorf("lot-1 remaining: expected 0, got %d", results[0].NewRemainingQty)
	}

	// Second lot: sell 50 (all remaining)
	if results[1].SellQty != 50 {
		t.Errorf("lot-2 sell qty: expected 50, got %d", results[1].SellQty)
	}
	if results[1].NewRemainingQty != 0 {
		t.Errorf("lot-2 remaining: expected 0, got %d", results[1].NewRemainingQty)
	}

	// grossPnL lot-1 = (15000-10000)*20 = 100,000
	if !almostEqual(results[0].GrossPnL, 100000, 0.01) {
		t.Errorf("lot-1 grossPnL: expected 100000, got %f", results[0].GrossPnL)
	}
	// grossPnL lot-2 = (15000-12000)*50 = 150,000
	if !almostEqual(results[1].GrossPnL, 150000, 0.01) {
		t.Errorf("lot-2 grossPnL: expected 150000, got %f", results[1].GrossPnL)
	}
}

// ---------------------------------------------------------------------------
// Test: sell more than available → error
// ---------------------------------------------------------------------------

func TestComputeFifoSell_InsufficientQty(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now(), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 50, Fee: 500},
	}

	_, err := ComputeFifoSell(lots, 15000, 60, 600, MarketKR)
	if err != ErrInsufficientQty {
		t.Fatalf("expected ErrInsufficientQty, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: invalid inputs
// ---------------------------------------------------------------------------

func TestComputeFifoSell_InvalidInputs(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now(), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 100, Fee: 500},
	}

	if _, err := ComputeFifoSell(lots, 15000, 0, 0, MarketKR); err != ErrInvalidQty {
		t.Errorf("expected ErrInvalidQty for qty=0, got %v", err)
	}
	if _, err := ComputeFifoSell(lots, 0, 10, 100, MarketKR); err != ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice for price=0, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: RecalculatePositionFromLots
// ---------------------------------------------------------------------------

func TestRecalculatePositionFromLots(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyPrice: 10000, OriginalQty: 100, RemainingQty: 20},
		{ID: "lot-2", BuyPrice: 12000, OriginalQty: 50, RemainingQty: 50},
		{ID: "lot-3", BuyPrice: 9000, OriginalQty: 30, RemainingQty: 0}, // exhausted
	}

	agg := RecalculatePositionFromLots(lots)

	// totalQty = 20 + 50 = 70
	if agg.Quantity != 70 {
		t.Errorf("expected quantity 70, got %d", agg.Quantity)
	}
	// totalCost = 10000*20 + 12000*50 = 200000 + 600000 = 800000
	if !almostEqual(agg.TotalCost, 800000, 0.01) {
		t.Errorf("expected totalCost 800000, got %f", agg.TotalCost)
	}
	// avgCost = 800000 / 70 ≈ 11428.57
	expectedAvg := 800000.0 / 70.0
	if !almostEqual(agg.AvgCostPrice, expectedAvg, 0.01) {
		t.Errorf("expected avgCost %f, got %f", expectedAvg, agg.AvgCostPrice)
	}
}

func TestRecalculatePositionFromLots_Empty(t *testing.T) {
	agg := RecalculatePositionFromLots(nil)
	if agg.Quantity != 0 || agg.AvgCostPrice != 0 || agg.TotalCost != 0 {
		t.Errorf("expected all zeros for empty lots, got %+v", agg)
	}
}

// ---------------------------------------------------------------------------
// Integration-style test: BUY 100@10k, BUY 50@12k, SELL 120@15k (KR)
// ---------------------------------------------------------------------------

func TestFifoIntegration_BuySellScenario(t *testing.T) {
	// After two buys we have:
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now().Add(-2 * time.Hour), BuyPrice: 78000, OriginalQty: 100, RemainingQty: 100, Fee: 1000},
		{ID: "lot-2", BuyDate: time.Now().Add(-1 * time.Hour), BuyPrice: 80000, OriginalQty: 50, RemainingQty: 50, Fee: 500},
	}

	// Position before sell
	aggBefore := RecalculatePositionFromLots(lots)
	if aggBefore.Quantity != 150 {
		t.Fatalf("pre-sell qty: expected 150, got %d", aggBefore.Quantity)
	}

	// Sell 120 @ 85,000
	results, err := ComputeFifoSell(lots, 85000, 120, 1200, MarketKR)
	if err != nil {
		t.Fatalf("sell error: %v", err)
	}

	// FIFO: lot-1 fully consumed (100), lot-2 partially consumed (20)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].SellQty != 100 {
		t.Errorf("lot-1 sell: expected 100, got %d", results[0].SellQty)
	}
	if results[1].SellQty != 20 {
		t.Errorf("lot-2 sell: expected 20, got %d", results[1].SellQty)
	}

	// After applying sell: lot-1 remaining=0, lot-2 remaining=30
	lotsAfter := []FifoLot{
		{ID: "lot-1", BuyPrice: 78000, OriginalQty: 100, RemainingQty: 0},
		{ID: "lot-2", BuyPrice: 80000, OriginalQty: 50, RemainingQty: 30},
	}
	aggAfter := RecalculatePositionFromLots(lotsAfter)
	if aggAfter.Quantity != 30 {
		t.Errorf("post-sell qty: expected 30, got %d", aggAfter.Quantity)
	}
	if !almostEqual(aggAfter.AvgCostPrice, 80000, 0.01) {
		t.Errorf("post-sell avgCost: expected 80000, got %f", aggAfter.AvgCostPrice)
	}
}
