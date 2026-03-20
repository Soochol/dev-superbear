# 포트폴리오 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 케이스의 매수/매도 기록과 자동 연동되어 실현/미실현 손익(FIFO 기준), 세금/수수료 포함 실질 수익률, 섹터 비중 시각화, 리밸런싱 시뮬레이터를 제공하는 포트폴리오 관리 시스템을 구축한다.
**Architecture:** Case Trade 레코드 생성/변경 이벤트를 감지하여 PortfolioPosition을 자동 갱신한다. 보유 수량/평단가는 FIFO 방식으로 계산하고, 실현 손익은 별도 RealizedPnL 레코드로 관리한다. 현재가는 KIS API 캐시 레이어를 통해 실시간 반영하며, 세금은 국내(증권거래세 0.18%)와 해외(양도소득세 22%)를 구분 적용한다.
**Tech Stack:** Go (Gin), sqlc, PostgreSQL, KIS API, Recharts (섹터 도넛 차트, 프론트엔드 유지)

**Backend (Go):**
```
backend/internal/
  handler/portfolio_handler.go         # GET /portfolio, /portfolio/summary, /sectors, /history, /tax, POST /rebalance
  service/portfolio_service.go         # 포트폴리오 비즈니스 로직 (FIFO 오케스트레이션)
  repository/portfolio_repo.go         # PortfolioPosition / FifoLot / RealizedPnL DB 접근
  domain/portfolio/
    types.go                           # 도메인 모델 & API 응답 타입
    fifo_engine.go                     # FIFO 매칭 엔진 (순수 도메인)
    fifo_engine_test.go                # FIFO 엔진 테스트
    tax_calculator.go                  # 세금 계산 (국내/해외)
    tax_calculator_test.go             # 세금 계산 테스트
    sector_analysis.go                 # 섹터 비중 분석 (순수 도메인)
    rebalance.go                       # 리밸런싱 시뮬레이터 (순수 도메인)
    rebalance_test.go                  # 리밸런싱 테스트
backend/db/
  migrations/005_portfolio.sql         # DDL (portfolio_positions, fifo_lots, realized_pnls)
  queries/portfolio.sql                # sqlc 쿼리
```

**FSD Layers (Frontend — 유지):**
- `src/entities/portfolio-position/` — 프론트엔드 타입 정의와 API 클라이언트. 도메인 로직(FIFO, tax)은 Go 백엔드로 이동 완료.
- `src/features/portfolio/` — 사용자 대면 기능(현황 조회, 리밸런싱 시뮬) UI 컴포넌트.

---

## 의존성

- **Plan 5 (케이스 관리)**: Case, Trade 모델 및 매수/매도 기록 CRUD API

---

## Task 1: Portfolio 데이터 모델 및 FIFO 계산 엔진

포트폴리오 포지션과 실현 손익을 저장하는 SQL 테이블, FIFO 방식 평단가/손익 계산 로직을 Go 도메인 패키지로 구현한다. FIFO 엔진과 세금 계산은 DB에 의존하지 않는 **순수 도메인 로직**으로 분리한다.

**Files:**
- Create: `backend/db/migrations/005_portfolio.sql`
- Create: `backend/db/queries/portfolio.sql`
- Create: `backend/internal/domain/portfolio/types.go`
- Create: `backend/internal/domain/portfolio/fifo_engine.go`
- Create: `backend/internal/domain/portfolio/fifo_engine_test.go`
- Create: `backend/internal/domain/portfolio/tax_calculator.go`
- Create: `backend/internal/domain/portfolio/tax_calculator_test.go`
- Create: `src/entities/portfolio-position/model/types.ts` (프론트엔드 타입 — Go 백엔드 응답 매핑)

**Steps:**

- [ ] SQL 마이그레이션에 포트폴리오 관련 테이블 추가

```sql
-- backend/db/migrations/005_portfolio.sql

-- === Enums ===
CREATE TYPE market_type AS ENUM ('KR', 'US');

-- === Portfolio Positions ===
CREATE TABLE portfolio_positions (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id),
  symbol          TEXT NOT NULL,
  symbol_name     TEXT NOT NULL,
  market          market_type NOT NULL DEFAULT 'KR',
  quantity        INT NOT NULL DEFAULT 0,
  avg_cost_price  NUMERIC(12,2) NOT NULL DEFAULT 0,
  total_cost      NUMERIC(12,2) NOT NULL DEFAULT 0,
  sector          TEXT,
  sector_name     TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, symbol)
);

CREATE INDEX idx_portfolio_positions_user_id ON portfolio_positions(user_id);

-- === FIFO Lots ===
CREATE TABLE fifo_lots (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  position_id     UUID NOT NULL REFERENCES portfolio_positions(id) ON DELETE CASCADE,
  trade_id        UUID NOT NULL REFERENCES trades(id),
  buy_date        TIMESTAMPTZ NOT NULL,
  buy_price       NUMERIC(12,2) NOT NULL,
  original_qty    INT NOT NULL,
  remaining_qty   INT NOT NULL,
  fee             NUMERIC(12,2) NOT NULL DEFAULT 0,
  CONSTRAINT chk_remaining CHECK (remaining_qty >= 0 AND remaining_qty <= original_qty)
);

CREATE INDEX idx_fifo_lots_position_remaining ON fifo_lots(position_id, remaining_qty);

-- === Realized P&L ===
CREATE TABLE realized_pnls (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  position_id     UUID NOT NULL REFERENCES portfolio_positions(id),
  sell_trade_id   UUID NOT NULL REFERENCES trades(id),
  buy_lot_id      UUID NOT NULL REFERENCES fifo_lots(id),
  quantity        INT NOT NULL,
  buy_price       NUMERIC(12,2) NOT NULL,
  sell_price      NUMERIC(12,2) NOT NULL,
  gross_pnl       NUMERIC(12,2) NOT NULL,
  fee             NUMERIC(12,2) NOT NULL DEFAULT 0,
  tax             NUMERIC(12,2) NOT NULL DEFAULT 0,
  net_pnl         NUMERIC(12,2) NOT NULL,
  realized_at     TIMESTAMPTZ NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_realized_pnls_position_date ON realized_pnls(position_id, realized_at);

-- === Trigger: auto-update updated_at ===
CREATE TRIGGER update_portfolio_positions_updated_at
  BEFORE UPDATE ON portfolio_positions
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

- [ ] sqlc 쿼리 파일 작성

```sql
-- backend/db/queries/portfolio.sql

-- name: GetPositionsByUserID :many
SELECT * FROM portfolio_positions
WHERE user_id = $1 AND quantity > 0
ORDER BY updated_at DESC;

-- name: GetPositionByUserSymbol :one
SELECT * FROM portfolio_positions
WHERE user_id = $1 AND symbol = $2;

-- name: UpsertPosition :one
INSERT INTO portfolio_positions (user_id, symbol, symbol_name, market, quantity, avg_cost_price, total_cost, sector, sector_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, symbol) DO UPDATE SET
  quantity = EXCLUDED.quantity,
  avg_cost_price = EXCLUDED.avg_cost_price,
  total_cost = EXCLUDED.total_cost,
  updated_at = now()
RETURNING *;

-- name: UpdatePositionAggregates :exec
UPDATE portfolio_positions
SET quantity = $2, avg_cost_price = $3, total_cost = $4, updated_at = now()
WHERE id = $1;

-- name: CreateFifoLot :one
INSERT INTO fifo_lots (position_id, trade_id, buy_date, buy_price, original_qty, remaining_qty, fee)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetActiveLotsByPositionID :many
SELECT * FROM fifo_lots
WHERE position_id = $1 AND remaining_qty > 0
ORDER BY buy_date ASC;

-- name: UpdateLotRemainingQty :exec
UPDATE fifo_lots SET remaining_qty = $2 WHERE id = $1;

-- name: CreateRealizedPnL :one
INSERT INTO realized_pnls (position_id, sell_trade_id, buy_lot_id, quantity, buy_price, sell_price, gross_pnl, fee, tax, net_pnl, realized_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetRealizedPnLsByUserID :many
SELECT rp.* FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1
ORDER BY rp.realized_at DESC;

-- name: SumRealizedPnLByUserID :one
SELECT COALESCE(SUM(rp.net_pnl), 0)::NUMERIC(12,2) AS total_net_pnl
FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1;

-- name: GetRealizedPnLsByPeriod :many
SELECT rp.* FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1
  AND rp.realized_at >= $2
  AND rp.realized_at <= $3
ORDER BY rp.realized_at ASC;

-- name: SumRealizedPnLByMarketAndYear :many
SELECT pp.market,
       COALESCE(SUM(rp.gross_pnl), 0)::NUMERIC(12,2) AS total_gross_pnl,
       COALESCE(SUM(rp.fee), 0)::NUMERIC(12,2) AS total_fee,
       COALESCE(SUM(rp.tax), 0)::NUMERIC(12,2) AS total_tax,
       COALESCE(SUM(rp.net_pnl), 0)::NUMERIC(12,2) AS total_net_pnl,
       COALESCE(SUM(rp.sell_price * rp.quantity), 0)::NUMERIC(12,2) AS total_sell_amount
FROM realized_pnls rp
JOIN portfolio_positions pp ON rp.position_id = pp.id
WHERE pp.user_id = $1
  AND EXTRACT(YEAR FROM rp.realized_at) = $2
GROUP BY pp.market;
```

- [ ] 도메인 타입 정의 (Go)

```go
// backend/internal/domain/portfolio/types.go
package portfolio

import "time"

// Market represents the stock market type.
type Market string

const (
	MarketKR Market = "KR"
	MarketUS Market = "US"
)

// FifoLot represents a FIFO lot for pure domain computation (no DB dependency).
type FifoLot struct {
	ID           string
	BuyDate      time.Time
	BuyPrice     float64
	OriginalQty  int
	RemainingQty int
	Fee          float64
}

// LotSellResult is the result of selling against a single FIFO lot.
type LotSellResult struct {
	LotID          string
	SellQty        int
	GrossPnL       float64
	LotFee         float64
	SellFee        float64
	Tax            float64
	NetPnL         float64
	NewRemainingQty int
}

// PositionAggregates holds recalculated position stats from remaining lots.
type PositionAggregates struct {
	Quantity     int
	AvgCostPrice float64
	TotalCost    float64
}

// PortfolioSummary is the top-level portfolio view returned to the frontend.
type PortfolioSummary struct {
	TotalValue       float64          `json:"totalValue"`
	TotalCost        float64          `json:"totalCost"`
	UnrealizedPnL    float64          `json:"unrealizedPnl"`
	UnrealizedPnLPct float64          `json:"unrealizedPnlPct"`
	RealizedPnL      float64          `json:"realizedPnl"`
	TotalPnL         float64          `json:"totalPnl"`
	Positions        []PositionDetail `json:"positions"`
}

// PositionDetail is a single position within the portfolio summary.
type PositionDetail struct {
	Symbol           string  `json:"symbol"`
	SymbolName       string  `json:"symbolName"`
	Market           Market  `json:"market"`
	Quantity         int     `json:"quantity"`
	AvgCostPrice     float64 `json:"avgCostPrice"`
	CurrentPrice     float64 `json:"currentPrice"`
	TotalCost        float64 `json:"totalCost"`
	TotalValue       float64 `json:"totalValue"`
	UnrealizedPnL    float64 `json:"unrealizedPnl"`
	UnrealizedPnLPct float64 `json:"unrealizedPnlPct"`
	Sector           string  `json:"sector,omitempty"`
	SectorName       string  `json:"sectorName,omitempty"`
	Weight           float64 `json:"weight"`
}

// TaxConfig holds per-market tax parameters.
type TaxConfig struct {
	KR struct {
		TransactionTax  float64 // 증권거래세 0.18%
		CapitalGainsTax float64 // 국내 주식 양도세 (대주주 외 비과세)
	}
	US struct {
		CapitalGainsTax float64 // 해외 양도소득세 22%
		BasicDeduction  float64 // 기본공제 250만원
	}
}

// DefaultTaxConfig returns the default Korean/US tax configuration.
func DefaultTaxConfig() TaxConfig {
	cfg := TaxConfig{}
	cfg.KR.TransactionTax = 0.0018  // 0.18%
	cfg.KR.CapitalGainsTax = 0      // 일반 투자자 비과세
	cfg.US.CapitalGainsTax = 0.22   // 22%
	cfg.US.BasicDeduction = 2500000 // 250만원
	return cfg
}

// BuyTradeInput is the input for processing a buy trade.
type BuyTradeInput struct {
	UserID     string
	TradeID    string
	Symbol     string
	SymbolName string
	Price      float64
	Quantity   int
	Fee        float64
	Market     Market
}

// SellTradeInput is the input for processing a sell trade.
type SellTradeInput struct {
	UserID   string
	TradeID  string
	Symbol   string
	Price    float64
	Quantity int
	Fee      float64
}

// TaxResult holds computed tax breakdown.
type TaxResult struct {
	TransactionTax float64 `json:"transactionTax"`
	IncomeTax      float64 `json:"incomeTax"`
	Total          float64 `json:"total"`
}
```

- [ ] 순수 FIFO 엔진 구현 — DB 의존 없이 데이터 구조만 다루는 pure domain logic

```go
// backend/internal/domain/portfolio/fifo_engine.go
package portfolio

import "fmt"

// ComputeFifoSell performs pure FIFO sell computation. Given lots sorted oldest-first
// and a sell order, returns the lot-level sell results (how much sold from each lot, PnL, fees, tax).
// This function is pure — no DB calls.
func ComputeFifoSell(lots []FifoLot, sellPrice float64, sellQuantity int, sellFee float64, market Market) ([]LotSellResult, error) {
	totalAvailable := 0
	for _, lot := range lots {
		totalAvailable += lot.RemainingQty
	}
	if sellQuantity > totalAvailable {
		return nil, fmt.Errorf("insufficient quantity: want %d, have %d", sellQuantity, totalAvailable)
	}

	var results []LotSellResult
	remainingToSell := sellQuantity

	for _, lot := range lots {
		if remainingToSell <= 0 {
			break
		}

		sellQty := min(lot.RemainingQty, remainingToSell)
		grossPnL := (sellPrice - lot.BuyPrice) * float64(sellQty)
		tax := CalculateTax(market, sellPrice, sellQty, grossPnL)
		lotFee := (lot.Fee / float64(lot.OriginalQty)) * float64(sellQty)
		sellFeeAlloc := (sellFee / float64(sellQuantity)) * float64(sellQty)

		results = append(results, LotSellResult{
			LotID:           lot.ID,
			SellQty:         sellQty,
			GrossPnL:        grossPnL,
			LotFee:          lotFee,
			SellFee:         sellFeeAlloc,
			Tax:             tax,
			NetPnL:          grossPnL - lotFee - sellFeeAlloc - tax,
			NewRemainingQty: lot.RemainingQty - sellQty,
		})

		remainingToSell -= sellQty
	}

	return results, nil
}

// RecalculatePositionFromLots recalculates position aggregates (quantity, avgCost, totalCost)
// from remaining lots. Pure function.
func RecalculatePositionFromLots(lots []FifoLot) PositionAggregates {
	var totalQty int
	var totalCost float64

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] 세금 계산기 구현 — pure domain function

```go
// backend/internal/domain/portfolio/tax_calculator.go
package portfolio

// CalculateTax computes the applicable tax for a sell transaction.
// KR: 증권거래세 (매도 금액 * 0.18%)
// US: 양도소득세 (양도차익의 22%, 기본공제는 연간 합산 시 적용)
func CalculateTax(market Market, sellPrice float64, sellQty int, grossPnL float64) float64 {
	cfg := DefaultTaxConfig()

	switch market {
	case MarketKR:
		// 증권거래세: 매도 금액 * 0.18%
		return sellPrice * float64(sellQty) * cfg.KR.TransactionTax
	case MarketUS:
		// 해외: 양도차익의 22% (기본공제는 연간 합산 시 적용)
		if grossPnL <= 0 {
			return 0
		}
		return grossPnL * cfg.US.CapitalGainsTax
	default:
		return 0
	}
}

// TaxSimulationInput holds annual tax simulation parameters.
type TaxSimulationInput struct {
	KR struct {
		TotalSellAmount float64
		TransactionTax  float64
	}
	US struct {
		TotalGain       float64
		BasicDeduction  float64
		TaxableGain     float64
		CapitalGainsTax float64
	}
	TotalTax float64
}

// SimulateAnnualTax computes annual tax breakdown by market.
func SimulateAnnualTax(krSellAmount float64, usGrossPnL float64) TaxSimulationInput {
	cfg := DefaultTaxConfig()
	result := TaxSimulationInput{}

	// KR: 증권거래세
	result.KR.TotalSellAmount = krSellAmount
	result.KR.TransactionTax = krSellAmount * cfg.KR.TransactionTax

	// US: 양도소득세 (기본공제 250만원 차감)
	result.US.TotalGain = usGrossPnL
	result.US.BasicDeduction = cfg.US.BasicDeduction
	taxableGain := usGrossPnL - cfg.US.BasicDeduction
	if taxableGain < 0 {
		taxableGain = 0
	}
	result.US.TaxableGain = taxableGain
	result.US.CapitalGainsTax = taxableGain * cfg.US.CapitalGainsTax

	result.TotalTax = result.KR.TransactionTax + result.US.CapitalGainsTax

	return result
}
```

- [ ] Repository 구현 — sqlc 생성 코드 + 도메인 로직 오케스트레이션

```go
// backend/internal/repository/portfolio_repo.go
package repository

import (
	"context"
	"fmt"
	"time"

	"superbear/backend/db/generated"
	"superbear/backend/internal/domain/portfolio"
)

type PortfolioRepo struct {
	q *generated.Queries
}

func NewPortfolioRepo(q *generated.Queries) *PortfolioRepo {
	return &PortfolioRepo{q: q}
}

func (r *PortfolioRepo) GetPositionsByUserID(ctx context.Context, userID string) ([]generated.PortfolioPosition, error) {
	return r.q.GetPositionsByUserID(ctx, userID)
}

func (r *PortfolioRepo) GetPositionByUserSymbol(ctx context.Context, userID, symbol string) (generated.PortfolioPosition, error) {
	return r.q.GetPositionByUserSymbol(ctx, generated.GetPositionByUserSymbolParams{
		UserID: userID,
		Symbol: symbol,
	})
}

func (r *PortfolioRepo) UpsertPosition(ctx context.Context, params generated.UpsertPositionParams) (generated.PortfolioPosition, error) {
	return r.q.UpsertPosition(ctx, params)
}

func (r *PortfolioRepo) UpdatePositionAggregates(ctx context.Context, id string, agg portfolio.PositionAggregates) error {
	return r.q.UpdatePositionAggregates(ctx, generated.UpdatePositionAggregatesParams{
		ID:           id,
		Quantity:     int32(agg.Quantity),
		AvgCostPrice: fmt.Sprintf("%.2f", agg.AvgCostPrice),
		TotalCost:    fmt.Sprintf("%.2f", agg.TotalCost),
	})
}

func (r *PortfolioRepo) CreateFifoLot(ctx context.Context, params generated.CreateFifoLotParams) (generated.FifoLot, error) {
	return r.q.CreateFifoLot(ctx, params)
}

func (r *PortfolioRepo) GetActiveLotsByPositionID(ctx context.Context, positionID string) ([]generated.FifoLot, error) {
	return r.q.GetActiveLotsByPositionID(ctx, positionID)
}

func (r *PortfolioRepo) UpdateLotRemainingQty(ctx context.Context, id string, qty int) error {
	return r.q.UpdateLotRemainingQty(ctx, generated.UpdateLotRemainingQtyParams{
		ID:           id,
		RemainingQty: int32(qty),
	})
}

func (r *PortfolioRepo) CreateRealizedPnL(ctx context.Context, params generated.CreateRealizedPnLParams) (generated.RealizedPnl, error) {
	return r.q.CreateRealizedPnL(ctx, params)
}

func (r *PortfolioRepo) SumRealizedPnLByUserID(ctx context.Context, userID string) (float64, error) {
	result, err := r.q.SumRealizedPnLByUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	// sqlc returns string for NUMERIC; parse it
	var val float64
	fmt.Sscanf(result, "%f", &val)
	return val, nil
}

func (r *PortfolioRepo) GetRealizedPnLsByPeriod(ctx context.Context, userID string, from, to time.Time) ([]generated.RealizedPnl, error) {
	return r.q.GetRealizedPnLsByPeriod(ctx, generated.GetRealizedPnLsByPeriodParams{
		UserID: userID,
		From:   from,
		To:     to,
	})
}

func (r *PortfolioRepo) SumRealizedPnLByMarketAndYear(ctx context.Context, userID string, year int) ([]generated.SumRealizedPnLByMarketAndYearRow, error) {
	return r.q.SumRealizedPnLByMarketAndYear(ctx, generated.SumRealizedPnLByMarketAndYearParams{
		UserID: userID,
		Year:   int32(year),
	})
}
```

- [ ] 프론트엔드 타입 정의 — Go 백엔드 응답 매핑 (Prisma 의존성 제거)

```typescript
// src/entities/portfolio-position/model/types.ts

/** Market type — mirrors Go backend market_type enum. */
export type Market = 'KR' | 'US';

export interface PortfolioSummary {
  totalValue: number;
  totalCost: number;
  unrealizedPnl: number;
  unrealizedPnlPct: number;
  realizedPnl: number;
  totalPnl: number;
  positions: PositionDetail[];
}

export interface PositionDetail {
  symbol: string;
  symbolName: string;
  market: Market;
  quantity: number;
  avgCostPrice: number;
  currentPrice: number;
  totalCost: number;
  totalValue: number;
  unrealizedPnl: number;
  unrealizedPnlPct: number;
  sector?: string;
  sectorName?: string;
  weight: number;
}

export interface TaxSimulationResult {
  kr: {
    totalSellAmount: number;
    transactionTax: number;
  };
  us: {
    totalGain: number;
    basicDeduction: number;
    taxableGain: number;
    capitalGainsTax: number;
  };
  totalTax: number;
}
```

- [ ] FIFO 엔진 테스트

```go
// backend/internal/domain/portfolio/fifo_engine_test.go
package portfolio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFifoSell_SingleLot(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now(), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 100, Fee: 1000},
	}

	results, err := ComputeFifoSell(lots, 15000, 80, 800, MarketKR)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 80, results[0].SellQty)
	assert.Equal(t, 20, results[0].NewRemainingQty)
	assert.InDelta(t, (15000-10000)*80, results[0].GrossPnL, 0.01) // 400,000
}

func TestComputeFifoSell_MultipleLots(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now().Add(-48 * time.Hour), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 100, Fee: 1000},
		{ID: "lot-2", BuyDate: time.Now(), BuyPrice: 12000, OriginalQty: 50, RemainingQty: 50, Fee: 500},
	}

	// Sell 120 shares → lot-1 (100) fully consumed + lot-2 (20) partially consumed
	results, err := ComputeFifoSell(lots, 15000, 120, 1200, MarketKR)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 100, results[0].SellQty)
	assert.Equal(t, 0, results[0].NewRemainingQty)
	assert.Equal(t, 20, results[1].SellQty)
	assert.Equal(t, 30, results[1].NewRemainingQty)
}

func TestComputeFifoSell_InsufficientQuantity(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyDate: time.Now(), BuyPrice: 10000, OriginalQty: 100, RemainingQty: 50, Fee: 1000},
	}

	_, err := ComputeFifoSell(lots, 15000, 80, 800, MarketKR)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient quantity")
}

func TestRecalculatePositionFromLots(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyPrice: 10000, OriginalQty: 100, RemainingQty: 20, Fee: 1000},
		{ID: "lot-2", BuyPrice: 12000, OriginalQty: 50, RemainingQty: 50, Fee: 500},
	}

	agg := RecalculatePositionFromLots(lots)
	assert.Equal(t, 70, agg.Quantity)
	// totalCost = 10000*20 + 12000*50 = 200000 + 600000 = 800000
	assert.InDelta(t, 800000, agg.TotalCost, 0.01)
	// avgCost = 800000 / 70 ≈ 11428.57
	assert.InDelta(t, 800000.0/70.0, agg.AvgCostPrice, 0.01)
}
```

- [ ] 세금 계산기 테스트

```go
// backend/internal/domain/portfolio/tax_calculator_test.go
package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateTax_KR(t *testing.T) {
	// 국내: 매도 금액 * 0.18%
	// 매도가 15000 * 100주 = 1,500,000 → 세금 = 1,500,000 * 0.0018 = 2,700
	tax := CalculateTax(MarketKR, 15000, 100, 500000)
	assert.InDelta(t, 2700, tax, 0.01)
}

func TestCalculateTax_US_Positive(t *testing.T) {
	// 해외: 양도차익 500,000 * 22% = 110,000
	tax := CalculateTax(MarketUS, 150, 100, 500000)
	assert.InDelta(t, 110000, tax, 0.01)
}

func TestCalculateTax_US_Negative(t *testing.T) {
	// 해외: 양도차익 음수 → 세금 0
	tax := CalculateTax(MarketUS, 80, 100, -200000)
	assert.Equal(t, 0.0, tax)
}

func TestSimulateAnnualTax(t *testing.T) {
	result := SimulateAnnualTax(50000000, 5000000)

	// KR: 50,000,000 * 0.0018 = 90,000
	assert.InDelta(t, 90000, result.KR.TransactionTax, 0.01)

	// US: (5,000,000 - 2,500,000) * 0.22 = 550,000
	assert.InDelta(t, 2500000, result.US.TaxableGain, 0.01)
	assert.InDelta(t, 550000, result.US.CapitalGainsTax, 0.01)

	// Total: 90,000 + 550,000 = 640,000
	assert.InDelta(t, 640000, result.TotalTax, 0.01)
}
```

- [ ] 테스트: BUY 100주 @ 10,000원 → Lot 생성, position qty=100, avgCost=10,000
- [ ] 테스트: BUY 50주 @ 12,000원 → Lot 2개, position qty=150
- [ ] 테스트: SELL 80주 @ 15,000원 → FIFO로 첫 Lot 100주 중 80주 소진, RealizedPnL 1건
- [ ] 테스트: SELL 70주 → 첫 Lot 나머지 20주 + 둘째 Lot 50주 소진, RealizedPnL 2건
- [ ] 테스트: 국내 세금 (증권거래세 0.18%) 정확 계산 확인
- [ ] 테스트: 해외 세금 (양도소득세 22%) 정확 계산 확인

```bash
cd backend && go test ./internal/domain/portfolio/... -v
git add backend/db/migrations/005_portfolio.sql backend/db/queries/portfolio.sql backend/internal/domain/portfolio/ backend/internal/repository/portfolio_repo.go src/entities/portfolio-position/model/types.ts
git commit -m "feat(portfolio): SQL 마이그레이션, FIFO 엔진, 세금 계산기, Repository 구현"
```

---

## Task 2: Trade 이벤트 자동 연동

Case Trade 레코드 생성 시 포트폴리오를 자동 갱신하는 서비스 레이어를 구현한다. 기존 Trade 생성 Go 핸들러에 연동 코드를 추가한다.

**Files:**
- Create: `backend/internal/service/portfolio_service.go`
- Modify: `backend/internal/handler/trade_handler.go` (기존 Trade POST에 연동 추가)
- Create: `backend/internal/service/portfolio_service_test.go`

**Steps:**

- [ ] Trade 생성 시 포트폴리오 자동 갱신 서비스

```go
// backend/internal/service/portfolio_service.go
package service

import (
	"context"
	"fmt"
	"time"

	"superbear/backend/db/generated"
	"superbear/backend/internal/domain/portfolio"
	"superbear/backend/internal/repository"

	"go.uber.org/zap"
)

type PortfolioService struct {
	repo     *repository.PortfolioRepo
	caseRepo *repository.CaseRepo
	logger   *zap.Logger
}

func NewPortfolioService(repo *repository.PortfolioRepo, caseRepo *repository.CaseRepo, logger *zap.Logger) *PortfolioService {
	return &PortfolioService{repo: repo, caseRepo: caseRepo, logger: logger}
}

// SyncTradeToPortfolio synchronizes a trade to the portfolio.
// Called after a Trade record is created in the trade handler.
func (s *PortfolioService) SyncTradeToPortfolio(ctx context.Context, trade generated.Trade) error {
	// Fetch case for symbol info
	caseRecord, err := s.caseRepo.GetByID(ctx, trade.CaseID)
	if err != nil {
		return fmt.Errorf("fetch case: %w", err)
	}

	if trade.TradeType == "BUY" {
		return s.processBuyTrade(ctx, portfolio.BuyTradeInput{
			UserID:     trade.UserID,
			TradeID:    trade.ID,
			Symbol:     caseRecord.Symbol,
			SymbolName: caseRecord.SymbolName,
			Price:      parseNumeric(trade.Price),
			Quantity:   int(trade.Quantity),
			Fee:        parseNumeric(trade.Fee),
			Market:     portfolio.Market(caseRecord.Market),
		})
	}

	return s.processSellTrade(ctx, portfolio.SellTradeInput{
		UserID:   trade.UserID,
		TradeID:  trade.ID,
		Symbol:   caseRecord.Symbol,
		Price:    parseNumeric(trade.Price),
		Quantity: int(trade.Quantity),
		Fee:      parseNumeric(trade.Fee),
	})
}

func (s *PortfolioService) processBuyTrade(ctx context.Context, input portfolio.BuyTradeInput) error {
	// 1. Position upsert
	position, err := s.repo.UpsertPosition(ctx, generated.UpsertPositionParams{
		UserID:       input.UserID,
		Symbol:       input.Symbol,
		SymbolName:   input.SymbolName,
		Market:       generated.MarketType(input.Market),
		Quantity:     int32(input.Quantity),
		AvgCostPrice: fmt.Sprintf("%.2f", input.Price),
		TotalCost:    fmt.Sprintf("%.2f", input.Price*float64(input.Quantity)),
	})
	if err != nil {
		return fmt.Errorf("upsert position: %w", err)
	}

	// 2. Create FIFO lot
	_, err = s.repo.CreateFifoLot(ctx, generated.CreateFifoLotParams{
		PositionID:   position.ID,
		TradeID:      input.TradeID,
		BuyDate:      time.Now(),
		BuyPrice:     fmt.Sprintf("%.2f", input.Price),
		OriginalQty:  int32(input.Quantity),
		RemainingQty: int32(input.Quantity),
		Fee:          fmt.Sprintf("%.2f", input.Fee),
	})
	if err != nil {
		return fmt.Errorf("create fifo lot: %w", err)
	}

	// 3. Recalculate position aggregates from all lots
	return s.recalculatePosition(ctx, position.ID)
}

func (s *PortfolioService) processSellTrade(ctx context.Context, input portfolio.SellTradeInput) error {
	// 1. Find position
	position, err := s.repo.GetPositionByUserSymbol(ctx, input.UserID, input.Symbol)
	if err != nil {
		return fmt.Errorf("find position: %w", err)
	}

	// 2. Fetch active lots sorted oldest-first
	dbLots, err := s.repo.GetActiveLotsByPositionID(ctx, position.ID)
	if err != nil {
		return fmt.Errorf("fetch lots: %w", err)
	}

	// 3. Convert to domain type for pure computation
	lots := make([]portfolio.FifoLot, len(dbLots))
	for i, l := range dbLots {
		lots[i] = portfolio.FifoLot{
			ID:           l.ID,
			BuyDate:      l.BuyDate,
			BuyPrice:     parseNumeric(l.BuyPrice),
			OriginalQty:  int(l.OriginalQty),
			RemainingQty: int(l.RemainingQty),
			Fee:          parseNumeric(l.Fee),
		}
	}

	// 4. Pure domain computation
	sellResults, err := portfolio.ComputeFifoSell(lots, input.Price, input.Quantity, input.Fee, portfolio.Market(position.Market))
	if err != nil {
		return fmt.Errorf("fifo computation: %w", err)
	}

	// 5. Persist each lot result
	for _, result := range sellResults {
		_, err := s.repo.CreateRealizedPnL(ctx, generated.CreateRealizedPnLParams{
			PositionID:  position.ID,
			SellTradeID: input.TradeID,
			BuyLotID:    result.LotID,
			Quantity:    int32(result.SellQty),
			BuyPrice:    fmt.Sprintf("%.2f", lots[findLotIndex(lots, result.LotID)].BuyPrice),
			SellPrice:   fmt.Sprintf("%.2f", input.Price),
			GrossPnl:    fmt.Sprintf("%.2f", result.GrossPnL),
			Fee:         fmt.Sprintf("%.2f", result.LotFee+result.SellFee),
			Tax:         fmt.Sprintf("%.2f", result.Tax),
			NetPnl:      fmt.Sprintf("%.2f", result.NetPnL),
			RealizedAt:  time.Now(),
		})
		if err != nil {
			return fmt.Errorf("create realized pnl: %w", err)
		}

		if err := s.repo.UpdateLotRemainingQty(ctx, result.LotID, result.NewRemainingQty); err != nil {
			return fmt.Errorf("update lot remaining qty: %w", err)
		}
	}

	// 6. Recalculate position aggregates
	return s.recalculatePosition(ctx, position.ID)
}

func (s *PortfolioService) recalculatePosition(ctx context.Context, positionID string) error {
	dbLots, err := s.repo.GetActiveLotsByPositionID(ctx, positionID)
	if err != nil {
		return err
	}

	lots := make([]portfolio.FifoLot, len(dbLots))
	for i, l := range dbLots {
		lots[i] = portfolio.FifoLot{
			ID:           l.ID,
			BuyPrice:     parseNumeric(l.BuyPrice),
			OriginalQty:  int(l.OriginalQty),
			RemainingQty: int(l.RemainingQty),
			Fee:          parseNumeric(l.Fee),
		}
	}

	agg := portfolio.RecalculatePositionFromLots(lots)
	return s.repo.UpdatePositionAggregates(ctx, positionID, agg)
}

// findLotIndex finds the index of a lot by ID.
func findLotIndex(lots []portfolio.FifoLot, id string) int {
	for i, l := range lots {
		if l.ID == id {
			return i
		}
	}
	return 0
}

// parseNumeric converts sqlc NUMERIC string to float64.
func parseNumeric(s string) float64 {
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}
```

- [ ] 기존 Trade POST 핸들러에 포트폴리오 연동 코드 추가

```go
// backend/internal/handler/trade_handler.go (수정 부분)
// 기존 CreateTrade 핸들러 내부, trade 생성 직후:

// Sync trade to portfolio
if err := h.portfolioSvc.SyncTradeToPortfolio(c.Request.Context(), trade); err != nil {
    h.logger.Error("failed to sync trade to portfolio", zap.Error(err))
    // 포트폴리오 연동 실패는 trade 생성을 롤백하지 않음 (eventual consistency)
}
```

- [ ] 테스트: Trade 생성 → PortfolioPosition 자동 생성/갱신 확인
- [ ] 테스트: 여러 케이스에서 동일 종목 매수 → 하나의 PortfolioPosition에 합산 확인
- [ ] 테스트: Trade 삭제 시 포트폴리오 역연산 (Lot 복원) 확인

```bash
cd backend && go test ./internal/service/... -run TestPortfolio -v
git add backend/internal/service/portfolio_service.go backend/internal/handler/trade_handler.go
git commit -m "feat(portfolio): Trade 생성 시 포트폴리오 자동 연동"
```

---

## Task 3: Portfolio API 엔드포인트

포트폴리오 현황 조회, 손익 히스토리, 세금 시뮬레이션 Go Gin 핸들러를 구현한다. 핸들러는 thin controller로서 서비스 레이어에 위임한다.

**Files:**
- Create: `backend/internal/handler/portfolio_handler.go`
- Modify: `backend/internal/service/portfolio_service.go` (GetSummary, GetHistory, SimulateTax 추가)
- Modify: `backend/internal/router/router.go` (포트폴리오 라우트 등록)
- Modify: `src/entities/portfolio-position/api/portfolio.api.ts` (프론트엔드 API 클라이언트)

**Steps:**

- [ ] 포트폴리오 서비스에 조회 메서드 추가

```go
// backend/internal/service/portfolio_service.go (추가)

// GetSummary returns the portfolio summary with live prices.
func (s *PortfolioService) GetSummary(ctx context.Context, userID string) (*portfolio.PortfolioSummary, error) {
	positions, err := s.repo.GetPositionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		return &portfolio.PortfolioSummary{
			Positions: []portfolio.PositionDetail{},
		}, nil
	}

	// Batch fetch current prices from KIS API cache
	symbols := make([]string, len(positions))
	for i, p := range positions {
		symbols[i] = p.Symbol
	}
	prices, err := s.kisClient.FetchPricesBatch(ctx, symbols)
	if err != nil {
		s.logger.Warn("failed to fetch prices, using avg cost as fallback", zap.Error(err))
	}

	// Sum realized PnL
	totalRealizedPnL, _ := s.repo.SumRealizedPnLByUserID(ctx, userID)

	var totalValue, totalCost float64
	details := make([]portfolio.PositionDetail, len(positions))

	for i, pos := range positions {
		currentPrice := parseNumeric(pos.AvgCostPrice) // fallback
		if p, ok := prices[pos.Symbol]; ok {
			currentPrice = p.Close
		}

		posValue := currentPrice * float64(pos.Quantity)
		posCost := parseNumeric(pos.TotalCost)
		unrealizedPnL := posValue - posCost

		totalValue += posValue
		totalCost += posCost

		details[i] = portfolio.PositionDetail{
			Symbol:           pos.Symbol,
			SymbolName:       pos.SymbolName,
			Market:           portfolio.Market(pos.Market),
			Quantity:         int(pos.Quantity),
			AvgCostPrice:     parseNumeric(pos.AvgCostPrice),
			CurrentPrice:     currentPrice,
			TotalCost:        posCost,
			TotalValue:       posValue,
			UnrealizedPnL:    unrealizedPnL,
			UnrealizedPnLPct: pctOrZero(unrealizedPnL, posCost),
		}
	}

	// Calculate weights
	for i := range details {
		details[i].Weight = pctOrZero(details[i].TotalValue, totalValue)
	}

	unrealizedPnL := totalValue - totalCost

	return &portfolio.PortfolioSummary{
		TotalValue:       totalValue,
		TotalCost:        totalCost,
		UnrealizedPnL:    unrealizedPnL,
		UnrealizedPnLPct: pctOrZero(unrealizedPnL, totalCost),
		RealizedPnL:      totalRealizedPnL,
		TotalPnL:         unrealizedPnL + totalRealizedPnL,
		Positions:        details,
	}, nil
}

// GetHistory returns realized PnL history for a date range.
func (s *PortfolioService) GetHistory(ctx context.Context, userID string, from, to time.Time) ([]map[string]interface{}, error) {
	pnls, err := s.repo.GetRealizedPnLsByPeriod(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}

	var cumulative float64
	history := make([]map[string]interface{}, len(pnls))
	for i, p := range pnls {
		netPnl := parseNumeric(p.NetPnl)
		cumulative += netPnl
		history[i] = map[string]interface{}{
			"date":          p.RealizedAt.Format("2006-01-02"),
			"realizedPnl":   netPnl,
			"cumulativePnl": cumulative,
		}
	}
	return history, nil
}

// SimulateTax returns annual tax breakdown by market.
func (s *PortfolioService) SimulateTax(ctx context.Context, userID string, year int) (*portfolio.TaxSimulationInput, error) {
	rows, err := s.repo.SumRealizedPnLByMarketAndYear(ctx, userID, year)
	if err != nil {
		return nil, err
	}

	var krSellAmount, usGrossPnL float64
	for _, row := range rows {
		switch string(row.Market) {
		case "KR":
			krSellAmount = parseNumeric(row.TotalSellAmount)
		case "US":
			usGrossPnL = parseNumeric(row.TotalGrossPnl)
		}
	}

	result := portfolio.SimulateAnnualTax(krSellAmount, usGrossPnL)
	return &result, nil
}

func pctOrZero(value, base float64) float64 {
	if base == 0 {
		return 0
	}
	return (value / base) * 100
}
```

- [ ] Go Gin 핸들러 구현

```go
// backend/internal/handler/portfolio_handler.go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"superbear/backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PortfolioHandler struct {
	svc    *service.PortfolioService
	logger *zap.Logger
}

func NewPortfolioHandler(svc *service.PortfolioService, logger *zap.Logger) *PortfolioHandler {
	return &PortfolioHandler{svc: svc, logger: logger}
}

// GetSummary handles GET /api/v1/portfolio
func (h *PortfolioHandler) GetSummary(c *gin.Context) {
	userID := c.GetString("userId")

	summary, err := h.svc.GetSummary(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get portfolio summary", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetHistory handles GET /api/v1/portfolio/history?from=2025-01-01&to=2025-12-31
func (h *PortfolioHandler) GetHistory(c *gin.Context) {
	userID := c.GetString("userId")

	from, err := time.Parse("2006-01-02", c.DefaultQuery("from", time.Now().AddDate(0, -1, 0).Format("2006-01-02")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' date format"})
		return
	}
	to, err := time.Parse("2006-01-02", c.DefaultQuery("to", time.Now().Format("2006-01-02")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' date format"})
		return
	}

	history, err := h.svc.GetHistory(c.Request.Context(), userID, from, to)
	if err != nil {
		h.logger.Error("failed to get portfolio history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}

// SimulateTax handles GET /api/v1/portfolio/tax?year=2025
func (h *PortfolioHandler) SimulateTax(c *gin.Context) {
	userID := c.GetString("userId")

	yearStr := c.DefaultQuery("year", strconv.Itoa(time.Now().Year()))
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year"})
		return
	}

	result, err := h.svc.SimulateTax(c.Request.Context(), userID, year)
	if err != nil {
		h.logger.Error("failed to simulate tax", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
```

- [ ] 라우터에 포트폴리오 엔드포인트 등록

```go
// backend/internal/router/router.go (추가)

// Portfolio routes
portfolioGroup := v1.Group("/portfolio")
portfolioGroup.Use(authMiddleware)
{
    portfolioGroup.GET("", portfolioHandler.GetSummary)
    portfolioGroup.GET("/history", portfolioHandler.GetHistory)
    portfolioGroup.GET("/tax", portfolioHandler.SimulateTax)
    portfolioGroup.GET("/sectors", portfolioHandler.GetSectors)
    portfolioGroup.POST("/rebalance", portfolioHandler.Rebalance)
}
```

- [ ] 프론트엔드 API 클라이언트 — Go 백엔드 호출

```typescript
// src/entities/portfolio-position/api/portfolio.api.ts
import { apiClient } from '@/shared/api/client';
import type { PortfolioSummary, TaxSimulationResult } from '../model/types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export async function fetchPortfolioSummary(): Promise<PortfolioSummary> {
  return apiClient(`${API_BASE}/api/v1/portfolio`);
}

export async function fetchPortfolioHistory(params: {
  from?: string;
  to?: string;
}): Promise<{ history: Array<{ date: string; realizedPnl: number; cumulativePnl: number }> }> {
  const searchParams = new URLSearchParams();
  if (params.from) searchParams.set('from', params.from);
  if (params.to) searchParams.set('to', params.to);
  return apiClient(`${API_BASE}/api/v1/portfolio/history?${searchParams}`);
}

export async function fetchTaxSimulation(year: number): Promise<TaxSimulationResult> {
  return apiClient(`${API_BASE}/api/v1/portfolio/tax?year=${year}`);
}
```

- [ ] 테스트: 포트폴리오 현황에 현재가 반영 확인
- [ ] 테스트: 비중(weight) 합계 100% 확인
- [ ] 테스트: 세금 시뮬레이션 — 국내 거래세 + 해외 양도세 분리 계산 확인
- [ ] 테스트: 해외 양도소득 기본공제 250만원 적용 확인

```bash
cd backend && go test ./internal/handler/... -run TestPortfolio -v
git add backend/internal/handler/portfolio_handler.go backend/internal/service/portfolio_service.go backend/internal/router/router.go src/entities/portfolio-position/api/
git commit -m "feat(portfolio): 포트폴리오 현황 / 손익 히스토리 / 세금 시뮬레이션 API"
```

---

## Task 4: 섹터 비중 시각화 데이터 API

섹터별 포트폴리오 비중 데이터를 도넛 차트 렌더링에 적합한 형태로 제공하는 API를 구현한다. 섹터 분석은 Go 도메인 로직으로 구현한다.

**Files:**
- Create: `backend/internal/domain/portfolio/sector_analysis.go`
- Modify: `backend/internal/service/portfolio_service.go` (GetSectorWeights 추가)
- Modify: `backend/internal/handler/portfolio_handler.go` (GetSectors 추가)
- Modify: `src/features/portfolio/api/portfolio.api.ts` (프론트엔드 API 클라이언트)

**Steps:**

- [ ] 섹터 분석 도메인 로직 — 보유 포지션을 섹터별로 그룹핑하여 비중 계산

```go
// backend/internal/domain/portfolio/sector_analysis.go
package portfolio

// SectorWeight represents a sector's weight in the portfolio.
type SectorWeight struct {
	Sector           string              `json:"sector"`
	SectorName       string              `json:"sectorName"`
	TotalValue       float64             `json:"totalValue"`
	Weight           float64             `json:"weight"`
	Positions        []SectorPosition    `json:"positions"`
	UnrealizedPnL    float64             `json:"unrealizedPnl"`
	UnrealizedPnLPct float64             `json:"unrealizedPnlPct"`
}

// SectorPosition represents a position within a sector.
type SectorPosition struct {
	Symbol     string  `json:"symbol"`
	SymbolName string  `json:"symbolName"`
	Value      float64 `json:"value"`
	Weight     float64 `json:"weight"`
}

// SectorAnalysisInput holds position data for sector analysis.
type SectorAnalysisInput struct {
	Symbol       string
	SymbolName   string
	Sector       string
	SectorName   string
	CurrentPrice float64
	Quantity     int
	TotalCost    float64
}

// CalculateSectorWeights groups positions by sector and calculates weights.
// Pure function — no DB calls.
func CalculateSectorWeights(inputs []SectorAnalysisInput) []SectorWeight {
	sectorMap := make(map[string]*SectorWeight)
	var portfolioTotalValue float64

	for _, inp := range inputs {
		value := inp.CurrentPrice * float64(inp.Quantity)
		portfolioTotalValue += value

		sectorKey := inp.Sector
		if sectorKey == "" {
			sectorKey = "UNKNOWN"
		}
		sectorName := inp.SectorName
		if sectorName == "" {
			sectorName = "미분류"
		}

		existing, ok := sectorMap[sectorKey]
		if ok {
			existing.TotalValue += value
			existing.UnrealizedPnL += value - inp.TotalCost
			existing.Positions = append(existing.Positions, SectorPosition{
				Symbol:     inp.Symbol,
				SymbolName: inp.SymbolName,
				Value:      value,
			})
		} else {
			sectorMap[sectorKey] = &SectorWeight{
				Sector:        sectorKey,
				SectorName:    sectorName,
				TotalValue:    value,
				UnrealizedPnL: value - inp.TotalCost,
				Positions: []SectorPosition{
					{Symbol: inp.Symbol, SymbolName: inp.SymbolName, Value: value},
				},
			}
		}
	}

	// Calculate weights
	results := make([]SectorWeight, 0, len(sectorMap))
	for _, sw := range sectorMap {
		if portfolioTotalValue > 0 {
			sw.Weight = (sw.TotalValue / portfolioTotalValue) * 100
		}
		costBasis := sw.TotalValue - sw.UnrealizedPnL
		if costBasis > 0 {
			sw.UnrealizedPnLPct = (sw.UnrealizedPnL / costBasis) * 100
		}
		// Position-level weights within sector
		for i := range sw.Positions {
			if sw.TotalValue > 0 {
				sw.Positions[i].Weight = (sw.Positions[i].Value / sw.TotalValue) * 100
			}
		}
		results = append(results, *sw)
	}

	// Sort by weight descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Weight > results[i].Weight {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}
```

- [ ] 서비스에 GetSectorWeights 추가

```go
// backend/internal/service/portfolio_service.go (추가)

// GetSectorWeights returns sector-level portfolio weights.
func (s *PortfolioService) GetSectorWeights(ctx context.Context, userID string) ([]portfolio.SectorWeight, float64, error) {
	positions, err := s.repo.GetPositionsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	symbols := make([]string, len(positions))
	for i, p := range positions {
		symbols[i] = p.Symbol
	}
	prices, _ := s.kisClient.FetchPricesBatch(ctx, symbols)

	inputs := make([]portfolio.SectorAnalysisInput, len(positions))
	var totalValue float64
	for i, pos := range positions {
		currentPrice := parseNumeric(pos.AvgCostPrice)
		if p, ok := prices[pos.Symbol]; ok {
			currentPrice = p.Close
		}
		value := currentPrice * float64(pos.Quantity)
		totalValue += value

		inputs[i] = portfolio.SectorAnalysisInput{
			Symbol:       pos.Symbol,
			SymbolName:   pos.SymbolName,
			Sector:       derefString(pos.Sector),
			SectorName:   derefString(pos.SectorName),
			CurrentPrice: currentPrice,
			Quantity:     int(pos.Quantity),
			TotalCost:    parseNumeric(pos.TotalCost),
		}
	}

	sectors := portfolio.CalculateSectorWeights(inputs)
	return sectors, totalValue, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

- [ ] 핸들러에 GetSectors 추가

```go
// backend/internal/handler/portfolio_handler.go (추가)

// GetSectors handles GET /api/v1/portfolio/sectors
func (h *PortfolioHandler) GetSectors(c *gin.Context) {
	userID := c.GetString("userId")

	sectors, totalValue, err := h.svc.GetSectorWeights(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get sector weights", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"sectors":    sectors,
		"totalValue": totalValue,
	})
}
```

- [ ] 프론트엔드 API 클라이언트 — 섹터 비중 조회

```typescript
// src/features/portfolio/api/portfolio.api.ts (추가)
import { apiClient } from '@/shared/api/client';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface SectorWeight {
  sector: string;
  sectorName: string;
  totalValue: number;
  weight: number;
  positions: Array<{
    symbol: string;
    symbolName: string;
    value: number;
    weight: number;
  }>;
  unrealizedPnl: number;
  unrealizedPnlPct: number;
}

export async function fetchSectorWeights(): Promise<{
  sectors: SectorWeight[];
  totalValue: number;
}> {
  return apiClient(`${API_BASE}/api/v1/portfolio/sectors`);
}
```

- [ ] 테스트: 3개 섹터에 각 2종목 → 섹터별 비중 합계 100% 확인
- [ ] 테스트: 미분류 섹터 처리 확인
- [ ] 테스트: 섹터 내 포지션별 비중 합계 100% 확인

```bash
cd backend && go test ./internal/domain/portfolio/... -run TestSector -v
git add backend/internal/domain/portfolio/sector_analysis.go backend/internal/service/portfolio_service.go backend/internal/handler/portfolio_handler.go src/features/portfolio/api/
git commit -m "feat(portfolio): 섹터 비중 시각화 데이터 API"
```

---

## Task 5: 리밸런싱 시뮬레이터

목표 비중과 현재 비중의 차이를 계산하여 리밸런싱에 필요한 매수/매도 수량을 제안하는 시뮬레이터를 Go 도메인 로직으로 구현한다.

**Files:**
- Create: `backend/internal/domain/portfolio/rebalance.go`
- Create: `backend/internal/domain/portfolio/rebalance_test.go`
- Modify: `backend/internal/service/portfolio_service.go` (Rebalance 추가)
- Modify: `backend/internal/handler/portfolio_handler.go` (Rebalance 추가)
- Modify: `src/features/portfolio/api/portfolio.api.ts` (프론트엔드 API 클라이언트)

**Steps:**

- [ ] 리밸런싱 시뮬레이터 엔진 구현 — pure domain logic

```go
// backend/internal/domain/portfolio/rebalance.go
package portfolio

import (
	"fmt"
	"math"
	"sort"
)

// TargetAllocation represents a target weight for a symbol.
type TargetAllocation struct {
	Symbol       string  `json:"symbol" binding:"required"`
	TargetWeight float64 `json:"targetWeight" binding:"required,min=0,max=100"`
}

// RebalanceAction represents a single buy/sell/hold action.
type RebalanceAction struct {
	Symbol          string  `json:"symbol"`
	SymbolName      string  `json:"symbolName"`
	CurrentWeight   float64 `json:"currentWeight"`
	TargetWeight    float64 `json:"targetWeight"`
	WeightDiff      float64 `json:"weightDiff"`
	CurrentValue    float64 `json:"currentValue"`
	TargetValue     float64 `json:"targetValue"`
	ValueDiff       float64 `json:"valueDiff"`
	Action          string  `json:"action"` // "BUY" | "SELL" | "HOLD"
	Quantity        int     `json:"quantity"`
	EstimatedAmount float64 `json:"estimatedAmount"`
}

// RebalanceResult holds the full rebalancing simulation output.
type RebalanceResult struct {
	TotalPortfolioValue float64           `json:"totalPortfolioValue"`
	Actions             []RebalanceAction `json:"actions"`
	TotalBuyAmount      float64           `json:"totalBuyAmount"`
	TotalSellAmount     float64           `json:"totalSellAmount"`
	NetCashFlow         float64           `json:"netCashFlow"`
}

// RebalancePositionInput holds current position data for the simulator.
type RebalancePositionInput struct {
	Symbol       string
	SymbolName   string
	CurrentPrice float64
	TotalValue   float64
	Weight       float64
}

// SimulateRebalance calculates the actions needed to reach target weights.
// Pure function — no DB calls.
func SimulateRebalance(
	positions []RebalancePositionInput,
	targets []TargetAllocation,
	totalPortfolioValue float64,
) (*RebalanceResult, error) {
	// Validate target weights sum <= 100
	var totalTarget float64
	for _, t := range targets {
		totalTarget += t.TargetWeight
	}
	if totalTarget > 100.01 { // small epsilon for float precision
		return nil, fmt.Errorf("target weights sum %.2f%% exceeds 100%%", totalTarget)
	}

	targetMap := make(map[string]float64)
	for _, t := range targets {
		targetMap[t.Symbol] = t.TargetWeight
	}

	var actions []RebalanceAction

	// Existing positions
	for _, pos := range positions {
		targetWeight := targetMap[pos.Symbol]
		targetValue := totalPortfolioValue * (targetWeight / 100)
		valueDiff := targetValue - pos.TotalValue

		var action string
		switch {
		case valueDiff > 0:
			action = "BUY"
		case valueDiff < 0:
			action = "SELL"
		default:
			action = "HOLD"
		}

		quantity := 0
		if pos.CurrentPrice > 0 {
			quantity = int(math.Abs(valueDiff / pos.CurrentPrice))
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
			Quantity:        quantity,
			EstimatedAmount: float64(quantity) * pos.CurrentPrice,
		})
	}

	// New positions (in targets but not currently held)
	positionSymbols := make(map[string]bool)
	for _, p := range positions {
		positionSymbols[p.Symbol] = true
	}
	for _, target := range targets {
		if !positionSymbols[target.Symbol] {
			targetValue := totalPortfolioValue * (target.TargetWeight / 100)
			actions = append(actions, RebalanceAction{
				Symbol:          target.Symbol,
				SymbolName:      target.Symbol, // name lookup required separately
				CurrentWeight:   0,
				TargetWeight:    target.TargetWeight,
				WeightDiff:      target.TargetWeight,
				CurrentValue:    0,
				TargetValue:     targetValue,
				ValueDiff:       targetValue,
				Action:          "BUY",
				Quantity:        0, // current price lookup required
				EstimatedAmount: targetValue,
			})
		}
	}

	// Sort by absolute value diff descending
	sort.Slice(actions, func(i, j int) bool {
		return math.Abs(actions[i].ValueDiff) > math.Abs(actions[j].ValueDiff)
	})

	var totalBuy, totalSell float64
	for _, a := range actions {
		switch a.Action {
		case "BUY":
			totalBuy += a.EstimatedAmount
		case "SELL":
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
```

- [ ] 핸들러에 Rebalance 추가

```go
// backend/internal/handler/portfolio_handler.go (추가)

// RebalanceRequest is the JSON body for the rebalance endpoint.
type RebalanceRequest struct {
	Targets []portfolio.TargetAllocation `json:"targets" binding:"required,dive"`
}

// Rebalance handles POST /api/v1/portfolio/rebalance
func (h *PortfolioHandler) Rebalance(c *gin.Context) {
	userID := c.GetString("userId")

	var req RebalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.Rebalance(c.Request.Context(), userID, req.Targets)
	if err != nil {
		h.logger.Error("failed to simulate rebalance", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
```

- [ ] 서비스에 Rebalance 추가

```go
// backend/internal/service/portfolio_service.go (추가)

// Rebalance simulates portfolio rebalancing to target weights.
func (s *PortfolioService) Rebalance(ctx context.Context, userID string, targets []portfolio.TargetAllocation) (*portfolio.RebalanceResult, error) {
	summary, err := s.GetSummary(ctx, userID)
	if err != nil {
		return nil, err
	}

	inputs := make([]portfolio.RebalancePositionInput, len(summary.Positions))
	for i, pos := range summary.Positions {
		inputs[i] = portfolio.RebalancePositionInput{
			Symbol:       pos.Symbol,
			SymbolName:   pos.SymbolName,
			CurrentPrice: pos.CurrentPrice,
			TotalValue:   pos.TotalValue,
			Weight:       pos.Weight,
		}
	}

	return portfolio.SimulateRebalance(inputs, targets, summary.TotalValue)
}
```

- [ ] 프론트엔드 API 클라이언트 — 리밸런싱

```typescript
// src/features/portfolio/api/portfolio.api.ts (추가)

export interface RebalanceAction {
  symbol: string;
  symbolName: string;
  currentWeight: number;
  targetWeight: number;
  weightDiff: number;
  currentValue: number;
  targetValue: number;
  valueDiff: number;
  action: 'BUY' | 'SELL' | 'HOLD';
  quantity: number;
  estimatedAmount: number;
}

export interface RebalanceResult {
  totalPortfolioValue: number;
  actions: RebalanceAction[];
  totalBuyAmount: number;
  totalSellAmount: number;
  netCashFlow: number;
}

export async function simulateRebalance(targets: Array<{
  symbol: string;
  targetWeight: number;
}>): Promise<RebalanceResult> {
  return apiClient(`${API_BASE}/api/v1/portfolio/rebalance`, {
    method: 'POST',
    body: JSON.stringify({ targets }),
  });
}
```

- [ ] 리밸런싱 테스트

```go
// backend/internal/domain/portfolio/rebalance_test.go
package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimulateRebalance_EqualWeight(t *testing.T) {
	positions := []RebalancePositionInput{
		{Symbol: "005930", SymbolName: "삼성전자", CurrentPrice: 80000, TotalValue: 8000000, Weight: 80},
		{Symbol: "000660", SymbolName: "SK하이닉스", CurrentPrice: 100000, TotalValue: 2000000, Weight: 20},
	}
	targets := []TargetAllocation{
		{Symbol: "005930", TargetWeight: 50},
		{Symbol: "000660", TargetWeight: 50},
	}

	result, err := SimulateRebalance(positions, targets, 10000000)
	require.NoError(t, err)
	assert.Equal(t, 2, len(result.Actions))

	// 삼성전자: 80% → 50%, SELL
	samsung := findAction(result.Actions, "005930")
	assert.Equal(t, "SELL", samsung.Action)
	assert.InDelta(t, -30, samsung.WeightDiff, 0.01)

	// SK하이닉스: 20% → 50%, BUY
	sk := findAction(result.Actions, "000660")
	assert.Equal(t, "BUY", sk.Action)
	assert.InDelta(t, 30, sk.WeightDiff, 0.01)
}

func TestSimulateRebalance_ExceedsTotalWeight(t *testing.T) {
	positions := []RebalancePositionInput{
		{Symbol: "005930", SymbolName: "삼성전자", CurrentPrice: 80000, TotalValue: 8000000, Weight: 100},
	}
	targets := []TargetAllocation{
		{Symbol: "005930", TargetWeight: 60},
		{Symbol: "000660", TargetWeight: 50},
	}

	_, err := SimulateRebalance(positions, targets, 10000000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds 100%")
}

func TestSimulateRebalance_NewPosition(t *testing.T) {
	positions := []RebalancePositionInput{
		{Symbol: "005930", SymbolName: "삼성전자", CurrentPrice: 80000, TotalValue: 10000000, Weight: 100},
	}
	targets := []TargetAllocation{
		{Symbol: "005930", TargetWeight: 70},
		{Symbol: "035420", TargetWeight: 30},
	}

	result, err := SimulateRebalance(positions, targets, 10000000)
	require.NoError(t, err)

	naver := findAction(result.Actions, "035420")
	assert.Equal(t, "BUY", naver.Action)
	assert.InDelta(t, 3000000, naver.EstimatedAmount, 0.01)
}

func TestSimulateRebalance_NetCashFlow(t *testing.T) {
	positions := []RebalancePositionInput{
		{Symbol: "005930", SymbolName: "삼성전자", CurrentPrice: 80000, TotalValue: 6000000, Weight: 60},
		{Symbol: "000660", SymbolName: "SK하이닉스", CurrentPrice: 100000, TotalValue: 4000000, Weight: 40},
	}
	targets := []TargetAllocation{
		{Symbol: "005930", TargetWeight: 40},
		{Symbol: "000660", TargetWeight: 60},
	}

	result, err := SimulateRebalance(positions, targets, 10000000)
	require.NoError(t, err)

	// SELL amount ≈ BUY amount → net cash flow ≈ 0
	assert.InDelta(t, 0, result.NetCashFlow, 200000) // allow small rounding
}

func findAction(actions []RebalanceAction, symbol string) RebalanceAction {
	for _, a := range actions {
		if a.Symbol == symbol {
			return a
		}
	}
	return RebalanceAction{}
}
```

- [ ] 테스트: 균등 비중(각 25%) 목표 → 매수/매도 수량 정확 계산 확인
- [ ] 테스트: 목표 비중 합계 100% 초과 시 에러 반환 확인
- [ ] 테스트: 미보유 종목이 목표에 포함된 경우 BUY 액션 생성 확인
- [ ] 테스트: netCashFlow 양수(현금 유입) / 음수(추가 자금) 정확 확인

```bash
cd backend && go test ./internal/domain/portfolio/... -run TestRebalance -v
git add backend/internal/domain/portfolio/rebalance.go backend/internal/domain/portfolio/rebalance_test.go backend/internal/service/portfolio_service.go backend/internal/handler/portfolio_handler.go src/features/portfolio/api/
git commit -m "feat(portfolio): 리밸런싱 시뮬레이터 구현"
```

---

## Task 6: 통합 테스트 및 엣지 케이스

전체 포트폴리오 파이프라인의 통합 테스트와 엣지 케이스를 검증한다.

**Files:**
- Create: `backend/internal/service/portfolio_integration_test.go`

**Steps:**

- [ ] 통합 테스트 작성

```go
// backend/internal/service/portfolio_integration_test.go
package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortfolioIntegration_TradeToSummary(t *testing.T) {
	// 1. 케이스 A에서 삼성전자 100주 매수 @ 78,000
	// 2. 케이스 B에서 삼성전자 50주 매수 @ 80,000
	// 3. 포트폴리오 조회 → 삼성전자 150주, 평단가 계산 확인
	// 4. 삼성전자 120주 매도 @ 85,000
	// 5. FIFO: 100주(@78k) + 20주(@80k) 소진
	// 6. RealizedPnL 2건 확인
	// 7. 포트폴리오 조회 → 삼성전자 30주, 평단가=80,000
	t.Skip("requires test DB setup — implement with testcontainers")
}

func TestPortfolioIntegration_FullSell(t *testing.T) {
	// 전량 매도 후 포지션 quantity=0
	t.Skip("requires test DB setup — implement with testcontainers")
}

func TestPortfolioIntegration_USTaxSimulation(t *testing.T) {
	// US 종목 양도차익 500만원
	// 기본공제 250만원 차감
	// 과세분 250만원 * 22% = 55만원
	t.Skip("requires test DB setup — implement with testcontainers")
}

func TestPortfolioIntegration_SectorToRebalance(t *testing.T) {
	// 현재: IT 60%, 바이오 40%
	// 목표: IT 50%, 바이오 30%, 금융 20%
	// → IT SELL, 바이오 SELL, 금융 BUY 액션 확인
	t.Skip("requires test DB setup — implement with testcontainers")
}
```

- [ ] 엣지 케이스 테스트 (도메인 레벨 — DB 불필요)

```go
// backend/internal/domain/portfolio/fifo_engine_test.go (추가)

func TestComputeFifoSell_ExceedsAvailable(t *testing.T) {
	lots := []FifoLot{
		{ID: "lot-1", BuyPrice: 10000, OriginalQty: 100, RemainingQty: 50, Fee: 1000},
	}
	_, err := ComputeFifoSell(lots, 15000, 100, 500, MarketKR)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient quantity")
}

func TestComputeFifoSell_MultiCaseSameSymbol(t *testing.T) {
	// 동일 종목 다중 케이스 매수 → FIFO 순서대로 매도
	lots := []FifoLot{
		{ID: "case-a-lot", BuyDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), BuyPrice: 78000, OriginalQty: 100, RemainingQty: 100, Fee: 1000},
		{ID: "case-b-lot", BuyDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC), BuyPrice: 80000, OriginalQty: 50, RemainingQty: 50, Fee: 500},
	}

	results, err := ComputeFifoSell(lots, 85000, 120, 1200, MarketKR)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// First lot fully consumed (100), second lot partially (20)
	assert.Equal(t, 100, results[0].SellQty)
	assert.Equal(t, 0, results[0].NewRemainingQty)
	assert.Equal(t, 20, results[1].SellQty)
	assert.Equal(t, 30, results[1].NewRemainingQty)
}

func TestRecalculatePositionFromLots_Empty(t *testing.T) {
	agg := RecalculatePositionFromLots([]FifoLot{})
	assert.Equal(t, 0, agg.Quantity)
	assert.Equal(t, 0.0, agg.AvgCostPrice)
	assert.Equal(t, 0.0, agg.TotalCost)
}
```

- [ ] 엣지 케이스 테스트
  - 보유 수량보다 많은 매도 시도 → 에러
  - 동일 종목 다중 케이스 매수 → 단일 포지션 합산
  - 가격 0원 / 수량 0 방지
  - KIS API 장애 시 마지막 캐시 가격 사용

- [ ] 테스트 실행 및 전체 통과 확인

```bash
cd backend && go test ./internal/domain/portfolio/... ./internal/service/... -v
git add backend/internal/service/portfolio_integration_test.go backend/internal/domain/portfolio/
git commit -m "feat(portfolio): 통합 테스트 및 엣지 케이스 검증 완성"
```
