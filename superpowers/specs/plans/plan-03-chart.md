# Chart Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** TradingView lightweight-charts 기반 캔들스틱 차트, 종목 리스트 사이드바(3탭), 보조 지표 패널, 하단 3칼럼 정보 패널을 구현하고 KIS API로 실시간 가격 데이터를 연동한다.
**Architecture:** FSD 아키텍처에 따라 차트 기능을 `features/chart/`에, UI 조합 위젯을 `widgets/`에, 지표 계산을 `entities/indicator/`에 배치한다. 차트 페이지는 좌측(차트 + 보조지표)과 우측(종목 리스트 사이드바)로 분할되며, 하단에 full-width 3칼럼 패널(Financials / AI Fusion / Sector Compare)을 배치한다. 검색 페이지에서 "Chart" 버튼 클릭 시 `entities/stock/` 공유 store를 통해 결과가 사이드바에 자동 로드된다. **백엔드는 Go (Gin) + sqlc 기반으로 구현한다.** KIS Open API와 DART API 클라이언트는 Go로 구현하고, 캔들/재무/관심종목 API를 Go handler로 제공한다. 프론트엔드는 Go 서버의 `/api/v1/` 엔드포인트를 호출한다.
**Tech Stack:** Next.js (App Router), lightweight-charts (TradingView), Zustand (feature-scoped chart store), TailwindCSS, **Go (Gin), sqlc, KIS Open API, DART Open API**

**Depends on:** Plan 1 (API 스캐폴드, DB, 인증), Plan 2 (`entities/stock/` store, 검색 결과 연동)

---

## Task 1: 차트 페이지 레이아웃 + Feature 상태 관리

스펙 Section 2.1의 레이아웃을 구현한다. 좌측 차트 영역, 우측 사이드바, 하단 정보 패널의 반응형 그리드를 구성한다. FSD 규칙에 따라 차트 상태를 `features/chart/model/`에, 레이아웃 위젯을 `widgets/`에 배치한다.

**Files:**
- Create: `src/app/(pages)/chart/page.tsx`
- Create: `src/app/(pages)/chart/layout.tsx`
- Create: `src/features/chart/model/chart.store.ts` (feature-scoped store — split from search)
- Create: `src/features/chart/model/types.ts`
- Create: `src/features/chart/index.ts` (barrel export)
- Create: `src/entities/candle/model/types.ts` (candle domain types)
- Create: `src/entities/candle/index.ts`
- Create: `src/widgets/main-chart/ui/ChartPageLayout.tsx`
- Create: `src/widgets/main-chart/ui/ChartTopbar.tsx`
- Create: `src/widgets/main-chart/index.ts`
- Test: `src/features/chart/__tests__/chart-store.test.ts`

### Steps

- [ ] Candle 엔티티 도메인 타입 정의 (`src/entities/candle/model/types.ts`)

```typescript
// src/entities/candle/model/types.ts

/** Normalized candle data — used across chart and indicator features */
export interface CandleData {
  time: string;  // YYYY-MM-DD or timestamp
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}
```

```typescript
// src/entities/candle/index.ts
export type { CandleData } from "./model/types";
```

- [ ] Chart feature 도메인 타입 (`src/features/chart/model/types.ts`)

```typescript
// src/features/chart/model/types.ts
export type Timeframe = "1m" | "5m" | "15m" | "1H" | "1D" | "1W" | "1M";
export type BottomPanelTab = "financials" | "ai-fusion" | "sector-compare";
```

- [ ] 테스트 먼저 작성 (`src/features/chart/__tests__/chart-store.test.ts`)

```typescript
// src/features/chart/__tests__/chart-store.test.ts
import { useChartStore } from "../model/chart.store";

describe("Chart Store", () => {
  beforeEach(() => {
    useChartStore.setState(useChartStore.getInitialState());
  });

  it("initializes with default timeframe 1D", () => {
    expect(useChartStore.getState().timeframe).toBe("1D");
  });

  it("sets current stock info", () => {
    useChartStore.getState().setCurrentStock({
      symbol: "005930",
      name: "Samsung Electronics",
      price: 78400,
      change: 1600,
      changePct: 2.08,
    });
    const state = useChartStore.getState();
    expect(state.currentStock?.symbol).toBe("005930");
    expect(state.currentStock?.price).toBe(78400);
  });

  it("switches timeframe", () => {
    useChartStore.getState().setTimeframe("1W");
    expect(useChartStore.getState().timeframe).toBe("1W");
  });

  it("toggles indicator overlays", () => {
    useChartStore.getState().toggleIndicator("ma20");
    expect(useChartStore.getState().activeIndicators).toContain("ma20");
    useChartStore.getState().toggleIndicator("ma20");
    expect(useChartStore.getState().activeIndicators).not.toContain("ma20");
  });

  it("manages sub-indicator panels", () => {
    useChartStore.getState().toggleSubIndicator("rsi");
    expect(useChartStore.getState().activeSubIndicators).toContain("rsi");
  });

  it("tracks candle data loading state", () => {
    useChartStore.getState().setIsLoading(true);
    expect(useChartStore.getState().isLoading).toBe(true);
  });
});
```

- [ ] 차트 상태 스토어 구현 (`src/features/chart/model/chart.store.ts`) — feature-scoped, split from search store

```typescript
// src/features/chart/model/chart.store.ts
import { create } from "zustand";
import type { Timeframe, BottomPanelTab } from "./types";
import type { StockInfo } from "@/entities/stock";
import type { CandleData } from "@/entities/candle";

interface ChartState {
  // 현재 종목
  currentStock: StockInfo | null;
  setCurrentStock: (stock: StockInfo | null) => void;

  // 타임프레임
  timeframe: Timeframe;
  setTimeframe: (tf: Timeframe) => void;

  // 캔들 데이터
  candles: CandleData[];
  setCandles: (candles: CandleData[]) => void;

  // 로딩 상태
  isLoading: boolean;
  setIsLoading: (v: boolean) => void;

  // 오버레이 지표 (MA, BB 등 — 메인 차트 위)
  activeIndicators: string[];
  toggleIndicator: (id: string) => void;

  // 보조 지표 패널 (RSI, MACD 등 — 차트 아래)
  activeSubIndicators: string[];
  toggleSubIndicator: (id: string) => void;

  // 하단 패널 활성 탭
  bottomPanelTab: BottomPanelTab;
  setBottomPanelTab: (tab: BottomPanelTab) => void;
}

export const useChartStore = create<ChartState>()((set) => ({
  currentStock: null,
  setCurrentStock: (stock) => set({ currentStock: stock }),

  timeframe: "1D",
  setTimeframe: (tf) => set({ timeframe: tf }),

  candles: [],
  setCandles: (candles) => set({ candles }),

  isLoading: false,
  setIsLoading: (v) => set({ isLoading: v }),

  activeIndicators: ["ma20", "ma60"],
  toggleIndicator: (id) =>
    set((state) => ({
      activeIndicators: state.activeIndicators.includes(id)
        ? state.activeIndicators.filter((i) => i !== id)
        : [...state.activeIndicators, id],
    })),

  activeSubIndicators: [],
  toggleSubIndicator: (id) =>
    set((state) => ({
      activeSubIndicators: state.activeSubIndicators.includes(id)
        ? state.activeSubIndicators.filter((i) => i !== id)
        : [...state.activeSubIndicators, id],
    })),

  bottomPanelTab: "financials",
  setBottomPanelTab: (tab) => set({ bottomPanelTab: tab }),
}));
```

- [ ] Barrel export (`src/features/chart/index.ts`)

```typescript
// src/features/chart/index.ts
export { useChartStore } from "./model/chart.store";
export type { Timeframe, BottomPanelTab } from "./model/types";
```

- [ ] 차트 페이지 레이아웃 위젯 구현 (`src/widgets/main-chart/ui/ChartPageLayout.tsx`) — 스펙 Section 2.1의 레이아웃 구조를 그리드로 구현

```typescript
// src/widgets/main-chart/ui/ChartPageLayout.tsx
"use client";

import { ChartTopbar } from "./ChartTopbar";
import { MainChart } from "@/features/chart/ui/MainChart";
import { SubIndicatorPanel } from "@/widgets/sub-indicator-panel";
import { StockListSidebar } from "@/widgets/stock-list-sidebar";
import { BottomInfoPanel } from "@/widgets/bottom-info-panel";

export function ChartPageLayout() {
  return (
    <div className="flex flex-col h-full">
      {/* Topbar: 종목 정보 + 타임프레임 선택 */}
      <ChartTopbar />

      {/* Main Area: 차트(좌) + 사이드바(우) */}
      <div className="flex flex-1 min-h-0">
        {/* 좌측: 메인 차트 + 보조 지표 */}
        <div className="flex-1 flex flex-col min-w-0">
          <div className="flex-1 min-h-0">
            <MainChart />
          </div>
          <SubIndicatorPanel />
        </div>

        {/* 우측: 종목 리스트 사이드바 */}
        <div className="w-72 border-l border-nexus-border flex-shrink-0">
          <StockListSidebar />
        </div>
      </div>

      {/* 하단: Financials | AI Fusion | Sector Compare (full-width 3칼럼) */}
      <BottomInfoPanel />
    </div>
  );
}
```

- [ ] 상단바 위젯 (`src/widgets/main-chart/ui/ChartTopbar.tsx`) — 종목코드, 종목명, 현재가, 등락률 + 타임프레임 버튼(1m/5m/15m/1H/1D/1W/1M). Imports from features/chart and entities/stock only

```typescript
// src/widgets/main-chart/ui/ChartTopbar.tsx
"use client";

import { useChartStore } from "@/features/chart";
import type { Timeframe } from "@/features/chart";

const TIMEFRAMES: Timeframe[] = ["1m", "5m", "15m", "1H", "1D", "1W", "1M"];

export function ChartTopbar() {
  const { currentStock, timeframe, setTimeframe } = useChartStore();

  return (
    <div className="flex items-center justify-between px-4 py-2 bg-nexus-surface border-b border-nexus-border">
      {/* 종목 정보 */}
      <div className="flex items-center gap-4">
        {currentStock ? (
          <>
            <span className="font-mono text-nexus-text-secondary text-sm">
              {currentStock.symbol}
            </span>
            <span className="font-semibold">{currentStock.name}</span>
            <span className="font-mono text-lg">
              {currentStock.price.toLocaleString()}
            </span>
            <span
              className={`font-mono text-sm ${
                currentStock.changePct >= 0 ? "text-nexus-success" : "text-nexus-failure"
              }`}
            >
              {currentStock.changePct >= 0 ? "+" : ""}
              {currentStock.changePct.toFixed(2)}%
            </span>
          </>
        ) : (
          <span className="text-nexus-text-muted">Select a stock</span>
        )}
      </div>

      {/* 타임프레임 선택 */}
      <div className="flex gap-1">
        {TIMEFRAMES.map((tf) => (
          <button
            key={tf}
            onClick={() => setTimeframe(tf)}
            className={`px-2 py-1 text-xs font-medium rounded transition-colors ${
              timeframe === tf
                ? "bg-nexus-accent text-white"
                : "text-nexus-text-secondary hover:text-nexus-text-primary"
            }`}
          >
            {tf}
          </button>
        ))}
      </div>
    </div>
  );
}
```

```typescript
// src/widgets/main-chart/index.ts
export { ChartPageLayout } from "./ui/ChartPageLayout";
export { ChartTopbar } from "./ui/ChartTopbar";
```

- [ ] 페이지 라우트 연결 (`src/app/(pages)/chart/page.tsx`) — URL 파라미터 `?symbol=005930`을 읽어 초기 종목 설정

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear
npx jest src/features/chart/__tests__/chart-store.test.ts
```

- [ ] 커밋

```bash
git add src/app/\(pages\)/chart/ src/features/chart/ src/entities/candle/ src/widgets/main-chart/
git commit -m "feat: chart page layout with feature-scoped store and FSD widget structure"
```

---

## Task 2: 종목 리스트 사이드바 위젯 (3탭: 검색결과 / 관심종목 / 최근)

스펙 Section 2.2의 사이드바를 `widgets/stock-list-sidebar/`로 구현한다. 검색결과 탭에는 Search 페이지 결과가 `entities/stock/` store에서 자동 로드되고, 관심종목과 최근 탭은 같은 entity store에서 관리한다. FSD 규칙: 다른 feature에서 직접 import하지 않고 entities/ 레이어를 통한다.

**Files:**
- Create: `src/widgets/stock-list-sidebar/ui/StockListSidebar.tsx`
- Create: `src/widgets/stock-list-sidebar/ui/StockListItem.tsx`
- Create: `src/widgets/stock-list-sidebar/ui/SidebarSearchInput.tsx`
- Create: `src/widgets/stock-list-sidebar/index.ts`
- Create: `src/features/watchlist/model/watchlist.actions.ts` (star toggle feature)
- Create: `src/features/watchlist/index.ts`
- Test: `src/widgets/stock-list-sidebar/__tests__/StockListSidebar.test.tsx`

### Steps

- [ ] 테스트 먼저 작성 (`src/widgets/stock-list-sidebar/__tests__/StockListSidebar.test.tsx`)

```typescript
// src/widgets/stock-list-sidebar/__tests__/StockListSidebar.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { StockListSidebar } from "../ui/StockListSidebar";
import { useStockListStore } from "@/entities/stock";

beforeEach(() => {
  useStockListStore.setState({
    searchResults: [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      { symbol: "247540", name: "ecoprobm", matchedValue: 15200000 },
    ],
    selectedSymbol: "005930",
    watchlist: [
      { symbol: "000660", name: "SK Hynix", matchedValue: 0 },
    ],
    recentStocks: [
      { symbol: "373220", name: "LG Energy", matchedValue: 0 },
    ],
  });
});

describe("StockListSidebar", () => {
  it("renders 3 tabs", () => {
    render(<StockListSidebar />);
    expect(screen.getByText(/검색결과/)).toBeInTheDocument();
    expect(screen.getByText(/관심/i)).toBeInTheDocument();
    expect(screen.getByText(/최근/i)).toBeInTheDocument();
  });

  it("shows search results in first tab", () => {
    render(<StockListSidebar />);
    expect(screen.getByText("Samsung")).toBeInTheDocument();
    expect(screen.getByText("ecoprobm")).toBeInTheDocument();
  });

  it("highlights the selected/active stock", () => {
    render(<StockListSidebar />);
    const samsungItem = screen.getByText("Samsung").closest("[data-testid]");
    expect(samsungItem).toHaveClass(/active|selected/i);
  });

  it("switches to watchlist tab and shows watchlist items", () => {
    render(<StockListSidebar />);
    fireEvent.click(screen.getByText(/관심/i));
    expect(screen.getByText("SK Hynix")).toBeInTheDocument();
  });

  it("switches to recent tab and shows recent items", () => {
    render(<StockListSidebar />);
    fireEvent.click(screen.getByText(/최근/i));
    expect(screen.getByText("LG Energy")).toBeInTheDocument();
  });

  it("renders search input at top", () => {
    render(<StockListSidebar />);
    expect(screen.getByPlaceholderText(/종목 검색|search/i)).toBeInTheDocument();
  });

  it("renders watchlist toggle (star icon) on each item", () => {
    render(<StockListSidebar />);
    const starButtons = screen.getAllByRole("button", { name: /watchlist|star|관심/i });
    expect(starButtons.length).toBeGreaterThan(0);
  });
});
```

- [ ] 사이드바 위젯 메인 컴포넌트 구현 (`src/widgets/stock-list-sidebar/ui/StockListSidebar.tsx`) — imports from entities/stock and features/chart only (no cross-feature imports)

```typescript
// src/widgets/stock-list-sidebar/ui/StockListSidebar.tsx
"use client";

import { useState } from "react";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";
import { StockListItem } from "./StockListItem";
import { SidebarSearchInput } from "./SidebarSearchInput";
import type { SearchResult } from "@/entities/search-result";

type SidebarTab = "results" | "watchlist" | "recent";

export function StockListSidebar() {
  const [activeTab, setActiveTab] = useState<SidebarTab>("results");
  const [filter, setFilter] = useState("");
  const { searchResults, watchlist, recentStocks, selectedSymbol, setSelectedSymbol, addToRecent } =
    useStockListStore();
  const { setCurrentStock } = useChartStore();

  const tabConfig: Record<SidebarTab, { label: string; items: SearchResult[] }> = {
    results: { label: "검색결과", items: searchResults },
    watchlist: { label: "관심종목", items: watchlist },
    recent: { label: "최근", items: recentStocks },
  };

  const items = tabConfig[activeTab].items.filter(
    (item) =>
      !filter ||
      item.symbol.includes(filter.toUpperCase()) ||
      item.name.toLowerCase().includes(filter.toLowerCase()),
  );

  const handleStockClick = (item: SearchResult) => {
    setSelectedSymbol(item.symbol);
    addToRecent(item);
    setCurrentStock({
      symbol: item.symbol,
      name: item.name,
      price: 0,   // Go 백엔드 KIS API에서 조회 후 업데이트
      change: 0,
      changePct: 0,
    });
  };

  return (
    <div className="flex flex-col h-full bg-nexus-surface">
      {/* 탭 바 */}
      <div className="flex border-b border-nexus-border">
        {(Object.keys(tabConfig) as SidebarTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 py-2 text-xs font-medium transition-colors ${
              activeTab === tab
                ? "text-nexus-accent border-b-2 border-nexus-accent"
                : "text-nexus-text-secondary hover:text-nexus-text-primary"
            }`}
          >
            {tabConfig[tab].label}
          </button>
        ))}
      </div>

      {/* 검색 입력 */}
      <SidebarSearchInput value={filter} onChange={setFilter} />

      {/* 종목 리스트 */}
      <div className="flex-1 overflow-y-auto">
        {items.map((item) => (
          <StockListItem
            key={item.symbol}
            item={item}
            isActive={item.symbol === selectedSymbol}
            onClick={() => handleStockClick(item)}
          />
        ))}
        {items.length === 0 && (
          <div className="p-4 text-center text-nexus-text-muted text-sm">
            {activeTab === "results" ? "검색 결과가 없습니다" : "항목이 없습니다"}
          </div>
        )}
      </div>
    </div>
  );
}
```

- [ ] 종목 리스트 아이템 컴포넌트 구현 (`src/widgets/stock-list-sidebar/ui/StockListItem.tsx`) — imports from entities/ only (no cross-feature imports from search store)

```typescript
// src/widgets/stock-list-sidebar/ui/StockListItem.tsx
"use client";

import { useStockListStore } from "@/entities/stock";
import type { SearchResult } from "@/entities/search-result";

interface Props {
  item: SearchResult;
  isActive: boolean;
  onClick: () => void;
}

export function StockListItem({ item, isActive, onClick }: Props) {
  const { isInWatchlist, addToWatchlist, removeFromWatchlist } = useStockListStore();
  const inWatchlist = isInWatchlist(item.symbol);

  const handleStarClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (inWatchlist) {
      removeFromWatchlist(item.symbol);
    } else {
      addToWatchlist(item);
    }
  };

  return (
    <div
      data-testid={`stock-item-${item.symbol}`}
      onClick={onClick}
      className={`flex items-center justify-between px-3 py-2 cursor-pointer transition-colors
        ${isActive ? "bg-nexus-accent/10 border-l-2 border-nexus-accent active" : "hover:bg-nexus-border/30"}`}
    >
      <div className="min-w-0">
        <div className="text-sm font-medium truncate">{item.name}</div>
        <div className="text-xs text-nexus-text-muted font-mono">{item.symbol}</div>
      </div>
      <button
        onClick={handleStarClick}
        aria-label={inWatchlist ? "Remove from watchlist" : "Add to watchlist"}
        className={`text-lg transition-colors flex-shrink-0 ml-2 ${
          inWatchlist ? "text-nexus-warning" : "text-nexus-text-muted hover:text-nexus-warning"
        }`}
      >
        {inWatchlist ? "\u2605" : "\u2606"}
      </button>
    </div>
  );
}
```

- [ ] Watchlist feature — star toggle actions (`src/features/watchlist/index.ts`)

```typescript
// src/features/watchlist/index.ts
// Watchlist feature actions delegate to entities/stock store
// This feature owns the UI interaction of toggling watchlist membership
export { useStockListStore as useWatchlistStore } from "@/entities/stock";
```

- [ ] 사이드바 검색 입력 컴포넌트 (`src/widgets/stock-list-sidebar/ui/SidebarSearchInput.tsx`)

```typescript
// src/widgets/stock-list-sidebar/index.ts
export { StockListSidebar } from "./ui/StockListSidebar";
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear
npx jest src/widgets/stock-list-sidebar/__tests__/StockListSidebar.test.tsx --env=jsdom
```

- [ ] 커밋

```bash
git add src/widgets/stock-list-sidebar/ src/features/watchlist/
git commit -m "feat: stock list sidebar widget with 3 tabs and watchlist feature"
```

---

## Task 3: Go 백엔드 — KIS API 클라이언트 + 캔들 데이터 Handler

KIS Open API를 호출하여 캔들 데이터를 조회하는 Go 클라이언트와 `GET /api/v1/candles/:symbol` 핸들러를 구현한다. KIS 클라이언트는 `backend/internal/infra/kis/`에 배치한다.

**Files:**
- Create: `backend/internal/infra/kis/client.go` (KIS API 클라이언트)
- Create: `backend/internal/infra/kis/types.go` (KIS API 타입)
- Create: `backend/internal/infra/kis/candles.go` (캔들 데이터 변환)
- Create: `backend/internal/service/candle_service.go` (캔들 비즈니스 로직)
- Create: `backend/internal/handler/candle_handler.go` — `GET /api/v1/candles/:symbol`
- Test: `backend/internal/infra/kis/candles_test.go`

### Steps

- [ ] KIS API 타입 정의 (`backend/internal/infra/kis/types.go`)

```go
// backend/internal/infra/kis/types.go
package kis

import "time"

// AuthToken holds the cached KIS access token.
type AuthToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

// KISCandle is the raw candle response from KIS Open API.
type KISCandle struct {
	StckBsopDate  string `json:"stck_bsop_date"`   // 영업일자 YYYYMMDD
	StckOprc      string `json:"stck_oprc"`         // 시가
	StckHgpr      string `json:"stck_hgpr"`         // 고가
	StckLwpr      string `json:"stck_lwpr"`         // 저가
	StckClpr      string `json:"stck_clpr"`         // 종가
	AcmlVol       string `json:"acml_vol"`          // 누적거래량
	AcmlTrPbmn    string `json:"acml_tr_pbmn"`      // 누적거래대금
}

// KISPriceResponse is the raw current-price response from KIS Open API.
type KISPriceResponse struct {
	StckPrpr   string `json:"stck_prpr"`    // 현재가
	PrdyVrss   string `json:"prdy_vrss"`    // 전일대비
	PrdyCtrt   string `json:"prdy_ctrt"`    // 전일대비율
	AcmlVol    string `json:"acml_vol"`     // 누적거래량
	Per        string `json:"per"`          // PER
	Eps        string `json:"eps"`          // EPS
	HtsKorIsnm string `json:"hts_kor_isnm"` // 한글 종목명
}

// NormalizedCandle is the clean candle format consumed by the frontend.
type NormalizedCandle struct {
	Time   string  `json:"time"`   // YYYY-MM-DD
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// CandleResponse wraps the candle API response.
type CandleResponse struct {
	Output2 []KISCandle `json:"output2"`
}

// PriceResponse wraps the current-price API response.
type PriceResponse struct {
	Output *KISPriceResponse `json:"output"`
}
```

- [ ] 테스트 먼저 작성 (`backend/internal/infra/kis/candles_test.go`)

```go
// backend/internal/infra/kis/candles_test.go
package kis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeKISCandles(t *testing.T) {
	rawCandles := []KISCandle{
		{
			StckBsopDate: "20260318",
			StckOprc:     "77000",
			StckHgpr:     "79000",
			StckLwpr:     "76500",
			StckClpr:     "78400",
			AcmlVol:      "15234000",
			AcmlTrPbmn:   "1189000000000",
		},
		{
			StckBsopDate: "20260317",
			StckOprc:     "76000",
			StckHgpr:     "77500",
			StckLwpr:     "75800",
			StckClpr:     "77000",
			AcmlVol:      "12100000",
			AcmlTrPbmn:   "932000000000",
		},
	}

	t.Run("converts KIS format to normalized candle data", func(t *testing.T) {
		result := NormalizeKISCandles(rawCandles)
		require.Len(t, result, 2)
		// sorted ascending, so index 1 is 2026-03-18
		assert.Equal(t, "2026-03-18", result[1].Time)
		assert.Equal(t, float64(77000), result[1].Open)
		assert.Equal(t, float64(79000), result[1].High)
		assert.Equal(t, float64(76500), result[1].Low)
		assert.Equal(t, float64(78400), result[1].Close)
		assert.Equal(t, int64(15234000), result[1].Volume)
	})

	t.Run("sorts candles by date ascending", func(t *testing.T) {
		result := NormalizeKISCandles(rawCandles)
		assert.Equal(t, "2026-03-17", result[0].Time)
		assert.Equal(t, "2026-03-18", result[1].Time)
	})

	t.Run("handles empty input", func(t *testing.T) {
		result := NormalizeKISCandles([]KISCandle{})
		assert.Empty(t, result)
	})

	t.Run("converts date format YYYYMMDD to YYYY-MM-DD", func(t *testing.T) {
		result := NormalizeKISCandles(rawCandles[:1])
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, result[0].Time)
	})
}
```

- [ ] 캔들 데이터 변환 유틸 구현 (`backend/internal/infra/kis/candles.go`)

```go
// backend/internal/infra/kis/candles.go
package kis

import (
	"fmt"
	"sort"
	"strconv"
)

// formatDate converts YYYYMMDD to YYYY-MM-DD.
func formatDate(yyyymmdd string) string {
	if len(yyyymmdd) != 8 {
		return yyyymmdd
	}
	return fmt.Sprintf("%s-%s-%s", yyyymmdd[:4], yyyymmdd[4:6], yyyymmdd[6:8])
}

// NormalizeKISCandles normalizes raw KIS candle data and sorts by date ascending.
func NormalizeKISCandles(raw []KISCandle) []NormalizedCandle {
	result := make([]NormalizedCandle, 0, len(raw))

	for _, c := range raw {
		open, _ := strconv.ParseFloat(c.StckOprc, 64)
		high, _ := strconv.ParseFloat(c.StckHgpr, 64)
		low, _ := strconv.ParseFloat(c.StckLwpr, 64)
		close, _ := strconv.ParseFloat(c.StckClpr, 64)
		volume, _ := strconv.ParseInt(c.AcmlVol, 10, 64)

		result = append(result, NormalizedCandle{
			Time:   formatDate(c.StckBsopDate),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Time < result[j].Time
	})

	return result
}
```

- [ ] KIS API 클라이언트 구현 (`backend/internal/infra/kis/client.go`) — 인증 토큰 관리 (자동 갱신), 일봉/주봉/월봉 캔들 조회, 현재가 조회

```go
// backend/internal/infra/kis/client.go
package kis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Client communicates with KIS Open API.
type Client struct {
	httpClient *http.Client
	appKey     string
	appSecret  string
	baseURL    string
	logger     *zap.Logger

	mu          sync.Mutex
	cachedToken *AuthToken
}

// NewClient creates a new KIS API client.
func NewClient(appKey, appSecret, baseURL string, logger *zap.Logger) *Client {
	if baseURL == "" {
		baseURL = "https://openapi.koreainvestment.com:9443"
	}
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		appKey:     appKey,
		appSecret:  appSecret,
		baseURL:    baseURL,
		logger:     logger,
	}
}

// getAccessToken returns a cached token or refreshes it.
func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != nil && time.Now().Before(c.cachedToken.ExpiresAt.Add(-time.Minute)) {
		return c.cachedToken.AccessToken, nil
	}

	c.logger.Info("KIS: refreshing access token")

	body := fmt.Sprintf(`{"grant_type":"client_credentials","appkey":"%s","appsecret":"%s"}`,
		c.appKey, c.appSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/oauth2/tokenP", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.ExpiresIn == 0 {
		tokenResp.ExpiresIn = 86400
	}

	c.cachedToken = &AuthToken{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	return c.cachedToken.AccessToken, nil
}

// authHeaders builds the common authentication headers.
func (c *Client) authHeaders(token string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json; charset=utf-8")
	h.Set("authorization", "Bearer "+token)
	h.Set("appkey", c.appKey)
	h.Set("appsecret", c.appSecret)
	return h
}

// GetCandles fetches daily candle data from KIS Open API.
func (c *Client) GetCandles(ctx context.Context, symbol, startDate, endDate string) ([]NormalizedCandle, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("FID_COND_MRKT_DIV_CODE", "J")
	params.Set("FID_INPUT_ISCD", symbol)
	params.Set("FID_INPUT_DATE_1", startDate)
	params.Set("FID_INPUT_DATE_2", endDate)
	params.Set("FID_PERIOD_DIV_CODE", "D")
	params.Set("FID_ORG_ADJ_PRC", "0")

	reqURL := fmt.Sprintf("%s/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice?%s",
		c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create candle request: %w", err)
	}
	req.Header = c.authHeaders(token)
	req.Header.Set("tr_id", "FHKST03010100")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("candle request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read candle response: %w", err)
	}

	var candleResp CandleResponse
	if err := json.Unmarshal(respBody, &candleResp); err != nil {
		return nil, fmt.Errorf("decode candle response: %w", err)
	}

	return NormalizeKISCandles(candleResp.Output2), nil
}

// GetCurrentPrice fetches the current price of a stock from KIS Open API.
func (c *Client) GetCurrentPrice(ctx context.Context, symbol string) (*KISPriceResponse, error) {
	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("FID_COND_MRKT_DIV_CODE", "J")
	params.Set("FID_INPUT_ISCD", symbol)

	reqURL := fmt.Sprintf("%s/uapi/domestic-stock/v1/quotations/inquire-price?%s",
		c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create price request: %w", err)
	}
	req.Header = c.authHeaders(token)
	req.Header.Set("tr_id", "FHKST01010100")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("price request failed: %w", err)
	}
	defer resp.Body.Close()

	var priceResp PriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&priceResp); err != nil {
		return nil, fmt.Errorf("decode price response: %w", err)
	}

	return priceResp.Output, nil
}
```

- [ ] 캔들 서비스 구현 (`backend/internal/service/candle_service.go`) — 비즈니스 로직 (기본 1년, 날짜 포맷)

```go
// backend/internal/service/candle_service.go
package service

import (
	"context"
	"time"

	"your-module/internal/infra/kis"

	"go.uber.org/zap"
)

// CandleService handles candle data retrieval business logic.
type CandleService struct {
	kisClient *kis.Client
	logger    *zap.Logger
}

// NewCandleService creates a new CandleService.
func NewCandleService(kisClient *kis.Client, logger *zap.Logger) *CandleService {
	return &CandleService{
		kisClient: kisClient,
		logger:    logger,
	}
}

// GetCandles fetches candle data. Defaults to 1-year range if dates are empty.
func (s *CandleService) GetCandles(ctx context.Context, symbol, startDate, endDate, period string) ([]kis.NormalizedCandle, error) {
	if endDate == "" {
		endDate = time.Now().Format("20060102")
	}
	if startDate == "" {
		startDate = time.Now().AddDate(-1, 0, 0).Format("20060102")
	}

	s.logger.Info("fetching candles",
		zap.String("symbol", symbol),
		zap.String("startDate", startDate),
		zap.String("endDate", endDate),
		zap.String("period", period),
	)

	candles, err := s.kisClient.GetCandles(ctx, symbol, startDate, endDate)
	if err != nil {
		s.logger.Error("failed to fetch candles",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		return nil, err
	}

	return candles, nil
}
```

- [ ] 캔들 핸들러 구현 (`backend/internal/handler/candle_handler.go`) — `GET /api/v1/candles/:symbol`

```go
// backend/internal/handler/candle_handler.go
package handler

import (
	"net/http"

	"your-module/internal/service"

	"github.com/gin-gonic/gin"
)

// CandleHandler handles candle-related HTTP requests.
type CandleHandler struct {
	candleSvc *service.CandleService
}

// NewCandleHandler creates a new CandleHandler.
func NewCandleHandler(candleSvc *service.CandleService) *CandleHandler {
	return &CandleHandler{candleSvc: candleSvc}
}

// GetCandles godoc
// @Summary     캔들 데이터 조회
// @Description KIS API를 통해 종목의 캔들(일봉) 데이터를 반환한다
// @Tags        candles
// @Param       symbol    path   string false "종목코드 (예: 005930)"
// @Param       period    query  string false "기간 (D/W/M, 기본 D)"
// @Param       startDate query  string false "시작일 YYYYMMDD"
// @Param       endDate   query  string false "종료일 YYYYMMDD"
// @Success     200 {object} map[string]interface{}
// @Failure     500 {object} map[string]interface{}
// @Router      /api/v1/candles/{symbol} [get]
func (h *CandleHandler) GetCandles(c *gin.Context) {
	symbol := c.Param("symbol")
	period := c.DefaultQuery("period", "D")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	candles, err := h.candleSvc.GetCandles(c.Request.Context(), symbol, startDate, endDate, period)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"candles": candles,
			"symbol":  symbol,
		},
	})
}
```

- [ ] `.env`에 KIS API 키 추가

```bash
# backend/.env
KIS_APP_KEY=your-kis-app-key
KIS_APP_SECRET=your-kis-app-secret
KIS_BASE_URL=https://openapi.koreainvestment.com:9443
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/infra/kis/... -v
```

- [ ] 커밋

```bash
git add backend/internal/infra/kis/ backend/internal/service/candle_service.go backend/internal/handler/candle_handler.go
git commit -m "feat: Go KIS API client with candle handler and service layer"
```

---

## Task 4: 메인 캔들스틱 차트 (lightweight-charts) + Indicator Entity

TradingView의 lightweight-charts 라이브러리로 캔들스틱 차트를 렌더링한다. MA(이동평균), BB(볼린저밴드) 오버레이를 지원한다. 지표 계산은 `entities/indicator/lib/`에 배치한다. **프론트엔드 API 호출은 Go 백엔드의 `/api/v1/candles/:symbol`을 사용한다.**

**Files:**
- Create: `src/features/chart/ui/MainChart.tsx`
- Create: `src/features/chart/api/chart-api.ts` (data fetching API client — Go 서버 호출)
- Create: `src/entities/indicator/lib/ma.ts` (이동평균 계산)
- Create: `src/entities/indicator/lib/bollinger.ts` (볼린저밴드 계산)
- Create: `src/entities/indicator/lib/rsi.ts` (RSI 계산)
- Create: `src/entities/indicator/index.ts`
- Create: `src/features/chart/lib/use-chart-data.ts` (데이터 fetching hook)
- Test: `src/entities/indicator/__tests__/indicators.test.ts`

### Steps

- [ ] lightweight-charts 설치

```bash
cd /home/dev/code/dev-superbear
npm install lightweight-charts
```

- [ ] 테스트 먼저 작성 (`src/entities/indicator/__tests__/indicators.test.ts`)

```typescript
// src/entities/indicator/__tests__/indicators.test.ts
import { calculateMA } from "../lib/ma";
import { calculateBollingerBands } from "../lib/bollinger";

const sampleData = [
  { close: 100 }, { close: 102 }, { close: 98 }, { close: 104 }, { close: 106 },
  { close: 103 }, { close: 107 }, { close: 110 }, { close: 108 }, { close: 112 },
];

describe("Technical Indicators", () => {
  describe("calculateMA", () => {
    it("calculates 5-day MA correctly", () => {
      const ma = calculateMA(sampleData.map((d) => d.close), 5);
      expect(ma).toHaveLength(10);
      // 처음 4개는 null (기간 미달)
      expect(ma[0]).toBeNull();
      expect(ma[3]).toBeNull();
      // 5번째부터 유효값
      expect(ma[4]).toBeCloseTo((100 + 102 + 98 + 104 + 106) / 5);
    });

    it("returns all null for period > data length", () => {
      const ma = calculateMA([100, 200], 5);
      expect(ma.every((v) => v === null)).toBe(true);
    });
  });

  describe("calculateBollingerBands", () => {
    it("returns upper, middle, lower bands", () => {
      const bb = calculateBollingerBands(sampleData.map((d) => d.close), 5, 2);
      expect(bb.upper).toHaveLength(10);
      expect(bb.middle).toHaveLength(10);
      expect(bb.lower).toHaveLength(10);
    });

    it("middle band equals MA", () => {
      const closes = sampleData.map((d) => d.close);
      const ma = calculateMA(closes, 5);
      const bb = calculateBollingerBands(closes, 5, 2);
      for (let i = 0; i < 10; i++) {
        expect(bb.middle[i]).toEqual(ma[i]);
      }
    });

    it("upper > middle > lower when valid", () => {
      const bb = calculateBollingerBands(sampleData.map((d) => d.close), 5, 2);
      for (let i = 4; i < 10; i++) {
        expect(bb.upper[i]!).toBeGreaterThan(bb.middle[i]!);
        expect(bb.middle[i]!).toBeGreaterThan(bb.lower[i]!);
      }
    });
  });
});
```

- [ ] 지표 계산 유틸 구현 — split into individual entity modules

```typescript
// src/entities/indicator/lib/ma.ts
/** N일 이동평균 계산. 기간 미달인 초기 값은 null */
export function calculateMA(closes: number[], period: number): (number | null)[] {
  const result: (number | null)[] = [];
  for (let i = 0; i < closes.length; i++) {
    if (i < period - 1) {
      result.push(null);
    } else {
      const sum = closes.slice(i - period + 1, i + 1).reduce((a, b) => a + b, 0);
      result.push(sum / period);
    }
  }
  return result;
}
```

```typescript
// src/entities/indicator/lib/bollinger.ts
import { calculateMA } from "./ma";

/** 볼린저밴드 계산 */
export function calculateBollingerBands(
  closes: number[],
  period: number,
  k: number,
): {
  upper: (number | null)[];
  middle: (number | null)[];
  lower: (number | null)[];
} {
  const middle = calculateMA(closes, period);
  const upper: (number | null)[] = [];
  const lower: (number | null)[] = [];

  for (let i = 0; i < closes.length; i++) {
    if (middle[i] === null) {
      upper.push(null);
      lower.push(null);
    } else {
      const slice = closes.slice(i - period + 1, i + 1);
      const mean = middle[i]!;
      const variance = slice.reduce((sum, val) => sum + (val - mean) ** 2, 0) / period;
      const stddev = Math.sqrt(variance);
      upper.push(mean + k * stddev);
      lower.push(mean - k * stddev);
    }
  }

  return { upper, middle, lower };
}
```

```typescript
// src/entities/indicator/lib/rsi.ts
/** RSI 계산 */
export function calculateRSI(closes: number[], period: number = 14): (number | null)[] {
  const result: (number | null)[] = [null]; // 첫 값은 변동 없음
  const gains: number[] = [];
  const losses: number[] = [];

  for (let i = 1; i < closes.length; i++) {
    const diff = closes[i] - closes[i - 1];
    gains.push(diff > 0 ? diff : 0);
    losses.push(diff < 0 ? -diff : 0);

    if (i < period) {
      result.push(null);
    } else if (i === period) {
      const avgGain = gains.slice(0, period).reduce((a, b) => a + b, 0) / period;
      const avgLoss = losses.slice(0, period).reduce((a, b) => a + b, 0) / period;
      result.push(avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss));
    } else {
      const avgGain = (gains[i - 2] * (period - 1) + gains[i - 1]) / period;
      const avgLoss = (losses[i - 2] * (period - 1) + losses[i - 1]) / period;
      result.push(avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss));
    }
  }

  return result;
}
```

```typescript
// src/entities/indicator/index.ts
export { calculateMA } from "./lib/ma";
export { calculateBollingerBands } from "./lib/bollinger";
export { calculateRSI } from "./lib/rsi";
```

- [ ] Chart API client — Go 백엔드 `/api/v1/` 호출 (`src/features/chart/api/chart-api.ts`)

```typescript
// src/features/chart/api/chart-api.ts
import type { CandleData } from "@/entities/candle";
import { logger } from "@/shared/lib/logger";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const chartApi = {
  /** Go 백엔드 GET /api/v1/candles/:symbol */
  async fetchCandles(symbol: string, timeframe: string): Promise<CandleData[]> {
    try {
      const res = await fetch(`${API_BASE}/api/v1/candles/${symbol}?period=${timeframe}`);
      const json = await res.json();
      return (json as { data?: { candles?: CandleData[] } }).data?.candles ?? [];
    } catch (err: unknown) {
      logger.error("Failed to fetch candles", {
        symbol,
        message: err instanceof Error ? err.message : String(err),
      });
      return [];
    }
  },

  /** Go 백엔드 GET /api/v1/candles/:symbol/price */
  async fetchCurrentPrice(symbol: string) {
    try {
      const res = await fetch(`${API_BASE}/api/v1/candles/${symbol}/price`);
      const json = await res.json();
      return (json as { data?: Record<string, unknown> }).data ?? null;
    } catch (err: unknown) {
      logger.error("Failed to fetch current price", {
        symbol,
        message: err instanceof Error ? err.message : String(err),
      });
      return null;
    }
  },
};
```

- [ ] 데이터 fetching 훅 구현 (`src/features/chart/lib/use-chart-data.ts`) — uses chart-api (Go backend), no console.log

```typescript
// src/features/chart/lib/use-chart-data.ts
"use client";

import { useEffect } from "react";
import { useChartStore } from "../model/chart.store";
import { useStockListStore } from "@/entities/stock";
import { chartApi } from "../api/chart-api";

export function useChartData() {
  const { currentStock, timeframe, setCandles, setIsLoading } = useChartStore();
  const { selectedSymbol } = useStockListStore();

  useEffect(() => {
    if (!currentStock?.symbol) return;

    const fetchCandles = async () => {
      setIsLoading(true);
      try {
        const candles = await chartApi.fetchCandles(currentStock.symbol, timeframe);
        setCandles(candles);
      } finally {
        setIsLoading(false);
      }
    };

    fetchCandles();
  }, [currentStock?.symbol, timeframe, setCandles, setIsLoading]);

  return { isLoading: useChartStore((s) => s.isLoading) };
}
```

- [ ] 메인 차트 컴포넌트 구현 (`src/features/chart/ui/MainChart.tsx`) — lightweight-charts의 `createChart`, `addCandlestickSeries`, 오버레이 `addLineSeries` 사용. Imports indicator calculations from `entities/indicator/`

```typescript
// src/features/chart/ui/MainChart.tsx (핵심 구조)
"use client";

import { useEffect, useRef } from "react";
import { createChart, IChartApi, ISeriesApi, CandlestickData, LineData } from "lightweight-charts";
import { useChartStore } from "../model/chart.store";
import { useChartData } from "../lib/use-chart-data";
import { calculateMA } from "@/entities/indicator";

export function MainChart() {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<"Candlestick"> | null>(null);
  const overlaySeriesRef = useRef<Map<string, ISeriesApi<"Line">>>(new Map());

  const { candles, activeIndicators, isLoading } = useChartStore();
  useChartData();

  // 차트 생성 (한 번)
  useEffect(() => {
    if (!chartContainerRef.current) return;

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { color: "#0a0a0f" },
        textColor: "#94a3b8",
      },
      grid: {
        vertLines: { color: "#1e1e2e" },
        horzLines: { color: "#1e1e2e" },
      },
      crosshair: { mode: 0 },
      rightPriceScale: { borderColor: "#1e1e2e" },
      timeScale: { borderColor: "#1e1e2e" },
    });

    const candleSeries = chart.addCandlestickSeries({
      upColor: "#22c55e",
      downColor: "#ef4444",
      borderDownColor: "#ef4444",
      borderUpColor: "#22c55e",
      wickDownColor: "#ef4444",
      wickUpColor: "#22c55e",
    });

    chartRef.current = chart;
    candleSeriesRef.current = candleSeries;

    const resizeObserver = new ResizeObserver((entries) => {
      const { width, height } = entries[0].contentRect;
      chart.applyOptions({ width, height });
    });
    resizeObserver.observe(chartContainerRef.current);

    return () => {
      resizeObserver.disconnect();
      chart.remove();
    };
  }, []);

  // 캔들 데이터 업데이트
  useEffect(() => {
    if (!candleSeriesRef.current || candles.length === 0) return;

    const candleData: CandlestickData[] = candles.map((c) => ({
      time: c.time as string & { __brand: "UTCDate" },
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
    }));

    candleSeriesRef.current.setData(candleData);

    // 오버레이 지표 업데이트
    updateOverlays();
  }, [candles, activeIndicators]);

  const updateOverlays = () => {
    if (!chartRef.current) return;

    const closes = candles.map((c) => c.close);
    const overlayConfigs: Record<string, { period: number; color: string }> = {
      ma5: { period: 5, color: "#f59e0b" },
      ma20: { period: 20, color: "#6366f1" },
      ma60: { period: 60, color: "#22c55e" },
      ma120: { period: 120, color: "#ef4444" },
      ma200: { period: 200, color: "#8b5cf6" },
    };

    // 기존 오버레이 제거
    overlaySeriesRef.current.forEach((series) => {
      chartRef.current!.removeSeries(series);
    });
    overlaySeriesRef.current.clear();

    // 활성 오버레이 추가
    for (const id of activeIndicators) {
      const config = overlayConfigs[id];
      if (!config) continue;

      const maValues = calculateMA(closes, config.period);
      const lineData: LineData[] = candles
        .map((c, i) => ({
          time: c.time as string & { __brand: "UTCDate" },
          value: maValues[i] ?? undefined,
        }))
        .filter((d): d is LineData => d.value !== undefined);

      const series = chartRef.current!.addLineSeries({
        color: config.color,
        lineWidth: 1,
        priceLineVisible: false,
      });
      series.setData(lineData);
      overlaySeriesRef.current.set(id, series);
    }
  };

  return (
    <div className="relative w-full h-full">
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-nexus-bg/50 z-10">
          <span className="text-nexus-text-muted">Loading chart data...</span>
        </div>
      )}
      <div ref={chartContainerRef} className="w-full h-full" />
    </div>
  );
}
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear
npx jest src/entities/indicator/__tests__/indicators.test.ts
```

- [ ] 커밋

```bash
git add src/features/chart/ src/entities/indicator/
git commit -m "feat: candlestick chart with indicator entity and MA/BB overlay"
```

---

## Task 5: 보조 지표 패널 위젯 (RSI, MACD, Revenue)

메인 차트 아래에 표시되는 보조 지표 패널을 `widgets/sub-indicator-panel/`로 구현한다. MACD 계산을 `entities/indicator/lib/macd.ts`에 추가한다.

**Files:**
- Create: `src/widgets/sub-indicator-panel/ui/SubIndicatorPanel.tsx`
- Create: `src/widgets/sub-indicator-panel/ui/RSIChart.tsx`
- Create: `src/widgets/sub-indicator-panel/ui/MACDChart.tsx`
- Create: `src/widgets/sub-indicator-panel/ui/RevenueChart.tsx`
- Create: `src/widgets/sub-indicator-panel/index.ts`
- Create: `src/entities/indicator/lib/macd.ts`
- Test: `src/entities/indicator/__tests__/macd.test.ts`

### Steps

- [ ] 테스트 먼저 작성 (`src/entities/indicator/__tests__/macd.test.ts`)

```typescript
// src/entities/indicator/__tests__/macd.test.ts
import { calculateMACD } from "../lib/macd";

describe("calculateMACD", () => {
  // 30일 이상의 샘플 데이터
  const closes = Array.from({ length: 40 }, (_, i) => 100 + Math.sin(i * 0.5) * 10);

  it("returns macd, signal, and histogram arrays", () => {
    const result = calculateMACD(closes, 12, 26, 9);
    expect(result.macd).toHaveLength(40);
    expect(result.signal).toHaveLength(40);
    expect(result.histogram).toHaveLength(40);
  });

  it("MACD line is null for first 25 values (26-period EMA not ready)", () => {
    const result = calculateMACD(closes, 12, 26, 9);
    for (let i = 0; i < 25; i++) {
      expect(result.macd[i]).toBeNull();
    }
    expect(result.macd[25]).not.toBeNull();
  });

  it("histogram = macd - signal", () => {
    const result = calculateMACD(closes, 12, 26, 9);
    for (let i = 33; i < 40; i++) {
      if (result.macd[i] !== null && result.signal[i] !== null) {
        expect(result.histogram[i]).toBeCloseTo(result.macd[i]! - result.signal[i]!, 5);
      }
    }
  });
});
```

- [ ] MACD 계산 유틸 구현 (`src/entities/indicator/lib/macd.ts`)

```typescript
// src/entities/indicator/lib/macd.ts

function ema(data: number[], period: number): (number | null)[] {
  const result: (number | null)[] = [];
  const k = 2 / (period + 1);

  for (let i = 0; i < data.length; i++) {
    if (i < period - 1) {
      result.push(null);
    } else if (i === period - 1) {
      const sum = data.slice(0, period).reduce((a, b) => a + b, 0);
      result.push(sum / period);
    } else {
      result.push(data[i] * k + (result[i - 1] as number) * (1 - k));
    }
  }

  return result;
}

export interface MACDResult {
  macd: (number | null)[];
  signal: (number | null)[];
  histogram: (number | null)[];
}

export function calculateMACD(
  closes: number[],
  shortPeriod: number = 12,
  longPeriod: number = 26,
  signalPeriod: number = 9,
): MACDResult {
  const shortEma = ema(closes, shortPeriod);
  const longEma = ema(closes, longPeriod);

  // MACD line = short EMA - long EMA
  const macdLine: (number | null)[] = closes.map((_, i) => {
    if (shortEma[i] === null || longEma[i] === null) return null;
    return shortEma[i]! - longEma[i]!;
  });

  // Signal line = EMA of MACD line
  const validMacd = macdLine.filter((v): v is number => v !== null);
  const signalEma = ema(validMacd, signalPeriod);

  // 정렬하여 원래 인덱스에 매핑
  const signal: (number | null)[] = [];
  let validIdx = 0;
  for (let i = 0; i < closes.length; i++) {
    if (macdLine[i] === null) {
      signal.push(null);
    } else {
      signal.push(signalEma[validIdx] ?? null);
      validIdx++;
    }
  }

  // Histogram = MACD - Signal
  const histogram: (number | null)[] = closes.map((_, i) => {
    if (macdLine[i] === null || signal[i] === null) return null;
    return macdLine[i]! - signal[i]!;
  });

  return { macd: macdLine, signal, histogram };
}
```

- [ ] Update indicator barrel export

```typescript
// src/entities/indicator/index.ts (updated)
export { calculateMA } from "./lib/ma";
export { calculateBollingerBands } from "./lib/bollinger";
export { calculateRSI } from "./lib/rsi";
export { calculateMACD } from "./lib/macd";
export type { MACDResult } from "./lib/macd";
```

- [ ] 보조 지표 패널 위젯 컨테이너 구현 (`src/widgets/sub-indicator-panel/ui/SubIndicatorPanel.tsx`) — 활성화된 보조 지표별로 미니 차트 렌더링. 지표 토글 버튼 바 포함

```typescript
// src/widgets/sub-indicator-panel/ui/SubIndicatorPanel.tsx
"use client";

import { useChartStore } from "@/features/chart";
import { RSIChart } from "./RSIChart";
import { MACDChart } from "./MACDChart";
import { RevenueChart } from "./RevenueChart";

const SUB_INDICATORS = [
  { id: "rsi", label: "RSI" },
  { id: "macd", label: "MACD" },
  { id: "revenue", label: "Revenue" },
];

export function SubIndicatorPanel() {
  const { activeSubIndicators, toggleSubIndicator } = useChartStore();

  return (
    <div className="border-t border-nexus-border">
      {/* 지표 선택 바 */}
      <div className="flex items-center gap-1 px-3 py-1 bg-nexus-surface border-b border-nexus-border">
        {SUB_INDICATORS.map((ind) => (
          <button
            key={ind.id}
            onClick={() => toggleSubIndicator(ind.id)}
            className={`px-2 py-0.5 text-xs rounded transition-colors ${
              activeSubIndicators.includes(ind.id)
                ? "bg-nexus-accent/20 text-nexus-accent"
                : "text-nexus-text-muted hover:text-nexus-text-secondary"
            }`}
          >
            [{ind.label}]
          </button>
        ))}
      </div>

      {/* 활성화된 지표 차트들 */}
      <div className="flex flex-col">
        {activeSubIndicators.includes("rsi") && (
          <div className="h-24 border-b border-nexus-border">
            <RSIChart />
          </div>
        )}
        {activeSubIndicators.includes("macd") && (
          <div className="h-24 border-b border-nexus-border">
            <MACDChart />
          </div>
        )}
        {activeSubIndicators.includes("revenue") && (
          <div className="h-32 border-b border-nexus-border">
            <RevenueChart />
          </div>
        )}
      </div>
    </div>
  );
}
```

```typescript
// src/widgets/sub-indicator-panel/index.ts
export { SubIndicatorPanel } from "./ui/SubIndicatorPanel";
```

- [ ] RSI 차트 컴포넌트 (`src/widgets/sub-indicator-panel/ui/RSIChart.tsx`) — lightweight-charts `addLineSeries`로 RSI 라인 + 30/70 수평선 렌더링

- [ ] MACD 차트 컴포넌트 (`src/widgets/sub-indicator-panel/ui/MACDChart.tsx`) — MACD 라인 + 시그널 라인 + 히스토그램 바

- [ ] Revenue 차트 컴포넌트 (`src/widgets/sub-indicator-panel/ui/RevenueChart.tsx`) — 분기별 매출/영업이익 바 차트 (데이터는 Go 백엔드 financials API에서)

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear
npx jest src/entities/indicator/__tests__/macd.test.ts
```

- [ ] 커밋

```bash
git add src/widgets/sub-indicator-panel/ src/entities/indicator/
git commit -m "feat: sub-indicator panel widget with RSI, MACD, and Revenue charts"
```

---

## Task 6: 하단 정보 패널 위젯 (Financials / AI Fusion / Sector Compare)

스펙 Section 2.4의 하단 full-width 3칼럼 정보 패널을 `widgets/bottom-info-panel/`로 구현한다. **프론트엔드는 Go 백엔드의 `/api/v1/financials/:symbol` 엔드포인트를 호출한다.**

**Files:**
- Create: `src/widgets/bottom-info-panel/ui/BottomInfoPanel.tsx`
- Create: `src/widgets/bottom-info-panel/ui/FinancialsPanel.tsx`
- Create: `src/widgets/bottom-info-panel/ui/AIFusionPanel.tsx`
- Create: `src/widgets/bottom-info-panel/ui/SectorComparePanel.tsx`
- Create: `src/widgets/bottom-info-panel/index.ts`
- Test: `src/widgets/bottom-info-panel/__tests__/BottomInfoPanel.test.tsx`

### Steps

- [ ] 테스트 먼저 작성 (`src/widgets/bottom-info-panel/__tests__/BottomInfoPanel.test.tsx`)

```typescript
// src/widgets/bottom-info-panel/__tests__/BottomInfoPanel.test.tsx
import { render, screen } from "@testing-library/react";
import { BottomInfoPanel } from "../ui/BottomInfoPanel";
import { useChartStore } from "@/features/chart";

beforeEach(() => {
  useChartStore.setState({
    ...useChartStore.getInitialState(),
    currentStock: {
      symbol: "005930",
      name: "Samsung Electronics",
      price: 78400,
      change: 1600,
      changePct: 2.08,
    },
  });
});

describe("BottomInfoPanel", () => {
  it("renders all 3 column headers", () => {
    render(<BottomInfoPanel />);
    expect(screen.getByText(/financials/i)).toBeInTheDocument();
    expect(screen.getByText(/ai fusion/i)).toBeInTheDocument();
    expect(screen.getByText(/sector compare/i)).toBeInTheDocument();
  });

  it("shows financial metrics labels", () => {
    render(<BottomInfoPanel />);
    expect(screen.getByText(/revenue/i)).toBeInTheDocument();
    expect(screen.getByText(/PER/)).toBeInTheDocument();
    expect(screen.getByText(/ROE/)).toBeInTheDocument();
  });

  it("shows empty state when no stock selected", () => {
    useChartStore.setState({ currentStock: null });
    render(<BottomInfoPanel />);
    expect(screen.getByText(/종목을 선택|select a stock/i)).toBeInTheDocument();
  });
});
```

- [ ] 하단 패널 위젯 컨테이너 구현 (`src/widgets/bottom-info-panel/ui/BottomInfoPanel.tsx`)

```typescript
// src/widgets/bottom-info-panel/ui/BottomInfoPanel.tsx
"use client";

import { useChartStore } from "@/features/chart";
import { FinancialsPanel } from "./FinancialsPanel";
import { AIFusionPanel } from "./AIFusionPanel";
import { SectorComparePanel } from "./SectorComparePanel";

export function BottomInfoPanel() {
  const { currentStock } = useChartStore();

  if (!currentStock) {
    return (
      <div className="h-48 border-t border-nexus-border bg-nexus-surface flex items-center justify-center">
        <span className="text-nexus-text-muted">Select a stock to view details</span>
      </div>
    );
  }

  return (
    <div className="border-t border-nexus-border bg-nexus-surface">
      <div className="grid grid-cols-3 divide-x divide-nexus-border min-h-[200px]">
        <FinancialsPanel symbol={currentStock.symbol} />
        <AIFusionPanel symbol={currentStock.symbol} />
        <SectorComparePanel symbol={currentStock.symbol} />
      </div>
    </div>
  );
}
```

- [ ] Financials 패널 구현 (`src/widgets/bottom-info-panel/ui/FinancialsPanel.tsx`) — Go 백엔드 `/api/v1/financials/:symbol` 호출. Uses logger instead of console

```typescript
// src/widgets/bottom-info-panel/ui/FinancialsPanel.tsx
"use client";

import { useEffect, useState } from "react";
import { logger } from "@/shared/lib/logger";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface FinancialData {
  revenue: number | null;
  operatingProfit: number | null;
  netMargin: number | null;
  per: number | null;
  pbr: number | null;
  roe: number | null;
}

export function FinancialsPanel({ symbol }: { symbol: string }) {
  const [data, setData] = useState<FinancialData | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch(`${API_BASE}/api/v1/financials/${symbol}`)
      .then((res) => res.json())
      .then((json) => setData((json as { data?: FinancialData }).data ?? null))
      .catch((err) => {
        logger.error("Failed to fetch financials", { symbol, message: String(err) });
        setData(null);
      })
      .finally(() => setLoading(false));
  }, [symbol]);

  const metrics = data
    ? [
        { label: "Revenue", value: data.revenue, format: "억원" },
        { label: "Op.Profit", value: data.operatingProfit, format: "억원" },
        { label: "Net Margin", value: data.netMargin, format: "%" },
        { label: "PER", value: data.per, format: "x" },
        { label: "PBR", value: data.pbr, format: "x" },
        { label: "ROE", value: data.roe, format: "%" },
      ]
    : [];

  return (
    <div className="p-4">
      <h3 className="text-xs font-semibold text-nexus-text-secondary uppercase mb-3">
        Financials
      </h3>
      {loading ? (
        <div className="text-nexus-text-muted text-sm">Loading...</div>
      ) : (
        <div className="space-y-2">
          {metrics.map((m) => (
            <div key={m.label} className="flex justify-between text-sm">
              <span className="text-nexus-text-secondary">{m.label}</span>
              <span className="font-mono">
                {m.value != null ? `${m.value.toLocaleString()}${m.format}` : "-"}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] AI Fusion 패널 구현 (`src/widgets/bottom-info-panel/ui/AIFusionPanel.tsx`) — placeholder

```typescript
// src/widgets/bottom-info-panel/ui/AIFusionPanel.tsx
"use client";

export function AIFusionPanel({ symbol }: { symbol: string }) {
  return (
    <div className="p-4">
      <h3 className="text-xs font-semibold text-nexus-text-secondary uppercase mb-3">
        AI Fusion
      </h3>
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-nexus-text-secondary">Signal</span>
          <span className="px-2 py-0.5 text-xs rounded-full bg-nexus-text-muted/20 text-nexus-text-muted">
            No analysis yet
          </span>
        </div>
        <div>
          <span className="text-sm text-nexus-text-secondary">Tags</span>
          <div className="flex flex-wrap gap-1 mt-1">
            <span className="text-xs text-nexus-text-muted">
              Run a pipeline to generate AI analysis
            </span>
          </div>
        </div>
        <div>
          <span className="text-sm text-nexus-text-secondary">Summary</span>
          <p className="text-xs text-nexus-text-muted mt-1">
            Execute a pipeline on this stock to see AI-generated cross-analysis of fundamental and technical factors.
          </p>
        </div>
      </div>
    </div>
  );
}
```

- [ ] Sector Compare 패널 구현 (`src/widgets/bottom-info-panel/ui/SectorComparePanel.tsx`) — Go 백엔드 호출, uses logger

```typescript
// src/widgets/bottom-info-panel/ui/SectorComparePanel.tsx
"use client";

import { useEffect, useState } from "react";
import { logger } from "@/shared/lib/logger";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface SectorStock {
  symbol: string;
  name: string;
  per: number | null;
  roe: number | null;
  rsi: number | null;
  changePct: number;
}

export function SectorComparePanel({ symbol }: { symbol: string }) {
  const [stocks, setStocks] = useState<SectorStock[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch(`${API_BASE}/api/v1/financials/${symbol}/sector`)
      .then((res) => res.json())
      .then((json) => setStocks((json as { data?: SectorStock[] }).data ?? []))
      .catch((err) => {
        logger.error("Failed to fetch sector data", { symbol, message: String(err) });
        setStocks([]);
      })
      .finally(() => setLoading(false));
  }, [symbol]);

  return (
    <div className="p-4">
      <h3 className="text-xs font-semibold text-nexus-text-secondary uppercase mb-3">
        Sector Compare
      </h3>
      {loading ? (
        <div className="text-nexus-text-muted text-sm">Loading...</div>
      ) : stocks.length === 0 ? (
        <div className="text-nexus-text-muted text-sm">No sector data available</div>
      ) : (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-nexus-text-muted">
              <th className="text-left py-1">Name</th>
              <th className="text-right py-1">PER</th>
              <th className="text-right py-1">ROE</th>
              <th className="text-right py-1">RSI</th>
              <th className="text-right py-1">Chg%</th>
            </tr>
          </thead>
          <tbody>
            {stocks.map((s) => (
              <tr
                key={s.symbol}
                className={s.symbol === symbol ? "text-nexus-accent font-medium" : ""}
              >
                <td className="py-1 truncate max-w-[100px]">{s.name}</td>
                <td className="text-right font-mono">{s.per?.toFixed(1) ?? "-"}</td>
                <td className="text-right font-mono">{s.roe?.toFixed(1) ?? "-"}%</td>
                <td className="text-right font-mono">{s.rsi?.toFixed(0) ?? "-"}</td>
                <td className={`text-right font-mono ${s.changePct >= 0 ? "text-nexus-success" : "text-nexus-failure"}`}>
                  {s.changePct >= 0 ? "+" : ""}{s.changePct.toFixed(2)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
```

```typescript
// src/widgets/bottom-info-panel/index.ts
export { BottomInfoPanel } from "./ui/BottomInfoPanel";
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear
npx jest src/widgets/bottom-info-panel/__tests__/BottomInfoPanel.test.tsx --env=jsdom
```

- [ ] 커밋

```bash
git add src/widgets/bottom-info-panel/
git commit -m "feat: bottom info panel widget with Financials, AI Fusion, and Sector Compare"
```

---

## Task 7: Go 백엔드 — DART API 클라이언트 + 재무제표/관심종목 Handler

차트 하단 패널에서 사용하는 재무제표 및 섹터 비교 API, 관심종목 CRUD를 Go 핸들러로 구현한다. DART 클라이언트는 `backend/internal/infra/dart/`에, 관심종목은 sqlc 기반 repository로 구현한다.

**Files:**
- Create: `backend/internal/infra/dart/client.go` (DART API 클라이언트)
- Create: `backend/internal/infra/dart/types.go` (DART API 타입)
- Create: `backend/internal/service/financials_service.go` (재무 비즈니스 로직)
- Create: `backend/internal/handler/financials_handler.go` — `GET /api/v1/financials/:symbol`, `GET /api/v1/financials/:symbol/sector`
- Create: `backend/internal/repository/watchlist_repo.go` (sqlc 관심종목)
- Create: `backend/internal/handler/watchlist_handler.go` — CRUD `/api/v1/watchlist`
- Create: `backend/db/queries/watchlist.sql` (sqlc 쿼리)
- Create: `backend/db/migrations/003_watchlist.sql`
- Test: `backend/internal/infra/dart/client_test.go`
- Test: `backend/internal/handler/watchlist_handler_test.go`

### Steps

- [ ] DART API 타입 정의 (`backend/internal/infra/dart/types.go`)

```go
// backend/internal/infra/dart/types.go
package dart

// RawFinancials is the raw response from DART Open API.
type RawFinancials struct {
	Revenue         string `json:"revenue"`
	OperatingProfit string `json:"operating_profit"`
	NetIncome       string `json:"net_income"`
}

// NormalizedFinancials is the cleaned financial data for the frontend.
type NormalizedFinancials struct {
	Revenue         *float64 `json:"revenue"`          // 억원
	OperatingProfit *float64 `json:"operatingProfit"`  // 억원
	NetMargin       *float64 `json:"netMargin"`        // %
	PER             *float64 `json:"per"`
	PBR             *float64 `json:"pbr"`
	ROE             *float64 `json:"roe"`
}

// DARTFinancialResponse wraps the DART API single-account response.
type DARTFinancialResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	List    []DARTFinancialItem `json:"list"`
}

// DARTFinancialItem is one line item in DART financial statements.
type DARTFinancialItem struct {
	RceptNo    string `json:"rcept_no"`
	BsnsYear   string `json:"bsns_year"`
	CorpCode   string `json:"corp_code"`
	AccountNm  string `json:"account_nm"`
	ThstrmAmt  string `json:"thstrm_amount"`
}
```

- [ ] 테스트 먼저 작성 (`backend/internal/infra/dart/client_test.go`)

```go
// backend/internal/infra/dart/client_test.go
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
		assert.InDelta(t, 678900.0, *result.Revenue, 1.0) // 억원 단위
		assert.NotNil(t, result.OperatingProfit)
		assert.InDelta(t, 123450.0, *result.OperatingProfit, 1.0)
	})

	t.Run("handles missing values gracefully", func(t *testing.T) {
		result := NormalizeFinancialStatements(RawFinancials{})
		assert.Nil(t, result.Revenue)
		assert.Nil(t, result.OperatingProfit)
	})
}
```

- [ ] DART API 클라이언트 구현 (`backend/internal/infra/dart/client.go`) — 재무제표 조회, 데이터 정규화

```go
// backend/internal/infra/dart/client.go
package dart

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/zap"
)

const dartBaseURL = "https://opendart.fss.or.kr/api"

// Client communicates with DART Open API.
type Client struct {
	httpClient *http.Client
	apiKey     string
	logger     *zap.Logger
}

// NewClient creates a new DART API client.
func NewClient(apiKey string, logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		apiKey:     apiKey,
		logger:     logger,
	}
}

// toEok converts a raw string amount (원) to 억원.
func toEok(val string) *float64 {
	if val == "" {
		return nil
	}
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return nil
	}
	result := math.Round(num / 100_000_000)
	return &result
}

// NormalizeFinancialStatements normalizes raw DART financial data.
func NormalizeFinancialStatements(raw RawFinancials) NormalizedFinancials {
	revenue := toEok(raw.Revenue)
	operatingProfit := toEok(raw.OperatingProfit)
	netIncome := toEok(raw.NetIncome)

	var netMargin *float64
	if revenue != nil && netIncome != nil && *revenue != 0 {
		m := (*netIncome / *revenue) * 100
		netMargin = &m
	}

	return NormalizedFinancials{
		Revenue:         revenue,
		OperatingProfit: operatingProfit,
		NetMargin:       netMargin,
		PER:             nil, // KIS API에서 별도 조회
		PBR:             nil,
		ROE:             nil,
	}
}

// FetchFinancialStatements fetches financial statements from DART Open API.
func (c *Client) FetchFinancialStatements(ctx context.Context, corpCode, year, reportCode string) (NormalizedFinancials, error) {
	if reportCode == "" {
		reportCode = "11011" // 사업보고서
	}

	params := url.Values{}
	params.Set("crtfc_key", c.apiKey)
	params.Set("corp_code", corpCode)
	params.Set("bsns_year", year)
	params.Set("reprt_code", reportCode)

	reqURL := fmt.Sprintf("%s/fnlttSinglAcnt.json?%s", dartBaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return NormalizedFinancials{}, fmt.Errorf("create DART request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NormalizedFinancials{}, fmt.Errorf("DART request failed: %w", err)
	}
	defer resp.Body.Close()

	var dartResp DARTFinancialResponse
	if err := json.NewDecoder(resp.Body).Decode(&dartResp); err != nil {
		return NormalizedFinancials{}, fmt.Errorf("decode DART response: %w", err)
	}

	// Extract revenue, operating profit, net income from items
	raw := RawFinancials{}
	for _, item := range dartResp.List {
		switch item.AccountNm {
		case "매출액", "수익(매출액)":
			raw.Revenue = item.ThstrmAmt
		case "영업이익", "영업이익(손실)":
			raw.OperatingProfit = item.ThstrmAmt
		case "당기순이익", "당기순이익(손실)":
			raw.NetIncome = item.ThstrmAmt
		}
	}

	c.logger.Info("DART: fetched financial statements",
		zap.String("corpCode", corpCode),
		zap.String("year", year),
	)

	return NormalizeFinancialStatements(raw), nil
}
```

- [ ] 재무 서비스 구현 (`backend/internal/service/financials_service.go`)

```go
// backend/internal/service/financials_service.go
package service

import (
	"context"
	"strconv"
	"time"

	"your-module/internal/infra/dart"

	"go.uber.org/zap"
)

// FinancialsService handles financial data retrieval business logic.
type FinancialsService struct {
	dartClient *dart.Client
	logger     *zap.Logger
}

// NewFinancialsService creates a new FinancialsService.
func NewFinancialsService(dartClient *dart.Client, logger *zap.Logger) *FinancialsService {
	return &FinancialsService{
		dartClient: dartClient,
		logger:     logger,
	}
}

// GetFinancials fetches financial data for a symbol.
func (s *FinancialsService) GetFinancials(ctx context.Context, symbol string) (dart.NormalizedFinancials, error) {
	// TODO: symbol -> corpCode 매핑 (매핑 테이블 구현 필요)
	corpCode := symbol
	year := strconv.Itoa(time.Now().Year() - 1)

	s.logger.Info("fetching financials",
		zap.String("symbol", symbol),
		zap.String("year", year),
	)

	return s.dartClient.FetchFinancialStatements(ctx, corpCode, year, "11011")
}
```

- [ ] 재무제표 핸들러 구현 (`backend/internal/handler/financials_handler.go`) — `GET /api/v1/financials/:symbol`, `GET /api/v1/financials/:symbol/sector`

```go
// backend/internal/handler/financials_handler.go
package handler

import (
	"net/http"

	"your-module/internal/service"

	"github.com/gin-gonic/gin"
)

// FinancialsHandler handles financial data HTTP requests.
type FinancialsHandler struct {
	financialsSvc *service.FinancialsService
}

// NewFinancialsHandler creates a new FinancialsHandler.
func NewFinancialsHandler(financialsSvc *service.FinancialsService) *FinancialsHandler {
	return &FinancialsHandler{financialsSvc: financialsSvc}
}

// GetFinancials godoc
// @Summary     재무제표 조회
// @Description DART API를 통해 종목의 재무제표 데이터를 반환한다
// @Tags        financials
// @Param       symbol path string true "종목코드 (예: 005930)"
// @Success     200 {object} map[string]interface{}
// @Failure     502 {object} map[string]interface{}
// @Router      /api/v1/financials/{symbol} [get]
func (h *FinancialsHandler) GetFinancials(c *gin.Context) {
	symbol := c.Param("symbol")

	data, err := h.financialsSvc.GetFinancials(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// GetSectorCompare godoc
// @Summary     섹터 비교
// @Description 동일 섹터 내 종목들의 재무/기술 지표 비교 데이터를 반환한다
// @Tags        financials
// @Param       symbol path string true "종목코드"
// @Success     200 {object} map[string]interface{}
// @Router      /api/v1/financials/{symbol}/sector [get]
func (h *FinancialsHandler) GetSectorCompare(c *gin.Context) {
	// TODO: KRX 섹터 분류에서 동일 섹터 종목 조회 후
	//       각 종목의 PER, ROE, RSI, 등락률을 비교
	c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
}
```

- [ ] 관심종목 DB 마이그레이션 (`backend/db/migrations/003_watchlist.sql`)

```sql
-- backend/db/migrations/003_watchlist.sql
CREATE TABLE IF NOT EXISTS watchlist (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    symbol     VARCHAR(20) NOT NULL,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, symbol)
);

CREATE INDEX idx_watchlist_user_id ON watchlist(user_id);
```

- [ ] sqlc 쿼리 정의 (`backend/db/queries/watchlist.sql`)

```sql
-- backend/db/queries/watchlist.sql

-- name: GetWatchlistByUser :many
SELECT id, user_id, symbol, name, created_at
FROM watchlist
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: AddToWatchlist :one
INSERT INTO watchlist (user_id, symbol, name)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, symbol) DO NOTHING
RETURNING id, user_id, symbol, name, created_at;

-- name: RemoveFromWatchlist :exec
DELETE FROM watchlist
WHERE user_id = $1 AND symbol = $2;

-- name: IsInWatchlist :one
SELECT EXISTS(
    SELECT 1 FROM watchlist WHERE user_id = $1 AND symbol = $2
) AS is_in_watchlist;
```

- [ ] 관심종목 리포지토리 구현 (`backend/internal/repository/watchlist_repo.go`)

```go
// backend/internal/repository/watchlist_repo.go
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WatchlistItem represents a watchlist entry.
type WatchlistItem struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Symbol    string    `json:"symbol"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// WatchlistRepo handles watchlist DB operations.
type WatchlistRepo struct {
	pool *pgxpool.Pool
}

// NewWatchlistRepo creates a new WatchlistRepo.
func NewWatchlistRepo(pool *pgxpool.Pool) *WatchlistRepo {
	return &WatchlistRepo{pool: pool}
}

// GetByUser returns all watchlist items for a user.
func (r *WatchlistRepo) GetByUser(ctx context.Context, userID int64) ([]WatchlistItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, symbol, name, created_at
		 FROM watchlist WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []WatchlistItem
	for rows.Next() {
		var item WatchlistItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.Symbol, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// Add inserts a new watchlist item (idempotent via ON CONFLICT).
func (r *WatchlistRepo) Add(ctx context.Context, userID int64, symbol, name string) (*WatchlistItem, error) {
	var item WatchlistItem
	err := r.pool.QueryRow(ctx,
		`INSERT INTO watchlist (user_id, symbol, name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, symbol) DO NOTHING
		 RETURNING id, user_id, symbol, name, created_at`,
		userID, symbol, name).
		Scan(&item.ID, &item.UserID, &item.Symbol, &item.Name, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Remove deletes a watchlist item.
func (r *WatchlistRepo) Remove(ctx context.Context, userID int64, symbol string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM watchlist WHERE user_id = $1 AND symbol = $2`, userID, symbol)
	return err
}
```

- [ ] 관심종목 핸들러 구현 (`backend/internal/handler/watchlist_handler.go`) — CRUD `/api/v1/watchlist`

```go
// backend/internal/handler/watchlist_handler.go
package handler

import (
	"net/http"

	"your-module/internal/repository"

	"github.com/gin-gonic/gin"
)

// WatchlistHandler handles watchlist HTTP requests.
type WatchlistHandler struct {
	repo *repository.WatchlistRepo
}

// NewWatchlistHandler creates a new WatchlistHandler.
func NewWatchlistHandler(repo *repository.WatchlistRepo) *WatchlistHandler {
	return &WatchlistHandler{repo: repo}
}

// GetWatchlist godoc
// @Summary     관심종목 목록 조회
// @Tags        watchlist
// @Success     200 {object} map[string]interface{}
// @Router      /api/v1/watchlist [get]
func (h *WatchlistHandler) GetWatchlist(c *gin.Context) {
	// TODO: 인증 미들웨어에서 userID 추출
	userID := int64(1)

	items, err := h.repo.GetByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

type addWatchlistRequest struct {
	Symbol string `json:"symbol" binding:"required"`
	Name   string `json:"name" binding:"required"`
}

// AddToWatchlist godoc
// @Summary     관심종목 추가
// @Tags        watchlist
// @Param       body body addWatchlistRequest true "종목 정보"
// @Success     201 {object} map[string]interface{}
// @Router      /api/v1/watchlist [post]
func (h *WatchlistHandler) AddToWatchlist(c *gin.Context) {
	userID := int64(1) // TODO: 인증 미들웨어에서 추출

	var req addWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.repo.Add(c.Request.Context(), userID, req.Symbol, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": item})
}

// RemoveFromWatchlist godoc
// @Summary     관심종목 삭제
// @Tags        watchlist
// @Param       symbol path string true "종목코드"
// @Success     204
// @Router      /api/v1/watchlist/{symbol} [delete]
func (h *WatchlistHandler) RemoveFromWatchlist(c *gin.Context) {
	userID := int64(1) // TODO: 인증 미들웨어에서 추출
	symbol := c.Param("symbol")

	if err := h.repo.Remove(c.Request.Context(), userID, symbol); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
```

- [ ] Go 라우터 등록 (참조용 — 기존 `backend/cmd/server/main.go`에 추가)

```go
// backend/cmd/server/main.go (라우터 등록 부분 발췌)

// --- Chart/Financial/Watchlist Routes ---
v1 := r.Group("/api/v1")
{
    // Candle
    candleHandler := handler.NewCandleHandler(candleSvc)
    v1.GET("/candles/:symbol", candleHandler.GetCandles)

    // Financials
    financialsHandler := handler.NewFinancialsHandler(financialsSvc)
    v1.GET("/financials/:symbol", financialsHandler.GetFinancials)
    v1.GET("/financials/:symbol/sector", financialsHandler.GetSectorCompare)

    // Watchlist
    watchlistHandler := handler.NewWatchlistHandler(watchlistRepo)
    v1.GET("/watchlist", watchlistHandler.GetWatchlist)
    v1.POST("/watchlist", watchlistHandler.AddToWatchlist)
    v1.DELETE("/watchlist/:symbol", watchlistHandler.RemoveFromWatchlist)
}
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/infra/dart/... -v
go test ./internal/handler/... -v
```

- [ ] 커밋

```bash
git add backend/internal/infra/dart/ backend/internal/service/financials_service.go \
  backend/internal/handler/financials_handler.go backend/internal/handler/watchlist_handler.go \
  backend/internal/repository/watchlist_repo.go backend/db/
git commit -m "feat: Go DART client, financials handler, watchlist CRUD with sqlc repository"
```

---

## Task 8: 검색->차트 네비게이션 통합 + E2E 확인

Search 페이지의 "Chart" 버튼 클릭 시 차트 페이지로 이동하면서 검색 결과가 사이드바에 로드되고, 선택한 종목의 차트가 표시되는 전체 흐름을 통합 테스트한다. Cross-feature communication은 `entities/stock/` store를 통한다. **프론트엔드는 Go 백엔드 `/api/v1/` 엔드포인트를 호출한다.**

**Files:**
- Modify: `src/features/search/ui/ResultsTable.tsx` (차트 네비게이션 로직 확인)
- Modify: `src/app/(pages)/chart/page.tsx` (URL 파라미터로 초기 상태 설정)
- Create: `src/features/chart/__tests__/navigation.test.tsx`
- Create: `src/shared/ui/AppNavigation.tsx` (전역 네비게이션 바)

### Steps

- [ ] 테스트 먼저 작성 (`src/features/chart/__tests__/navigation.test.tsx`)

```typescript
// src/features/chart/__tests__/navigation.test.tsx
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "../model/chart.store";

describe("Search -> Chart Navigation", () => {
  beforeEach(() => {
    useStockListStore.setState({
      searchResults: [],
      selectedSymbol: null,
      watchlist: [],
      recentStocks: [],
    });
    useChartStore.setState(useChartStore.getInitialState());
  });

  it("stock list store receives search results when navigating", () => {
    const mockResults = [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      { symbol: "247540", name: "ecoprobm", matchedValue: 15200000 },
    ];

    // 검색 결과 페이지에서 Chart 클릭 시 entity store에 데이터 설정
    useStockListStore.getState().setSearchResults(mockResults);
    useStockListStore.getState().setSelectedSymbol("005930");

    // 차트 페이지에서 entity store 읽기
    const state = useStockListStore.getState();
    expect(state.searchResults).toHaveLength(2);
    expect(state.selectedSymbol).toBe("005930");
  });

  it("chart store updates currentStock from entity store", () => {
    useStockListStore.getState().setSelectedSymbol("005930");

    // 차트 페이지 마운트 시 entity store에서 현재 종목 설정
    useChartStore.getState().setCurrentStock({
      symbol: "005930",
      name: "Samsung Electronics",
      price: 0,
      change: 0,
      changePct: 0,
    });

    expect(useChartStore.getState().currentStock?.symbol).toBe("005930");
  });

  it("recent stocks are updated on navigation", () => {
    const item = { symbol: "005930", name: "Samsung", matchedValue: 0 };
    useStockListStore.getState().addToRecent(item);

    expect(useStockListStore.getState().recentStocks).toContainEqual(item);
  });

  it("recent stocks are capped at 30 items", () => {
    for (let i = 0; i < 35; i++) {
      useStockListStore.getState().addToRecent({
        symbol: String(i).padStart(6, "0"),
        name: `Stock ${i}`,
        matchedValue: 0,
      });
    }

    expect(useStockListStore.getState().recentStocks).toHaveLength(30);
  });
});
```

- [ ] 차트 페이지에서 URL 파라미터 처리 (`src/app/(pages)/chart/page.tsx`) — reads from entities/stock store, not from other features

```typescript
// src/app/(pages)/chart/page.tsx
"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";
import { ChartPageLayout } from "@/widgets/main-chart";

export default function ChartPage() {
  const searchParams = useSearchParams();
  const symbol = searchParams.get("symbol");
  const { selectedSymbol, searchResults } = useStockListStore();
  const { setCurrentStock } = useChartStore();

  // URL 파라미터 또는 entity store에서 종목 설정
  useEffect(() => {
    const targetSymbol = symbol ?? selectedSymbol;
    if (!targetSymbol) return;

    const stockInfo = searchResults.find((r) => r.symbol === targetSymbol);

    setCurrentStock({
      symbol: targetSymbol,
      name: stockInfo?.name ?? targetSymbol,
      price: 0,
      change: 0,
      changePct: 0,
    });
  }, [symbol, selectedSymbol, searchResults, setCurrentStock]);

  return <ChartPageLayout />;
}
```

- [ ] 전역 네비게이션 바 구현 (`src/shared/ui/AppNavigation.tsx`) — Search, Chart, Pipeline, Cases 페이지 간 이동

```typescript
// src/shared/ui/AppNavigation.tsx
"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const NAV_ITEMS = [
  { href: "/search", label: "Search" },
  { href: "/chart", label: "Chart" },
  { href: "/pipeline", label: "Pipeline" },
  { href: "/cases", label: "Cases" },
];

export function AppNavigation() {
  const pathname = usePathname();

  return (
    <nav className="flex items-center gap-6 px-6 py-3 bg-nexus-surface border-b border-nexus-border">
      <Link href="/" className="text-lg font-bold text-nexus-accent">
        NEXUS
      </Link>
      <div className="flex gap-1">
        {NAV_ITEMS.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
              pathname.startsWith(item.href)
                ? "bg-nexus-accent/10 text-nexus-accent"
                : "text-nexus-text-secondary hover:text-nexus-text-primary"
            }`}
          >
            {item.label}
          </Link>
        ))}
      </div>
    </nav>
  );
}
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear
npx jest src/features/chart/__tests__/navigation.test.tsx
```

- [ ] 커밋

```bash
git add src/app/\(pages\)/chart/ src/features/ src/shared/ui/ src/widgets/
git commit -m "feat: search-to-chart navigation with entity store and shared navigation widget"
```
