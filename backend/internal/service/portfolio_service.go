// Package service implements the portfolio business logic, orchestrating
// domain computations and repository calls.
package service

import (
	"context"
	"fmt"
	"time"

	domain "backend/internal/domain/portfolio"
	"backend/internal/repository"
)

// ---------------------------------------------------------------------------
// PriceFetcher interface — decouples from KIS API implementation
// ---------------------------------------------------------------------------

// PriceFetcher fetches current market prices for a list of symbols.
type PriceFetcher interface {
	FetchPrices(ctx context.Context, symbols []string) (map[string]float64, error)
}

// ---------------------------------------------------------------------------
// PortfolioService
// ---------------------------------------------------------------------------

// PortfolioService orchestrates portfolio operations by combining the
// repository (DB access) with pure domain logic (FIFO, tax, rebalance).
type PortfolioService struct {
	repo         *repository.PortfolioRepository
	priceFetcher PriceFetcher
}

// NewPortfolioService creates a new service.
func NewPortfolioService(repo *repository.PortfolioRepository, pf PriceFetcher) *PortfolioService {
	return &PortfolioService{repo: repo, priceFetcher: pf}
}

// ---------------------------------------------------------------------------
// ProcessBuyTrade — record a buy and update position aggregates
// ---------------------------------------------------------------------------

// ProcessBuyTrade records a buy trade into the portfolio system:
//  1. Upsert the PortfolioPosition
//  2. Create a FifoLot
//  3. Recalculate position aggregates from all active lots
func (s *PortfolioService) ProcessBuyTrade(ctx context.Context, input domain.BuyTradeInput) error {
	// 1. Upsert position
	pos, err := s.repo.UpsertPosition(ctx, input)
	if err != nil {
		return fmt.Errorf("process buy: %w", err)
	}

	// 2. Create FIFO lot
	_, err = s.repo.CreateFifoLot(ctx, pos.ID, input.TradeID, time.Now(), input.Price, input.Quantity, input.Fee)
	if err != nil {
		return fmt.Errorf("process buy: create lot: %w", err)
	}

	// 3. Recalculate position aggregates
	return s.recalculatePosition(ctx, pos.ID)
}

// ---------------------------------------------------------------------------
// ProcessSellTrade — FIFO matching, PnL recording, position update
// ---------------------------------------------------------------------------

// ProcessSellTrade executes a sell trade:
//  1. Look up the position
//  2. Fetch active lots (oldest first)
//  3. Run FIFO matching (pure domain)
//  4. Persist lot updates and realized PnL records
//  5. Recalculate position aggregates
func (s *PortfolioService) ProcessSellTrade(ctx context.Context, input domain.SellTradeInput) error {
	// 1. Get position
	pos, err := s.repo.GetPositionByUserSymbol(ctx, input.UserID, input.Symbol)
	if err != nil {
		return fmt.Errorf("process sell: %w", err)
	}

	// 2. Get active lots
	lots, err := s.repo.ListActiveLots(ctx, pos.ID)
	if err != nil {
		return fmt.Errorf("process sell: list lots: %w", err)
	}

	// 3. Pure FIFO computation
	results, err := domain.ComputeFifoSell(lots, input.Price, input.Quantity, input.Fee, pos.Market)
	if err != nil {
		return fmt.Errorf("process sell: fifo: %w", err)
	}

	// 4. Persist each lot result
	now := time.Now()
	for _, r := range results {
		// Find the original lot to get buyPrice
		var buyPrice float64
		for _, lot := range lots {
			if lot.ID == r.LotID {
				buyPrice = lot.BuyPrice
				break
			}
		}

		err := s.repo.CreateRealizedPnL(ctx,
			pos.ID, input.TradeID, r.LotID,
			r.SellQty, buyPrice, input.Price,
			r.GrossPnL, r.LotFee+r.SellFee, r.Tax, r.NetPnL,
			now,
		)
		if err != nil {
			return fmt.Errorf("process sell: create pnl: %w", err)
		}

		if err := s.repo.UpdateLotRemainingQty(ctx, r.LotID, r.NewRemainingQty); err != nil {
			return fmt.Errorf("process sell: update lot: %w", err)
		}
	}

	// 5. Recalculate position aggregates
	return s.recalculatePosition(ctx, pos.ID)
}

// recalculatePosition fetches active lots and recomputes aggregates.
func (s *PortfolioService) recalculatePosition(ctx context.Context, positionID string) error {
	lots, err := s.repo.ListActiveLots(ctx, positionID)
	if err != nil {
		return fmt.Errorf("recalculate: list lots: %w", err)
	}

	agg := domain.RecalculatePositionFromLots(lots)
	return s.repo.UpdatePositionAggregates(ctx, positionID, agg)
}

// ---------------------------------------------------------------------------
// GetPortfolioSummary — with live prices
// ---------------------------------------------------------------------------

// GetPortfolioSummary returns a full portfolio overview with live-priced
// position details.
func (s *PortfolioService) GetPortfolioSummary(ctx context.Context, userID string) (*domain.PortfolioSummary, error) {
	positions, err := s.repo.ListActivePositions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get summary: %w", err)
	}

	if len(positions) == 0 {
		return &domain.PortfolioSummary{
			Positions: []domain.PositionDetail{},
		}, nil
	}

	// Batch-fetch current prices
	symbols := make([]string, len(positions))
	for i, p := range positions {
		symbols[i] = p.Symbol
	}

	prices, err := s.priceFetcher.FetchPrices(ctx, symbols)
	if err != nil {
		// Fallback: use avg cost price if price fetch fails
		prices = make(map[string]float64)
		for _, p := range positions {
			prices[p.Symbol] = p.AvgCostPrice
		}
	}

	// Cumulative realized PnL
	realizedPnL, err := s.repo.SumRealizedPnLByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get summary: realized pnl: %w", err)
	}

	var totalValue, totalCost float64
	details := make([]domain.PositionDetail, 0, len(positions))

	for _, pos := range positions {
		currentPrice, ok := prices[pos.Symbol]
		if !ok {
			currentPrice = pos.AvgCostPrice
		}

		posValue := currentPrice * float64(pos.Quantity)
		posCost := pos.TotalCost
		unrealizedPnL := posValue - posCost

		totalValue += posValue
		totalCost += posCost

		unrealizedPnLPct := 0.0
		if posCost > 0 {
			unrealizedPnLPct = (unrealizedPnL / posCost) * 100
		}

		details = append(details, domain.PositionDetail{
			Symbol:           pos.Symbol,
			SymbolName:       pos.SymbolName,
			Market:           pos.Market,
			Quantity:         pos.Quantity,
			AvgCostPrice:     pos.AvgCostPrice,
			CurrentPrice:     currentPrice,
			TotalCost:        posCost,
			TotalValue:       posValue,
			UnrealizedPnL:    unrealizedPnL,
			UnrealizedPnLPct: unrealizedPnLPct,
			Sector:           pos.Sector,
			SectorName:       pos.SectorName,
			Weight:           0, // computed below
		})
	}

	// Compute portfolio weights
	for i := range details {
		if totalValue > 0 {
			details[i].Weight = (details[i].TotalValue / totalValue) * 100
		}
	}

	unrealizedPnL := totalValue - totalCost
	unrealizedPnLPct := 0.0
	if totalCost > 0 {
		unrealizedPnLPct = (unrealizedPnL / totalCost) * 100
	}

	return &domain.PortfolioSummary{
		TotalValue:       totalValue,
		TotalCost:        totalCost,
		UnrealizedPnL:    unrealizedPnL,
		UnrealizedPnLPct: unrealizedPnLPct,
		RealizedPnL:      realizedPnL,
		TotalPnL:         unrealizedPnL + realizedPnL,
		Positions:        details,
	}, nil
}

// ---------------------------------------------------------------------------
// GetSectorWeights
// ---------------------------------------------------------------------------

// GetSectorWeights returns sector-grouped portfolio data for donut chart rendering.
func (s *PortfolioService) GetSectorWeights(ctx context.Context, userID string) ([]domain.SectorWeight, float64, error) {
	summary, err := s.GetPortfolioSummary(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	sectors := domain.ComputeSectorWeights(summary.Positions)
	return sectors, summary.TotalValue, nil
}

// ---------------------------------------------------------------------------
// GetPnLHistory — realized PnL over time
// ---------------------------------------------------------------------------

// GetPnLHistory returns a chronological list of realized PnL data points.
func (s *PortfolioService) GetPnLHistory(ctx context.Context, userID string, from, to time.Time) ([]domain.PnLHistoryEntry, error) {
	records, err := s.repo.ListRealizedPnLByPeriod(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get pnl history: %w", err)
	}

	entries := make([]domain.PnLHistoryEntry, 0, len(records))
	cumulative := 0.0
	for _, r := range records {
		cumulative += r.NetPnL
		entries = append(entries, domain.PnLHistoryEntry{
			Date:          r.RealizedAt.Format("2006-01-02"),
			RealizedPnL:   r.NetPnL,
			CumulativePnL: cumulative,
		})
	}

	return entries, nil
}

// ---------------------------------------------------------------------------
// SimulateTax — annual tax simulation
// ---------------------------------------------------------------------------

// SimulateTax computes the tax simulation for domestic and overseas holdings.
func (s *PortfolioService) SimulateTax(ctx context.Context, userID string) (*domain.TaxSimulation, error) {
	// KR market aggregates
	_, krGross, krTax, krSellAmt, err := s.repo.SumRealizedPnLByMarket(ctx, userID, domain.MarketKR)
	if err != nil {
		return nil, fmt.Errorf("simulate tax KR: %w", err)
	}
	_ = krGross
	_ = krTax

	// US market aggregates
	_, usGross, _, _, err := s.repo.SumRealizedPnLByMarket(ctx, userID, domain.MarketUS)
	if err != nil {
		return nil, fmt.Errorf("simulate tax US: %w", err)
	}

	// KR: 증권거래세 재계산
	krTransactionTax := krSellAmt * domain.KRTransactionTaxRate

	// US: 연간 양도소득세 with 기본공제
	usTax := domain.CalculateAnnualUSTax(usGross)

	totalTax := krTransactionTax + usTax.TotalTax

	taxableGain := usGross - domain.USBasicDeduction
	if taxableGain < 0 {
		taxableGain = 0
	}

	return &domain.TaxSimulation{
		KR: domain.KRTaxSummary{
			TotalSellAmount: krSellAmt,
			TransactionTax:  krTransactionTax,
		},
		US: domain.USTaxSummary{
			TotalGain:       usGross,
			BasicDeduction:  domain.USBasicDeduction,
			TaxableGain:     taxableGain,
			CapitalGainsTax: usTax.IncomeTax,
		},
		TotalTax: totalTax,
	}, nil
}

// ---------------------------------------------------------------------------
// SimulateRebalance
// ---------------------------------------------------------------------------

// SimulateRebalance wraps the pure domain rebalance simulator with live data.
func (s *PortfolioService) SimulateRebalance(ctx context.Context, userID string, targets []domain.TargetAllocation) (*domain.RebalanceResult, error) {
	summary, err := s.GetPortfolioSummary(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("rebalance: %w", err)
	}

	positions := make([]domain.PositionForRebalance, 0, len(summary.Positions))
	for _, p := range summary.Positions {
		positions = append(positions, domain.PositionForRebalance{
			Symbol:       p.Symbol,
			SymbolName:   p.SymbolName,
			CurrentPrice: p.CurrentPrice,
			TotalValue:   p.TotalValue,
			Weight:       p.Weight,
		})
	}

	return domain.SimulateRebalance(positions, targets, summary.TotalValue)
}
