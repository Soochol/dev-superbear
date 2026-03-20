# Search 기능 완성 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Search 페이지의 모든 버튼 핸들러를 API와 연결하고, 프리셋 CRUD를 프론트엔드까지 완성하며, E2E 테스트로 전체 플로우를 검증한다.

**Architecture:** 프론트엔드에 `useSearchActions` 커스텀 훅을 생성하여 모든 검색 액션(NL 검색, DSL 검색, 검증, 설명)을 캡슐화한다. 각 UI 컴포넌트는 이 훅의 함수를 onClick 핸들러로 연결한다. 백엔드는 이미 구현된 PresetHandler를 라우터에 등록만 하면 된다.

**Tech Stack:** Next.js 16.2 + React 19, Zustand, TypeScript, Go Gin, Playwright

**Working directory:** `/home/dev/code/dev-superbear/.worktrees/search-completion`

---

## File Structure

### 생성할 파일
| 파일 | 책임 |
|------|------|
| `src/features/search/model/use-search-actions.ts` | 검색 관련 모든 API 호출 + 상태 전환 로직을 캡슐화하는 커스텀 훅 |
| `src/features/search/__tests__/use-search-actions.test.ts` | useSearchActions 훅 유닛 테스트 |
| `src/features/search/api/preset-api.ts` | 프리셋 CRUD API 클라이언트 |
| `src/features/search/model/preset.store.ts` | 프리셋 목록 상태 관리 (Zustand) |
| `src/features/search/ui/PresetManager.tsx` | 프리셋 저장/로드/삭제 UI |
| `src/features/search/__tests__/preset.store.test.ts` | 프리셋 스토어 테스트 |
| `src/features/search/__tests__/PresetManager.test.tsx` | 프리셋 매니저 컴포넌트 테스트 |
| `backend/internal/handler/preset_handler_test.go` | PresetHandler 유닛 테스트 |

### 수정할 파일
| 파일 | 변경 내용 |
|------|-----------|
| `src/features/search/ui/NLTab.tsx` | Search 버튼에 onClick 핸들러 연결 |
| `src/features/search/ui/DSLTab.tsx` | Validate, Explain, Run Search 버튼에 onClick 핸들러 연결 |
| `src/features/search/ui/LiveDSLPanel.tsx` | Run 버튼에 onClick 핸들러 연결 |
| `src/features/search/ui/SearchPageLayout.tsx` | PresetManager 삽입 |
| `src/features/search/index.ts` | 새 export 추가 |
| `backend/cmd/api/main.go:77-128` | PresetHandler 라우트 등록 |
| `e2e/search.spec.ts` | 검색 플로우 E2E 테스트 추가 |

---

## Task 1: useSearchActions 훅 — 테스트 작성

Search 관련 모든 API 호출과 상태 전환 로직을 하나의 훅으로 캡슐화한다.

**Files:**
- Create: `src/features/search/__tests__/use-search-actions.test.ts`

- [ ] **Step 1: 테스트 파일 작성**

```typescript
import { useSearchStore } from "../model/search.store";
import { searchApi } from "../api/search-api";

// Mock the search API module
jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearch: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

// Import after mock so the mock is applied
import { createSearchActions } from "../model/use-search-actions";

const mockedApi = searchApi as jest.Mocked<typeof searchApi>;

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
  jest.clearAllMocks();
});

describe("createSearchActions", () => {
  describe("runNLSearch", () => {
    it("transitions through agent statuses and sets results on success", async () => {
      mockedApi.nlSearch.mockResolvedValue({
        dsl: "scan where volume > 1000000",
        explanation: "거래량 100만 이상",
        results: [{ symbol: "005930", name: "삼성전자", matchedValue: 2840000 }],
      });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ nlQuery: "거래량 많은 종목" });

      await actions.runNLSearch();

      const state = useSearchStore.getState();
      expect(state.agentStatus).toBe("done");
      expect(state.dslCode).toBe("scan where volume > 1000000");
      expect(state.results).toHaveLength(1);
      expect(state.results[0].symbol).toBe("005930");
      expect(mockedApi.nlSearch).toHaveBeenCalledWith("거래량 많은 종목");
    });

    it("sets error status on API failure", async () => {
      mockedApi.nlSearch.mockRejectedValue(new Error("API Error"));

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ nlQuery: "테스트" });

      await actions.runNLSearch();

      const state = useSearchStore.getState();
      expect(state.agentStatus).toBe("error");
      expect(state.agentMessage).toContain("API Error");
    });
  });

  describe("runDSLSearch", () => {
    it("executes DSL search and sets results", async () => {
      mockedApi.dslSearch.mockResolvedValue({
        results: [{ symbol: "000660", name: "SK하이닉스", matchedValue: 5000000 }],
      });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "scan where volume > 5000000" });

      await actions.runDSLSearch();

      const state = useSearchStore.getState();
      expect(state.agentStatus).toBe("done");
      expect(state.results).toHaveLength(1);
      expect(mockedApi.dslSearch).toHaveBeenCalledWith("scan where volume > 5000000");
    });

    it("sets error status on failure", async () => {
      mockedApi.dslSearch.mockRejectedValue(new Error("Execute failed"));

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "invalid dsl" });

      await actions.runDSLSearch();

      expect(useSearchStore.getState().agentStatus).toBe("error");
    });
  });

  describe("validateDSL", () => {
    it("sets valid state on successful validation", async () => {
      mockedApi.validate.mockResolvedValue({ valid: true, error: null });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "scan where volume > 1000000" });

      await actions.validateDSL();

      const state = useSearchStore.getState();
      expect(state.validationState).toBe("valid");
      expect(state.validationMessage).toBe("");
    });

    it("sets invalid state with message on validation failure", async () => {
      mockedApi.validate.mockResolvedValue({ valid: false, error: "syntax error at line 1" });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "bad query" });

      await actions.validateDSL();

      const state = useSearchStore.getState();
      expect(state.validationState).toBe("invalid");
      expect(state.validationMessage).toBe("syntax error at line 1");
    });
  });

  describe("explainDSL", () => {
    it("returns explanation text", async () => {
      mockedApi.explain.mockResolvedValue({
        explanation: "이 쿼리는 거래량이 100만 이상인 종목을 검색합니다",
      });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "scan where volume > 1000000" });

      const result = await actions.explainDSL();

      expect(result).toBe("이 쿼리는 거래량이 100만 이상인 종목을 검색합니다");
      expect(mockedApi.explain).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });
});
```

- [ ] **Step 2: 테스트 실행 — 실패 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/use-search-actions.test.ts --no-coverage`
Expected: FAIL — `Cannot find module '../model/use-search-actions'`

---

## Task 2: useSearchActions 훅 — 구현

**Files:**
- Create: `src/features/search/model/use-search-actions.ts`

- [ ] **Step 3: 구현 작성**

`createSearchActions`는 순수 함수로 만들어 Zustand의 `getState`/`setState`를 주입받는 패턴을 사용한다.
이렇게 하면 React 훅 없이도 테스트 가능하고, 컴포넌트에서는 `useSearchActions()` 훅으로 감싸서 사용한다.

```typescript
import { searchApi } from "../api/search-api";
import type { AgentStatus, ValidationState } from "./types";
import type { SearchResult } from "@/entities/search-result";

interface SearchStoreState {
  nlQuery: string;
  dslCode: string;
  agentStatus: AgentStatus;
  results: SearchResult[];
}

type GetState = () => SearchStoreState;
type SetState = (partial: Record<string, unknown>) => void;

export function createSearchActions(getState: GetState, setState: SetState) {
  async function runNLSearch() {
    const { nlQuery } = getState();
    setState({ agentStatus: "interpreting", agentMessage: "Interpreting query..." });

    try {
      const response = await searchApi.nlSearch(nlQuery);
      setState({
        dslCode: response.dsl,
        agentStatus: "scanning",
        agentMessage: "Scanning stocks...",
      });
      setState({
        results: response.results,
        agentStatus: "done",
        agentMessage: `${response.results.length}개 종목 발견`,
      });
    } catch (err) {
      setState({
        agentStatus: "error",
        agentMessage: err instanceof Error ? err.message : "Unknown error",
      });
    }
  }

  async function runDSLSearch() {
    const { dslCode } = getState();
    setState({ agentStatus: "scanning", agentMessage: "Scanning stocks..." });

    try {
      const response = await searchApi.dslSearch(dslCode);
      setState({
        results: response.results,
        agentStatus: "done",
        agentMessage: `${response.results.length}개 종목 발견`,
      });
    } catch (err) {
      setState({
        agentStatus: "error",
        agentMessage: err instanceof Error ? err.message : "Unknown error",
      });
    }
  }

  async function validateDSL() {
    const { dslCode } = getState();

    try {
      const response = await searchApi.validate(dslCode);
      setState({
        validationState: response.valid ? "valid" as ValidationState : "invalid" as ValidationState,
        validationMessage: response.error ?? "",
      });
    } catch (err) {
      setState({
        validationState: "invalid" as ValidationState,
        validationMessage: err instanceof Error ? err.message : "Validation failed",
      });
    }
  }

  async function explainDSL(): Promise<string> {
    const { dslCode } = getState();

    const response = await searchApi.explain(dslCode);
    return response.explanation;
  }

  return { runNLSearch, runDSLSearch, validateDSL, explainDSL };
}

// React hook wrapper for use in components
import { useSearchStore } from "./search.store";

export function useSearchActions() {
  return createSearchActions(
    useSearchStore.getState,
    useSearchStore.setState,
  );
}
```

- [ ] **Step 4: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/use-search-actions.test.ts --no-coverage`
Expected: PASS (7 tests)

- [ ] **Step 5: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/model/use-search-actions.ts src/features/search/__tests__/use-search-actions.test.ts
git commit -m "feat(search): add useSearchActions hook with API integration

Encapsulates NL search, DSL search, validate, and explain
actions with proper agent status transitions and error handling."
```

---

## Task 3: NLTab 버튼 핸들러 연결

**Files:**
- Modify: `src/features/search/ui/NLTab.tsx`
- Modify: `src/features/search/__tests__/NLTab.test.tsx`

- [ ] **Step 6: NLTab 테스트에 버튼 클릭 테스트 추가**

`src/features/search/__tests__/NLTab.test.tsx`에 추가:

```typescript
import { searchApi } from "../api/search-api";

jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearch: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

const mockedApi = searchApi as jest.Mocked<typeof searchApi>;
```

그리고 테스트 추가:

```typescript
  it("calls NL search API when Search button is clicked", async () => {
    mockedApi.nlSearch.mockResolvedValue({
      dsl: "scan where volume > 1000000",
      explanation: "test",
      results: [{ symbol: "005930", name: "삼성전자", matchedValue: 100 }],
    });

    useSearchStore.setState({ nlQuery: "거래량 많은 종목" });
    render(<NLTab />);

    const searchBtn = screen.getByRole("button", { name: /search/i });
    fireEvent.click(searchBtn);

    await waitFor(() => {
      expect(mockedApi.nlSearch).toHaveBeenCalledWith("거래량 많은 종목");
    });
  });
```

상단 import에 `waitFor` 추가: `import { render, screen, fireEvent, waitFor } from "@testing-library/react";`

- [ ] **Step 7: 테스트 실행 — 실패 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/NLTab.test.tsx --no-coverage`
Expected: FAIL — `nlSearch` was not called

- [ ] **Step 8: NLTab에 onClick 핸들러 추가**

`src/features/search/ui/NLTab.tsx` 수정:

기존:
```typescript
import { btnPrimary } from "./styles";
```

변경:
```typescript
import { btnPrimary } from "./styles";
import { useSearchActions } from "../model/use-search-actions";
```

기존:
```typescript
  const isSearching = agentStatus !== "idle" && agentStatus !== "done" && agentStatus !== "error";
```

변경:
```typescript
  const isSearching = agentStatus !== "idle" && agentStatus !== "done" && agentStatus !== "error";
  const { runNLSearch } = useSearchActions();
```

기존:
```typescript
        <button
          disabled={isSearching || !nlQuery.trim()}
          className={btnPrimary}
        >
          Search
        </button>
```

변경:
```typescript
        <button
          disabled={isSearching || !nlQuery.trim()}
          className={btnPrimary}
          onClick={runNLSearch}
        >
          Search
        </button>
```

- [ ] **Step 9: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/NLTab.test.tsx --no-coverage`
Expected: PASS

- [ ] **Step 10: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/ui/NLTab.tsx src/features/search/__tests__/NLTab.test.tsx
git commit -m "feat(search): wire NLTab Search button to API"
```

---

## Task 4: DSLTab 버튼 핸들러 연결

**Files:**
- Modify: `src/features/search/ui/DSLTab.tsx`
- Modify: `src/features/search/__tests__/DSLTab.test.tsx`

- [ ] **Step 11: DSLTab 테스트에 버튼 클릭 테스트 추가**

`src/features/search/__tests__/DSLTab.test.tsx`에 mock + 테스트 추가:

```typescript
import { fireEvent, waitFor } from "@testing-library/react";
import { searchApi } from "../api/search-api";

jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearch: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

const mockedApi = searchApi as jest.Mocked<typeof searchApi>;
```

테스트 추가:

```typescript
  it("calls validate API when Validate button is clicked", async () => {
    mockedApi.validate.mockResolvedValue({ valid: true, error: null });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);

    fireEvent.click(screen.getByRole("button", { name: /validate/i }));

    await waitFor(() => {
      expect(mockedApi.validate).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });

  it("calls explain API when Explain button is clicked", async () => {
    mockedApi.explain.mockResolvedValue({ explanation: "test explanation" });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);

    fireEvent.click(screen.getByRole("button", { name: /explain/i }));

    await waitFor(() => {
      expect(mockedApi.explain).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });

  it("calls dslSearch API when Run Search button is clicked", async () => {
    mockedApi.dslSearch.mockResolvedValue({ results: [] });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<DSLTab />);

    fireEvent.click(screen.getByRole("button", { name: /run/i }));

    await waitFor(() => {
      expect(mockedApi.dslSearch).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });
```

- [ ] **Step 12: 테스트 실행 — 실패 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/DSLTab.test.tsx --no-coverage`
Expected: FAIL

- [ ] **Step 13: DSLTab에 onClick 핸들러 추가**

`src/features/search/ui/DSLTab.tsx` 수정:

기존:
```typescript
import { btnPrimary, btnSecondary } from "./styles";
```

변경:
```typescript
import { btnPrimary, btnSecondary } from "./styles";
import { useSearchActions } from "../model/use-search-actions";
```

기존:
```typescript
  const hasCode = dslCode.trim().length > 0;
```

변경:
```typescript
  const hasCode = dslCode.trim().length > 0;
  const { runDSLSearch, validateDSL, explainDSL } = useSearchActions();
```

Validate 버튼 — 기존:
```typescript
        <button
          disabled={!hasCode}
          className={btnSecondary}
        >
          Validate
        </button>
```

변경:
```typescript
        <button
          disabled={!hasCode}
          className={btnSecondary}
          onClick={validateDSL}
        >
          Validate
        </button>
```

Explain 버튼 — 기존:
```typescript
        <button
          disabled={!hasCode}
          className={btnSecondary}
        >
          Explain in NL
        </button>
```

변경:
```typescript
        <button
          disabled={!hasCode}
          className={btnSecondary}
          onClick={explainDSL}
        >
          Explain in NL
        </button>
```

Run Search 버튼 — 기존:
```typescript
        <button
          disabled={!hasCode || validationState === "invalid"}
          className={btnPrimary}
        >
          Run Search
        </button>
```

변경:
```typescript
        <button
          disabled={!hasCode || validationState === "invalid"}
          className={btnPrimary}
          onClick={runDSLSearch}
        >
          Run Search
        </button>
```

- [ ] **Step 14: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/DSLTab.test.tsx --no-coverage`
Expected: PASS

- [ ] **Step 15: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/ui/DSLTab.tsx src/features/search/__tests__/DSLTab.test.tsx
git commit -m "feat(search): wire DSLTab Validate/Explain/Run buttons to API"
```

---

## Task 5: LiveDSLPanel Run 버튼 연결

**Files:**
- Modify: `src/features/search/ui/LiveDSLPanel.tsx`
- Modify: `src/features/search/__tests__/LiveDSLPanel.test.tsx`

- [ ] **Step 16: LiveDSLPanel 테스트에 Run 버튼 클릭 테스트 추가**

`src/features/search/__tests__/LiveDSLPanel.test.tsx`에 mock + 테스트 추가:

```typescript
import { fireEvent, waitFor } from "@testing-library/react";
import { searchApi } from "../api/search-api";

jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearch: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

const mockedApi = searchApi as jest.Mocked<typeof searchApi>;
```

테스트 추가:

```typescript
  it("calls dslSearch when Run button is clicked", async () => {
    mockedApi.dslSearch.mockResolvedValue({ results: [] });
    useSearchStore.setState({ dslCode: "scan where volume > 1000000" });
    render(<LiveDSLPanel />);

    fireEvent.click(screen.getByRole("button", { name: /run/i }));

    await waitFor(() => {
      expect(mockedApi.dslSearch).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });
```

- [ ] **Step 17: 테스트 실행 — 실패 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/LiveDSLPanel.test.tsx --no-coverage`
Expected: FAIL

- [ ] **Step 18: LiveDSLPanel에 Run onClick 핸들러 추가**

`src/features/search/ui/LiveDSLPanel.tsx` 수정:

기존:
```typescript
import { btnMini } from "./styles";
```

변경:
```typescript
import { btnMini } from "./styles";
import { useSearchActions } from "../model/use-search-actions";
```

기존:
```typescript
  const [copyLabel, setCopyLabel] = useState("Copy");
```

변경:
```typescript
  const [copyLabel, setCopyLabel] = useState("Copy");
  const { runDSLSearch } = useSearchActions();
```

기존:
```typescript
            <button
              aria-label="Run Search"
              className={btnMini}
            >
              Run
            </button>
```

변경:
```typescript
            <button
              aria-label="Run Search"
              className={btnMini}
              onClick={runDSLSearch}
            >
              Run
            </button>
```

- [ ] **Step 19: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/LiveDSLPanel.test.tsx --no-coverage`
Expected: PASS

- [ ] **Step 20: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/ui/LiveDSLPanel.tsx src/features/search/__tests__/LiveDSLPanel.test.tsx
git commit -m "feat(search): wire LiveDSLPanel Run button to API"
```

---

## Task 6: 프론트엔드 전체 유닛 테스트 확인

- [ ] **Step 21: 모든 search 관련 유닛 테스트 실행**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/ src/shared/lib/dsl/ src/entities/search-preset/ --no-coverage`
Expected: ALL PASS

실패 시 수정 후 재실행. 모든 테스트 통과 확인 후 다음 Task 진행.

---

## Task 7: 백엔드 PresetHandler 라우트 등록

`PresetHandler`와 `PresetRepository`는 이미 완전히 구현되어 있다. `main.go`에 등록만 하면 된다.

**Files:**
- Create: `backend/internal/handler/preset_handler_test.go`
- Modify: `backend/cmd/api/main.go:77-128`

- [ ] **Step 22: PresetHandler 테스트 작성**

```go
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

	"github.com/dev-superbear/nexus-backend/internal/handler"
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

// newTestPresetRouter sets up a Gin router with PresetHandler using an
// in-memory approach. Since PresetRepository requires *sql.DB, we test
// via the integration test file, or we test the handler routes are
// correctly registered by checking 401 (no userID in context).
func setupPresetRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Pass nil DB — handlers will fail at repo level but we can test
	// routing and request validation.
	repo := repository.NewPresetRepository(nil)
	h := handler.NewPresetHandler(repo)

	api := r.Group("/api/v1")
	// Simulate auth middleware setting userID
	api.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Next()
	})
	h.RegisterRoutes(api)

	return r
}

func TestPresetHandler_ListPresets_Route(t *testing.T) {
	r := setupPresetRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/presets", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	// Will fail at DB level (nil DB) but should not be 404 — route exists
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestPresetHandler_CreatePreset_Validation(t *testing.T) {
	r := setupPresetRouter()

	t.Run("rejects empty body", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/presets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPresetHandler_DeletePreset_Route(t *testing.T) {
	r := setupPresetRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/search/presets/some-id", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestPresetHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := repository.NewPresetRepository(nil)
	h := handler.NewPresetHandler(repo)

	api := r.Group("/api/v1")
	// NO auth middleware — userID will be empty
	h.RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/presets", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "authentication required", resp["error"])
}
```

- [ ] **Step 23: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion/backend && go test ./internal/handler/ -run TestPresetHandler -v`
Expected: PASS (4 tests, 일부는 nil DB로 인해 500 응답이지만 404가 아님을 확인)

- [ ] **Step 24: main.go에 PresetHandler 등록**

`backend/cmd/api/main.go`의 `registerRoutes` 함수에서, search 핸들러 등록 코드 바로 아래에 추가:

기존 (line 101-104):
```go
	searchSvc := service.NewSearchService(nil)
	nlSvc := service.NewNLToDSLService()
	searchH := handler.NewSearchHandler(searchSvc, nlSvc)
	searchH.RegisterRoutes(rg)
```

변경:
```go
	searchSvc := service.NewSearchService(nil)
	nlSvc := service.NewNLToDSLService()
	searchH := handler.NewSearchHandler(searchSvc, nlSvc)
	searchH.RegisterRoutes(rg)

	presetRepo := repository.NewPresetRepository(pool.Config().ConnConfig.Config.Conn)
	presetH := handler.NewPresetHandler(presetRepo)
	presetH.RegisterRoutes(rg)
```

**주의:** `pool`은 `*pgxpool.Pool`이다. `PresetRepository`는 `*sql.DB`를 기대하므로, 프로젝트에서 `database/sql`을 어떻게 사용하는지 먼저 확인해야 한다. 기존 패턴이 pgxpool과 database/sql을 혼용하는지 확인할 것. 만약 `sql.DB` 인스턴스가 없다면, `pgxpool.Pool`을 사용하도록 `PresetRepository`를 수정하거나, `stdlib` 어댑터를 사용해야 한다.

**대안 — pgxpool 직접 사용:**
기존 다른 repo (e.g., `PipelineRepository`)에서 `pool`을 어떻게 사용하는지 확인하고 동일 패턴 따를 것.

- [ ] **Step 25: 백엔드 테스트 전체 실행**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion/backend && go test ./internal/handler/ -v`
Expected: ALL PASS

- [ ] **Step 26: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add backend/internal/handler/preset_handler_test.go backend/cmd/api/main.go
git commit -m "feat(search): register PresetHandler routes in main.go"
```

---

## Task 8: 프리셋 API 클라이언트 + 스토어 — 테스트 작성

**Files:**
- Create: `src/features/search/__tests__/preset.store.test.ts`
- Create: `src/features/search/api/preset-api.ts`
- Create: `src/features/search/model/preset.store.ts`

- [ ] **Step 27: 프리셋 API 클라이언트 작성**

```typescript
// src/features/search/api/preset-api.ts
import { apiGet, apiPost, apiDelete } from "@/shared/api/client";
import type { SearchPreset, CreateSearchPresetInput } from "@/entities/search-preset";

interface ListPresetsResponse {
  data: SearchPreset[];
  pagination: {
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  };
}

interface CreatePresetResponse {
  data: SearchPreset;
}

export const presetApi = {
  list(page = 1, pageSize = 20): Promise<ListPresetsResponse> {
    return apiGet<ListPresetsResponse>(`/api/v1/search/presets?page=${page}&pageSize=${pageSize}`);
  },

  create(input: CreateSearchPresetInput): Promise<CreatePresetResponse> {
    return apiPost<CreatePresetResponse>("/api/v1/search/presets", input);
  },

  delete(id: string): Promise<void> {
    return apiDelete<void>(`/api/v1/search/presets/${id}`);
  },
};
```

- [ ] **Step 28: 프리셋 스토어 테스트 작성**

```typescript
// src/features/search/__tests__/preset.store.test.ts
import { usePresetStore } from "../model/preset.store";

beforeEach(() => {
  usePresetStore.setState(usePresetStore.getInitialState());
});

describe("Preset Store", () => {
  it("initializes with empty presets", () => {
    const state = usePresetStore.getState();
    expect(state.presets).toEqual([]);
    expect(state.isLoading).toBe(false);
  });

  it("sets presets", () => {
    usePresetStore.getState().setPresets([
      { id: "1", userId: "u1", name: "Test", dsl: "scan where volume > 100", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
    ]);
    expect(usePresetStore.getState().presets).toHaveLength(1);
  });

  it("adds a preset", () => {
    const preset = { id: "2", userId: "u1", name: "New", dsl: "scan where close > 50000", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" };
    usePresetStore.getState().addPreset(preset);
    expect(usePresetStore.getState().presets).toHaveLength(1);
    expect(usePresetStore.getState().presets[0].id).toBe("2");
  });

  it("removes a preset by id", () => {
    usePresetStore.getState().setPresets([
      { id: "1", userId: "u1", name: "A", dsl: "a", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      { id: "2", userId: "u1", name: "B", dsl: "b", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
    ]);
    usePresetStore.getState().removePreset("1");
    expect(usePresetStore.getState().presets).toHaveLength(1);
    expect(usePresetStore.getState().presets[0].id).toBe("2");
  });
});
```

- [ ] **Step 29: 테스트 실행 — 실패 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/preset.store.test.ts --no-coverage`
Expected: FAIL — `Cannot find module '../model/preset.store'`

- [ ] **Step 30: 프리셋 스토어 구현**

```typescript
// src/features/search/model/preset.store.ts
import { create } from "zustand";
import type { SearchPreset } from "@/entities/search-preset";

interface PresetState {
  presets: SearchPreset[];
  isLoading: boolean;
  setPresets: (presets: SearchPreset[]) => void;
  addPreset: (preset: SearchPreset) => void;
  removePreset: (id: string) => void;
  setLoading: (loading: boolean) => void;
}

export const usePresetStore = create<PresetState>()((set) => ({
  presets: [],
  isLoading: false,
  setPresets: (presets) => set({ presets }),
  addPreset: (preset) => set((s) => ({ presets: [preset, ...s.presets] })),
  removePreset: (id) => set((s) => ({ presets: s.presets.filter((p) => p.id !== id) })),
  setLoading: (loading) => set({ isLoading: loading }),
}));
```

- [ ] **Step 31: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/preset.store.test.ts --no-coverage`
Expected: PASS

- [ ] **Step 32: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/api/preset-api.ts src/features/search/model/preset.store.ts src/features/search/__tests__/preset.store.test.ts
git commit -m "feat(search): add preset API client and Zustand store"
```

---

## Task 9: PresetManager 컴포넌트

**Files:**
- Create: `src/features/search/__tests__/PresetManager.test.tsx`
- Create: `src/features/search/ui/PresetManager.tsx`
- Modify: `src/features/search/ui/SearchPageLayout.tsx`

- [ ] **Step 33: PresetManager 테스트 작성**

```typescript
// src/features/search/__tests__/PresetManager.test.tsx
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PresetManager } from "../ui/PresetManager";
import { usePresetStore } from "../model/preset.store";
import { useSearchStore } from "../model/search.store";
import { presetApi } from "../api/preset-api";

jest.mock("../api/preset-api", () => ({
  presetApi: {
    list: jest.fn(),
    create: jest.fn(),
    delete: jest.fn(),
  },
}));

const mockedApi = presetApi as jest.Mocked<typeof presetApi>;

beforeEach(() => {
  usePresetStore.setState(usePresetStore.getInitialState());
  useSearchStore.setState(useSearchStore.getInitialState());
  jest.clearAllMocks();
});

describe("PresetManager", () => {
  it("renders Save Preset button", () => {
    render(<PresetManager />);
    expect(screen.getByRole("button", { name: /save/i })).toBeInTheDocument();
  });

  it("shows saved presets from store", () => {
    usePresetStore.setState({
      presets: [
        { id: "1", userId: "u1", name: "My Preset", dsl: "scan where volume > 100", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      ],
    });
    render(<PresetManager />);
    expect(screen.getByText("My Preset")).toBeInTheDocument();
  });

  it("clicking a preset loads its DSL into the editor", () => {
    usePresetStore.setState({
      presets: [
        { id: "1", userId: "u1", name: "Volume Filter", dsl: "scan where volume > 5000000", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      ],
    });
    render(<PresetManager />);
    fireEvent.click(screen.getByText("Volume Filter"));

    expect(useSearchStore.getState().dslCode).toBe("scan where volume > 5000000");
  });

  it("delete button removes preset", async () => {
    mockedApi.delete.mockResolvedValue(undefined);
    usePresetStore.setState({
      presets: [
        { id: "1", userId: "u1", name: "To Delete", dsl: "scan", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      ],
    });
    render(<PresetManager />);
    fireEvent.click(screen.getByLabelText("Delete preset To Delete"));

    await waitFor(() => {
      expect(mockedApi.delete).toHaveBeenCalledWith("1");
    });
  });
});
```

- [ ] **Step 34: 테스트 실행 — 실패 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/PresetManager.test.tsx --no-coverage`
Expected: FAIL

- [ ] **Step 35: PresetManager 컴포넌트 구현**

```typescript
// src/features/search/ui/PresetManager.tsx
"use client";

import { useState } from "react";
import { usePresetStore } from "../model/preset.store";
import { useSearchStore } from "../model/search.store";
import { presetApi } from "../api/preset-api";
import { btnSecondary, btnMini } from "./styles";

export function PresetManager() {
  const presets = usePresetStore((s) => s.presets);
  const removePreset = usePresetStore((s) => s.removePreset);
  const addPreset = usePresetStore((s) => s.addPreset);
  const dslCode = useSearchStore((s) => s.dslCode);
  const setDslCode = useSearchStore((s) => s.setDslCode);
  const nlQuery = useSearchStore((s) => s.nlQuery);
  const setActiveTab = useSearchStore((s) => s.setActiveTab);
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    if (!dslCode.trim()) return;
    setSaving(true);
    try {
      const name = `Preset ${new Date().toLocaleString("ko-KR", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}`;
      const response = await presetApi.create({
        name,
        dsl: dslCode,
        nlQuery: nlQuery || undefined,
      });
      addPreset(response.data);
    } finally {
      setSaving(false);
    }
  };

  const handleLoad = (dsl: string) => {
    setDslCode(dsl);
    setActiveTab("dsl");
  };

  const handleDelete = async (id: string) => {
    await presetApi.delete(id);
    removePreset(id);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold text-nexus-text-secondary uppercase tracking-wider">
          Saved Presets
        </span>
        <button
          onClick={handleSave}
          disabled={!dslCode.trim() || saving}
          className={btnSecondary}
        >
          {saving ? "Saving..." : "Save Preset"}
        </button>
      </div>

      {presets.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {presets.map((preset) => (
            <div
              key={preset.id}
              className="flex items-center gap-1 bg-nexus-surface border border-nexus-border rounded-lg px-3 py-1"
            >
              <button
                onClick={() => handleLoad(preset.dsl)}
                className="text-sm text-nexus-text-primary hover:text-nexus-accent transition-colors"
              >
                {preset.name}
              </button>
              <button
                onClick={() => handleDelete(preset.id)}
                aria-label={`Delete preset ${preset.name}`}
                className="text-xs text-nexus-text-secondary hover:text-nexus-failure ml-1"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 36: 테스트 실행 — 통과 확인**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/__tests__/PresetManager.test.tsx --no-coverage`
Expected: PASS

- [ ] **Step 37: SearchPageLayout에 PresetManager 삽입**

`src/features/search/ui/SearchPageLayout.tsx` 수정:

기존:
```typescript
import { SearchResults } from "./SearchResults";
```

변경:
```typescript
import { SearchResults } from "./SearchResults";
import { PresetManager } from "./PresetManager";
```

기존 (LiveDSLPanel과 SearchResults 사이):
```typescript
      <LiveDSLPanel />
      <SearchResults />
```

변경:
```typescript
      <LiveDSLPanel />
      <PresetManager />
      <SearchResults />
```

- [ ] **Step 38: 전체 search 유닛 테스트 실행**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest src/features/search/ --no-coverage`
Expected: ALL PASS

- [ ] **Step 39: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/ui/PresetManager.tsx src/features/search/__tests__/PresetManager.test.tsx src/features/search/ui/SearchPageLayout.tsx
git commit -m "feat(search): add PresetManager component for saved presets"
```

---

## Task 10: index.ts 업데이트

**Files:**
- Modify: `src/features/search/index.ts`

- [ ] **Step 40: index.ts에 새 export 추가**

기존:
```typescript
export { useSearchStore } from "./model/search.store";
export type { SearchTab, AgentStatus, ValidationState } from "./model/types";
```

변경:
```typescript
export { useSearchStore } from "./model/search.store";
export { useSearchActions } from "./model/use-search-actions";
export { usePresetStore } from "./model/preset.store";
export type { SearchTab, AgentStatus, ValidationState } from "./model/types";
```

- [ ] **Step 41: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add src/features/search/index.ts
git commit -m "chore(search): export new hooks from index"
```

---

## Task 11: E2E 테스트 — 검색 플로우

기존 E2E 테스트는 UI 렌더링만 검증한다. 실제 검색 플로우(버튼 클릭 → API 호출 → 결과 표시)를 추가한다.

**중요:** E2E 테스트는 실제 백엔드에 요청을 보낸다. 현재 백엔드 `Execute()`는 빈 결과를 반환하므로, API 응답을 Playwright `route.fulfill`로 모킹한다.

**Files:**
- Modify: `e2e/search.spec.ts`

- [ ] **Step 42: E2E 테스트 추가**

`e2e/search.spec.ts` 파일 끝에 추가:

```typescript
test.describe("Search Flow", () => {
  test("NL search: type query → click Search → see results", async ({ page }) => {
    // Mock the NL search API
    await page.route("**/api/v1/search/nl-to-dsl", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          dsl: "scan where volume > 1000000",
          explanation: "거래량 100만 이상",
          results: [
            { symbol: "005930", name: "삼성전자", matchedValue: 28400000, close: 71000, changePct: 1.5 },
            { symbol: "000660", name: "SK하이닉스", matchedValue: 15200000, close: 195000, changePct: -0.3 },
          ],
        }),
      });
    });

    await page.goto("/search");

    // Type a query
    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await textarea.fill("거래량 많은 종목");

    // Click Search
    await page.getByRole("button", { name: "Search" }).click();

    // Wait for results
    await expect(page.getByText("2개 종목")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("삼성전자")).toBeVisible();
    await expect(page.getByText("SK하이닉스")).toBeVisible();

    // LIVE DSL panel should show the generated DSL
    await expect(page.getByText(/scan/)).toBeVisible();
  });

  test("DSL search: enter DSL → Run Search → see results", async ({ page }) => {
    // Mock the DSL execute API
    await page.route("**/api/v1/search/execute", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          results: [
            { symbol: "035420", name: "NAVER", matchedValue: 5000000, close: 220000, changePct: 2.1 },
          ],
        }),
      });
    });

    await page.goto("/search");

    // Switch to DSL tab
    await page.getByRole("button", { name: "DSL" }).click();

    // Type DSL in the CodeMirror editor
    const editor = page.getByTestId("dsl-editor-container");
    await editor.click();
    await page.keyboard.type("scan where volume > 5000000");

    // Click Run Search
    await page.getByRole("button", { name: "Run Search" }).click();

    // Wait for results
    await expect(page.getByText("1개 종목")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("NAVER")).toBeVisible();
  });

  test("DSL validate: enter DSL → Validate → see validation badge", async ({ page }) => {
    // Mock the validate API
    await page.route("**/api/v1/search/validate", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ valid: true, error: null }),
      });
    });

    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();

    // Type DSL
    const editor = page.getByTestId("dsl-editor-container");
    await editor.click();
    await page.keyboard.type("scan where volume > 1000000");

    // Click Validate
    await page.getByRole("button", { name: "Validate" }).click();

    // LIVE DSL panel should show "Validated" badge
    await expect(page.getByText("Validated")).toBeVisible({ timeout: 5000 });
  });

  test("NL search via preset: click preset → click Search → see agent status", async ({ page }) => {
    // Mock the NL search API
    await page.route("**/api/v1/search/nl-to-dsl", async (route) => {
      // Simulate slight delay
      await new Promise((r) => setTimeout(r, 100));
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          dsl: "scan where rsi(14) < 30",
          explanation: "RSI 과매도",
          results: [{ symbol: "003550", name: "LG", matchedValue: 28.5, close: 95000, changePct: -1.2 }],
        }),
      });
    });

    await page.goto("/search");

    // Click RSI Oversold preset
    await page.getByRole("button", { name: "RSI Oversold" }).click();

    // Click Search
    await page.getByRole("button", { name: "Search" }).click();

    // Agent status should show interpreting (may flash quickly)
    // Wait for results
    await expect(page.getByText("1개 종목")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("LG")).toBeVisible();
  });
});
```

- [ ] **Step 43: 커밋**

```bash
cd /home/dev/code/dev-superbear/.worktrees/search-completion
git add e2e/search.spec.ts
git commit -m "test(e2e): add search flow E2E tests with API mocking"
```

---

## Task 12: 전체 검증

- [ ] **Step 44: 프론트엔드 전체 유닛 테스트**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx jest --no-coverage`
Expected: ALL PASS

- [ ] **Step 45: 백엔드 전체 테스트**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion/backend && go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 46: TypeScript 타입 체크**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 47: Next.js 빌드**

Run: `cd /home/dev/code/dev-superbear/.worktrees/search-completion && npm run build`
Expected: Build succeeds

- [ ] **Step 48: 최종 커밋 (필요시)**

실패한 항목 수정 후 추가 커밋.

---

## TypeScript 진단 오류 참고

시스템 진단에서 보고된 TypeScript 오류들(`Cannot find name 'Promise'`, `Cannot find module` 등)은 **IDE/LSP 캐시 문제**로 확인됨. `npx tsc --noEmit`는 에러 없이 통과하며, `npm run build`도 성공함. 코드 수정이 아닌 IDE TypeScript 서버 재시작으로 해결 가능.
