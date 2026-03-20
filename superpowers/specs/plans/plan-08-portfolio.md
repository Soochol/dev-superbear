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

**FSD Layers:**
- `src/entities/portfolio-position/` — PortfolioPosition은 비즈니스 엔티티이므로 entity 레이어에 배치. 순수 도메인 로직(FIFO, tax)은 `lib/`, DB 접근은 `api/`에 배치.
- `src/features/portfolio/` — 사용자 대면 기능(현황 조회, 리밸런싱 시뮬)은 feature 레이어에 배치.

---

## 의존성

- **Plan 5 (케이스 관리)**: Case, Trade 모델 및 매수/매도 기록 CRUD API

---

## Task 1: Portfolio 데이터 모델 및 FIFO 계산 엔진

포트폴리오 포지션과 실현 손익을 저장하는 모델, FIFO 방식 평단가/손익 계산 로직을 구현한다. FIFO 엔진과 세금 계산은 Prisma에 의존하지 않는 **순수 도메인 로직**으로 분리한다.

**Files:**
- Modify: `prisma/schema.prisma`
- Create: `src/entities/portfolio-position/model/types.ts`
- Create: `src/entities/portfolio-position/lib/fifo-engine.ts`
- Create: `src/entities/portfolio-position/lib/tax-calculator.ts`
- Create: `src/entities/portfolio-position/api/portfolio.repository.ts`
- Create: `src/entities/portfolio-position/lib/__tests__/fifo-engine.test.ts`
- Create: `src/entities/portfolio-position/lib/__tests__/tax-calculator.test.ts`

**Steps:**

- [ ] Prisma 스키마에 포트폴리오 관련 모델 추가 — **uuid()** for all IDs, use Prisma Market enum

```prisma
model PortfolioPosition {
  id              String    @id @default(uuid())
  userId          String
  symbol          String
  symbolName      String
  market          Market    @default(KR)    // KR | US
  quantity        Int       @default(0)     // 현재 보유 수량
  avgCostPrice    Float     @default(0)     // FIFO 기준 평균 매입가
  totalCost       Float     @default(0)     // 총 매입 원가
  sector          String?                   // 섹터 코드
  sectorName      String?                   // 섹터명
  createdAt       DateTime  @default(now())
  updatedAt       DateTime  @updatedAt

  user            User      @relation(fields: [userId], references: [id])
  lots            FifoLot[]
  realizedPnls    RealizedPnL[]

  @@unique([userId, symbol])
  @@index([userId])
}

enum Market {
  KR
  US
}

model FifoLot {
  id              String    @id @default(uuid())
  positionId      String
  tradeId         String
  buyDate         DateTime
  buyPrice        Float
  originalQty     Int       // 최초 매수 수량
  remainingQty    Int       // 잔여 수량
  fee             Float     @default(0)

  position        PortfolioPosition @relation(fields: [positionId], references: [id])
  trade           Trade     @relation(fields: [tradeId], references: [id])

  @@index([positionId, remainingQty])
}

model RealizedPnL {
  id              String    @id @default(uuid())
  positionId      String
  sellTradeId     String
  buyLotId        String
  quantity        Int
  buyPrice        Float
  sellPrice       Float
  grossPnl        Float     // (sellPrice - buyPrice) * quantity
  fee             Float     // 매수 + 매도 수수료
  tax             Float     // 세금
  netPnl          Float     // grossPnl - fee - tax
  realizedAt      DateTime

  position        PortfolioPosition @relation(fields: [positionId], references: [id])

  @@index([positionId, realizedAt])
}
```

- [ ] 마이그레이션 실행

```bash
npx prisma migrate dev --name add-portfolio-models
```

- [ ] 도메인 타입 정의 — use Prisma Market enum

```typescript
// src/entities/portfolio-position/model/types.ts
import type { Market as PrismaMarket } from '@prisma/client';

/** Re-export Prisma Market enum as the canonical Market type. */
export type Market = PrismaMarket;

export interface PortfolioSummary {
  totalValue: number;          // 총 평가금액
  totalCost: number;           // 총 매입금액
  unrealizedPnl: number;       // 미실현 손익
  unrealizedPnlPct: number;    // 미실현 수익률
  realizedPnl: number;         // 실현 손익 (누적)
  totalPnl: number;            // 전체 손익
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
  weight: number;              // 포트폴리오 내 비중 (%)
}

export interface TaxConfig {
  KR: {
    transactionTax: number;    // 증권거래세 0.18%
    capitalGainsTax: number;   // 국내 주식 양도세 (대주주 외 비과세)
  };
  US: {
    capitalGainsTax: number;   // 해외 양도소득세 22%
    basicDeduction: number;    // 기본공제 250만원
  };
}

export const DEFAULT_TAX_CONFIG: TaxConfig = {
  KR: {
    transactionTax: 0.0018,    // 0.18%
    capitalGainsTax: 0,        // 일반 투자자 비과세
  },
  US: {
    capitalGainsTax: 0.22,     // 22%
    basicDeduction: 2500000,   // 250만원
  },
};

/** Input for a buy trade — uses object parameter (not positional args). */
export interface BuyTradeInput {
  userId: string;
  tradeId: string;
  symbol: string;
  symbolName: string;
  price: number;
  quantity: number;
  fee: number;
  market: Market;
}

/** Input for a sell trade — uses object parameter (not positional args). */
export interface SellTradeInput {
  userId: string;
  tradeId: string;
  symbol: string;
  price: number;
  quantity: number;
  fee: number;
}

/** A FIFO lot for pure domain computation (no Prisma dependency). */
export interface FifoLotData {
  id: string;
  buyPrice: number;
  originalQty: number;
  remainingQty: number;
  fee: number;
}

/** Result of selling against a single lot. */
export interface LotSellResult {
  lotId: string;
  sellQty: number;
  grossPnl: number;
  lotFee: number;
  sellFee: number;
  tax: number;
  netPnl: number;
  newRemainingQty: number;
}
```

- [ ] 순수 FIFO 엔진 구현 — Prisma 의존 없이 데이터 구조만 다루는 pure domain logic

```typescript
// src/entities/portfolio-position/lib/fifo-engine.ts
import type { FifoLotData, LotSellResult, Market } from '../model/types';
import { calculateTax } from './tax-calculator';

/**
 * Pure FIFO sell computation. Given lots sorted oldest-first and a sell order,
 * returns the lot-level sell results (how much sold from each lot, PnL, fees, tax).
 * This function is pure — no DB calls.
 */
export function computeFifoSell(params: {
  lots: FifoLotData[];
  sellPrice: number;
  sellQuantity: number;
  sellFee: number;
  market: Market;
}): LotSellResult[] {
  const { lots, sellPrice, sellQuantity, sellFee, market } = params;
  const results: LotSellResult[] = [];
  let remainingToSell = sellQuantity;

  for (const lot of lots) {
    if (remainingToSell <= 0) break;

    const sellQty = Math.min(lot.remainingQty, remainingToSell);
    const grossPnl = (sellPrice - lot.buyPrice) * sellQty;
    const tax = calculateTax({ market, sellPrice, sellQty, grossPnl });
    const lotFee = (lot.fee / lot.originalQty) * sellQty;
    const sellFeeAlloc = (sellFee / sellQuantity) * sellQty;

    results.push({
      lotId: lot.id,
      sellQty,
      grossPnl,
      lotFee,
      sellFee: sellFeeAlloc,
      tax,
      netPnl: grossPnl - lotFee - sellFeeAlloc - tax,
      newRemainingQty: lot.remainingQty - sellQty,
    });

    remainingToSell -= sellQty;
  }

  return results;
}

/**
 * Recalculate position aggregates (quantity, avgCost, totalCost) from remaining lots.
 * Pure function.
 */
export function recalculatePositionFromLots(
  lots: FifoLotData[]
): { quantity: number; avgCostPrice: number; totalCost: number } {
  const activeLots = lots.filter((l) => l.remainingQty > 0);
  const totalQty = activeLots.reduce((sum, l) => sum + l.remainingQty, 0);
  const totalCost = activeLots.reduce((sum, l) => sum + l.buyPrice * l.remainingQty, 0);
  const avgCost = totalQty > 0 ? totalCost / totalQty : 0;
  return { quantity: totalQty, avgCostPrice: avgCost, totalCost };
}
```

- [ ] 세금 계산기 추출 — pure domain function

```typescript
// src/entities/portfolio-position/lib/tax-calculator.ts
import { DEFAULT_TAX_CONFIG, type Market } from '../model/types';

export function calculateTax(params: {
  market: Market;
  sellPrice: number;
  sellQty: number;
  grossPnl: number;
}): number {
  const { market, sellPrice, sellQty, grossPnl } = params;

  if (market === 'KR') {
    // 증권거래세: 매도 금액 * 0.18%
    return sellPrice * sellQty * DEFAULT_TAX_CONFIG.KR.transactionTax;
  } else {
    // 해외: 양도차익의 22% (기본공제는 연간 합산 시 적용)
    if (grossPnl <= 0) return 0;
    return grossPnl * DEFAULT_TAX_CONFIG.US.capitalGainsTax;
  }
}
```

- [ ] Repository — orchestrates Prisma calls + pure domain logic

```typescript
// src/entities/portfolio-position/api/portfolio.repository.ts
import { prisma } from '@/shared/lib/prisma';
import { logger } from '@/shared/lib/logger';
import type { BuyTradeInput, SellTradeInput } from '../model/types';
import { computeFifoSell, recalculatePositionFromLots } from '../lib/fifo-engine';

export async function processBuyTrade(input: BuyTradeInput): Promise<void> {
  const { userId, tradeId, symbol, symbolName, price, quantity, fee, market } = input;

  // 1. Position upsert
  const position = await prisma.portfolioPosition.upsert({
    where: { userId_symbol: { userId, symbol } },
    create: {
      userId, symbol, symbolName, market,
      quantity, avgCostPrice: price, totalCost: price * quantity,
    },
    update: {},
  });

  // 2. FifoLot 생성
  await prisma.fifoLot.create({
    data: {
      positionId: position.id,
      tradeId,
      buyDate: new Date(),
      buyPrice: price,
      originalQty: quantity,
      remainingQty: quantity,
      fee,
    },
  });

  // 3. Position 평단가 재계산 (pure FIFO logic)
  await recalculatePosition(position.id);
}

export async function processSellTrade(input: SellTradeInput): Promise<void> {
  const { userId, tradeId, symbol, price, quantity, fee } = input;

  const position = await prisma.portfolioPosition.findUniqueOrThrow({
    where: { userId_symbol: { userId, symbol } },
  });

  // Fetch lots sorted oldest-first
  const lots = await prisma.fifoLot.findMany({
    where: { positionId: position.id, remainingQty: { gt: 0 } },
    orderBy: { buyDate: 'asc' },
  });

  // Pure domain computation
  const sellResults = computeFifoSell({
    lots: lots.map((l) => ({
      id: l.id,
      buyPrice: l.buyPrice,
      originalQty: l.originalQty,
      remainingQty: l.remainingQty,
      fee: l.fee,
    })),
    sellPrice: price,
    sellQuantity: quantity,
    sellFee: fee,
    market: position.market,
  });

  // Persist each lot result
  for (const result of sellResults) {
    const lot = lots.find((l) => l.id === result.lotId)!;

    await prisma.realizedPnL.create({
      data: {
        positionId: position.id,
        sellTradeId: tradeId,
        buyLotId: result.lotId,
        quantity: result.sellQty,
        buyPrice: lot.buyPrice,
        sellPrice: price,
        grossPnl: result.grossPnl,
        fee: result.lotFee + result.sellFee,
        tax: result.tax,
        netPnl: result.netPnl,
        realizedAt: new Date(),
      },
    });

    await prisma.fifoLot.update({
      where: { id: result.lotId },
      data: { remainingQty: result.newRemainingQty },
    });
  }

  await recalculatePosition(position.id);
}

async function recalculatePosition(positionId: string): Promise<void> {
  const lots = await prisma.fifoLot.findMany({
    where: { positionId, remainingQty: { gt: 0 } },
  });

  const aggregates = recalculatePositionFromLots(
    lots.map((l) => ({
      id: l.id,
      buyPrice: l.buyPrice,
      originalQty: l.originalQty,
      remainingQty: l.remainingQty,
      fee: l.fee,
    }))
  );

  await prisma.portfolioPosition.update({
    where: { id: positionId },
    data: aggregates,
  });
}
```

- [ ] 테스트: BUY 100주 @ 10,000원 → Lot 생성, position qty=100, avgCost=10,000
- [ ] 테스트: BUY 50주 @ 12,000원 → Lot 2개, position qty=150
- [ ] 테스트: SELL 80주 @ 15,000원 → FIFO로 첫 Lot 100주 중 80주 소진, RealizedPnL 1건
- [ ] 테스트: SELL 70주 → 첫 Lot 나머지 20주 + 둘째 Lot 50주 소진, RealizedPnL 2건
- [ ] 테스트: 국내 세금 (증권거래세 0.18%) 정확 계산 확인
- [ ] 테스트: 해외 세금 (양도소득세 22%) 정확 계산 확인

```bash
git add prisma/ src/entities/portfolio-position/
git commit -m "feat(portfolio): 데이터 모델 및 FIFO 손익 계산 엔진 구현"
```

---

## Task 2: Trade 이벤트 자동 연동

Case Trade 레코드 생성 시 포트폴리오를 자동 갱신하는 이벤트 핸들러를 구현한다.

**Files:**
- Create: `src/features/portfolio/lib/trade-sync.ts`
- Modify: `src/app/api/cases/[id]/trades/route.ts` (기존 Trade POST에 연동 추가)
- Create: `src/features/portfolio/lib/__tests__/trade-sync.test.ts`

**Steps:**

- [ ] Trade 생성 시 포트폴리오 자동 갱신 핸들러 — uses object parameters

```typescript
// src/features/portfolio/lib/trade-sync.ts
import type { Trade } from '@prisma/client';
import { processBuyTrade, processSellTrade } from '@/entities/portfolio-position/api/portfolio.repository';
import { prisma } from '@/shared/lib/prisma';
import { logger } from '@/shared/lib/logger';

export async function syncTradeToPortfolio(trade: Trade): Promise<void> {
  // 케이스에서 종목 정보 조회
  const caseRecord = await prisma.case.findUniqueOrThrow({
    where: { id: trade.caseId },
  });

  if (trade.type === 'BUY') {
    await processBuyTrade({
      userId: trade.userId,
      tradeId: trade.id,
      symbol: caseRecord.symbol,
      symbolName: caseRecord.symbolName ?? caseRecord.symbol,
      price: trade.price,
      quantity: trade.quantity,
      fee: trade.fee,
      market: caseRecord.market ?? 'KR',
    });
  } else {
    await processSellTrade({
      userId: trade.userId,
      tradeId: trade.id,
      symbol: caseRecord.symbol,
      price: trade.price,
      quantity: trade.quantity,
      fee: trade.fee,
    });
  }
}
```

- [ ] 기존 Trade POST API에 연동 코드 추가 — thin controller delegates

```typescript
// src/app/api/cases/[id]/trades/route.ts (수정)
// 기존 Trade 생성 후:
import { syncTradeToPortfolio } from '@/features/portfolio/lib/trade-sync';

// trade = await prisma.trade.create({ ... });
await syncTradeToPortfolio(trade);
```

- [ ] 테스트: Trade 생성 → PortfolioPosition 자동 생성/갱신 확인
- [ ] 테스트: 여러 케이스에서 동일 종목 매수 → 하나의 PortfolioPosition에 합산 확인
- [ ] 테스트: Trade 삭제 시 포트폴리오 역연산 (Lot 복원) 확인

```bash
git add src/features/portfolio/lib/trade-sync.ts src/app/api/cases/
git commit -m "feat(portfolio): Trade 생성 시 포트폴리오 자동 연동"
```

---

## Task 3: Portfolio API 엔드포인트

포트폴리오 현황 조회, 손익 히스토리, 세금 시뮬레이션 API를 구현한다. Route handler는 thin controller로서 feature service에 위임한다.

**Files:**
- Create: `src/app/api/portfolio/route.ts`
- Create: `src/app/api/portfolio/history/route.ts`
- Create: `src/app/api/portfolio/tax/route.ts`
- Create: `src/features/portfolio/lib/portfolio-service.ts`
- Create: `src/features/portfolio/lib/__tests__/portfolio-service.test.ts`

**Steps:**

- [ ] 포트폴리오 서비스 구현 — 현재가 반영 실시간 포지션 계산

```typescript
// src/features/portfolio/lib/portfolio-service.ts
import { prisma } from '@/shared/lib/prisma';
import { fetchPricesBatch } from '@/shared/api/kis/batch-client';
import { logger } from '@/shared/lib/logger';
import type { PortfolioSummary, PositionDetail, Market } from '@/entities/portfolio-position/model/types';

export async function getPortfolioSummary(
  userId: string
): Promise<PortfolioSummary> {
  const positions = await prisma.portfolioPosition.findMany({
    where: { userId, quantity: { gt: 0 } },
  });

  if (positions.length === 0) {
    return {
      totalValue: 0, totalCost: 0,
      unrealizedPnl: 0, unrealizedPnlPct: 0,
      realizedPnl: 0, totalPnl: 0, positions: [],
    };
  }

  // 현재가 배치 조회
  const symbols = positions.map((p) => p.symbol);
  const prices = await fetchPricesBatch(symbols);

  // 실현 손익 합산
  const realizedPnls = await prisma.realizedPnL.aggregate({
    where: { position: { userId } },
    _sum: { netPnl: true },
  });

  let totalValue = 0;
  let totalCost = 0;

  const positionDetails: PositionDetail[] = positions.map((pos) => {
    const currentPrice = prices.get(pos.symbol)?.close ?? pos.avgCostPrice;
    const posValue = currentPrice * pos.quantity;
    const posCost = pos.totalCost;
    const unrealizedPnl = posValue - posCost;

    totalValue += posValue;
    totalCost += posCost;

    return {
      symbol: pos.symbol,
      symbolName: pos.symbolName,
      market: pos.market as Market,
      quantity: pos.quantity,
      avgCostPrice: pos.avgCostPrice,
      currentPrice,
      totalCost: posCost,
      totalValue: posValue,
      unrealizedPnl,
      unrealizedPnlPct: posCost > 0 ? (unrealizedPnl / posCost) * 100 : 0,
      sector: pos.sector ?? undefined,
      sectorName: pos.sectorName ?? undefined,
      weight: 0, // 아래에서 계산
    };
  });

  // 비중 계산
  positionDetails.forEach((p) => {
    p.weight = totalValue > 0 ? (p.totalValue / totalValue) * 100 : 0;
  });

  const unrealizedPnl = totalValue - totalCost;
  const realizedPnl = realizedPnls._sum.netPnl ?? 0;

  return {
    totalValue,
    totalCost,
    unrealizedPnl,
    unrealizedPnlPct: totalCost > 0 ? (unrealizedPnl / totalCost) * 100 : 0,
    realizedPnl,
    totalPnl: unrealizedPnl + realizedPnl,
    positions: positionDetails,
  };
}
```

- [ ] 포트폴리오 현황 API — thin controller

```typescript
// src/app/api/portfolio/route.ts
// GET /api/portfolio
// delegates to portfolio-service.getPortfolioSummary()
// Response: PortfolioSummary
```

- [ ] 손익 히스토리 API — thin controller

```typescript
// src/app/api/portfolio/history/route.ts
// GET /api/portfolio/history?period=daily|monthly&from=2025-01-01&to=2025-12-31
// Response: { history: [{ date, realizedPnl, cumulativePnl }] }
```

- [ ] 세금 시뮬레이션 API — thin controller

```typescript
// src/app/api/portfolio/tax/route.ts
// GET /api/portfolio/tax?year=2025
// Response: {
//   kr: { totalSellAmount, transactionTax },
//   us: { totalGain, basicDeduction, taxableGain, capitalGainsTax },
//   totalTax
// }
```

- [ ] 테스트: 포트폴리오 현황에 현재가 반영 확인
- [ ] 테스트: 비중(weight) 합계 100% 확인
- [ ] 테스트: 세금 시뮬레이션 — 국내 거래세 + 해외 양도세 분리 계산 확인
- [ ] 테스트: 해외 양도소득 기본공제 250만원 적용 확인

```bash
git add src/app/api/portfolio/ src/features/portfolio/lib/portfolio-service.ts
git commit -m "feat(portfolio): 포트폴리오 현황 / 손익 히스토리 / 세금 시뮬레이션 API"
```

---

## Task 4: 섹터 비중 시각화 데이터 API

섹터별 포트폴리오 비중 데이터를 도넛 차트 렌더링에 적합한 형태로 제공하는 API를 구현한다.

**Files:**
- Create: `src/app/api/portfolio/sectors/route.ts`
- Create: `src/features/portfolio/lib/sector-analysis.ts`
- Create: `src/features/portfolio/lib/__tests__/sector-analysis.test.ts`

**Steps:**

- [ ] 섹터 분석 서비스 — 보유 포지션을 섹터별로 그룹핑하여 비중 계산

```typescript
// src/features/portfolio/lib/sector-analysis.ts
import { prisma } from '@/shared/lib/prisma';
import { fetchPricesBatch } from '@/shared/api/kis/batch-client';
import { logger } from '@/shared/lib/logger';

export interface SectorWeight {
  sector: string;
  sectorName: string;
  totalValue: number;
  weight: number;           // 비중 (%)
  positions: Array<{
    symbol: string;
    symbolName: string;
    value: number;
    weight: number;
  }>;
  unrealizedPnl: number;
  unrealizedPnlPct: number;
}

export async function getSectorWeights(
  userId: string
): Promise<SectorWeight[]> {
  const positions = await prisma.portfolioPosition.findMany({
    where: { userId, quantity: { gt: 0 } },
  });

  const prices = await fetchPricesBatch(positions.map((p) => p.symbol));

  // 섹터별 그룹핑
  const sectorMap = new Map<string, SectorWeight>();
  let portfolioTotalValue = 0;

  for (const pos of positions) {
    const currentPrice = prices.get(pos.symbol)?.close ?? pos.avgCostPrice;
    const value = currentPrice * pos.quantity;
    portfolioTotalValue += value;

    const sectorKey = pos.sector || 'UNKNOWN';
    const existing = sectorMap.get(sectorKey);

    if (existing) {
      existing.totalValue += value;
      existing.unrealizedPnl += value - pos.totalCost;
      existing.positions.push({
        symbol: pos.symbol,
        symbolName: pos.symbolName,
        value,
        weight: 0,
      });
    } else {
      sectorMap.set(sectorKey, {
        sector: sectorKey,
        sectorName: pos.sectorName || '미분류',
        totalValue: value,
        weight: 0,
        positions: [{
          symbol: pos.symbol,
          symbolName: pos.symbolName,
          value,
          weight: 0,
        }],
        unrealizedPnl: value - pos.totalCost,
        unrealizedPnlPct: 0,
      });
    }
  }

  // 비중 계산
  const results = Array.from(sectorMap.values()).map((sw) => ({
    ...sw,
    weight: portfolioTotalValue > 0
      ? (sw.totalValue / portfolioTotalValue) * 100
      : 0,
    unrealizedPnlPct: sw.totalValue > 0
      ? (sw.unrealizedPnl / (sw.totalValue - sw.unrealizedPnl)) * 100
      : 0,
    positions: sw.positions.map((p) => ({
      ...p,
      weight: sw.totalValue > 0
        ? (p.value / sw.totalValue) * 100
        : 0,
    })),
  }));

  return results.sort((a, b) => b.weight - a.weight);
}
```

- [ ] 섹터 비중 API 엔드포인트 — thin controller

```typescript
// src/app/api/portfolio/sectors/route.ts
// GET /api/portfolio/sectors
// delegates to sector-analysis.getSectorWeights()
// Response: { sectors: SectorWeight[], totalValue: number }
```

- [ ] 테스트: 3개 섹터에 각 2종목 → 섹터별 비중 합계 100% 확인
- [ ] 테스트: 미분류 섹터 처리 확인
- [ ] 테스트: 섹터 내 포지션별 비중 합계 100% 확인

```bash
git add src/app/api/portfolio/sectors/ src/features/portfolio/lib/sector-analysis.ts
git commit -m "feat(portfolio): 섹터 비중 시각화 데이터 API"
```

---

## Task 5: 리밸런싱 시뮬레이터

목표 비중과 현재 비중의 차이를 계산하여 리밸런싱에 필요한 매수/매도 수량을 제안하는 시뮬레이터를 구현한다.

**Files:**
- Create: `src/app/api/portfolio/rebalance/route.ts`
- Create: `src/features/portfolio/lib/rebalance-simulator.ts`
- Create: `src/features/portfolio/lib/__tests__/rebalance-simulator.test.ts`

**Steps:**

- [ ] 리밸런싱 시뮬레이터 엔진 구현

```typescript
// src/features/portfolio/lib/rebalance-simulator.ts
export interface TargetAllocation {
  symbol: string;
  targetWeight: number;  // 목표 비중 (%)
}

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
  quantity: number;        // 매수/매도 수량
  estimatedAmount: number; // 예상 금액
}

export interface RebalanceResult {
  totalPortfolioValue: number;
  actions: RebalanceAction[];
  totalBuyAmount: number;
  totalSellAmount: number;
  netCashFlow: number;    // 양수 = 현금 유입, 음수 = 추가 자금 필요
}

export function simulateRebalance(
  positions: Array<{
    symbol: string;
    symbolName: string;
    currentPrice: number;
    totalValue: number;
    weight: number;
  }>,
  targets: TargetAllocation[],
  totalPortfolioValue: number
): RebalanceResult {
  const targetMap = new Map(targets.map((t) => [t.symbol, t.targetWeight]));

  const actions: RebalanceAction[] = positions.map((pos) => {
    const targetWeight = targetMap.get(pos.symbol) ?? 0;
    const targetValue = totalPortfolioValue * (targetWeight / 100);
    const valueDiff = targetValue - pos.totalValue;
    const quantity = Math.abs(Math.floor(valueDiff / pos.currentPrice));

    return {
      symbol: pos.symbol,
      symbolName: pos.symbolName,
      currentWeight: pos.weight,
      targetWeight,
      weightDiff: targetWeight - pos.weight,
      currentValue: pos.totalValue,
      targetValue,
      valueDiff,
      action: valueDiff > 0 ? 'BUY' : valueDiff < 0 ? 'SELL' : 'HOLD',
      quantity,
      estimatedAmount: quantity * pos.currentPrice,
    };
  });

  // 목표에 있지만 현재 미보유 종목 추가
  for (const target of targets) {
    if (!positions.find((p) => p.symbol === target.symbol)) {
      const targetValue = totalPortfolioValue * (target.targetWeight / 100);
      actions.push({
        symbol: target.symbol,
        symbolName: target.symbol,
        currentWeight: 0,
        targetWeight: target.targetWeight,
        weightDiff: target.targetWeight,
        currentValue: 0,
        targetValue,
        valueDiff: targetValue,
        action: 'BUY',
        quantity: 0, // 현재가 조회 필요
        estimatedAmount: targetValue,
      });
    }
  }

  const totalBuyAmount = actions
    .filter((a) => a.action === 'BUY')
    .reduce((sum, a) => sum + a.estimatedAmount, 0);
  const totalSellAmount = actions
    .filter((a) => a.action === 'SELL')
    .reduce((sum, a) => sum + a.estimatedAmount, 0);

  return {
    totalPortfolioValue,
    actions: actions.sort((a, b) => Math.abs(b.valueDiff) - Math.abs(a.valueDiff)),
    totalBuyAmount,
    totalSellAmount,
    netCashFlow: totalSellAmount - totalBuyAmount,
  };
}
```

- [ ] 리밸런싱 API 엔드포인트 — thin controller

```typescript
// src/app/api/portfolio/rebalance/route.ts
// POST /api/portfolio/rebalance
// Body: { targets: [{ symbol, targetWeight }] }
// delegates to rebalance-simulator.simulateRebalance()
// Response: RebalanceResult
```

- [ ] 테스트: 균등 비중(각 25%) 목표 → 매수/매도 수량 정확 계산 확인
- [ ] 테스트: 목표 비중 합계 100% 초과 시 에러 반환 확인
- [ ] 테스트: 미보유 종목이 목표에 포함된 경우 BUY 액션 생성 확인
- [ ] 테스트: netCashFlow 양수(현금 유입) / 음수(추가 자금) 정확 확인

```bash
git add src/app/api/portfolio/rebalance/ src/features/portfolio/lib/rebalance-simulator.ts
git commit -m "feat(portfolio): 리밸런싱 시뮬레이터 구현"
```

---

## Task 6: 통합 테스트 및 엣지 케이스

전체 포트폴리오 파이프라인의 통합 테스트와 엣지 케이스를 검증한다.

**Files:**
- Create: `src/features/portfolio/lib/__tests__/integration.test.ts`

**Steps:**

- [ ] 통합 테스트 작성

```typescript
// src/features/portfolio/lib/__tests__/integration.test.ts
describe('Portfolio Integration', () => {
  it('Trade 생성 → FIFO 포지션 갱신 → 포트폴리오 현황 조회', async () => {
    // 1. 케이스 A에서 삼성전자 100주 매수 @ 78,000
    // 2. 케이스 B에서 삼성전자 50주 매수 @ 80,000
    // 3. 포트폴리오 조회 → 삼성전자 150주, 평단가 계산 확인
    // 4. 삼성전자 120주 매도 @ 85,000
    // 5. FIFO: 100주(@78k) + 20주(@80k) 소진
    // 6. RealizedPnL 2건 확인
    // 7. 포트폴리오 조회 → 삼성전자 30주, 평단가=80,000
  });

  it('전체 매도 → 포지션 quantity=0', async () => {
    // 전량 매도 후 포트폴리오에서 사라짐 (quantity=0)
  });

  it('해외 종목 세금 시뮬레이션', async () => {
    // US 종목 양도차익 500만원
    // 기본공제 250만원 차감
    // 과세분 250만원 * 22% = 55만원
  });

  it('섹터 비중 → 리밸런싱 시뮬 연동', async () => {
    // 현재: IT 60%, 바이오 40%
    // 목표: IT 50%, 바이오 30%, 금융 20%
    // → IT SELL, 바이오 SELL, 금융 BUY 액션 확인
  });
});
```

- [ ] 엣지 케이스 테스트
  - 보유 수량보다 많은 매도 시도 → 에러
  - 동일 종목 다중 케이스 매수 → 단일 포지션 합산
  - 가격 0원 / 수량 0 방지
  - KIS API 장애 시 마지막 캐시 가격 사용

- [ ] 테스트 실행 및 전체 통과 확인

```bash
npm test -- --testPathPattern=portfolio
git add src/features/portfolio/lib/__tests__/ src/entities/portfolio-position/lib/__tests__/
git commit -m "feat(portfolio): 통합 테스트 및 엣지 케이스 검증 완성"
```
