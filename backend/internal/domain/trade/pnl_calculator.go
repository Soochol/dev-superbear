package trade

import (
	"sort"
	"time"
)

type TradeInput struct {
	Type     string
	Price    float64
	Quantity int
	Fee      float64
	TradedAt time.Time
}

type PnLSummary struct {
	TotalBuyQuantity  int     `json:"total_buy_quantity"`
	TotalSellQuantity int     `json:"total_sell_quantity"`
	RemainingQuantity int     `json:"remaining_quantity"`
	AverageBuyPrice   float64 `json:"average_buy_price"`
	RealizedPnL       float64 `json:"realized_pnl"`
	RealizedReturn    float64 `json:"realized_return"`
	UnrealizedPnL     float64 `json:"unrealized_pnl"`
	UnrealizedReturn  float64 `json:"unrealized_return"`
	TotalFees         float64 `json:"total_fees"`
}

func CalculatePnL(trades []TradeInput, currentPrice float64) PnLSummary {
	if len(trades) == 0 {
		return PnLSummary{}
	}

	sorted := make([]TradeInput, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TradedAt.Before(sorted[j].TradedAt)
	})

	type lot struct {
		price    float64
		quantity int
	}
	var buyQueue []lot

	var totalBuyQty, totalSellQty int
	var totalBuyCost, totalFees, realizedPnL float64

	for _, t := range sorted {
		totalFees += t.Fee

		if t.Type == "BUY" {
			buyQueue = append(buyQueue, lot{price: t.Price, quantity: t.Quantity})
			totalBuyQty += t.Quantity
			totalBuyCost += t.Price * float64(t.Quantity)
		} else {
			totalSellQty += t.Quantity
			remaining := t.Quantity

			for remaining > 0 && len(buyQueue) > 0 {
				front := &buyQueue[0]
				matched := min(remaining, front.quantity)

				realizedPnL += float64(matched) * (t.Price - front.price)
				front.quantity -= matched
				remaining -= matched

				if front.quantity == 0 {
					buyQueue = buyQueue[1:]
				}
			}
		}
	}

	remainingQty := 0
	remainingCost := 0.0
	for _, lot := range buyQueue {
		remainingQty += lot.quantity
		remainingCost += lot.price * float64(lot.quantity)
	}

	avgBuyPrice := 0.0
	if totalBuyQty > 0 {
		avgBuyPrice = totalBuyCost / float64(totalBuyQty)
	}

	unrealizedPnL := 0.0
	if remainingQty > 0 {
		avgRemaining := remainingCost / float64(remainingQty)
		unrealizedPnL = float64(remainingQty) * (currentPrice - avgRemaining)
	}

	realizedReturn := 0.0
	if totalSellQty > 0 && avgBuyPrice > 0 {
		realizedReturn = realizedPnL / (avgBuyPrice * float64(totalSellQty)) * 100
	}
	unrealizedReturn := 0.0
	if remainingQty > 0 && remainingCost > 0 {
		unrealizedReturn = unrealizedPnL / remainingCost * 100
	}

	return PnLSummary{
		TotalBuyQuantity:  totalBuyQty,
		TotalSellQuantity: totalSellQty,
		RemainingQuantity: remainingQty,
		AverageBuyPrice:   avgBuyPrice,
		RealizedPnL:       realizedPnL,
		RealizedReturn:    realizedReturn,
		UnrealizedPnL:     unrealizedPnL,
		UnrealizedReturn:  unrealizedReturn,
		TotalFees:         totalFees,
	}
}
