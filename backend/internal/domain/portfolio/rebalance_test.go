package portfolio

import (
	"testing"
)

func TestSimulateRebalance_EqualWeight(t *testing.T) {
	positions := []PositionForRebalance{
		{Symbol: "005930", SymbolName: "삼성전자", CurrentPrice: 78000, TotalValue: 7800000, Weight: 60},
		{Symbol: "035420", SymbolName: "NAVER", CurrentPrice: 200000, TotalValue: 4000000, Weight: 30.77},
		{Symbol: "068270", SymbolName: "셀트리온", CurrentPrice: 180000, TotalValue: 1200000, Weight: 9.23},
	}
	total := 13000000.0

	targets := []TargetAllocation{
		{Symbol: "005930", TargetWeight: 33.33},
		{Symbol: "035420", TargetWeight: 33.33},
		{Symbol: "068270", TargetWeight: 33.34},
	}

	result, err := SimulateRebalance(positions, targets, total)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(result.Actions))
	}

	// 삼성전자 overweight → SELL
	for _, a := range result.Actions {
		if a.Symbol == "005930" && a.Action != "SELL" {
			t.Errorf("005930 should SELL, got %s", a.Action)
		}
		if a.Symbol == "068270" && a.Action != "BUY" {
			t.Errorf("068270 should BUY, got %s", a.Action)
		}
	}
}

func TestSimulateRebalance_ExceedsWeight(t *testing.T) {
	targets := []TargetAllocation{
		{Symbol: "A", TargetWeight: 60},
		{Symbol: "B", TargetWeight: 50},
	}
	_, err := SimulateRebalance(nil, targets, 1000000)
	if err != ErrTargetWeightExceeds100 {
		t.Errorf("expected ErrTargetWeightExceeds100, got %v", err)
	}
}

func TestSimulateRebalance_NewSymbol(t *testing.T) {
	positions := []PositionForRebalance{
		{Symbol: "A", SymbolName: "Stock A", CurrentPrice: 1000, TotalValue: 500000, Weight: 100},
	}
	targets := []TargetAllocation{
		{Symbol: "A", TargetWeight: 50},
		{Symbol: "B", TargetWeight: 50},
	}

	result, err := SimulateRebalance(positions, targets, 500000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have action for B as BUY
	found := false
	for _, a := range result.Actions {
		if a.Symbol == "B" {
			found = true
			if a.Action != "BUY" {
				t.Errorf("B should BUY, got %s", a.Action)
			}
		}
	}
	if !found {
		t.Error("expected action for symbol B")
	}
}

func TestSimulateRebalance_NetCashFlow(t *testing.T) {
	positions := []PositionForRebalance{
		{Symbol: "A", SymbolName: "Stock A", CurrentPrice: 10000, TotalValue: 6000000, Weight: 60},
		{Symbol: "B", SymbolName: "Stock B", CurrentPrice: 10000, TotalValue: 4000000, Weight: 40},
	}
	targets := []TargetAllocation{
		{Symbol: "A", TargetWeight: 50},
		{Symbol: "B", TargetWeight: 50},
	}

	result, err := SimulateRebalance(positions, targets, 10000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A sells ~1M, B buys ~1M → net cash flow ≈ 0
	if result.TotalBuyAmount == 0 || result.TotalSellAmount == 0 {
		t.Errorf("expected non-zero buy/sell amounts")
	}
}
