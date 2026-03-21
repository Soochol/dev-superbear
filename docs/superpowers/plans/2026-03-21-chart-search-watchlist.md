# 차트 종목 검색 팝업 + 관심종목 DB 연동 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 차트 페이지의 종목 검색을 센터 팝업 모달로 전환하고, 관심종목을 DB에 영속화한다.

**Architecture:** 기존 `StockListSidebar` 위젯을 제거하고 `StockSearchModal` 위젯으로 대체. 모달은 텍스트 사이드메뉴(검색/관심/최근) + 컨텐츠 영역 구조. 관심종목은 기존 백엔드 `WatchlistHandler` API와 연동하여 DB에 저장.

**Tech Stack:** Next.js 16, React 19, TypeScript, Zustand 5, Tailwind CSS 4, Go/Gin, PostgreSQL

**Spec:** `docs/superpowers/specs/2026-03-21-chart-enhancement-design.md` (서브시스템 1, 2)

---

## File Structure

### 새로 생성
| 파일 | 책임 |
|------|------|
| `src/widgets/stock-search-modal/index.ts` | barrel export |
| `src/widgets/stock-search-modal/model/search-modal.store.ts` | 모달 열기/닫기 + 활성 탭 상태 |
| `src/widgets/stock-search-modal/ui/StockSearchModal.tsx` | 모달 컨테이너 (backdrop + 센터 + 키보드) |
| `src/widgets/stock-search-modal/ui/SearchSideNav.tsx` | 좌측 텍스트 사이드메뉴 |
| `src/widgets/stock-search-modal/ui/SearchContent.tsx` | 우측 컨텐츠 (검색입력 + 리스트) |
| `src/widgets/stock-search-modal/ui/SearchStockItem.tsx` | 종목 아이템 행 (이름, 코드, 가격, star) |
| `src/widgets/stock-search-modal/__tests__/StockSearchModal.test.tsx` | 모달 통합 테스트 |
| `src/features/watchlist/api/watchlist-api.ts` | Watchlist REST 클라이언트 |
| `src/features/watchlist/api/__tests__/watchlist-api.test.ts` | API 클라이언트 테스트 |
| `backend/internal/handler/stock_search_handler.go` | 종목명/코드 검색 핸들러 |
| `backend/internal/handler/stock_search_handler_test.go` | 핸들러 테스트 |
| `backend/internal/repository/stock_repo.go` | stocks 테이블 쿼리 |

### 수정
| 파일 | 변경 내용 |
|------|----------|
| `src/widgets/main-chart/ui/ChartPageLayout.tsx` | StockListSidebar 제거, StockSearchModal 추가 |
| `src/widgets/main-chart/ui/ChartTopbar.tsx` | 종목 영역 클릭 → 모달 열기 트리거 |
| `src/widgets/main-chart/__tests__/ChartPageLayout.test.tsx` | 사이드바 mock 제거, 모달 mock 추가 |
| `src/entities/stock/model/stock-list.store.ts` | loadWatchlist, API 연동 액션 추가 |
| `src/features/watchlist/index.ts` | API export 추가 |
| `backend/cmd/api/main.go` | watchlist + stock search 라우트 등록 |

### 삭제
| 파일 | 이유 |
|------|------|
| `src/widgets/stock-list-sidebar/` (전체) | 기능이 모달로 이전 |

---

## Task 1: 검색 모달 Store

**Files:**
- Create: `src/widgets/stock-search-modal/model/search-modal.store.ts`
- Test: `src/widgets/stock-search-modal/__tests__/search-modal.store.test.ts`

- [ ] **Step 1: Write the failing test**

```typescript
/** @jest-environment jsdom */
import { useSearchModalStore } from "../model/search-modal.store";

describe("search-modal.store", () => {
  beforeEach(() => {
    useSearchModalStore.setState(useSearchModalStore.getInitialState());
  });

  it("starts closed with search tab", () => {
    const state = useSearchModalStore.getState();
    expect(state.isOpen).toBe(false);
    expect(state.activeTab).toBe("search");
  });

  it("openModal sets isOpen true", () => {
    useSearchModalStore.getState().openModal();
    expect(useSearchModalStore.getState().isOpen).toBe(true);
  });

  it("closeModal sets isOpen false and resets tab", () => {
    useSearchModalStore.getState().openModal();
    useSearchModalStore.getState().setActiveTab("watchlist");
    useSearchModalStore.getState().closeModal();
    const state = useSearchModalStore.getState();
    expect(state.isOpen).toBe(false);
    expect(state.activeTab).toBe("search");
  });

  it("setActiveTab changes tab", () => {
    useSearchModalStore.getState().setActiveTab("recent");
    expect(useSearchModalStore.getState().activeTab).toBe("recent");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx jest src/widgets/stock-search-modal/__tests__/search-modal.store.test.ts --no-coverage`
Expected: FAIL — module not found

- [ ] **Step 3: Write the store**

```typescript
// src/widgets/stock-search-modal/model/search-modal.store.ts
import { create } from "zustand";

export type SearchModalTab = "search" | "watchlist" | "recent";

interface SearchModalState {
  isOpen: boolean;
  activeTab: SearchModalTab;
  openModal: () => void;
  closeModal: () => void;
  setActiveTab: (tab: SearchModalTab) => void;
}

export const useSearchModalStore = create<SearchModalState>()((set) => ({
  isOpen: false,
  activeTab: "search" as SearchModalTab,
  openModal: () => set({ isOpen: true }),
  closeModal: () => set({ isOpen: false, activeTab: "search" }),
  setActiveTab: (tab) => set({ activeTab: tab }),
}));
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx jest src/widgets/stock-search-modal/__tests__/search-modal.store.test.ts --no-coverage`
Expected: PASS — 4 tests

- [ ] **Step 5: Commit**

```bash
git add src/widgets/stock-search-modal/model/search-modal.store.ts src/widgets/stock-search-modal/__tests__/search-modal.store.test.ts
git commit -m "feat(chart): add search modal zustand store"
```

---

## Task 2: SearchStockItem 컴포넌트

**Files:**
- Create: `src/widgets/stock-search-modal/ui/SearchStockItem.tsx`
- Test: `src/widgets/stock-search-modal/__tests__/SearchStockItem.test.tsx`

- [ ] **Step 1: Write the failing test**

```typescript
/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { SearchStockItem } from "../ui/SearchStockItem";

const mockItem = {
  symbol: "005930",
  name: "삼성전자",
  matchedValue: "005930" as string | number,
  close: 71200,
  changePct: 2.3,
};

describe("SearchStockItem", () => {
  it("renders stock name, symbol, price, change", () => {
    render(
      <SearchStockItem item={mockItem} isInWatchlist={false} onSelect={jest.fn()} onToggleWatchlist={jest.fn()} />,
    );
    expect(screen.getByText("삼성전자")).toBeInTheDocument();
    expect(screen.getByText("005930")).toBeInTheDocument();
    expect(screen.getByText("71,200")).toBeInTheDocument();
    expect(screen.getByText("+2.30%")).toBeInTheDocument();
  });

  it("calls onSelect when clicked", () => {
    const onSelect = jest.fn();
    render(
      <SearchStockItem item={mockItem} isInWatchlist={false} onSelect={onSelect} onToggleWatchlist={jest.fn()} />,
    );
    fireEvent.click(screen.getByTestId("search-stock-item-005930"));
    expect(onSelect).toHaveBeenCalledWith(mockItem);
  });

  it("shows filled star when in watchlist", () => {
    render(
      <SearchStockItem item={mockItem} isInWatchlist={true} onSelect={jest.fn()} onToggleWatchlist={jest.fn()} />,
    );
    expect(screen.getByLabelText("Remove from watchlist")).toBeInTheDocument();
  });

  it("star click calls onToggleWatchlist without triggering onSelect", () => {
    const onSelect = jest.fn();
    const onToggle = jest.fn();
    render(
      <SearchStockItem item={mockItem} isInWatchlist={false} onSelect={onSelect} onToggleWatchlist={onToggle} />,
    );
    fireEvent.click(screen.getByLabelText("Add to watchlist"));
    expect(onToggle).toHaveBeenCalledWith(mockItem);
    expect(onSelect).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx jest src/widgets/stock-search-modal/__tests__/SearchStockItem.test.tsx --no-coverage`
Expected: FAIL

- [ ] **Step 3: Write the component**

```tsx
// src/widgets/stock-search-modal/ui/SearchStockItem.tsx
"use client";

import type { SearchResult } from "@/entities/search-result";

interface Props {
  item: SearchResult;
  isInWatchlist: boolean;
  onSelect: (item: SearchResult) => void;
  onToggleWatchlist: (item: SearchResult) => void;
}

export function SearchStockItem({ item, isInWatchlist, onSelect, onToggleWatchlist }: Props) {
  return (
    <div
      data-testid={`search-stock-item-${item.symbol}`}
      onClick={() => onSelect(item)}
      className="flex items-center justify-between px-3 py-2.5 rounded-lg cursor-pointer transition-colors hover:bg-nexus-border/30"
    >
      <div className="flex items-center gap-3 min-w-0">
        <span className="text-sm font-medium text-nexus-text-primary truncate">{item.name}</span>
        <span className="text-xs text-nexus-text-muted font-mono flex-shrink-0">{item.symbol}</span>
      </div>
      <div className="flex items-center gap-3 flex-shrink-0">
        {item.close != null && (
          <div className="text-right">
            <span className="text-sm text-nexus-text-primary font-mono">{item.close.toLocaleString()}</span>
            {item.changePct != null && (
              <span
                className={`text-xs font-mono ml-2 ${
                  item.changePct >= 0 ? "text-nexus-success" : "text-nexus-failure"
                }`}
              >
                {item.changePct >= 0 ? "+" : ""}{item.changePct.toFixed(2)}%
              </span>
            )}
          </div>
        )}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggleWatchlist(item);
          }}
          aria-label={isInWatchlist ? "Remove from watchlist" : "Add to watchlist"}
          className={`text-base transition-colors ${
            isInWatchlist ? "text-nexus-warning" : "text-nexus-text-muted hover:text-nexus-warning"
          }`}
        >
          {isInWatchlist ? "\u2605" : "\u2606"}
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx jest src/widgets/stock-search-modal/__tests__/SearchStockItem.test.tsx --no-coverage`
Expected: PASS — 4 tests

- [ ] **Step 5: Commit**

```bash
git add src/widgets/stock-search-modal/ui/SearchStockItem.tsx src/widgets/stock-search-modal/__tests__/SearchStockItem.test.tsx
git commit -m "feat(chart): add SearchStockItem component for modal"
```

---

## Task 3: SearchSideNav 컴포넌트

**Files:**
- Create: `src/widgets/stock-search-modal/ui/SearchSideNav.tsx`
- Test: `src/widgets/stock-search-modal/__tests__/SearchSideNav.test.tsx`

- [ ] **Step 1: Write the failing test**

```typescript
/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { SearchSideNav } from "../ui/SearchSideNav";
import type { SearchModalTab } from "../model/search-modal.store";

describe("SearchSideNav", () => {
  const onTabChange = jest.fn();

  it("renders 3 navigation items", () => {
    render(<SearchSideNav activeTab="search" onTabChange={onTabChange} />);
    expect(screen.getByText("종목 검색")).toBeInTheDocument();
    expect(screen.getByText("관심 종목")).toBeInTheDocument();
    expect(screen.getByText("최근 본 종목")).toBeInTheDocument();
  });

  it("highlights active tab", () => {
    render(<SearchSideNav activeTab="watchlist" onTabChange={onTabChange} />);
    const watchlistItem = screen.getByText("관심 종목").closest("[data-tab]");
    expect(watchlistItem?.getAttribute("data-active")).toBe("true");
  });

  it("calls onTabChange on click", () => {
    render(<SearchSideNav activeTab="search" onTabChange={onTabChange} />);
    fireEvent.click(screen.getByText("관심 종목"));
    expect(onTabChange).toHaveBeenCalledWith("watchlist");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Write the component**

```tsx
// src/widgets/stock-search-modal/ui/SearchSideNav.tsx
"use client";

import type { SearchModalTab } from "../model/search-modal.store";

interface Props {
  activeTab: SearchModalTab;
  onTabChange: (tab: SearchModalTab) => void;
}

const NAV_ITEMS: { tab: SearchModalTab; icon: string; label: string }[] = [
  { tab: "search", icon: "🔍", label: "종목 검색" },
  { tab: "watchlist", icon: "⭐", label: "관심 종목" },
  { tab: "recent", icon: "🕐", label: "최근 본 종목" },
];

export function SearchSideNav({ activeTab, onTabChange }: Props) {
  return (
    <nav className="w-[140px] bg-nexus-bg border-r border-nexus-border flex flex-col py-4 flex-shrink-0">
      {NAV_ITEMS.map(({ tab, icon, label }) => (
        <button
          key={tab}
          data-tab={tab}
          data-active={activeTab === tab}
          onClick={() => onTabChange(tab)}
          className={`flex items-center gap-2 px-4 py-2 text-xs transition-colors text-left ${
            activeTab === tab
              ? "text-nexus-accent bg-nexus-accent/10 border-r-2 border-nexus-accent font-semibold"
              : "text-nexus-text-secondary hover:text-nexus-text-primary"
          }`}
        >
          <span>{icon}</span>
          {label}
        </button>
      ))}
    </nav>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add src/widgets/stock-search-modal/ui/SearchSideNav.tsx src/widgets/stock-search-modal/__tests__/SearchSideNav.test.tsx
git commit -m "feat(chart): add SearchSideNav component"
```

---

## Task 4: SearchContent 컴포넌트

**Files:**
- Create: `src/widgets/stock-search-modal/ui/SearchContent.tsx`
- Test: `src/widgets/stock-search-modal/__tests__/SearchContent.test.tsx`

- [ ] **Step 1: Write the failing test**

```typescript
/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { SearchContent } from "../ui/SearchContent";
import type { SearchResult } from "@/entities/search-result";

const mockItems: SearchResult[] = [
  { symbol: "005930", name: "삼성전자", matchedValue: "005930", close: 71200, changePct: 2.3 },
  { symbol: "000660", name: "SK하이닉스", matchedValue: "000660", close: 178500, changePct: -1.1 },
];

describe("SearchContent", () => {
  it("renders title and search input", () => {
    render(
      <SearchContent
        title="종목 검색"
        items={mockItems}
        watchlistSymbols={new Set()}
        onSelect={jest.fn()}
        onToggleWatchlist={jest.fn()}
      />,
    );
    expect(screen.getByText("종목 검색")).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/종목명 또는 코드/)).toBeInTheDocument();
  });

  it("filters items by search query", () => {
    render(
      <SearchContent
        title="종목 검색"
        items={mockItems}
        watchlistSymbols={new Set()}
        onSelect={jest.fn()}
        onToggleWatchlist={jest.fn()}
      />,
    );
    fireEvent.change(screen.getByPlaceholderText(/종목명 또는 코드/), { target: { value: "삼성" } });
    expect(screen.getByText("삼성전자")).toBeInTheDocument();
    expect(screen.queryByText("SK하이닉스")).not.toBeInTheDocument();
  });

  it("shows empty message when no items match", () => {
    render(
      <SearchContent
        title="종목 검색"
        items={[]}
        watchlistSymbols={new Set()}
        onSelect={jest.fn()}
        onToggleWatchlist={jest.fn()}
      />,
    );
    expect(screen.getByText(/항목이 없습니다/)).toBeInTheDocument();
  });

  it("shows keyboard hints at bottom", () => {
    render(
      <SearchContent
        title="종목 검색"
        items={mockItems}
        watchlistSymbols={new Set()}
        onSelect={jest.fn()}
        onToggleWatchlist={jest.fn()}
      />,
    );
    expect(screen.getByText(/Esc/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Write the component**

```tsx
// src/widgets/stock-search-modal/ui/SearchContent.tsx
"use client";

import { useState } from "react";
import type { SearchResult } from "@/entities/search-result";
import { SearchStockItem } from "./SearchStockItem";

interface Props {
  title: string;
  items: SearchResult[];
  watchlistSymbols: Set<string>;
  onSelect: (item: SearchResult) => void;
  onToggleWatchlist: (item: SearchResult) => void;
}

export function SearchContent({ title, items, watchlistSymbols, onSelect, onToggleWatchlist }: Props) {
  const [filter, setFilter] = useState("");

  const filtered = items.filter(
    (item) =>
      !filter ||
      item.symbol.includes(filter.toUpperCase()) ||
      item.name.toLowerCase().includes(filter.toLowerCase()),
  );

  return (
    <div className="flex-1 flex flex-col min-w-0">
      <div className="flex items-center justify-between px-4 py-3 border-b border-nexus-border">
        <h3 className="text-sm font-semibold text-nexus-text-primary">{title}</h3>
      </div>
      <div className="px-4 py-3 border-b border-nexus-border">
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="종목명 또는 코드를 검색하세요..."
          className="w-full bg-nexus-border/50 rounded-lg px-3 py-2 text-sm text-nexus-text-primary placeholder:text-nexus-text-muted outline-none focus:ring-1 focus:ring-nexus-accent"
          autoFocus
        />
      </div>
      <div className="flex-1 overflow-y-auto px-2 py-1">
        {filtered.length > 0 ? (
          filtered.map((item) => (
            <SearchStockItem
              key={item.symbol}
              item={item}
              isInWatchlist={watchlistSymbols.has(item.symbol)}
              onSelect={onSelect}
              onToggleWatchlist={onToggleWatchlist}
            />
          ))
        ) : (
          <div className="flex items-center justify-center h-full text-nexus-text-muted text-sm">
            항목이 없습니다
          </div>
        )}
      </div>
      <div className="px-4 py-2 border-t border-nexus-border flex justify-between">
        <span className="text-[10px] text-nexus-text-muted">↑↓ 이동 · Enter 선택 · Esc 닫기</span>
        <span className="text-[10px] text-nexus-text-muted">{filtered.length} 종목</span>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add src/widgets/stock-search-modal/ui/SearchContent.tsx src/widgets/stock-search-modal/__tests__/SearchContent.test.tsx
git commit -m "feat(chart): add SearchContent component with filtering"
```

---

## Task 5: StockSearchModal 컴포넌트 (조합)

**Files:**
- Create: `src/widgets/stock-search-modal/ui/StockSearchModal.tsx`
- Create: `src/widgets/stock-search-modal/index.ts`
- Test: `src/widgets/stock-search-modal/__tests__/StockSearchModal.test.tsx`

- [ ] **Step 1: Write the failing test**

```typescript
/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { StockSearchModal } from "../ui/StockSearchModal";
import { useSearchModalStore } from "../model/search-modal.store";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";

beforeEach(() => {
  useSearchModalStore.setState({ isOpen: true, activeTab: "search" });
  useStockListStore.setState({
    ...useStockListStore.getInitialState(),
    searchResults: [
      { symbol: "005930", name: "삼성전자", matchedValue: "005930", close: 71200, changePct: 2.3 },
    ],
    watchlist: [{ symbol: "000660", name: "SK하이닉스", matchedValue: "000660" }],
    recentStocks: [{ symbol: "035420", name: "NAVER", matchedValue: "035420" }],
  });
  useChartStore.setState(useChartStore.getInitialState());
});

describe("StockSearchModal", () => {
  it("renders when isOpen is true", () => {
    render(<StockSearchModal />);
    expect(screen.getByText("종목 검색")).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    useSearchModalStore.setState({ isOpen: false });
    render(<StockSearchModal />);
    expect(screen.queryByText("종목 검색")).not.toBeInTheDocument();
  });

  it("closes on Esc key", () => {
    render(<StockSearchModal />);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(useSearchModalStore.getState().isOpen).toBe(false);
  });

  it("closes on backdrop click", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByTestId("search-modal-backdrop"));
    expect(useSearchModalStore.getState().isOpen).toBe(false);
  });

  it("selects stock → updates chart store → closes modal", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByTestId("search-stock-item-005930"));
    expect(useChartStore.getState().currentStock?.symbol).toBe("005930");
    expect(useSearchModalStore.getState().isOpen).toBe(false);
  });

  it("switches to watchlist tab and shows watchlist items", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByText("관심 종목"));
    expect(screen.getByText("SK하이닉스")).toBeInTheDocument();
  });

  it("switches to recent tab and shows recent items", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByText("최근 본 종목"));
    expect(screen.getByText("NAVER")).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Write the modal component**

```tsx
// src/widgets/stock-search-modal/ui/StockSearchModal.tsx
"use client";

import { useEffect, useCallback, useMemo } from "react";
import { useSearchModalStore } from "../model/search-modal.store";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";
import { SearchSideNav } from "./SearchSideNav";
import { SearchContent } from "./SearchContent";
import type { SearchResult } from "@/entities/search-result";

const TAB_TITLES = {
  search: "종목 검색",
  watchlist: "관심 종목",
  recent: "최근 본 종목",
} as const;

export function StockSearchModal() {
  const { isOpen, activeTab, closeModal, setActiveTab } = useSearchModalStore();
  const { searchResults, watchlist, recentStocks, addToRecent, isInWatchlist, addToWatchlist, removeFromWatchlist } =
    useStockListStore();
  const setCurrentStock = useChartStore((s) => s.setCurrentStock);

  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") closeModal();
    },
    [closeModal],
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [isOpen, handleEscape]);

  const handleSelect = useCallback(
    (item: SearchResult) => {
      setCurrentStock({
        symbol: item.symbol,
        name: item.name,
        price: item.close ?? 0,
        change: item.change ?? 0,
        changePct: item.changePct ?? 0,
      });
      addToRecent(item);
      closeModal();
    },
    [setCurrentStock, addToRecent, closeModal],
  );

  const handleToggleWatchlist = useCallback(
    (item: SearchResult) => {
      if (isInWatchlist(item.symbol)) {
        removeFromWatchlist(item.symbol);
      } else {
        addToWatchlist(item);
      }
    },
    [isInWatchlist, addToWatchlist, removeFromWatchlist],
  );

  const watchlistSymbols = useMemo(
    () => new Set(watchlist.map((w) => w.symbol)),
    [watchlist],
  );

  const tabItems = { search: searchResults, watchlist, recent: recentStocks };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        data-testid="search-modal-backdrop"
        className="absolute inset-0 bg-black/60"
        onClick={closeModal}
      />
      <div className="relative w-[560px] max-h-[420px] bg-nexus-surface border border-nexus-border rounded-2xl shadow-2xl flex overflow-hidden">
        <SearchSideNav activeTab={activeTab} onTabChange={setActiveTab} />
        <SearchContent
          title={TAB_TITLES[activeTab]}
          items={tabItems[activeTab]}
          watchlistSymbols={watchlistSymbols}
          onSelect={handleSelect}
          onToggleWatchlist={handleToggleWatchlist}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Write barrel export**

```typescript
// src/widgets/stock-search-modal/index.ts
export { StockSearchModal } from "./ui/StockSearchModal";
export { useSearchModalStore } from "./model/search-modal.store";
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npx jest src/widgets/stock-search-modal/__tests__/StockSearchModal.test.tsx --no-coverage`
Expected: PASS — 7 tests

- [ ] **Step 6: Commit**

```bash
git add src/widgets/stock-search-modal/
git commit -m "feat(chart): add StockSearchModal widget with side nav and content"
```

---

## Task 6: ChartTopbar 검색 트리거 + ChartPageLayout 업데이트

**Files:**
- Modify: `src/widgets/main-chart/ui/ChartTopbar.tsx`
- Modify: `src/widgets/main-chart/ui/ChartPageLayout.tsx`
- Modify: `src/widgets/main-chart/__tests__/ChartPageLayout.test.tsx`

- [ ] **Step 1: Update ChartTopbar — 종목 영역 클릭으로 모달 열기**

`ChartTopbar.tsx`에서 좌측 종목 정보를 클릭 가능하게 변경:

```tsx
// src/widgets/main-chart/ui/ChartTopbar.tsx
"use client";

import { useChartStore } from "@/features/chart";
import { useSearchModalStore } from "@/widgets/stock-search-modal";
import type { Timeframe } from "@/features/chart";

const TIMEFRAMES: Timeframe[] = ["1m", "5m", "15m", "1H", "1D", "1W", "1M"];

export function ChartTopbar() {
  const { currentStock, timeframe, setTimeframe } = useChartStore();
  const openModal = useSearchModalStore((s) => s.openModal);

  return (
    <div className="flex items-center justify-between px-4 py-2 bg-nexus-surface border-b border-nexus-border">
      <button
        onClick={openModal}
        className="flex items-center gap-4 hover:bg-nexus-border/30 rounded-lg px-3 py-1 transition-colors"
      >
        {currentStock ? (
          <>
            <span className="font-mono text-nexus-text-secondary text-sm">{currentStock.symbol}</span>
            <span className="font-semibold">{currentStock.name}</span>
            <span className="font-mono text-lg">{currentStock.price.toLocaleString()}</span>
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
          <span className="text-nexus-text-muted flex items-center gap-2">
            <span>🔍</span> 종목을 검색하세요
          </span>
        )}
      </button>
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

- [ ] **Step 2: Update ChartPageLayout — 사이드바 제거, 모달 추가**

```tsx
// src/widgets/main-chart/ui/ChartPageLayout.tsx
"use client";

import { MainChart } from "@/features/chart";
import { ChartTopbar } from "./ChartTopbar";
import { StockSearchModal } from "@/widgets/stock-search-modal";
import { BottomInfoPanel } from "@/widgets/bottom-info-panel";

export function ChartPageLayout() {
  return (
    <div className="flex flex-col h-full">
      <ChartTopbar />
      <div className="flex-1 min-h-0">
        <MainChart />
      </div>
      <BottomInfoPanel />
      <StockSearchModal />
    </div>
  );
}
```

- [ ] **Step 3: Update ChartPageLayout test**

```tsx
// src/widgets/main-chart/__tests__/ChartPageLayout.test.tsx
/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";
import { ChartPageLayout } from "../ui/ChartPageLayout";

jest.mock("../ui/ChartTopbar", () => ({ ChartTopbar: () => <div data-testid="topbar" /> }));
jest.mock("@/widgets/stock-search-modal", () => ({
  StockSearchModal: () => <div data-testid="search-modal" />,
  useSearchModalStore: jest.fn(),
}));
jest.mock("@/widgets/bottom-info-panel", () => ({ BottomInfoPanel: () => <div data-testid="bottom-panel" /> }));
jest.mock("@/features/chart", () => ({
  MainChart: () => <div data-testid="main-chart" />,
}));

describe("ChartPageLayout", () => {
  it("renders MainChart, topbar, bottom panel, and search modal", () => {
    render(<ChartPageLayout />);
    expect(screen.getByTestId("main-chart")).toBeInTheDocument();
    expect(screen.getByTestId("topbar")).toBeInTheDocument();
    expect(screen.getByTestId("bottom-panel")).toBeInTheDocument();
    expect(screen.getByTestId("search-modal")).toBeInTheDocument();
  });

  it("does not render stock-list-sidebar", () => {
    render(<ChartPageLayout />);
    expect(screen.queryByTestId("sidebar")).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 4: Run tests**

Run: `npx jest src/widgets/main-chart/__tests__/ --no-coverage`
Expected: PASS

- [ ] **Step 5: Delete old StockListSidebar widget**

```bash
rm -rf src/widgets/stock-list-sidebar/
```

- [ ] **Step 6: Commit**

```bash
git add -A src/widgets/main-chart/ src/widgets/stock-list-sidebar/
git commit -m "feat(chart): replace sidebar with search modal, update layout"
```

---

## Task 7: 백엔드 종목 검색 API

**Files:**
- Create: `backend/internal/repository/stock_repo.go`
- Create: `backend/internal/handler/stock_search_handler.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Create stock repository**

```go
// backend/internal/repository/stock_repo.go
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StockSearchResult struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type StockRepository struct {
	pool *pgxpool.Pool
}

func NewStockRepository(pool *pgxpool.Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

func (r *StockRepository) Search(ctx context.Context, query string, limit int) ([]StockSearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx,
		`SELECT symbol, name FROM stocks
		 WHERE name ILIKE '%' || $1 || '%' OR symbol ILIKE '%' || $1 || '%'
		 ORDER BY
		   CASE WHEN symbol = UPPER($1) THEN 0
		        WHEN symbol LIKE UPPER($1) || '%' THEN 1
		        WHEN name LIKE $1 || '%' THEN 2
		        ELSE 3
		   END,
		   name
		 LIMIT $2`,
		query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StockSearchResult
	for rows.Next() {
		var r StockSearchResult
		if err := rows.Scan(&r.Symbol, &r.Name); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
```

- [ ] **Step 2: Create search handler**

```go
// backend/internal/handler/stock_search_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"your-module/internal/repository"
)

type StockSearchHandler struct {
	repo *repository.StockRepository
}

func NewStockSearchHandler(repo *repository.StockRepository) *StockSearchHandler {
	return &StockSearchHandler{repo: repo}
}

func (h *StockSearchHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}

	results, err := h.repo.Search(c.Request.Context(), query, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
```

Note: `your-module`은 실제 Go module path로 교체할 것. `go.mod`에서 확인.

- [ ] **Step 3: Register routes in main.go**

`backend/cmd/api/main.go`의 `registerRoutes` 함수에 추가:

```go
// Stock search
stockRepo := repository.NewStockRepository(pool)
stockSearchH := handler.NewStockSearchHandler(stockRepo)
rg.GET("/stocks/search", stockSearchH.Search)

// Watchlist (already implemented, just register)
watchlistRepo := repository.NewWatchlistRepository(pool)
watchlistH := handler.NewWatchlistHandler(watchlistRepo)
rg.GET("/watchlist", watchlistH.GetWatchlist)
rg.POST("/watchlist", watchlistH.AddToWatchlist)
rg.DELETE("/watchlist/:symbol", watchlistH.RemoveFromWatchlist)
```

- [ ] **Step 4: Verify build**

Run: `cd backend && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/stock_repo.go backend/internal/handler/stock_search_handler.go backend/cmd/api/main.go
git commit -m "feat(backend): add stock search endpoint and register watchlist routes"
```

---

## Task 8: Watchlist API 클라이언트 + Store 연동

**Files:**
- Create: `src/features/watchlist/api/watchlist-api.ts`
- Modify: `src/entities/stock/model/stock-list.store.ts`
- Modify: `src/features/watchlist/index.ts`
- Test: `src/features/watchlist/api/__tests__/watchlist-api.test.ts`

- [ ] **Step 1: Write the API client test**

```typescript
/** @jest-environment jsdom */
import { watchlistApi } from "../watchlist-api";

// Mock global fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

beforeEach(() => {
  mockFetch.mockReset();
});

describe("watchlistApi", () => {
  it("fetchWatchlist returns mapped items", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: [
          { id: 1, user_id: "u1", symbol: "005930", name: "삼성전자", created_at: "2026-03-21" },
        ],
      }),
    });

    const result = await watchlistApi.fetchWatchlist();
    expect(result).toEqual([
      { symbol: "005930", name: "삼성전자", matchedValue: "005930" },
    ]);
  });

  it("addItem sends POST with symbol and name", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ data: { id: 1, symbol: "005930", name: "삼성전자" } }),
    });

    await watchlistApi.addItem("005930", "삼성전자");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/watchlist"),
      expect.objectContaining({ method: "POST" }),
    );
  });

  it("removeItem sends DELETE", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, json: async () => ({}) });

    await watchlistApi.removeItem("005930");
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/watchlist/005930"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });
});
```

- [ ] **Step 2: Write the API client**

```typescript
// src/features/watchlist/api/watchlist-api.ts
import { apiGet, apiPost, apiDelete } from "@/shared/api/client";
import type { SearchResult } from "@/entities/search-result";

interface WatchlistItem {
  id: number;
  user_id: string;
  symbol: string;
  name: string;
  created_at: string;
}

function toSearchResult(item: WatchlistItem): SearchResult {
  return {
    symbol: item.symbol,
    name: item.name,
    matchedValue: item.symbol,
  };
}

export const watchlistApi = {
  async fetchWatchlist(): Promise<SearchResult[]> {
    const res = await apiGet<{ data: WatchlistItem[] }>("/api/v1/watchlist");
    return (res.data ?? []).map(toSearchResult);
  },

  async addItem(symbol: string, name: string): Promise<void> {
    await apiPost("/api/v1/watchlist", { symbol, name });
  },

  async removeItem(symbol: string): Promise<void> {
    await apiDelete(`/api/v1/watchlist/${symbol}`);
  },
};
```

- [ ] **Step 3: Update stock-list.store with API integration**

`src/entities/stock/model/stock-list.store.ts` — `loadWatchlist`, `addToWatchlist`, `removeFromWatchlist` 액션을 API 연동으로 변경:

```typescript
import { create } from "zustand";
import type { SearchResult } from "@/entities/search-result";
import { watchlistApi } from "@/features/watchlist/api/watchlist-api";
import { logger } from "@/shared/lib/logger";

interface StockListState {
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;

  selectedSymbol: string | null;
  setSelectedSymbol: (symbol: string | null) => void;

  watchlist: SearchResult[];
  watchlistLoaded: boolean;
  loadWatchlist: () => Promise<void>;
  addToWatchlist: (item: SearchResult) => Promise<void>;
  removeFromWatchlist: (symbol: string) => Promise<void>;
  isInWatchlist: (symbol: string) => boolean;

  recentStocks: SearchResult[];
  addToRecent: (item: SearchResult) => void;
}

export const useStockListStore = create<StockListState>()((set, get) => ({
  searchResults: [],
  setSearchResults: (results) => set({ searchResults: results }),

  selectedSymbol: null,
  setSelectedSymbol: (symbol) => set({ selectedSymbol: symbol }),

  watchlist: [],
  watchlistLoaded: false,
  loadWatchlist: async () => {
    if (get().watchlistLoaded) return;
    try {
      const items = await watchlistApi.fetchWatchlist();
      set({ watchlist: items, watchlistLoaded: true });
    } catch (err) {
      logger.error("Failed to load watchlist", { error: err });
    }
  },
  addToWatchlist: async (item) => {
    if (get().watchlist.some((w) => w.symbol === item.symbol)) return;
    try {
      await watchlistApi.addItem(item.symbol, item.name);
      set((state) => ({ watchlist: [...state.watchlist, item] }));
    } catch (err) {
      logger.error("Failed to add to watchlist", { error: err });
    }
  },
  removeFromWatchlist: async (symbol) => {
    try {
      await watchlistApi.removeItem(symbol);
      set((state) => ({
        watchlist: state.watchlist.filter((w) => w.symbol !== symbol),
      }));
    } catch (err) {
      logger.error("Failed to remove from watchlist", { error: err });
    }
  },
  isInWatchlist: (symbol) => get().watchlist.some((w) => w.symbol === symbol),

  recentStocks: [],
  addToRecent: (item) =>
    set((state) => {
      const filtered = state.recentStocks.filter((r) => r.symbol !== item.symbol);
      return { recentStocks: [item, ...filtered].slice(0, 30) };
    }),
}));
```

- [ ] **Step 4: Update watchlist feature index**

```typescript
// src/features/watchlist/index.ts
export { useStockListStore as useWatchlistStore } from "@/entities/stock";
export { watchlistApi } from "./api/watchlist-api";
```

- [ ] **Step 5: Run tests**

Run: `npx jest src/features/watchlist/ --no-coverage`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add src/features/watchlist/ src/entities/stock/model/stock-list.store.ts
git commit -m "feat(chart): add watchlist API client and DB persistence"
```

---

## Task 9: 모달에서 Watchlist 로드 트리거 + StockSearchModal 업데이트

**Files:**
- Modify: `src/widgets/stock-search-modal/ui/StockSearchModal.tsx`

- [ ] **Step 1: Add loadWatchlist call on modal open**

`StockSearchModal.tsx`에 `useEffect` 추가 — 모달이 열릴 때 `loadWatchlist()` 호출:

```typescript
// StockSearchModal.tsx 내부, 기존 useEffect 아래에 추가
const loadWatchlist = useStockListStore((s) => s.loadWatchlist);

useEffect(() => {
  if (isOpen) {
    loadWatchlist();
  }
}, [isOpen, loadWatchlist]);
```

- [ ] **Step 2: Run all modal tests**

Run: `npx jest src/widgets/stock-search-modal/ --no-coverage`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add src/widgets/stock-search-modal/ui/StockSearchModal.tsx
git commit -m "feat(chart): trigger watchlist load on modal open"
```

---

## Task 10: 전체 테스트 실행 + 정리

- [ ] **Step 1: Run all tests**

Run: `npx jest --no-coverage`
Expected: All tests PASS. 기존 `StockListSidebar` 테스트는 삭제됨.

- [ ] **Step 2: Check for broken imports**

Run: `npx tsc --noEmit`
Expected: No type errors

- [ ] **Step 3: Fix any issues found**

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "chore: cleanup after stock search modal migration"
```

---

## Verification Checklist

- [ ] 모달이 ChartTopbar 클릭으로 열리는가
- [ ] 모달에 좌측 사이드메뉴 (검색/관심/최근) 탭이 있는가
- [ ] 종목 클릭 → 모달 닫힘 → 차트 전환이 되는가
- [ ] Esc / backdrop 클릭으로 모달이 닫히는가
- [ ] 관심종목 추가/삭제가 DB에 영속화되는가
- [ ] 기존 StockListSidebar가 완전히 제거되었는가
- [ ] 차트가 전체 너비를 사용하는가
- [ ] 모든 테스트가 통과하는가
- [ ] TypeScript 타입 에러가 없는가
