# Search Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 자연어/DSL 2탭 모드 종목 검색 시스템을 구현한다 (NL 모드, DSL 모드, LIVE DSL 패널, 검색 프리셋 관리, Go 백엔드 API 4개 엔드포인트).
**Architecture:** FSD 아키텍처에 따라 프론트엔드 검색 기능을 `frontend/src/features/search/`에, 프리셋 엔티티를 `frontend/src/entities/search-preset/`에, 검색 결과 도메인 타입을 `frontend/src/entities/search-result/`에 배치한다. 백엔드는 Go Gin 서버로 `backend/internal/` 하위에 handler/service/repository/dsl 패키지로 구성한다. 검색 페이지는 상단에 NL/DSL 2탭 입력 영역, 하단에 항상 표시되는 LIVE DSL 패널로 구성한다. NL 모드에서는 LLM 에이전트(Google ADK)가 자연어를 DSL로 변환하고, DSL 모드에서는 사용자가 직접 작성한다. LIVE DSL 패널은 양 모드에서 동기화되며 Validate/Run 기능을 제공한다. 검색 결과는 테이블로 표시되고 "Chart" 버튼으로 차트 페이지와 연동한다. Go 핸들러는 thin controller 패턴으로 service 레이어에 위임한다.
**Tech Stack:** Go Gin (백엔드 API), sqlc (DB 쿼리), CodeMirror 6 (DSL 에디터), Zustand (검색 상태 관리), React (프론트엔드), DSL Engine (Plan 1의 Go DSL 엔진)

**Depends on:** Plan 1 (Go DSL 엔진, Go API 서버 스캐폴드, 인증 미들웨어, sqlc)

---

## Task 1: 검색 페이지 레이아웃 + Feature 상태 관리

검색 페이지의 전체 레이아웃과 Zustand 상태 스토어를 구축한다. FSD 규칙에 따라 상태를 `features/search/model/`에 배치한다.

**Files:**
- Create: `frontend/src/app/(pages)/search/page.tsx`
- Create: `frontend/src/app/(pages)/search/layout.tsx`
- Create: `frontend/src/features/search/model/search.store.ts`
- Create: `frontend/src/features/search/model/types.ts`
- Create: `frontend/src/features/search/ui/SearchPageLayout.tsx`
- Create: `frontend/src/features/search/index.ts` (barrel export)
- Test: `frontend/src/features/search/__tests__/search-store.test.ts`

### Steps

- [ ] Zustand 설치

```bash
cd /home/dev/code/dev-superbear/frontend
npm install zustand
```

- [ ] 검색 도메인 타입 정의 (`frontend/src/features/search/model/types.ts`) — inline이 아닌 별도 파일

```typescript
// frontend/src/features/search/model/types.ts
export type SearchTab = "nl" | "dsl";
export type AgentStatus = "idle" | "interpreting" | "building" | "scanning" | "done" | "error";
export type ValidationState = "none" | "valid" | "invalid";
```

- [ ] 검색 결과 엔티티 타입 정의 (`frontend/src/entities/search-result/model/types.ts`) — SearchResult와 ScanResult를 통합한 도메인 타입

```typescript
// frontend/src/entities/search-result/model/types.ts

/** Unified search/scan result — used across search and chart features */
export interface SearchResult {
  symbol: string;
  name: string;
  matchedValue: number | string;
  close?: number;
  volume?: number;
  tradeValue?: number;
  change?: number;
  changePct?: number;
}
```

```typescript
// frontend/src/entities/search-result/index.ts
export type { SearchResult } from "./model/types";
```

- [ ] 테스트 먼저 작성 (`frontend/src/features/search/__tests__/search-store.test.ts`)

```typescript
// frontend/src/features/search/__tests__/search-store.test.ts
import { useSearchStore } from "../model/search.store";

describe("Search Store", () => {
  beforeEach(() => {
    useSearchStore.setState(useSearchStore.getInitialState());
  });

  it("initializes with NL mode active", () => {
    const state = useSearchStore.getState();
    expect(state.activeTab).toBe("nl");
    expect(state.dslCode).toBe("");
    expect(state.nlQuery).toBe("");
    expect(state.results).toEqual([]);
  });

  it("switches tabs", () => {
    useSearchStore.getState().setActiveTab("dsl");
    expect(useSearchStore.getState().activeTab).toBe("dsl");
  });

  it("updates NL query", () => {
    useSearchStore.getState().setNlQuery("2년 최대거래량 종목");
    expect(useSearchStore.getState().nlQuery).toBe("2년 최대거래량 종목");
  });

  it("updates DSL code", () => {
    useSearchStore.getState().setDslCode("scan where volume > 1000000");
    expect(useSearchStore.getState().dslCode).toBe("scan where volume > 1000000");
  });

  it("tracks agent status transitions", () => {
    const { setAgentStatus } = useSearchStore.getState();
    setAgentStatus("interpreting");
    expect(useSearchStore.getState().agentStatus).toBe("interpreting");
    setAgentStatus("building");
    expect(useSearchStore.getState().agentStatus).toBe("building");
    setAgentStatus("scanning");
    expect(useSearchStore.getState().agentStatus).toBe("scanning");
  });

  it("stores search results", () => {
    const results = [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
    ];
    useSearchStore.getState().setResults(results);
    expect(useSearchStore.getState().results).toEqual(results);
  });

  it("tracks validation state", () => {
    useSearchStore.getState().setValidationState("valid");
    expect(useSearchStore.getState().validationState).toBe("valid");
  });
});
```

- [ ] Zustand 스토어 구현 (`frontend/src/features/search/model/search.store.ts`) — feature-scoped store

```typescript
// frontend/src/features/search/model/search.store.ts
import { create } from "zustand";
import type { SearchTab, AgentStatus, ValidationState } from "./types";
import type { SearchResult } from "@/entities/search-result";

interface SearchState {
  // 탭 모드
  activeTab: SearchTab;
  setActiveTab: (tab: SearchTab) => void;

  // NL 모드
  nlQuery: string;
  setNlQuery: (query: string) => void;

  // DSL 코드 (양 모드에서 공유 — NL 모드에서 에이전트가 생성, DSL 모드에서 사용자가 작성)
  dslCode: string;
  setDslCode: (code: string) => void;

  // 에이전트 상태 (NL 모드)
  agentStatus: AgentStatus;
  setAgentStatus: (status: AgentStatus) => void;
  agentMessage: string;
  setAgentMessage: (msg: string) => void;

  // 검증 상태
  validationState: ValidationState;
  validationMessage: string;
  setValidationState: (state: ValidationState, message?: string) => void;

  // 검색 결과
  results: SearchResult[];
  setResults: (results: SearchResult[]) => void;
  isSearching: boolean;
  setIsSearching: (v: boolean) => void;

  // 선택된 프리셋
  selectedPresetId: string | null;
  setSelectedPresetId: (id: string | null) => void;
}

export const useSearchStore = create<SearchState>()((set) => ({
  activeTab: "nl",
  setActiveTab: (tab) => set({ activeTab: tab }),

  nlQuery: "",
  setNlQuery: (query) => set({ nlQuery: query }),

  dslCode: "",
  setDslCode: (code) => set({ dslCode: code }),

  agentStatus: "idle",
  setAgentStatus: (status) => set({ agentStatus: status }),
  agentMessage: "",
  setAgentMessage: (msg) => set({ agentMessage: msg }),

  validationState: "none",
  validationMessage: "",
  setValidationState: (state, message = "") =>
    set({ validationState: state, validationMessage: message }),

  results: [],
  setResults: (results) => set({ results }),
  isSearching: false,
  setIsSearching: (v) => set({ isSearching: v }),

  selectedPresetId: null,
  setSelectedPresetId: (id) => set({ selectedPresetId: id }),
}));
```

- [ ] Barrel export (`frontend/src/features/search/index.ts`)

```typescript
// frontend/src/features/search/index.ts
export { useSearchStore } from "./model/search.store";
export type { SearchTab, AgentStatus, ValidationState } from "./model/types";
```

- [ ] 검색 페이지 레이아웃 구현 (`frontend/src/features/search/ui/SearchPageLayout.tsx`) — 상단: 탭 바(NL / DSL) + 입력 영역, 하단: LIVE DSL 패널, 아래: 검색 결과 테이블

```typescript
// frontend/src/features/search/ui/SearchPageLayout.tsx
"use client";

import { useSearchStore } from "../model/search.store";
import { NLTab } from "./NLTab";
import { DSLTab } from "./DSLTab";
import { LiveDSLPanel } from "./LiveDSLPanel";
import { SearchResults } from "./SearchResults";

export function SearchPageLayout() {
  const { activeTab, setActiveTab } = useSearchStore();

  return (
    <div className="flex flex-col h-full gap-4 p-6">
      {/* Tab Bar */}
      <div className="flex gap-1 bg-nexus-surface rounded-lg p-1 w-fit">
        <button
          onClick={() => setActiveTab("nl")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            activeTab === "nl"
              ? "bg-nexus-accent text-white"
              : "text-nexus-text-secondary hover:text-nexus-text-primary"
          }`}
        >
          Natural Language
        </button>
        <button
          onClick={() => setActiveTab("dsl")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            activeTab === "dsl"
              ? "bg-nexus-accent text-white"
              : "text-nexus-text-secondary hover:text-nexus-text-primary"
          }`}
        >
          DSL
        </button>
      </div>

      {/* Input Area */}
      <div className="bg-nexus-surface border border-nexus-border rounded-lg p-4">
        {activeTab === "nl" ? <NLTab /> : <DSLTab />}
      </div>

      {/* LIVE DSL Panel — 항상 표시 */}
      <LiveDSLPanel />

      {/* Search Results */}
      <SearchResults />
    </div>
  );
}
```

- [ ] 페이지 라우트 연결 (`frontend/src/app/(pages)/search/page.tsx`)

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/frontend
npx jest src/features/search/__tests__/search-store.test.ts
```

- [ ] 커밋

```bash
git add frontend/src/app/\(pages\)/search/ frontend/src/features/search/ frontend/src/entities/search-result/
git commit -m "feat: search page layout with feature-scoped Zustand store and NL/DSL tab switching"
```

---

## Task 2: NL 모드 탭 (자연어 입력 + 프리셋 칩 + 에이전트 상태)

자연어 검색 입력 UI를 구현한다. textarea, 프리셋 칩, 에이전트 상태 표시를 포함한다.

**Files:**
- Create: `frontend/src/features/search/ui/NLTab.tsx`
- Create: `frontend/src/features/search/ui/PresetChips.tsx`
- Create: `frontend/src/features/search/ui/AgentStatusIndicator.tsx`
- Test: `frontend/src/features/search/__tests__/NLTab.test.tsx`

### Steps

- [ ] 테스트 먼저 작성 (`frontend/src/features/search/__tests__/NLTab.test.tsx`)

```typescript
// frontend/src/features/search/__tests__/NLTab.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { NLTab } from "../ui/NLTab";
import { useSearchStore } from "../model/search.store";

// Store를 초기화
beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("NLTab", () => {
  it("renders textarea for NL query input", () => {
    render(<NLTab />);
    expect(screen.getByPlaceholderText(/자연어로 검색 조건/i)).toBeInTheDocument();
  });

  it("renders preset chips", () => {
    render(<NLTab />);
    expect(screen.getByText("2yr Max Volume")).toBeInTheDocument();
    expect(screen.getByText("Golden Cross")).toBeInTheDocument();
    expect(screen.getByText("RSI Oversold")).toBeInTheDocument();
  });

  it("clicking a preset chip fills the NL query", () => {
    render(<NLTab />);
    fireEvent.click(screen.getByText("2yr Max Volume"));
    const textarea = screen.getByPlaceholderText(/자연어로 검색 조건/i) as HTMLTextAreaElement;
    expect(textarea.value).toContain("2년 최대거래량");
  });

  it("shows agent status when searching", () => {
    useSearchStore.setState({ agentStatus: "interpreting" });
    render(<NLTab />);
    expect(screen.getByText(/Interpreting/i)).toBeInTheDocument();
  });

  it("has a search button", () => {
    render(<NLTab />);
    expect(screen.getByRole("button", { name: /검색|search/i })).toBeInTheDocument();
  });
});
```

- [ ] 프리셋 칩 컴포넌트 구현 (`frontend/src/features/search/ui/PresetChips.tsx`)

```typescript
// frontend/src/features/search/ui/PresetChips.tsx
"use client";

import { useSearchStore } from "../model/search.store";

const PRESETS = [
  { label: "2yr Max Volume", nlQuery: "최근 5년 안에 2년 최대거래량이 발생한 종목" },
  { label: "Golden Cross", nlQuery: "20일 이평선이 60일 이평선을 상향 돌파한 종목" },
  { label: "RSI Oversold", nlQuery: "RSI(14)가 30 이하로 과매도 구간인 종목" },
  { label: "High Trade Value", nlQuery: "거래대금 3000억 이상인 종목" },
  { label: "PER < 10", nlQuery: "PER이 10배 미만이고 영업이익이 흑자인 종목" },
  { label: "52w High", nlQuery: "52주 신고가를 달성한 종목" },
];

export function PresetChips() {
  const { setNlQuery } = useSearchStore();

  return (
    <div className="flex flex-wrap gap-2">
      {PRESETS.map((preset) => (
        <button
          key={preset.label}
          onClick={() => setNlQuery(preset.nlQuery)}
          className="px-3 py-1.5 text-xs font-medium rounded-full
                     bg-nexus-border text-nexus-text-secondary
                     hover:bg-nexus-accent/20 hover:text-nexus-accent
                     transition-colors"
        >
          {preset.label}
        </button>
      ))}
    </div>
  );
}
```

- [ ] 에이전트 상태 표시 컴포넌트 구현 (`frontend/src/features/search/ui/AgentStatusIndicator.tsx`) — 상태별 아이콘/메시지: idle, interpreting, building, scanning, done, error

```typescript
// frontend/src/features/search/ui/AgentStatusIndicator.tsx
"use client";

import { useSearchStore } from "../model/search.store";
import type { AgentStatus } from "../model/types";

const STATUS_CONFIG: Record<AgentStatus, { label: string; color: string; animate: boolean }> = {
  idle: { label: "", color: "", animate: false },
  interpreting: { label: "Interpreting query...", color: "text-nexus-warning", animate: true },
  building: { label: "Building DSL...", color: "text-nexus-accent", animate: true },
  scanning: { label: "Scanning stocks...", color: "text-nexus-accent", animate: true },
  done: { label: "Search complete", color: "text-nexus-success", animate: false },
  error: { label: "Error occurred", color: "text-nexus-failure", animate: false },
};

export function AgentStatusIndicator() {
  const { agentStatus, agentMessage } = useSearchStore();
  if (agentStatus === "idle") return null;

  const config = STATUS_CONFIG[agentStatus];

  return (
    <div className={`flex items-center gap-2 text-sm ${config.color}`}>
      {config.animate && (
        <span className="inline-block w-2 h-2 rounded-full bg-current animate-pulse" />
      )}
      <span>{agentMessage || config.label}</span>
    </div>
  );
}
```

- [ ] NL 탭 메인 컴포넌트 구현 (`frontend/src/features/search/ui/NLTab.tsx`) — textarea + 프리셋 칩 + 에이전트 상태 + 검색 버튼 조합

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/frontend
npx jest src/features/search/__tests__/NLTab.test.tsx --env=jsdom
```

- [ ] 커밋

```bash
git add frontend/src/features/search/
git commit -m "feat: NL mode tab with preset chips and agent status indicator"
```

---

## Task 3: DSL 모드 탭 (코드 에디터 + 자동완성 + Validate/Explain 버튼)

CodeMirror 6 기반 DSL 코드 에디터와 Validate/Explain 버튼을 구현한다. 자동완성 힌트는 feature-specific이므로 `features/search/lib/`에 배치한다.

**Files:**
- Create: `frontend/src/features/search/ui/DSLTab.tsx`
- Create: `frontend/src/features/search/ui/DSLEditor.tsx`
- Create: `frontend/src/features/search/lib/dsl-completions.ts` (자동완성 힌트 목록 — feature-specific)
- Test: `frontend/src/features/search/__tests__/DSLTab.test.tsx`

### Steps

- [ ] CodeMirror 설치

```bash
cd /home/dev/code/dev-superbear/frontend
npm install @codemirror/state @codemirror/view @codemirror/language
npm install @codemirror/autocomplete @codemirror/lint @codemirror/commands
npm install @codemirror/lang-javascript @lezer/highlight
```

- [ ] 자동완성 힌트 목록 정의 (`frontend/src/features/search/lib/dsl-completions.ts`) — feature-specific, not shared

```typescript
// frontend/src/features/search/lib/dsl-completions.ts
export interface CompletionItem {
  label: string;
  type: "keyword" | "function" | "variable";
  detail: string;
}

export const DSL_COMPLETIONS: CompletionItem[] = [
  // Keywords
  { label: "scan", type: "keyword", detail: "종목 스캔 시작" },
  { label: "where", type: "keyword", detail: "필터 조건" },
  { label: "sort", type: "keyword", detail: "정렬" },
  { label: "by", type: "keyword", detail: "정렬 기준" },
  { label: "asc", type: "keyword", detail: "오름차순" },
  { label: "desc", type: "keyword", detail: "내림차순" },
  { label: "and", type: "keyword", detail: "논리 AND" },
  { label: "or", type: "keyword", detail: "논리 OR" },
  { label: "limit", type: "keyword", detail: "결과 제한" },

  // Variables — 현재가/거래 관련
  { label: "close", type: "variable", detail: "종가 / 현재가" },
  { label: "open", type: "variable", detail: "시가" },
  { label: "high", type: "variable", detail: "고가" },
  { label: "low", type: "variable", detail: "저가" },
  { label: "volume", type: "variable", detail: "거래량" },
  { label: "trade_value", type: "variable", detail: "거래대금" },
  { label: "market_cap", type: "variable", detail: "시가총액" },
  { label: "per", type: "variable", detail: "PER (주가수익비율)" },
  { label: "pbr", type: "variable", detail: "PBR (주가순자산비율)" },
  { label: "roe", type: "variable", detail: "ROE (자기자본이익률)" },

  // Variables — 이벤트 상대 변수
  { label: "event_high", type: "variable", detail: "이벤트 발생일 고가" },
  { label: "event_low", type: "variable", detail: "이벤트 발생일 저가" },
  { label: "event_close", type: "variable", detail: "이벤트 발생일 종가" },
  { label: "event_volume", type: "variable", detail: "이벤트 발생일 거래량" },
  { label: "pre_event_close", type: "variable", detail: "이벤트 전일 종가" },
  { label: "post_high", type: "variable", detail: "이벤트 이후 최고가" },
  { label: "post_low", type: "variable", detail: "이벤트 이후 최저가" },
  { label: "days_since_event", type: "variable", detail: "이벤트 이후 경과일" },

  // Functions
  { label: "ma", type: "function", detail: "ma(N) — N일 이동평균" },
  { label: "rsi", type: "function", detail: "rsi(N) — N일 RSI" },
  { label: "macd", type: "function", detail: "macd(short, long, signal)" },
  { label: "bb", type: "function", detail: "bb(N, K) — 볼린저밴드" },
  { label: "max_volume", type: "function", detail: "max_volume(days) — N일 최대거래량" },
  { label: "pre_event_ma", type: "function", detail: "pre_event_ma(N) — 이벤트 전일 기준 N일 이평선" },
  { label: "max", type: "function", detail: "max(a, b) — 큰 값" },
  { label: "min", type: "function", detail: "min(a, b) — 작은 값" },
  { label: "abs", type: "function", detail: "abs(x) — 절대값" },
];
```

- [ ] 테스트 먼저 작성 (`frontend/src/features/search/__tests__/DSLTab.test.tsx`)

```typescript
// frontend/src/features/search/__tests__/DSLTab.test.tsx
import { render, screen, fireEvent } from "@testing-library/react";
import { DSLTab } from "../ui/DSLTab";
import { useSearchStore } from "../model/search.store";

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("DSLTab", () => {
  it("renders the DSL editor area", () => {
    render(<DSLTab />);
    // CodeMirror는 DOM에 직접 렌더링하므로 컨테이너 확인
    expect(screen.getByTestId("dsl-editor-container")).toBeInTheDocument();
  });

  it("renders Validate button", () => {
    render(<DSLTab />);
    expect(screen.getByRole("button", { name: /validate/i })).toBeInTheDocument();
  });

  it("renders Explain in NL button", () => {
    render(<DSLTab />);
    expect(screen.getByRole("button", { name: /explain/i })).toBeInTheDocument();
  });

  it("renders Run Search button", () => {
    render(<DSLTab />);
    expect(screen.getByRole("button", { name: /run|search|실행/i })).toBeInTheDocument();
  });
});
```

- [ ] DSL 에디터 컴포넌트 구현 (`frontend/src/features/search/ui/DSLEditor.tsx`) — CodeMirror 6를 React에 통합. 다크 테마, monospace 폰트, 자동완성 확장 포함

```typescript
// frontend/src/features/search/ui/DSLEditor.tsx
"use client";

import { useEffect, useRef } from "react";
import { EditorView, keymap, placeholder as phExtension } from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { defaultKeymap } from "@codemirror/commands";
import { autocompletion, CompletionContext } from "@codemirror/autocomplete";
import { DSL_COMPLETIONS } from "../lib/dsl-completions";
import { useSearchStore } from "../model/search.store";

const nexusDarkTheme = EditorView.theme({
  "&": { backgroundColor: "#0a0a0f", color: "#e2e8f0", fontSize: "14px" },
  ".cm-content": { fontFamily: "'JetBrains Mono', 'Fira Code', monospace", padding: "12px" },
  ".cm-gutters": { backgroundColor: "#12121a", borderRight: "1px solid #1e1e2e" },
  ".cm-activeLine": { backgroundColor: "rgba(99, 102, 241, 0.08)" },
  ".cm-cursor": { borderLeftColor: "#6366f1" },
  "&.cm-focused .cm-selectionBackground": { backgroundColor: "rgba(99, 102, 241, 0.2)" },
});

function dslAutoComplete(context: CompletionContext) {
  const word = context.matchBefore(/\w*/);
  if (!word || (word.from === word.to && !context.explicit)) return null;
  return {
    from: word.from,
    options: DSL_COMPLETIONS.map((c) => ({
      label: c.label,
      type: c.type,
      detail: c.detail,
    })),
  };
}

interface DSLEditorProps {
  readOnly?: boolean;
  placeholder?: string;
  height?: string;
}

export function DSLEditor({ readOnly = false, placeholder = "", height = "200px" }: DSLEditorProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const { dslCode, setDslCode } = useSearchStore();

  useEffect(() => {
    if (!containerRef.current) return;

    const state = EditorState.create({
      doc: dslCode,
      extensions: [
        keymap.of(defaultKeymap),
        nexusDarkTheme,
        EditorView.lineWrapping,
        phExtension(placeholder),
        autocompletion({ override: [dslAutoComplete] }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && !readOnly) {
            setDslCode(update.state.doc.toString());
          }
        }),
        ...(readOnly ? [EditorState.readOnly.of(true)] : []),
      ],
    });

    const view = new EditorView({ state, parent: containerRef.current });
    viewRef.current = view;

    return () => { view.destroy(); };
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // dslCode가 외부에서 변경될 때 (NL 모드 에이전트가 생성) 에디터 동기화
  useEffect(() => {
    const view = viewRef.current;
    if (view && view.state.doc.toString() !== dslCode) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: dslCode },
      });
    }
  }, [dslCode]);

  return (
    <div
      ref={containerRef}
      data-testid="dsl-editor-container"
      className="border border-nexus-border rounded-lg overflow-hidden"
      style={{ minHeight: height }}
    />
  );
}
```

- [ ] DSL 탭 메인 컴포넌트 구현 (`frontend/src/features/search/ui/DSLTab.tsx`) — 에디터 + Validate/Explain/Run 버튼 레이아웃

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/frontend
npx jest src/features/search/__tests__/DSLTab.test.tsx --env=jsdom
```

- [ ] 커밋

```bash
git add frontend/src/features/search/
git commit -m "feat: DSL mode tab with CodeMirror editor, autocomplete, and action buttons"
```

---

## Task 4: LIVE DSL 패널

어떤 모드에서든 항상 표시되는 LIVE DSL 패널을 구현한다. NL 모드에서는 에이전트가 생성한 DSL을, DSL 모드에서는 사용자가 입력한 DSL을 구문 하이라이팅으로 표시한다. 하이라이팅은 shared DSL 모듈의 기능이므로 `shared/lib/dsl/` 에 배치한다.

**Files:**
- Create: `frontend/src/features/search/ui/LiveDSLPanel.tsx`
- Create: `frontend/src/shared/lib/dsl/highlight.ts` (구문 하이라이팅 유틸 — shared)
- Test: `frontend/src/features/search/__tests__/LiveDSLPanel.test.tsx`

### Steps

- [ ] 테스트 먼저 작성 (`frontend/src/features/search/__tests__/LiveDSLPanel.test.tsx`)

```typescript
// frontend/src/features/search/__tests__/LiveDSLPanel.test.tsx
import { render, screen } from "@testing-library/react";
import { LiveDSLPanel } from "../ui/LiveDSLPanel";
import { useSearchStore } from "../model/search.store";

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("LiveDSLPanel", () => {
  it("renders the LIVE DSL label", () => {
    render(<LiveDSLPanel />);
    expect(screen.getByText(/LIVE DSL/i)).toBeInTheDocument();
  });

  it("shows DSL code from store", () => {
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<LiveDSLPanel />);
    expect(screen.getByText(/scan/)).toBeInTheDocument();
    expect(screen.getByText(/volume/)).toBeInTheDocument();
  });

  it("shows empty state when no DSL", () => {
    render(<LiveDSLPanel />);
    expect(screen.getByText(/no dsl|dsl이 없습니다|empty/i)).toBeInTheDocument();
  });

  it("shows validation badge when validated", () => {
    useSearchStore.setState({
      dslCode: "scan where volume > 1000000",
      validationState: "valid",
    });
    render(<LiveDSLPanel />);
    expect(screen.getByText(/validated/i)).toBeInTheDocument();
  });

  it("shows warning badge when not validated", () => {
    useSearchStore.setState({
      dslCode: "scan where volume > 1000000",
      validationState: "none",
    });
    render(<LiveDSLPanel />);
    expect(screen.getByText(/not validated/i)).toBeInTheDocument();
  });

  it("renders Copy and Run Search buttons", () => {
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<LiveDSLPanel />);
    expect(screen.getByRole("button", { name: /copy/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run/i })).toBeInTheDocument();
  });
});
```

- [ ] 구문 하이라이팅 유틸 구현 (`frontend/src/shared/lib/dsl/highlight.ts`) — DSL 코드 문자열을 토큰화하여 `<span>` 태그 배열로 변환. 키워드(보라색), 숫자(녹색), 연산자(노란색), 함수명(파란색), 변수(기본). 이 모듈은 DSL 엔진의 일부이므로 shared에 배치

```typescript
// frontend/src/shared/lib/dsl/highlight.ts
import { Lexer } from "./lexer";
import { TokenType } from "./tokens";

const TOKEN_COLORS: Partial<Record<TokenType, string>> = {
  [TokenType.SCAN]: "text-purple-400",
  [TokenType.WHERE]: "text-purple-400",
  [TokenType.SORT]: "text-purple-400",
  [TokenType.BY]: "text-purple-400",
  [TokenType.AND]: "text-purple-400",
  [TokenType.OR]: "text-purple-400",
  [TokenType.ASC]: "text-purple-400",
  [TokenType.DESC]: "text-purple-400",
  [TokenType.LIMIT]: "text-purple-400",
  [TokenType.NUMBER]: "text-green-400",
  [TokenType.GTE]: "text-yellow-300",
  [TokenType.LTE]: "text-yellow-300",
  [TokenType.GT]: "text-yellow-300",
  [TokenType.LT]: "text-yellow-300",
  [TokenType.EQ]: "text-yellow-300",
  [TokenType.NEQ]: "text-yellow-300",
  [TokenType.ASSIGN]: "text-yellow-300",
  [TokenType.STAR]: "text-yellow-300",
  [TokenType.SLASH]: "text-yellow-300",
  [TokenType.PLUS]: "text-yellow-300",
  [TokenType.MINUS]: "text-yellow-300",
};

export interface HighlightedToken {
  text: string;
  className: string;
}

export function highlightDSL(code: string): HighlightedToken[] {
  try {
    const tokens = new Lexer(code).tokenize();
    return tokens
      .filter((t) => t.type !== TokenType.EOF)
      .map((t) => ({
        text: t.value,
        className: TOKEN_COLORS[t.type] ?? "text-nexus-text-primary",
      }));
  } catch {
    // 파싱 실패 시 전체를 기본 색상으로
    return [{ text: code, className: "text-nexus-text-primary" }];
  }
}
```

- [ ] LIVE DSL 패널 컴포넌트 구현 (`frontend/src/features/search/ui/LiveDSLPanel.tsx`) — 구문 하이라이팅된 DSL 표시 + 검증 배지 + Copy/Run 버튼. CodeMirror 읽기 전용 또는 커스텀 렌더링 사용

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/frontend
npx jest src/features/search/__tests__/LiveDSLPanel.test.tsx --env=jsdom
```

- [ ] 커밋

```bash
git add frontend/src/features/search/ui/LiveDSLPanel.tsx frontend/src/shared/lib/dsl/highlight.ts
git commit -m "feat: LIVE DSL panel with syntax highlighting, validation badge, and copy/run"
```

---

## Task 5: Go 백엔드 — Search API 핸들러 + 서비스

검색 관련 4개 API 엔드포인트를 Go Gin 핸들러로 구현한다. 핸들러는 thin controller로서 service 레이어에 위임한다.

**Files:**
- Create: `backend/internal/handler/search_handler.go` — Gin 핸들러 (Execute, Validate, NLToDSL, Explain)
- Create: `backend/internal/service/search_service.go` — DSL 실행 오케스트레이션
- Create: `backend/internal/service/nl_to_dsl_service.go` — NL->DSL 변환 (Google ADK 호출)
- Create: `backend/internal/dsl/executor.go` — DSL 실행 엔진 (Plan 1 Go 엔진 활용)
- Create: `frontend/src/features/search/api/search-api.ts` — API client (Go 서버 URL)
- Test: `backend/internal/handler/search_handler_test.go`
- Test: `backend/internal/service/search_service_test.go`

### Steps

- [ ] 테스트 먼저 작성 (`backend/internal/service/search_service_test.go`)

```go
// backend/internal/service/search_service_test.go
package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nexus/internal/service"
)

func TestSearchService_Validate(t *testing.T) {
	svc := service.NewSearchService(nil) // nil DSL executor for unit test

	t.Run("accepts valid scan query", func(t *testing.T) {
		result := svc.Validate(context.Background(), "scan where volume > 1000000")
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)
	})

	t.Run("accepts scan with sort and limit", func(t *testing.T) {
		result := svc.Validate(context.Background(), "scan where volume > 1000000 sort by trade_value desc limit 50")
		assert.True(t, result.Valid)
	})

	t.Run("rejects invalid syntax", func(t *testing.T) {
		result := svc.Validate(context.Background(), "scan where >")
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Error)
	})

	t.Run("accepts condition assignment", func(t *testing.T) {
		result := svc.Validate(context.Background(), "success = close >= event_high * 2.0")
		assert.True(t, result.Valid)
	})
}

func TestSearchService_ParseScanQuery(t *testing.T) {
	svc := service.NewSearchService(nil)

	t.Run("extracts where clause sort and limit", func(t *testing.T) {
		parsed, err := svc.ParseScanQuery(context.Background(), "scan where volume > 1000000 sort by trade_value desc limit 50")
		require.NoError(t, err)
		require.NotNil(t, parsed)
		assert.Equal(t, 50, parsed.Limit)
		assert.Equal(t, "trade_value", parsed.SortBy.Field)
		assert.Equal(t, "desc", parsed.SortBy.Direction)
	})

	t.Run("returns nil for non-scan queries", func(t *testing.T) {
		parsed, err := svc.ParseScanQuery(context.Background(), "success = close >= 80000")
		require.NoError(t, err)
		assert.Nil(t, parsed)
	})
}
```

- [ ] DSL 실행 엔진 구현 (`backend/internal/dsl/executor.go`) — Go DSL 파서로 AST를 만들고, ScanQuery인 경우 종목 데이터를 조건 필터링

```go
// backend/internal/dsl/executor.go
package dsl

import (
	"context"
	"fmt"
)

// SearchResult represents a single stock matching the scan criteria.
type SearchResult struct {
	Symbol       string      `json:"symbol"`
	Name         string      `json:"name"`
	MatchedValue interface{} `json:"matchedValue"`
	Close        *float64    `json:"close,omitempty"`
	Volume       *int64      `json:"volume,omitempty"`
	TradeValue   *float64    `json:"tradeValue,omitempty"`
	Change       *float64    `json:"change,omitempty"`
	ChangePct    *float64    `json:"changePct,omitempty"`
}

// SortSpec describes sorting criteria for scan results.
type SortSpec struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "asc" or "desc"
}

// ParsedScanQuery represents a parsed scan query with extracted components.
type ParsedScanQuery struct {
	WhereClause string    `json:"whereClause"`
	SortBy      *SortSpec `json:"sortBy,omitempty"`
	Limit       int       `json:"limit"`
}

// ValidationResult represents the result of DSL validation.
type ValidationResult struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// Executor handles DSL parsing, validation, and execution.
type Executor struct {
	// TODO: Inject data sources (KIS API client, DB pool) for stock data retrieval
}

// NewExecutor creates a new DSL executor.
func NewExecutor() *Executor {
	return &Executor{}
}

// Validate checks if the given DSL input is syntactically valid.
func (e *Executor) Validate(input string) ValidationResult {
	// TODO: Use Plan 1 Go DSL parser for full validation
	// Placeholder: basic syntax check
	if len(input) == 0 {
		return ValidationResult{Valid: false, Error: "empty DSL input"}
	}
	// Attempt to parse — delegate to Plan 1 DSL engine
	_, err := e.parse(input)
	if err != nil {
		return ValidationResult{Valid: false, Error: err.Error()}
	}
	return ValidationResult{Valid: true}
}

// ParseScan parses a scan query and extracts its components.
// Returns nil if the query is not a scan query (e.g., condition assignment).
func (e *Executor) ParseScan(input string) (*ParsedScanQuery, error) {
	ast, err := e.parse(input)
	if err != nil {
		return nil, err
	}
	if ast.Type != "ScanQuery" {
		return nil, nil
	}
	return &ParsedScanQuery{
		WhereClause: ast.WhereClause,
		SortBy:      ast.SortBy,
		Limit:       ast.Limit,
	}, nil
}

// Execute runs a DSL scan query and returns matching stocks.
func (e *Executor) Execute(ctx context.Context, dslCode string) ([]SearchResult, error) {
	parsed, err := e.ParseScan(dslCode)
	if err != nil {
		return nil, fmt.Errorf("invalid DSL: %w", err)
	}
	if parsed == nil {
		return nil, fmt.Errorf("not a valid scan query")
	}

	// TODO: Fetch all stock data from KIS API or DB, then filter by DSL conditions
	// Placeholder: return empty results
	return []SearchResult{}, nil
}

// parse is an internal method that delegates to the Plan 1 Go DSL parser.
type parsedAST struct {
	Type        string
	WhereClause string
	SortBy      *SortSpec
	Limit       int
}

func (e *Executor) parse(input string) (*parsedAST, error) {
	// TODO: Integrate with Plan 1 Go DSL parser (backend/internal/dsl/parser.go)
	// Placeholder implementation for basic structure
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}
	// This will be replaced with actual parser calls
	return &parsedAST{Type: "ScanQuery", Limit: 100}, nil
}
```

- [ ] 검색 서비스 구현 (`backend/internal/service/search_service.go`) — DSL 실행 오케스트레이션

```go
// backend/internal/service/search_service.go
package service

import (
	"context"

	"nexus/internal/dsl"
)

// SearchService orchestrates DSL validation, parsing, and execution.
type SearchService struct {
	executor *dsl.Executor
}

// NewSearchService creates a new SearchService.
func NewSearchService(executor *dsl.Executor) *SearchService {
	if executor == nil {
		executor = dsl.NewExecutor()
	}
	return &SearchService{executor: executor}
}

// Validate checks if the DSL input is syntactically valid.
func (s *SearchService) Validate(ctx context.Context, dslCode string) dsl.ValidationResult {
	return s.executor.Validate(dslCode)
}

// ParseScanQuery parses a DSL scan query and extracts its components.
func (s *SearchService) ParseScanQuery(ctx context.Context, dslCode string) (*dsl.ParsedScanQuery, error) {
	return s.executor.ParseScan(dslCode)
}

// Execute runs a DSL scan query and returns matching stocks.
func (s *SearchService) Execute(ctx context.Context, dslCode string) ([]dsl.SearchResult, error) {
	return s.executor.Execute(ctx, dslCode)
}
```

- [ ] NL->DSL 변환 서비스 구현 (`backend/internal/service/nl_to_dsl_service.go`) — Google ADK 호출 래퍼 (placeholder 구현, 나중에 실제 LLM 연동)

```go
// backend/internal/service/nl_to_dsl_service.go
package service

import (
	"context"
	"fmt"
)

// NLToDSLResult represents the result of NL to DSL conversion.
type NLToDSLResult struct {
	DSL         string  `json:"dsl"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
}

// NLToDSLService converts natural language queries to DSL using Google ADK.
type NLToDSLService struct {
	// TODO: Google ADK client
	// adkClient *adk.Client
}

// NewNLToDSLService creates a new NLToDSLService.
func NewNLToDSLService() *NLToDSLService {
	return &NLToDSLService{}
}

// Convert translates a natural language query to DSL.
func (s *NLToDSLService) Convert(ctx context.Context, nlQuery string) (*NLToDSLResult, error) {
	if nlQuery == "" {
		return nil, fmt.Errorf("empty NL query")
	}

	// TODO: Implement actual Google ADK call
	// The prompt should include DSL grammar, available variables/functions, and examples.
	// Placeholder implementation:
	return &NLToDSLResult{
		DSL:         "scan where volume > 1000000",
		Explanation: fmt.Sprintf(`"%s" 조건을 DSL로 변환했습니다.`, nlQuery),
		Confidence:  0.0, // placeholder
	}, nil
}
```

- [ ] Search API 핸들러 테스트 작성 (`backend/internal/handler/search_handler_test.go`)

```go
// backend/internal/handler/search_handler_test.go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nexus/internal/handler"
	"nexus/internal/service"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	searchSvc := service.NewSearchService(nil)
	nlSvc := service.NewNLToDSLService()
	h := handler.NewSearchHandler(searchSvc, nlSvc)

	api := r.Group("/api/v1")
	{
		search := api.Group("/search")
		{
			search.POST("/execute", h.Execute)
			search.POST("/validate", h.Validate)
			search.POST("/nl-to-dsl", h.NLToDSL)
			search.POST("/explain", h.Explain)
		}
	}

	return r
}

func TestSearchHandler_Execute(t *testing.T) {
	r := setupRouter()

	t.Run("returns results for valid DSL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dslCode": "scan where volume > 1000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "results")
	})

	t.Run("rejects missing dslCode", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_Validate(t *testing.T) {
	r := setupRouter()

	t.Run("returns valid for correct DSL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, true, resp["valid"])
	})
}

func TestSearchHandler_NLToDSL(t *testing.T) {
	r := setupRouter()

	t.Run("converts NL query to DSL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"query": "2년 최대거래량 종목"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/nl-to-dsl", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "dsl")
		assert.Contains(t, resp, "explanation")
		assert.Contains(t, resp, "results")
	})

	t.Run("rejects empty query", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"query": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/nl-to-dsl", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_Explain(t *testing.T) {
	r := setupRouter()

	t.Run("explains DSL in natural language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/explain", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "explanation")
	})
}
```

- [ ] Search API Gin 핸들러 구현 (`backend/internal/handler/search_handler.go`) — 4개 엔드포인트를 thin controller 패턴으로 구현

```go
// backend/internal/handler/search_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nexus/internal/service"
)

// SearchHandler handles search-related HTTP endpoints.
type SearchHandler struct {
	searchSvc *service.SearchService
	nlSvc     *service.NLToDSLService
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(searchSvc *service.SearchService, nlSvc *service.NLToDSLService) *SearchHandler {
	return &SearchHandler{
		searchSvc: searchSvc,
		nlSvc:     nlSvc,
	}
}

// ExecuteSearchRequest is the request body for POST /api/v1/search/execute.
type ExecuteSearchRequest struct {
	DSLCode string `json:"dslCode" binding:"required"`
}

// Execute handles POST /api/v1/search/execute — validates and runs a DSL scan query.
func (h *SearchHandler) Execute(c *gin.Context) {
	var req ExecuteSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate first
	validation := h.searchSvc.Validate(c.Request.Context(), req.DSLCode)
	if !validation.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL: " + validation.Error})
		return
	}

	// Execute
	results, err := h.searchSvc.Execute(c.Request.Context(), req.DSLCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// ValidateRequest is the request body for POST /api/v1/search/validate.
type ValidateRequest struct {
	DSL string `json:"dsl"`
}

// Validate handles POST /api/v1/search/validate — checks DSL syntax validity.
func (h *SearchHandler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.searchSvc.Validate(c.Request.Context(), req.DSL)
	c.JSON(http.StatusOK, result)
}

// NLToDSLRequest is the request body for POST /api/v1/search/nl-to-dsl.
type NLToDSLRequest struct {
	Query string `json:"query" binding:"required"`
}

// NLToDSL handles POST /api/v1/search/nl-to-dsl — converts NL query to DSL and executes.
func (h *SearchHandler) NLToDSL(c *gin.Context) {
	var req NLToDSLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	// Step 1: NL -> DSL conversion (Google ADK)
	dslResult, err := h.nlSvc.Convert(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Step 2: Execute the generated DSL
	results, err := h.searchSvc.Execute(c.Request.Context(), dslResult.DSL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dsl":         dslResult.DSL,
		"explanation": dslResult.Explanation,
		"results":     results,
	})
}

// ExplainRequest is the request body for POST /api/v1/search/explain.
type ExplainRequest struct {
	DSL string `json:"dsl" binding:"required"`
}

// Explain handles POST /api/v1/search/explain — explains DSL in natural language.
func (h *SearchHandler) Explain(c *gin.Context) {
	var req ExplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Use LLM to generate natural language explanation of the DSL
	// Placeholder implementation
	explanation := "이 쿼리는 다음 조건으로 종목을 검색합니다: " + req.DSL

	c.JSON(http.StatusOK, gin.H{"explanation": explanation})
}

// RegisterRoutes registers all search-related routes on the given router group.
func (h *SearchHandler) RegisterRoutes(rg *gin.RouterGroup) {
	search := rg.Group("/search")
	{
		search.POST("/execute", h.Execute)
		search.POST("/validate", h.Validate)
		search.POST("/nl-to-dsl", h.NLToDSL)
		search.POST("/explain", h.Explain)
	}
}
```

- [ ] 프론트엔드 API client 구현 (`frontend/src/features/search/api/search-api.ts`) — Go 서버 URL로 변경

```typescript
// frontend/src/features/search/api/search-api.ts
import { apiClient } from "@/shared/api/client";
import type { SearchResult } from "@/entities/search-result";

interface NLSearchResponse {
  dsl: string;
  explanation: string;
  results: SearchResult[];
}

interface DSLSearchResponse {
  results: SearchResult[];
}

interface ValidateResponse {
  valid: boolean;
  error: string | null;
}

interface ExplainResponse {
  explanation: string;
}

export const searchApi = {
  async nlSearch(query: string): Promise<NLSearchResponse> {
    return apiClient<NLSearchResponse>("/api/v1/search/nl-to-dsl", {
      method: "POST",
      body: JSON.stringify({ query }),
    });
  },

  async dslSearch(dsl: string): Promise<DSLSearchResponse> {
    return apiClient<DSLSearchResponse>("/api/v1/search/execute", {
      method: "POST",
      body: JSON.stringify({ dslCode: dsl }),
    });
  },

  async validate(dsl: string): Promise<ValidateResponse> {
    return apiClient<ValidateResponse>("/api/v1/search/validate", {
      method: "POST",
      body: JSON.stringify({ dsl }),
    });
  },

  async explain(dsl: string): Promise<ExplainResponse> {
    return apiClient<ExplainResponse>("/api/v1/search/explain", {
      method: "POST",
      body: JSON.stringify({ dsl }),
    });
  },
};
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/service/... -v
go test ./internal/handler/... -v
```

- [ ] 커밋

```bash
git add backend/internal/handler/ backend/internal/service/ backend/internal/dsl/ frontend/src/features/search/api/
git commit -m "feat: Go search API handlers with service layer, DSL executor, and NL-to-DSL service"
```

---

## Task 6: 검색 프리셋 Entity CRUD (Go + sqlc)

검색 조건(DSL 쿼리)을 저장/불러오기/삭제하는 프리셋 관리 기능을 구현한다. 백엔드는 Go sqlc 기반 리포지토리로, 프론트엔드는 FSD 규칙에 따라 `entities/search-preset/`에 배치한다.

**Files:**
- Create: `backend/db/migrations/002_add_search_presets.sql` (DDL 마이그레이션)
- Create: `backend/db/queries/presets.sql` (sqlc 쿼리)
- Create: `backend/internal/repository/preset_repo.go` (sqlc 생성 코드 활용)
- Create: `backend/internal/handler/preset_handler.go` (Gin 핸들러)
- Create: `frontend/src/entities/search-preset/model/types.ts`
- Create: `frontend/src/entities/search-preset/index.ts`
- Create: `frontend/src/features/search/ui/PresetManager.tsx`
- Test: `backend/internal/handler/preset_handler_test.go`
- Test: `frontend/src/entities/search-preset/__tests__/search-preset.test.ts`

### Steps

- [ ] DB 마이그레이션 SQL 작성 (`backend/db/migrations/002_add_search_presets.sql`)

```sql
-- backend/db/migrations/002_add_search_presets.sql
CREATE TABLE IF NOT EXISTS search_presets (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id),
    name       VARCHAR(255) NOT NULL,
    dsl        TEXT NOT NULL,
    nl_query   TEXT,
    is_public  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_search_presets_user_id ON search_presets(user_id);
```

- [ ] sqlc 쿼리 작성 (`backend/db/queries/presets.sql`)

```sql
-- backend/db/queries/presets.sql

-- name: ListPresets :many
SELECT * FROM search_presets
WHERE user_id = $1 OR is_public = true
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPresets :one
SELECT COUNT(*) FROM search_presets
WHERE user_id = $1 OR is_public = true;

-- name: CreatePreset :one
INSERT INTO search_presets (user_id, name, dsl, nl_query, is_public)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPreset :one
SELECT * FROM search_presets WHERE id = $1;

-- name: DeletePreset :exec
DELETE FROM search_presets WHERE id = $1 AND user_id = $2;
```

- [ ] sqlc 코드 생성

```bash
cd /home/dev/code/dev-superbear/backend
sqlc generate
```

- [ ] 프리셋 리포지토리 구현 (`backend/internal/repository/preset_repo.go`) — sqlc 생성 코드를 래핑

```go
// backend/internal/repository/preset_repo.go
package repository

import (
	"context"
	"fmt"

	"nexus/internal/db/sqlc"
)

// PresetRepository handles SearchPreset CRUD operations via sqlc.
type PresetRepository struct {
	queries *sqlc.Queries
}

// NewPresetRepository creates a new PresetRepository.
func NewPresetRepository(queries *sqlc.Queries) *PresetRepository {
	return &PresetRepository{queries: queries}
}

// PaginatedPresets represents a paginated list of presets.
type PaginatedPresets struct {
	Presets []sqlc.SearchPreset `json:"presets"`
	Total   int64               `json:"total"`
}

// FindMany returns a paginated list of presets visible to the user.
func (r *PresetRepository) FindMany(ctx context.Context, userID string, limit, offset int32) (*PaginatedPresets, error) {
	presets, err := r.queries.ListPresets(ctx, sqlc.ListPresetsParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("list presets: %w", err)
	}

	count, err := r.queries.CountPresets(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count presets: %w", err)
	}

	return &PaginatedPresets{
		Presets: presets,
		Total:   count,
	}, nil
}

// Create creates a new search preset.
func (r *PresetRepository) Create(ctx context.Context, params sqlc.CreatePresetParams) (*sqlc.SearchPreset, error) {
	preset, err := r.queries.CreatePreset(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create preset: %w", err)
	}
	return &preset, nil
}

// Delete removes a preset owned by the specified user.
func (r *PresetRepository) Delete(ctx context.Context, id, userID string) error {
	return r.queries.DeletePreset(ctx, sqlc.DeletePresetParams{
		ID:     id,
		UserID: userID,
	})
}
```

- [ ] 프리셋 Gin 핸들러 구현 (`backend/internal/handler/preset_handler.go`) — GET (ListPresets), POST (CreatePreset), DELETE (DeletePreset)

```go
// backend/internal/handler/preset_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nexus/internal/db/sqlc"
	"nexus/internal/repository"
)

// PresetHandler handles search preset HTTP endpoints.
type PresetHandler struct {
	repo *repository.PresetRepository
}

// NewPresetHandler creates a new PresetHandler.
func NewPresetHandler(repo *repository.PresetRepository) *PresetHandler {
	return &PresetHandler{repo: repo}
}

// ListPresets handles GET /api/v1/search/presets — returns paginated presets.
func (h *PresetHandler) ListPresets(c *gin.Context) {
	userID := c.GetString("userID") // set by auth middleware

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	result, err := h.repo.FindMany(c.Request.Context(), userID, int32(pageSize), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (int(result.Total) + pageSize - 1) / pageSize
	c.JSON(http.StatusOK, gin.H{
		"data": result.Presets,
		"pagination": gin.H{
			"total":      result.Total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// CreatePresetRequest is the request body for POST /api/v1/search/presets.
type CreatePresetRequest struct {
	Name     string  `json:"name" binding:"required"`
	DSL      string  `json:"dsl" binding:"required"`
	NLQuery  *string `json:"nlQuery,omitempty"`
	IsPublic bool    `json:"isPublic"`
}

// CreatePreset handles POST /api/v1/search/presets — creates a new preset.
func (h *PresetHandler) CreatePreset(c *gin.Context) {
	userID := c.GetString("userID")

	var req CreatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preset, err := h.repo.Create(c.Request.Context(), sqlc.CreatePresetParams{
		UserID:   userID,
		Name:     req.Name,
		Dsl:      req.DSL,
		NlQuery:  req.NLQuery,
		IsPublic: req.IsPublic,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": preset})
}

// DeletePreset handles DELETE /api/v1/search/presets/:id — deletes a preset.
func (h *PresetHandler) DeletePreset(c *gin.Context) {
	userID := c.GetString("userID")
	presetID := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), presetID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// RegisterRoutes registers all preset-related routes on the given router group.
func (h *PresetHandler) RegisterRoutes(rg *gin.RouterGroup) {
	presets := rg.Group("/search/presets")
	{
		presets.GET("", h.ListPresets)
		presets.POST("", h.CreatePreset)
		presets.DELETE("/:id", h.DeletePreset)
	}
}
```

- [ ] 프론트엔드 Entity 타입 정의 (`frontend/src/entities/search-preset/model/types.ts`)

```typescript
// frontend/src/entities/search-preset/model/types.ts

/** SearchPreset — matches Go backend search_presets table */
export interface SearchPreset {
  id: string;
  userId: string;
  name: string;
  dsl: string;
  nlQuery: string | null;
  isPublic: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateSearchPresetInput {
  name: string;
  dsl: string;
  nlQuery?: string;
  isPublic?: boolean;
}
```

```typescript
// frontend/src/entities/search-preset/index.ts
export type { SearchPreset, CreateSearchPresetInput } from "./model/types";
```

- [ ] 테스트 먼저 작성 (`backend/internal/handler/preset_handler_test.go`)

```go
// backend/internal/handler/preset_handler_test.go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresetHandler_CreatePreset(t *testing.T) {
	t.Run("requires name and dsl fields", func(t *testing.T) {
		validPayload := map[string]interface{}{
			"name":    "2yr Max Volume",
			"dsl":     "scan where max_volume(730) == volume and trade_value >= 300000000000",
			"nlQuery": "2년 최대거래량 + 거래대금 3000억",
		}
		assert.NotEmpty(t, validPayload["name"])
		assert.NotEmpty(t, validPayload["dsl"])
	})

	t.Run("rejects missing name", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		// Handler would return 400 for missing required fields
		payload := map[string]string{"dsl": "scan where volume > 1000000"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/presets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		assert.NotNil(t, req)
	})
}

func TestPresetHandler_ListPresets(t *testing.T) {
	t.Run("returns paginated list format", func(t *testing.T) {
		expectedFormat := map[string]interface{}{
			"data": []interface{}{
				map[string]interface{}{
					"id": "uuid", "name": "preset name", "dsl": "scan ...", "nlQuery": nil, "createdAt": "date",
				},
			},
			"pagination": map[string]interface{}{
				"total": 1, "page": 1, "pageSize": 20, "totalPages": 1,
			},
		}
		data, ok := expectedFormat["data"].([]interface{})
		require.True(t, ok)
		assert.Len(t, data, 1)

		pagination, ok := expectedFormat["pagination"].(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, pagination, "total")
	})
}
```

- [ ] 프론트엔드 프리셋 테스트 (`frontend/src/entities/search-preset/__tests__/search-preset.test.ts`)

```typescript
// frontend/src/entities/search-preset/__tests__/search-preset.test.ts
describe("Search Presets", () => {
  it("POST /api/v1/search/presets requires name and dsl", () => {
    const validPayload = {
      name: "2yr Max Volume",
      dsl: "scan where max_volume(730) == volume and trade_value >= 300000000000",
      nlQuery: "2년 최대거래량 + 거래대금 3000억",
    };
    expect(validPayload.name).toBeTruthy();
    expect(validPayload.dsl).toBeTruthy();
  });

  it("GET /api/v1/search/presets returns list format", () => {
    const expectedFormat = {
      data: [
        { id: "uuid", name: "preset name", dsl: "scan ...", nlQuery: null, createdAt: "date" },
      ],
      pagination: { total: 1, page: 1, pageSize: 20, totalPages: 1 },
    };
    expect(expectedFormat.data).toBeInstanceOf(Array);
    expect(expectedFormat.pagination).toHaveProperty("total");
  });
});
```

- [ ] 프리셋 매니저 UI 컴포넌트 (`frontend/src/features/search/ui/PresetManager.tsx`) — 저장된 프리셋 목록 드롭다운, 이름 입력 + 저장 버튼, 삭제 버튼

- [ ] 마이그레이션 실행

```bash
cd /home/dev/code/dev-superbear/backend
go run cmd/migrate/main.go up
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/handler/... -v -run TestPreset
cd /home/dev/code/dev-superbear/frontend
npx jest src/entities/search-preset/__tests__/search-preset.test.ts
```

- [ ] 커밋

```bash
git add backend/db/ backend/internal/repository/ backend/internal/handler/preset_handler.go frontend/src/entities/search-preset/ frontend/src/features/search/ui/PresetManager.tsx
git commit -m "feat: search preset CRUD with Go sqlc repository and Gin handlers"
```

---

## Task 7: 검색 결과 테이블 + 차트 연동 (Stock Entity 활용)

검색 결과를 테이블로 표시하고, "Chart" 버튼으로 차트 페이지에 결과를 전달하는 기능을 구현한다. `chart-bridge-store`를 제거하고 `entities/stock/` 공유 엔티티를 사용한다.

**Files:**
- Create: `frontend/src/features/search/ui/SearchResults.tsx`
- Create: `frontend/src/features/search/ui/ResultsTable.tsx`
- Create: `frontend/src/entities/stock/model/stock-list.store.ts` (shared entity store — replaces chart-bridge-store)
- Create: `frontend/src/entities/stock/model/types.ts`
- Create: `frontend/src/entities/stock/index.ts`
- Test: `frontend/src/features/search/__tests__/SearchResults.test.tsx`

### Steps

- [ ] Stock 엔티티 도메인 타입 정의 (`frontend/src/entities/stock/model/types.ts`) — shared between search and chart features

```typescript
// frontend/src/entities/stock/model/types.ts

/** Stock information shared between search and chart features */
export interface StockInfo {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePct: number;
}

/** Stock list item — used in sidebars and search results */
export interface StockListItem {
  symbol: string;
  name: string;
  matchedValue?: number | string;
}
```

- [ ] Stock list 공유 store (`frontend/src/entities/stock/model/stock-list.store.ts`) — replaces chart-bridge-store, shared between features through entities/ layer

```typescript
// frontend/src/entities/stock/model/stock-list.store.ts
import { create } from "zustand";
import type { SearchResult } from "@/entities/search-result";

interface StockListState {
  /** 검색 결과에서 넘어온 종목 리스트 */
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;

  /** 선택된(활성) 종목 */
  selectedSymbol: string | null;
  setSelectedSymbol: (symbol: string | null) => void;

  /** 관심종목 */
  watchlist: SearchResult[];
  addToWatchlist: (item: SearchResult) => void;
  removeFromWatchlist: (symbol: string) => void;
  isInWatchlist: (symbol: string) => boolean;

  /** 최근 본 종목 */
  recentStocks: SearchResult[];
  addToRecent: (item: SearchResult) => void;
}

export const useStockListStore = create<StockListState>()((set, get) => ({
  searchResults: [],
  setSearchResults: (results) => set({ searchResults: results }),

  selectedSymbol: null,
  setSelectedSymbol: (symbol) => set({ selectedSymbol: symbol }),

  watchlist: [],
  addToWatchlist: (item) =>
    set((state) => ({
      watchlist: state.watchlist.some((w) => w.symbol === item.symbol)
        ? state.watchlist
        : [...state.watchlist, item],
    })),
  removeFromWatchlist: (symbol) =>
    set((state) => ({
      watchlist: state.watchlist.filter((w) => w.symbol !== symbol),
    })),
  isInWatchlist: (symbol) => get().watchlist.some((w) => w.symbol === symbol),

  recentStocks: [],
  addToRecent: (item) =>
    set((state) => {
      const filtered = state.recentStocks.filter((r) => r.symbol !== item.symbol);
      return { recentStocks: [item, ...filtered].slice(0, 30) }; // 최근 30개 유지
    }),
}));
```

```typescript
// frontend/src/entities/stock/index.ts
export type { StockInfo, StockListItem } from "./model/types";
export { useStockListStore } from "./model/stock-list.store";
```

- [ ] 테스트 먼저 작성 (`frontend/src/features/search/__tests__/SearchResults.test.tsx`)

```typescript
// frontend/src/features/search/__tests__/SearchResults.test.tsx
import { render, screen } from "@testing-library/react";
import { SearchResults } from "../ui/SearchResults";
import { useSearchStore } from "../model/search.store";

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("SearchResults", () => {
  it("shows empty state when no results", () => {
    render(<SearchResults />);
    expect(screen.getByText(/검색 결과가 없습니다|no results/i)).toBeInTheDocument();
  });

  it("renders results table with stock data", () => {
    useSearchStore.setState({
      results: [
        { symbol: "005930", name: "Samsung Electronics", matchedValue: 28400000 },
        { symbol: "247540", name: "ecoprobm", matchedValue: 15200000 },
      ],
    });
    render(<SearchResults />);
    expect(screen.getByText("Samsung Electronics")).toBeInTheDocument();
    expect(screen.getByText("ecoprobm")).toBeInTheDocument();
    expect(screen.getByText("005930")).toBeInTheDocument();
  });

  it("shows result count", () => {
    useSearchStore.setState({
      results: [
        { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      ],
    });
    render(<SearchResults />);
    expect(screen.getByText(/1/)).toBeInTheDocument();
  });

  it("renders Chart button for each row", () => {
    useSearchStore.setState({
      results: [
        { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      ],
    });
    render(<SearchResults />);
    expect(screen.getByRole("button", { name: /chart/i })).toBeInTheDocument();
  });

  it("shows loading state while searching", () => {
    useSearchStore.setState({ isSearching: true });
    render(<SearchResults />);
    expect(screen.getByText(/searching|검색 중/i)).toBeInTheDocument();
  });
});
```

- [ ] 결과 테이블 컴포넌트 구현 (`frontend/src/features/search/ui/ResultsTable.tsx`) — imports from entities/stock (not from other features). 종목코드, 종목명, 매칭값, 종가, 거래량, 등락률 컬럼 + Chart 버튼

```typescript
// frontend/src/features/search/ui/ResultsTable.tsx
"use client";

import { useRouter } from "next/navigation";
import { useSearchStore } from "../model/search.store";
import { useStockListStore } from "@/entities/stock";
import type { SearchResult } from "@/entities/search-result";

export function ResultsTable() {
  const router = useRouter();
  const { results } = useSearchStore();
  const { setSearchResults, setSelectedSymbol, addToRecent } = useStockListStore();

  const handleChartClick = (symbol: string) => {
    // 검색 결과 전체를 stock list entity store에 저장
    setSearchResults(results);
    setSelectedSymbol(symbol);
    addToRecent(results.find((r) => r.symbol === symbol)!);
    // 차트 페이지로 이동
    router.push(`/chart?symbol=${symbol}`);
  };

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-nexus-border text-nexus-text-secondary">
          <th className="text-left py-2 px-3">Code</th>
          <th className="text-left py-2 px-3">Name</th>
          <th className="text-right py-2 px-3">Matched Value</th>
          <th className="text-right py-2 px-3">Close</th>
          <th className="text-right py-2 px-3">Change %</th>
          <th className="text-center py-2 px-3"></th>
        </tr>
      </thead>
      <tbody>
        {results.map((row: SearchResult) => (
          <tr key={row.symbol} className="border-b border-nexus-border/50 hover:bg-nexus-border/20">
            <td className="py-2 px-3 font-mono text-nexus-text-secondary">{row.symbol}</td>
            <td className="py-2 px-3">{row.name}</td>
            <td className="py-2 px-3 text-right font-mono">{String(row.matchedValue)}</td>
            <td className="py-2 px-3 text-right font-mono">{row.close ?? "-"}</td>
            <td className="py-2 px-3 text-right font-mono">
              {row.changePct != null
                ? `${row.changePct > 0 ? "+" : ""}${row.changePct}%`
                : "-"}
            </td>
            <td className="py-2 px-3 text-center">
              <button
                onClick={() => handleChartClick(row.symbol)}
                aria-label="Chart"
                className="px-3 py-1 text-xs rounded bg-nexus-accent/20 text-nexus-accent
                           hover:bg-nexus-accent/30 transition-colors"
              >
                Chart
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
```

- [ ] 검색 결과 래퍼 컴포넌트 구현 (`frontend/src/features/search/ui/SearchResults.tsx`) — 빈 상태, 로딩 상태, 결과 수 표시, 테이블 렌더링

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/frontend
npx jest src/features/search/__tests__/SearchResults.test.tsx --env=jsdom
```

- [ ] 커밋

```bash
git add frontend/src/features/search/ frontend/src/entities/stock/
git commit -m "feat: search results table with stock entity store and chart navigation"
```
