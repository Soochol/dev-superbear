package portfolio

import (
	"errors"
	"math"
	"sort"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// TargetAllocation describes one symbol's desired weight.
type TargetAllocation struct {
	Symbol       string  `json:"symbol"`
	TargetWeight float64 `json:"targetWeight"` // %
}

// RebalanceAction is the per-symbol recommendation.
type RebalanceAction struct {
	Symbol          string  `json:"symbol"`
	SymbolName      string  `json:"symbolName"`
	CurrentWeight   float64 `json:"currentWeight"`
	TargetWeight    float64 `json:"targetWeight"`
	WeightDiff      float64 `json:"weightDiff"`
	CurrentValue    float64 `json:"currentValue"`
	TargetValue     float64 `json:"targetValue"`
	ValueDiff       float64 `json:"valueDiff"`
	Action          string  `json:"action"` // BUY | SELL | HOLD
	Quantity        int     `json:"quantity"`
	EstimatedAmount float64 `json:"estimatedAmount"`
}

// RebalanceResult is the full simulation output.
type RebalanceResult struct {
	TotalPortfolioValue float64           `json:"totalPortfolioValue"`
	Actions             []RebalanceAction `json:"actions"`
	TotalBuyAmount      float64           `json:"totalBuyAmount"`
	TotalSellAmount     float64           `json:"totalSellAmount"`
	NetCashFlow         float64           `json:"netCashFlow"` // positive = cash inflow
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrTargetWeightExceeds100 = errors.New("portfolio: target weight sum exceeds 100%")
)

// ---------------------------------------------------------------------------
// SimulateRebalance — pure computation
// ---------------------------------------------------------------------------

// PositionForRebalance carries the minimal data needed by the simulator.
type PositionForRebalance struct {
	Symbol       string
	SymbolName   string
	CurrentPrice float64
	TotalValue   float64
	Weight       float64
}

// SimulateRebalance computes buy/sell/hold actions to move from the current
// allocation towards the target allocation.
func SimulateRebalance(
	positions []PositionForRebalance,
	targets []TargetAllocation,
	totalPortfolioValue float64,
) (*RebalanceResult, error) {
	// Validate target weights
	totalWeight := 0.0
	for _, t := range targets {
		totalWeight += t.TargetWeight
	}
	if totalWeight > 100.01 { // small epsilon for floats
		return nil, ErrTargetWeightExceeds100
	}

	targetMap := make(map[string]float64, len(targets))
	for _, t := range targets {
		targetMap[t.Symbol] = t.TargetWeight
	}

	posMap := make(map[string]bool, len(positions))
	actions := make([]RebalanceAction, 0, len(positions)+len(targets))

	// Existing positions
	for _, pos := range positions {
		posMap[pos.Symbol] = true
		targetWeight := targetMap[pos.Symbol] // 0 if not in targets
		targetValue := totalPortfolioValue * (targetWeight / 100)
		valueDiff := targetValue - pos.TotalValue

		action := "HOLD"
		if valueDiff > 0 {
			action = "BUY"
		} else if valueDiff < 0 {
			action = "SELL"
		}

		qty := 0
		if pos.CurrentPrice > 0 {
			qty = int(math.Abs(valueDiff) / pos.CurrentPrice)
		}

		actions = append(actions, RebalanceAction{
			Symbol:          pos.Symbol,
			SymbolName:      pos.SymbolName,
			CurrentWeight:   pos.Weight,
			TargetWeight:    targetWeight,
			WeightDiff:      targetWeight - pos.Weight,
			CurrentValue:    pos.TotalValue,
			TargetValue:     targetValue,
			ValueDiff:       valueDiff,
			Action:          action,
			Quantity:        qty,
			EstimatedAmount: float64(qty) * pos.CurrentPrice,
		})
	}

	// Target symbols not currently held
	for _, target := range targets {
		if posMap[target.Symbol] {
			continue
		}
		targetValue := totalPortfolioValue * (target.TargetWeight / 100)
		actions = append(actions, RebalanceAction{
			Symbol:          target.Symbol,
			SymbolName:      target.Symbol, // name unknown for unowned symbols
			CurrentWeight:   0,
			TargetWeight:    target.TargetWeight,
			WeightDiff:      target.TargetWeight,
			CurrentValue:    0,
			TargetValue:     targetValue,
			ValueDiff:       targetValue,
			Action:          "BUY",
			Quantity:        0, // current price unknown
			EstimatedAmount: targetValue,
		})
	}

	// Sort by absolute value diff descending
	sort.Slice(actions, func(i, j int) bool {
		return math.Abs(actions[i].ValueDiff) > math.Abs(actions[j].ValueDiff)
	})

	totalBuy := 0.0
	totalSell := 0.0
	for _, a := range actions {
		if a.Action == "BUY" {
			totalBuy += a.EstimatedAmount
		} else if a.Action == "SELL" {
			totalSell += a.EstimatedAmount
		}
	}

	return &RebalanceResult{
		TotalPortfolioValue: totalPortfolioValue,
		Actions:             actions,
		TotalBuyAmount:      totalBuy,
		TotalSellAmount:     totalSell,
		NetCashFlow:         totalSell - totalBuy,
	}, nil
}
