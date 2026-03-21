# NL-to-DSL LLM Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Search 페이지에서 `claude -p` + MCP 도구로 자연어→DSL 변환을 구현하고, DSL 에디터에 문맥 인식 자동완성 + 실시간 lint를 추가한다.

**Architecture:** Go 백엔드에 `LLMProvider` 인터페이스를 도입하고, 1차 구현체로 `ClaudeCLIProvider`(subprocess)를 만든다. MCP 서버는 별도 Go 바이너리로 DSL 문법 참조/검증 도구를 제공. 프론트는 SSE로 실시간 이벤트를 수신.

**Tech Stack:** Go 1.25 (Gin), Next.js 16.2, React 19, CodeMirror 6, Zustand, `claude -p` CLI

**Spec:** `docs/superpowers/specs/2026-03-21-nl-to-dsl-llm-design.md`

---

## File Map

### Backend — New Files

| File | Responsibility |
|------|---------------|
| `backend/internal/llm/provider.go` | `Provider` interface, `Event`/`EventType` 타입 |
| `backend/internal/llm/provider_test.go` | Event 타입 검증 테스트 |
| `backend/internal/llm/claudecli/provider.go` | `claude -p` subprocess 실행, stdout 파싱, Event 채널 변환 |
| `backend/internal/llm/claudecli/provider_test.go` | mock subprocess로 이벤트 스트림 검증 |
| `backend/internal/mcp/server.go` | JSON-RPC 2.0 over stdio MCP 서버 |
| `backend/internal/mcp/server_test.go` | JSON-RPC 요청/응답 검증 |
| `backend/internal/mcp/tools.go` | 3개 도구 핸들러 (get_dsl_grammar, list_available_fields, validate_dsl) |
| `backend/internal/mcp/tools_test.go` | 각 도구 출력 검증 |
| `backend/cmd/mcp-server/main.go` | MCP 서버 바이너리 진입점 |
| `backend/internal/llm/prompts/nl-to-dsl.txt` | NL→DSL 시스템 프롬프트 |
| `backend/internal/llm/prompts/explain.txt` | DSL 설명 시스템 프롬프트 |
| `backend/mcp-config.json` | claude -p MCP 서버 설정 |

### Backend — Modified Files

| File | Change |
|------|--------|
| `backend/internal/config/config.go` | `LLMConfig` 구조체 + 환경변수 로딩 추가 |
| `backend/internal/service/nl_to_dsl_service.go` | `Provider` 위임, `Stream()`/`Explain()` 메서드 |
| `backend/internal/handler/search_handler.go` | `NLToDSL()` SSE 응답, `Explain()` Provider 위임 |
| `backend/internal/handler/search_handler_test.go` | SSE 응답 테스트, mock Provider |
| `backend/cmd/api/main.go` | LLM Provider 초기화, DI 수정 |
| `backend/Makefile` | `build-mcp` 타겟 추가 |

### Frontend — New Files

| File | Responsibility |
|------|---------------|
| `src/features/search/lib/sse-parser.ts` | SSE 텍스트 스트림 → 이벤트 파서 |
| `src/features/search/lib/__tests__/sse-parser.test.ts` | SSE 파서 유닛 테스트 |
| `src/features/search/lib/dsl-linter.ts` | CodeMirror lint extension |
| `src/features/search/lib/__tests__/dsl-linter.test.ts` | lint 로직 유닛 테스트 |
| `src/features/search/lib/__tests__/dsl-completions.test.ts` | 문맥 인식 자동완성 테스트 |

### Frontend — Modified Files

| File | Change |
|------|--------|
| `src/features/search/model/types.ts` | `SSEEvent` union 타입 추가 |
| `src/features/search/api/search-api.ts` | `nlSearchStream()` AsyncGenerator 추가 |
| `src/features/search/model/use-search-actions.ts` | `runNLSearch` SSE 소비로 변경 |
| `src/features/search/__tests__/use-search-actions.test.ts` | SSE mock 테스트 |
| `src/features/search/lib/dsl-completions.ts` | 문맥 인식 로직, 미지원 필드 제거 |
| `src/features/search/ui/DSLEditor.tsx` | lint extension 연결 |

---

## Task 1: LLM Provider Interface (Go 타입 정의)

**Files:**
- Create: `backend/internal/llm/provider.go`
- Create: `backend/internal/llm/provider_test.go`

- [ ] **Step 1: Write the test**

```go
// backend/internal/llm/provider_test.go
package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/llm"
)

func TestEventType_Constants(t *testing.T) {
	assert.Equal(t, llm.EventType("thinking"), llm.EventThinking)
	assert.Equal(t, llm.EventType("tool_call"), llm.EventToolCall)
	assert.Equal(t, llm.EventType("tool_result"), llm.EventToolResult)
	assert.Equal(t, llm.EventType("dsl_ready"), llm.EventDSLReady)
	assert.Equal(t, llm.EventType("done"), llm.EventDone)
	assert.Equal(t, llm.EventType("error"), llm.EventError)
}

func TestEvent_JSONTags(t *testing.T) {
	e := llm.Event{
		Type:        llm.EventDSLReady,
		Message:     "test",
		DSL:         "scan where volume > 100",
		Explanation: "설명",
	}
	assert.Equal(t, llm.EventDSLReady, e.Type)
	assert.Equal(t, "scan where volume > 100", e.DSL)
	assert.Equal(t, "설명", e.Explanation)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/llm/... -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/llm/provider.go
package llm

import "context"

type EventType string

const (
	EventThinking   EventType = "thinking"
	EventToolCall   EventType = "tool_call"
	EventToolResult EventType = "tool_result"
	EventDSLReady   EventType = "dsl_ready"
	EventDone       EventType = "done"
	EventError      EventType = "error"
)

type Event struct {
	Type        EventType `json:"type"`
	Message     string    `json:"message"`
	DSL         string    `json:"dsl,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
	Data        any       `json:"data,omitempty"`
}

// Provider abstracts LLM backends for NL-to-DSL conversion.
// Implementations: ClaudeCLIProvider (claude -p subprocess), future: ClaudeAPIProvider, GoogleADKProvider.
type Provider interface {
	// NLToDSL streams events converting natural language to DSL.
	// The final meaningful event MUST be EventDSLReady (with DSL + Explanation) or EventError.
	// The provider does NOT execute the DSL — the handler does that after receiving dsl_ready.
	NLToDSL(ctx context.Context, query string) (<-chan Event, error)

	// Explain returns a natural language explanation of a DSL query (synchronous).
	Explain(ctx context.Context, dsl string) (string, error)

	// Name returns the provider identifier (e.g. "claude-cli").
	Name() string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/llm/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/llm/
git commit -m "feat(llm): add Provider interface and Event types"
```

---

## Task 2: Config — LLMConfig 추가

**Files:**
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Write the implementation**

`backend/internal/config/config.go`에 `LLMConfig`를 `Config` 구조체에 추가:

```go
// Config struct에 필드 추가 (기존 RedisPassword 아래)
type LLMConfig struct {
	Provider       string
	ClaudeCLIPath  string
	MCPConfigPath  string
	AnthropicKey   string
	MaxConcurrent  int
	TimeoutSeconds int
}

// Config struct에 추가:
// LLM LLMConfig
```

`Load()` 함수 끝에 LLM 설정 로드 추가 (return 전):

```go
import "strconv"

// LLM config (provider-agnostic)
maxConcurrent, _ := strconv.Atoi(getEnv("LLM_MAX_CONCURRENT", "5"))
timeoutSec, _ := strconv.Atoi(getEnv("LLM_TIMEOUT_SECONDS", "60"))
cfg.LLM = LLMConfig{
	Provider:       getEnv("LLM_PROVIDER", "claude-cli"),
	ClaudeCLIPath:  getEnv("CLAUDE_CLI_PATH", "claude"),
	MCPConfigPath:  getEnv("MCP_CONFIG_PATH", ""),
	AnthropicKey:   getEnv("ANTHROPIC_API_KEY", ""),
	MaxConcurrent:  maxConcurrent,
	TimeoutSeconds: timeoutSec,
}
```

- [ ] **Step 2: Write the config test**

```go
// backend/internal/config/config_test.go
package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/config"
)

func TestLoad_LLMDefaults(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	defer os.Unsetenv("APP_ENV")

	cfg := config.Load()

	assert.Equal(t, "claude-cli", cfg.LLM.Provider)
	assert.Equal(t, "claude", cfg.LLM.ClaudeCLIPath)
	assert.Equal(t, 5, cfg.LLM.MaxConcurrent)
	assert.Equal(t, 60, cfg.LLM.TimeoutSeconds)
}

func TestLoad_LLMFromEnv(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	os.Setenv("LLM_PROVIDER", "claude-api")
	os.Setenv("LLM_MAX_CONCURRENT", "10")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("LLM_PROVIDER")
		os.Unsetenv("LLM_MAX_CONCURRENT")
	}()

	cfg := config.Load()

	assert.Equal(t, "claude-api", cfg.LLM.Provider)
	assert.Equal(t, 10, cfg.LLM.MaxConcurrent)
}
```

- [ ] **Step 3: Run tests**

Run: `cd backend && go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/config/
git commit -m "feat(config): add LLMConfig for provider settings"
```

---

## Task 3: MCP Server — Tool Handlers

**Files:**
- Create: `backend/internal/mcp/tools.go`
- Create: `backend/internal/mcp/tools_test.go`

- [ ] **Step 1: Write the tests**

```go
// backend/internal/mcp/tools_test.go
package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/mcp"
)

func TestGetDSLGrammar(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())
	result := s.HandleGetDSLGrammar()
	assert.Contains(t, result, "scan")
	assert.Contains(t, result, "where")
	assert.Contains(t, result, "sort")
	assert.Contains(t, result, "limit")
	assert.Contains(t, result, "AND")
}

func TestListAvailableFields(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())
	result := s.HandleListAvailableFields()
	assert.Contains(t, result, "volume")
	assert.Contains(t, result, "close")
	assert.Contains(t, result, "trade_value")
	assert.Contains(t, result, "change_pct")
}

func TestValidateDSL(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())

	t.Run("valid query", func(t *testing.T) {
		result, err := s.HandleValidateDSL("scan where volume > 1000000")
		require.NoError(t, err)
		assert.Contains(t, result, `"valid":true`)
	})

	t.Run("invalid query", func(t *testing.T) {
		_, err := s.HandleValidateDSL("bad query")
		assert.Error(t, err)
	})

	t.Run("empty query", func(t *testing.T) {
		_, err := s.HandleValidateDSL("")
		assert.Error(t, err)
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/mcp/... -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write the implementation**

```go
// backend/internal/mcp/tools.go
package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

type Server struct {
	executor *dsl.Executor
}

func NewServer(executor *dsl.Executor) *Server {
	return &Server{executor: executor}
}

func (s *Server) HandleGetDSLGrammar() string {
	return `DSL Grammar:
  scan where <conditions> [sort by <field> [asc|desc]] [limit N]

Conditions:
  <field> <operator> <value>
  Multiple conditions joined with AND (OR is NOT supported)

Operators: >, <, >=, <=, =

Defaults:
  limit: 50 (max: 500)
  sort: volume DESC

Example:
  scan where volume > 1000000 and close > 50000 sort by trade_value desc limit 20`
}

func (s *Server) HandleListAvailableFields() string {
	return `Available fields:
  close       — 종가/현재가 (numeric, KRW)
  open        — 시가 (numeric, KRW)
  high        — 고가 (numeric, KRW)
  low         — 저가 (numeric, KRW)
  volume      — 거래량 (integer, shares)
  trade_value — 거래대금 (numeric, close × volume)
  change_pct  — 전일 대비 등락률 (numeric, %)

All fields support operators: >, <, >=, <=, =
All fields can be used in sort by clause.`
}

func (s *Server) HandleValidateDSL(dslCode string) (string, error) {
	result := s.executor.Validate(dslCode)
	if !result.Valid {
		return "", fmt.Errorf("%s", result.Error)
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/mcp/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/mcp/
git commit -m "feat(mcp): add DSL tool handlers (grammar, fields, validate)"
```

---

## Task 4: MCP Server — JSON-RPC Protocol

**Files:**
- Create: `backend/internal/mcp/server.go` (Run method 추가)
- Modify: `backend/internal/mcp/server_test.go`

- [ ] **Step 1: Write the test**

```go
// backend/internal/mcp/server_test.go에 추가
func TestServer_HandleRequest(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())

	t.Run("initialize", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), `"name":"nexus-dsl"`)
	})

	t.Run("tools/list", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "get_dsl_grammar")
		assert.Contains(t, string(resp), "list_available_fields")
		assert.Contains(t, string(resp), "validate_dsl")
	})

	t.Run("tools/call get_dsl_grammar", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_dsl_grammar","arguments":{}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "scan")
	})

	t.Run("tools/call validate_dsl valid", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"validate_dsl","arguments":{"dsl":"scan where volume > 1000000"}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "valid")
		assert.NotContains(t, string(resp), "isError")
	})

	t.Run("tools/call validate_dsl invalid", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"validate_dsl","arguments":{"dsl":"bad"}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "isError")
	})

	t.Run("tools/call unknown tool", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"unknown","arguments":{}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "error")
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/mcp/... -v -run TestServer_HandleRequest`
Expected: FAIL — `HandleRequest` method does not exist

- [ ] **Step 3: Write the implementation**

`backend/internal/mcp/server.go`에 JSON-RPC 프로토콜 핸들링을 구현. `HandleRequest([]byte) ([]byte, error)` 메서드로 단일 요청 처리. `Run(ctx)` 메서드는 stdin에서 줄 단위로 읽어 HandleRequest에 위임 후 stdout에 응답을 쓴다.

핵심 구조:
- `jsonrpcRequest` struct: `Jsonrpc`, `ID`, `Method`, `Params` 필드
- `jsonrpcResponse` struct: `Jsonrpc`, `ID`, `Result`, `Error` 필드
- method dispatch: `initialize` → 서버 capabilities 반환, `tools/list` → 도구 스키마 목록, `tools/call` → 도구명으로 핸들러 dispatch

MCP tool 응답 포맷:
- 성공: `{"content": [{"type": "text", "text": "..."}]}`
- 실패: `{"content": [{"type": "text", "text": "error msg"}], "isError": true}`

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/mcp/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/mcp/
git commit -m "feat(mcp): add JSON-RPC 2.0 protocol handler"
```

---

## Task 5: MCP Server Binary

**Files:**
- Create: `backend/cmd/mcp-server/main.go`
- Create: `backend/mcp-config.json`
- Modify: `backend/Makefile`

- [ ] **Step 1: Write mcp-server main.go**

```go
// backend/cmd/mcp-server/main.go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/mcp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	dbURL := os.Getenv("DATABASE_URL")
	var pool *pgxpool.Pool
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), dbURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		slog.Info("mcp-server: connected to database")
	} else {
		slog.Warn("mcp-server: no DATABASE_URL, validate-only mode")
	}

	executor := dsl.NewExecutor(pool)
	server := mcp.NewServer(executor)

	if err := server.Run(context.Background()); err != nil {
		slog.Error("mcp-server exited with error", "error", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Write mcp-config.json**

```json
{
  "mcpServers": {
    "nexus-dsl": {
      "command": "./bin/mcp-server",
      "env": {
        "DATABASE_URL": "${DATABASE_URL}"
      }
    }
  }
}
```

Note: `env` 필드로 DATABASE_URL을 명시적으로 전달. MCP 서버 내부에서 `os.Getenv("DATABASE_URL")`로 읽는다.

- [ ] **Step 3: Add build-mcp to Makefile**

`backend/Makefile`의 `.PHONY` 줄과 `build` 타겟 수정:

```makefile
.PHONY: build build-mcp run test lint migrate sqlc

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
	go build -o bin/mcp-server ./cmd/mcp-server

build-mcp:
	go build -o bin/mcp-server ./cmd/mcp-server
```

- [ ] **Step 4: Verify build**

Run: `cd backend && make build-mcp`
Expected: `bin/mcp-server` binary created

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/mcp-server/ backend/mcp-config.json backend/Makefile
git commit -m "feat(mcp): add mcp-server binary and config"
```

---

## Task 6: System Prompts

**Files:**
- Create: `backend/internal/llm/prompts/nl-to-dsl.txt`
- Create: `backend/internal/llm/prompts/explain.txt`

- [ ] **Step 1: Write NL-to-DSL system prompt**

```text
You are a Korean stock market search DSL expert.

Your job: convert the user's natural language query into a valid DSL query.

WORKFLOW:
1. Call get_dsl_grammar to understand the syntax
2. Call list_available_fields to see what fields are available
3. Generate the DSL query based on the user's request
4. Call validate_dsl to verify your query is valid
5. If validation fails, fix the query and re-validate (max 3 attempts)
6. Return your final answer in this exact format:

DSL: <the valid DSL query>
EXPLANATION: <Korean explanation of what the query does>

RULES:
- Only use AND to combine conditions (OR is NOT supported)
- Use reasonable defaults: limit 50 if not specified
- All numeric comparisons only (no string matching)
- If the user's request is ambiguous, make a reasonable interpretation and explain your choices
- Always respond in Korean for the explanation
```

- [ ] **Step 2: Write Explain system prompt**

```text
You are a Korean stock market DSL interpreter.

Given a DSL query, explain what it does in natural Korean.
- Describe each condition clearly
- Explain the overall purpose of the query
- Mention sort order and limit if present
- Keep the explanation concise (2-3 sentences)

Respond ONLY with the Korean explanation, no other text.
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/llm/prompts/
git commit -m "feat(llm): add system prompts for NL-to-DSL and explain"
```

---

## Task 7: ClaudeCLIProvider

**Files:**
- Create: `backend/internal/llm/claudecli/provider.go`
- Create: `backend/internal/llm/claudecli/provider_test.go`

- [ ] **Step 1: Write the test**

```go
// backend/internal/llm/claudecli/provider_test.go
package claudecli_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/claudecli"
)

func TestProvider_ImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*claudecli.Provider)(nil)
}

func TestProvider_Name(t *testing.T) {
	p := claudecli.New(config.LLMConfig{})
	assert.Equal(t, "claude-cli", p.Name())
}

func TestProvider_NLToDSL_ContextCancellation(t *testing.T) {
	p := claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  "sleep",   // use sleep as a slow command
		TimeoutSeconds: 1,
		MaxConcurrent:  1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := p.NLToDSL(ctx, "test")
	if err != nil {
		// Expected: command might fail immediately
		return
	}

	// Drain events — should get error due to cancellation
	var lastEvent llm.Event
	for e := range ch {
		lastEvent = e
	}
	assert.Equal(t, llm.EventError, lastEvent.Type)
}

func TestParseStreamJSON(t *testing.T) {
	// Test the stream JSON line parser
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"DSL: scan where volume > 1000000\nEXPLANATION: 거래량 100만 이상"}]}}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	require.NoError(t, err)
	if event != nil {
		assert.NotEmpty(t, event.Type)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/llm/claudecli/... -v`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write the implementation**

`claudecli.Provider` 구현 핵심:
- `New(cfg config.LLMConfig)` — 세마포어(`sync` 패키지의 `Semaphore`) 초기화
- `NLToDSL(ctx, query)` — 세마포어 acquire → subprocess 실행 → goroutine에서 stdout 파싱 → Event 채널 전송
- subprocess 명령: `claude -p --output-format stream-json --mcp-config <path> --system-prompt <prompt> <query>`
- stdout 각 줄을 `ParseStreamLine()`으로 파싱
- `claude -p stream-json` 출력 포맷 파싱: `{"type":"assistant","message":{...}}` 등
- tool_use 이벤트 → EventToolCall, tool_result → EventToolResult 변환
- 최종 텍스트에서 `DSL:` / `EXPLANATION:` 라인 추출 → EventDSLReady
- ctx 취소 시 `cmd.Process.Kill()` + EventError
- `Explain(ctx, dsl)` — 동기, subprocess 실행 후 결과 텍스트 반환

`ParseStreamLine()` 함수는 exported로 만들어 테스트 가능하게.

`golang.org/x/sync/semaphore` 패키지 사용 (이미 `go.sum`에 `golang.org/x/sync` 존재하나 indirect이므로 `go mod tidy` 필요).

- [ ] **Step 4: Run go mod tidy**

Run: `cd backend && go mod tidy`
Expected: `golang.org/x/sync`가 direct dependency로 승격

- [ ] **Step 5: Run test to verify it passes**

Run: `cd backend && go test ./internal/llm/claudecli/... -v`
Expected: PASS (ContextCancellation 테스트는 sleep 명령 사용하므로 timeout으로 통과)

- [ ] **Step 6: Commit**

```bash
git add backend/internal/llm/claudecli/ backend/go.mod backend/go.sum
git commit -m "feat(llm): add ClaudeCLIProvider (claude -p subprocess)"
```

---

## Task 8: Backend Service & Handler — SSE Integration

**Files:**
- Modify: `backend/internal/service/nl_to_dsl_service.go`
- Modify: `backend/internal/handler/search_handler.go`
- Modify: `backend/internal/handler/search_handler_test.go`
- Modify: `backend/cmd/api/main.go`

- [ ] **Step 1: Rewrite the handler tests for SSE**

기존 `search_handler_test.go` **전체 재작성**:
- `setupSearchRouter()` → mock Provider 주입
- 기존 `TestSearchHandler_NLToDSL` (JSON 응답 기대) → **삭제** (SSE로 변경되므로)
- 기존 `TestSearchHandler_Explain` → **수정** (Provider 위임 반영)
- 새로운 SSE 테스트 추가
- `TestSearchHandler_Execute`, `TestSearchHandler_Validate`는 유지 (변경 없음)

```go
// backend/internal/handler/search_handler_test.go — 전체 재작성
// MockProvider 구현
type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Explain(_ context.Context, dsl string) (string, error) {
	return "Mock explanation for: " + dsl, nil
}
func (m *mockProvider) NLToDSL(_ context.Context, _ string) (<-chan llm.Event, error) {
	ch := make(chan llm.Event, 3)
	ch <- llm.Event{Type: llm.EventThinking, Message: "분석 중..."}
	ch <- llm.Event{Type: llm.EventDSLReady, DSL: "scan where volume > 1000000", Explanation: "거래량 100만 이상", Message: "생성 완료"}
	close(ch)
	return ch, nil
}

// setupSearchRouter updated:
// searchSvc := service.NewSearchService(nil)
// nlSvc := service.NewNLToDSLService(&mockProvider{})
// searchH := handler.NewSearchHandler(searchSvc, nlSvc)
```

SSE 테스트 추가:

```go
func TestSearchHandler_NLToDSL_SSE(t *testing.T) {
	r := setupSearchRouter()

	body, _ := json.Marshal(map[string]string{"query": "거래량 많은 종목"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/nl-to-dsl", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	// SSE 이벤트 파싱
	responseBody := w.Body.String()
	assert.Contains(t, responseBody, "event: thinking")
	assert.Contains(t, responseBody, "event: dsl_ready")
	assert.Contains(t, responseBody, "event: done")
}

func TestSearchHandler_Explain_WithProvider(t *testing.T) {
	r := setupSearchRouter()

	body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/explain", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["explanation"], "Mock explanation")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/handler/... -v -run "TestSearchHandler_NLToDSL_SSE|TestSearchHandler_Explain_WithProvider"`
Expected: FAIL — interface changes

- [ ] **Step 3: Modify nl_to_dsl_service.go**

```go
// backend/internal/service/nl_to_dsl_service.go — 전체 재작성
package service

import (
	"context"

	"github.com/dev-superbear/nexus-backend/internal/llm"
)

type NLToDSLService struct {
	provider llm.Provider
}

func NewNLToDSLService(provider llm.Provider) *NLToDSLService {
	return &NLToDSLService{provider: provider}
}

func (s *NLToDSLService) Stream(ctx context.Context, query string) (<-chan llm.Event, error) {
	return s.provider.NLToDSL(ctx, query)
}

func (s *NLToDSLService) Explain(ctx context.Context, dsl string) (string, error) {
	return s.provider.Explain(ctx, dsl)
}
```

- [ ] **Step 4: Modify search_handler.go**

핵심 변경:
- `NLToDSL()`: SSE 헤더 설정 → `nlSvc.Stream()` → 이벤트 루프 → `dsl_ready` 시 `searchSvc.Execute()` → `done` 이벤트
- `Explain()`: `nlSvc.Explain()` 사용
- SSE 헬퍼 함수 `writeSSE(c *gin.Context, event llm.Event)` 추가

```go
func writeSSE(c *gin.Context, eventType string, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, b)
	c.Writer.Flush()
}
```

- [ ] **Step 5: Modify main.go DI**

`backend/cmd/api/main.go` 변경 — line 103-106:

```go
// 기존:
// searchSvc := service.NewSearchService(dsl.NewExecutor(pool))
// nlSvc := service.NewNLToDSLService()

// 변경:
import "github.com/dev-superbear/nexus-backend/internal/llm/claudecli"

llmProvider := claudecli.New(cfg.LLM)
searchSvc := service.NewSearchService(dsl.NewExecutor(pool))
nlSvc := service.NewNLToDSLService(llmProvider)
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd backend && go test ./internal/handler/... -v`
Expected: PASS

Run: `cd backend && go test ./... -v`
Expected: ALL PASS (기존 테스트 포함)

- [ ] **Step 7: Commit**

```bash
git add backend/internal/service/nl_to_dsl_service.go backend/internal/handler/search_handler.go backend/internal/handler/search_handler_test.go backend/cmd/api/main.go
git commit -m "feat(search): integrate LLM provider with SSE streaming"
```

---

## Task 9: Frontend — SSE Types & Parser

**Files:**
- Modify: `src/features/search/model/types.ts`
- Create: `src/features/search/lib/sse-parser.ts`
- Create: `src/features/search/lib/__tests__/sse-parser.test.ts`

- [ ] **Step 1: Write the SSE parser test**

```typescript
// src/features/search/lib/__tests__/sse-parser.test.ts
import { parseSSEBuffer } from "../sse-parser";

describe("parseSSEBuffer", () => {
  it("parses a complete SSE event", () => {
    const buffer = 'event: thinking\ndata: {"message":"분석 중..."}\n\n';
    const { events, remaining } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "thinking", message: "분석 중..." });
    expect(remaining).toBe("");
  });

  it("parses multiple events", () => {
    const buffer =
      'event: thinking\ndata: {"message":"a"}\n\nevent: dsl_ready\ndata: {"dsl":"scan where volume > 100","explanation":"설명"}\n\n';
    const { events, remaining } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(2);
    expect(events[0].type).toBe("thinking");
    expect(events[1].type).toBe("dsl_ready");
  });

  it("keeps incomplete event in remaining", () => {
    const buffer = 'event: thinking\ndata: {"mess';
    const { events, remaining } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(0);
    expect(remaining).toBe(buffer);
  });

  it("handles empty buffer", () => {
    const { events, remaining } = parseSSEBuffer("");
    expect(events).toHaveLength(0);
    expect(remaining).toBe("");
  });

  it("parses done event with results", () => {
    const buffer =
      'event: done\ndata: {"results":[{"symbol":"005930","name":"삼성전자","matchedValue":100}],"count":1}\n\n';
    const { events } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe("done");
    if (events[0].type === "done") {
      expect(events[0].results).toHaveLength(1);
      expect(events[0].count).toBe(1);
    }
  });

  it("parses error event", () => {
    const buffer = 'event: error\ndata: {"message":"timeout"}\n\n';
    const { events } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "error", message: "timeout" });
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx jest src/features/search/lib/__tests__/sse-parser.test.ts --no-coverage`
Expected: FAIL — module not found

- [ ] **Step 3: Add SSEEvent type to types.ts**

```typescript
// src/features/search/model/types.ts — append
import type { SearchResult } from "@/entities/search-result";

export type SSEEventType = "thinking" | "tool_call" | "tool_result" | "dsl_ready" | "done" | "error";

export type SSEEvent =
  | { type: "thinking"; message: string }
  | { type: "tool_call"; tool: string; message: string }
  | { type: "tool_result"; tool: string; message: string }
  | { type: "dsl_ready"; dsl: string; explanation: string }
  | { type: "done"; results: SearchResult[]; count: number }
  | { type: "error"; message: string };
```

- [ ] **Step 4: Implement SSE parser**

```typescript
// src/features/search/lib/sse-parser.ts
import type { SSEEvent } from "../model/types";

interface ParseResult {
  events: SSEEvent[];
  remaining: string;
}

export function parseSSEBuffer(buffer: string): ParseResult {
  const events: SSEEvent[] = [];
  const blocks = buffer.split("\n\n");

  // Last block may be incomplete
  const remaining = blocks[blocks.length - 1];

  for (let i = 0; i < blocks.length - 1; i++) {
    const block = blocks[i].trim();
    if (!block) continue;

    let eventType = "";
    let data = "";

    for (const line of block.split("\n")) {
      if (line.startsWith("event: ")) {
        eventType = line.slice(7);
      } else if (line.startsWith("data: ")) {
        data = line.slice(6);
      }
    }

    if (eventType && data) {
      try {
        const parsed = JSON.parse(data);
        events.push({ type: eventType, ...parsed } as SSEEvent);
      } catch {
        // skip malformed events
      }
    }
  }

  return { events, remaining };
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `npx jest src/features/search/lib/__tests__/sse-parser.test.ts --no-coverage`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add src/features/search/model/types.ts src/features/search/lib/sse-parser.ts src/features/search/lib/__tests__/sse-parser.test.ts
git commit -m "feat(search): add SSE event types and parser"
```

---

## Task 10: Frontend — SSE API Client & Actions

**Files:**
- Modify: `src/features/search/api/search-api.ts`
- Modify: `src/features/search/model/use-search-actions.ts`
- Modify: `src/features/search/__tests__/use-search-actions.test.ts`

- [ ] **Step 1: Update the test**

`src/features/search/__tests__/use-search-actions.test.ts` — **전체 재작성**:
- mock에서 `nlSearch` → `nlSearchStream`으로 교체
- 기존 `"transitions through agent statuses and sets results on success"` 테스트 (`mockedApi.nlSearch` 사용) → **삭제**
- 기존 `"sets error status on API failure"` 테스트 → **삭제**
- SSE 기반 새 테스트로 교체
- `runDSLSearch`, `validateDSL`, `explainDSL` 테스트는 유지 (변경 없음)

```typescript
// mock 변경: nlSearch → nlSearchStream
jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearchStream: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

// nlSearchStream mock은 AsyncGenerator를 반환:
describe("runNLSearch (SSE)", () => {
  it("transitions through statuses from SSE events", async () => {
    async function* mockStream() {
      yield { type: "thinking" as const, message: "분석 중..." };
      yield { type: "dsl_ready" as const, dsl: "scan where volume > 1000000", explanation: "거래량 100만 이상" };
      yield { type: "done" as const, results: [{ symbol: "005930", name: "삼성전자", matchedValue: 2840000 }], count: 1 };
    }
    mockedApi.nlSearchStream.mockReturnValue(mockStream());

    const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
    useSearchStore.setState({ nlQuery: "거래량 많은 종목" });

    await actions.runNLSearch();

    const state = useSearchStore.getState();
    expect(state.agentStatus).toBe("done");
    expect(state.dslCode).toBe("scan where volume > 1000000");
    expect(state.results).toHaveLength(1);
  });

  it("handles error event", async () => {
    async function* mockStream() {
      yield { type: "error" as const, message: "timeout" };
    }
    mockedApi.nlSearchStream.mockReturnValue(mockStream());

    const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
    useSearchStore.setState({ nlQuery: "테스트" });

    await actions.runNLSearch();

    expect(useSearchStore.getState().agentStatus).toBe("error");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx jest src/features/search/__tests__/use-search-actions.test.ts --no-coverage`
Expected: FAIL — `nlSearchStream` not defined

- [ ] **Step 3: Replace nlSearch with nlSearchStream in search-api.ts**

기존 `nlSearch` 메서드를 **삭제**하고 `nlSearchStream`으로 교체. `NLSearchResponse` 인터페이스도 삭제 (SSEEvent로 대체됨).

```typescript
// src/features/search/api/search-api.ts — nlSearch 삭제 후 교체
import { API_BASE_URL } from "@/shared/config/constants";
import { parseSSEBuffer } from "../lib/sse-parser";
import type { SSEEvent } from "../model/types";

// 기존 searchApi 객체에 메서드 추가:
async *nlSearchStream(query: string): AsyncGenerator<SSEEvent> {
  const response = await fetch(`${API_BASE_URL}/api/v1/search/nl-to-dsl`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ query }),
  });

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }

  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const { events, remaining } = parseSSEBuffer(buffer);
    buffer = remaining;

    for (const event of events) {
      yield event;
    }
  }
},
```

- [ ] **Step 4: Update use-search-actions.ts runNLSearch**

```typescript
async function runNLSearch(): Promise<void> {
  const { nlQuery } = getState();
  setState({ agentStatus: "interpreting", agentMessage: "쿼리 분석 중..." });

  try {
    for await (const event of searchApi.nlSearchStream(nlQuery)) {
      switch (event.type) {
        case "thinking":
          setState({ agentStatus: "interpreting", agentMessage: event.message });
          break;
        case "tool_call":
        case "tool_result":
          setState({ agentStatus: "building", agentMessage: event.message });
          break;
        case "dsl_ready":
          setState({
            dslCode: event.dsl,
            explanation: event.explanation,
            agentStatus: "scanning",
            agentMessage: "검색 중...",
          });
          break;
        case "done":
          setState({
            results: event.results,
            agentStatus: "done",
            agentMessage: `${event.count}개 종목 발견`,
          });
          break;
        case "error":
          setState({ agentStatus: "error", agentMessage: event.message });
          break;
      }
    }
  } catch (err) {
    setError(err);
  }
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `npx jest src/features/search/__tests__/use-search-actions.test.ts --no-coverage`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add src/features/search/api/search-api.ts src/features/search/model/use-search-actions.ts src/features/search/__tests__/use-search-actions.test.ts
git commit -m "feat(search): SSE streaming for NL search"
```

---

## Task 11: DSL Context-Aware Completions

**Files:**
- Modify: `src/features/search/lib/dsl-completions.ts`
- Create: `src/features/search/lib/__tests__/dsl-completions.test.ts`

- [ ] **Step 1: Write the test**

```typescript
// src/features/search/lib/__tests__/dsl-completions.test.ts
import { getContextualCompletions } from "../dsl-completions";

describe("getContextualCompletions", () => {
  it("suggests 'scan' at empty input", () => {
    const items = getContextualCompletions("");
    expect(items.map((c) => c.label)).toEqual(["scan"]);
  });

  it("suggests 'where' after scan", () => {
    const items = getContextualCompletions("scan ");
    expect(items.map((c) => c.label)).toEqual(["where"]);
  });

  it("suggests fields after 'where'", () => {
    const items = getContextualCompletions("scan where ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("volume");
    expect(labels).toContain("close");
    expect(labels).toContain("change_pct");
    expect(labels).not.toContain("market_cap");
    expect(labels).not.toContain("ma");
  });

  it("suggests operators after field", () => {
    const items = getContextualCompletions("scan where volume ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain(">");
    expect(labels).toContain(">=");
  });

  it("suggests 'and', 'sort', 'limit' after value", () => {
    const items = getContextualCompletions("scan where volume > 1000000 ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("and");
    expect(labels).toContain("sort");
    expect(labels).toContain("limit");
  });

  it("suggests fields after 'and'", () => {
    const items = getContextualCompletions("scan where volume > 1000000 and ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("close");
  });

  it("suggests 'by' after 'sort'", () => {
    const items = getContextualCompletions("scan where volume > 1000000 sort ");
    expect(items.map((c) => c.label)).toEqual(["by"]);
  });

  it("suggests fields after 'sort by'", () => {
    const items = getContextualCompletions("scan where volume > 1000000 sort by ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("volume");
    expect(labels).toContain("trade_value");
  });

  it("suggests 'asc'/'desc' after sort field", () => {
    const items = getContextualCompletions("scan where volume > 1000000 sort by volume ");
    const labels = items.map((c) => c.label);
    expect(labels).toContain("asc");
    expect(labels).toContain("desc");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx jest src/features/search/lib/__tests__/dsl-completions.test.ts --no-coverage`
Expected: FAIL — `getContextualCompletions` not exported

- [ ] **Step 3: Rewrite dsl-completions.ts**

핵심 변경:
- 미지원 필드/함수 제거 (market_cap, per, pbr, roe, event_*, ma, rsi 등)
- `change_pct` 추가
- `or` 키워드 제거
- `getContextualCompletions(input: string): CompletionItem[]` 함수 추가 — TS Lexer로 토큰화 후 마지막 의미 있는 토큰의 타입에 따라 적절한 completion 반환
- 기존 `DSL_COMPLETIONS` 배열은 유지하되 백엔드 지원 필드로 정리
- `dslAutoComplete` CodeMirror 함수도 `getContextualCompletions` 사용하도록 변경

문맥 판단 로직:
1. 입력을 Lexer로 토큰화
2. WHITESPACE/EOF를 제외한 마지막 토큰의 type 확인
3. switch(lastToken.type)로 다음 제안 결정

- [ ] **Step 4: Run test to verify it passes**

Run: `npx jest src/features/search/lib/__tests__/dsl-completions.test.ts --no-coverage`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add src/features/search/lib/dsl-completions.ts src/features/search/lib/__tests__/dsl-completions.test.ts
git commit -m "feat(search): context-aware DSL autocompletions"
```

---

## Task 12: DSL Linter

**Files:**
- Create: `src/features/search/lib/dsl-linter.ts`
- Create: `src/features/search/lib/__tests__/dsl-linter.test.ts`
- Modify: `src/features/search/ui/DSLEditor.tsx`

- [ ] **Step 1: Write the test**

```typescript
// src/features/search/lib/__tests__/dsl-linter.test.ts
import { lintDSL, type DSLDiagnostic } from "../dsl-linter";

describe("lintDSL", () => {
  it("returns no diagnostics for valid DSL", () => {
    expect(lintDSL("scan where volume > 1000000")).toEqual([]);
  });

  it("returns no diagnostics for valid DSL with sort and limit", () => {
    expect(lintDSL("scan where volume > 1000000 sort by trade_value desc limit 50")).toEqual([]);
  });

  it("reports missing 'scan' keyword", () => {
    const diags = lintDSL("where volume > 100");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("scan");
  });

  it("reports missing 'where' keyword", () => {
    const diags = lintDSL("scan volume > 100");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("where");
  });

  it("reports unknown field", () => {
    const diags = lintDSL("scan where market_cap > 100");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("market_cap");
  });

  it("reports OR not supported", () => {
    const diags = lintDSL("scan where volume > 100 or close > 50000");
    expect(diags).toHaveLength(1);
    expect(diags[0].message).toContain("OR");
  });

  it("returns empty for empty input", () => {
    expect(lintDSL("")).toEqual([]);
  });

  it("reports error position", () => {
    const diags = lintDSL("scan where unknown_field > 100");
    expect(diags[0].from).toBeGreaterThan(0);
    expect(diags[0].to).toBeGreaterThan(diags[0].from);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx jest src/features/search/lib/__tests__/dsl-linter.test.ts --no-coverage`
Expected: FAIL — module not found

- [ ] **Step 3: Implement dsl-linter.ts**

```typescript
// src/features/search/lib/dsl-linter.ts
import { Lexer } from "@/shared/lib/dsl/lexer";
import { TokenType } from "@/shared/lib/dsl/tokens";

export interface DSLDiagnostic {
  from: number;
  to: number;
  severity: "error" | "warning";
  message: string;
}

const ALLOWED_FIELDS = new Set([
  "close", "open", "high", "low", "volume", "trade_value", "change_pct",
]);

export function lintDSL(input: string): DSLDiagnostic[] {
  if (!input.trim()) return [];

  const lexer = new Lexer(input);
  const tokens = lexer.tokenize();
  const meaningful = tokens.filter(
    (t) => t.type !== TokenType.WHITESPACE && t.type !== TokenType.EOF,
  );

  const diagnostics: DSLDiagnostic[] = [];

  if (meaningful.length === 0) return [];

  // Check: first token must be SCAN
  if (meaningful[0].type !== TokenType.SCAN) {
    diagnostics.push({
      from: meaningful[0].position,
      to: meaningful[0].position + meaningful[0].value.length,
      severity: "error",
      message: "쿼리는 'scan'으로 시작해야 합니다",
    });
    return diagnostics;
  }

  // Check: second meaningful token must be WHERE
  if (meaningful.length > 1 && meaningful[1].type !== TokenType.WHERE) {
    diagnostics.push({
      from: meaningful[1].position,
      to: meaningful[1].position + meaningful[1].value.length,
      severity: "error",
      message: "'scan' 다음에 'where'가 필요합니다",
    });
    return diagnostics;
  }

  // Check for OR usage
  for (const token of meaningful) {
    if (token.type === TokenType.OR) {
      diagnostics.push({
        from: token.position,
        to: token.position + token.value.length,
        severity: "error",
        message: "OR은 지원되지 않습니다. AND를 사용하세요",
      });
    }
  }

  // Check field names after WHERE and AND
  for (let i = 0; i < meaningful.length; i++) {
    const token = meaningful[i];
    if (
      (token.type === TokenType.WHERE || token.type === TokenType.AND) &&
      i + 1 < meaningful.length
    ) {
      const next = meaningful[i + 1];
      if (
        next.type === TokenType.IDENTIFIER &&
        !ALLOWED_FIELDS.has(next.value.toLowerCase())
      ) {
        diagnostics.push({
          from: next.position,
          to: next.position + next.value.length,
          severity: "error",
          message: `알 수 없는 필드: ${next.value}. 사용 가능: ${[...ALLOWED_FIELDS].join(", ")}`,
        });
      }
    }
  }

  return diagnostics;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx jest src/features/search/lib/__tests__/dsl-linter.test.ts --no-coverage`
Expected: PASS

- [ ] **Step 5: Install @codemirror/lint**

Run: `npm install @codemirror/lint`
Expected: package added to package.json

- [ ] **Step 6: Integrate lint into DSLEditor.tsx**

`src/features/search/ui/DSLEditor.tsx` 수정 — `@codemirror/lint` 패키지의 `linter()` extension 추가:

```typescript
import { linter, type Diagnostic } from "@codemirror/lint";
import { lintDSL } from "../lib/dsl-linter";

const dslLintExtension = linter((view) => {
  const doc = view.state.doc.toString();
  return lintDSL(doc).map((d) => ({
    from: d.from,
    to: d.to,
    severity: d.severity,
    message: d.message,
  }));
}, { delay: 300 });

// EditorState.create extensions 배열에 추가:
// dslLintExtension,
```

- [ ] **Step 6: Run all search tests**

Run: `npx jest src/features/search/ --no-coverage`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add src/features/search/lib/dsl-linter.ts src/features/search/lib/__tests__/dsl-linter.test.ts src/features/search/ui/DSLEditor.tsx
git commit -m "feat(search): add real-time DSL linter with CodeMirror integration"
```

---

## Task 13: E2E Test

**Files:**
- Create: `e2e/search-sse.spec.ts`

Note: 유저 피드백 — "UI 대량 추가 시 plan에 없더라도 E2E 테스트를 반드시 확인/추가할 것"

- [ ] **Step 1: Write E2E test for NL search SSE flow**

```typescript
// e2e/search-sse.spec.ts
import { test, expect } from "@playwright/test";

test.describe("Search SSE Integration", () => {
  test("NL tab shows streaming status during search", async ({ page }) => {
    await page.goto("/search");

    // NL 탭이 기본 선택됨
    const textarea = page.locator("textarea");
    await textarea.fill("거래량 100만 이상 종목");

    // Search 버튼 클릭
    const searchBtn = page.getByRole("button", { name: "Search" });
    await searchBtn.click();

    // 에이전트 상태가 표시되어야 함 (interpreting, building, scanning 중 하나)
    const statusIndicator = page.locator("[class*='animate-pulse']");
    // SSE가 진행 중이면 상태 표시가 나타남 (타임아웃 허용 — LLM 호출이 느릴 수 있음)
    await expect(statusIndicator.or(page.getByText(/종목 발견/))).toBeVisible({ timeout: 120000 });
  });

  test("DSL tab shows lint errors for invalid field", async ({ page }) => {
    await page.goto("/search");

    // DSL 탭 클릭
    await page.getByRole("tab", { name: /DSL/i }).click();

    // CodeMirror에 잘못된 DSL 입력
    const editor = page.locator("[data-testid='dsl-editor-container'] .cm-content");
    await editor.click();
    await page.keyboard.type("scan where unknown_field > 100");

    // lint 에러가 표시될 때까지 대기 (debounce 300ms + 렌더링)
    await expect(page.locator(".cm-lint-marker-error, .cm-lintRange-error")).toBeVisible({ timeout: 2000 });
  });

  test("DSL tab validates and runs valid query", async ({ page }) => {
    await page.goto("/search");

    // DSL 탭 클릭
    await page.getByRole("tab", { name: /DSL/i }).click();

    const editor = page.locator("[data-testid='dsl-editor-container'] .cm-content");
    await editor.click();
    await page.keyboard.type("scan where volume > 1000000");

    // Validate 클릭
    await page.getByRole("button", { name: "Validate" }).click();

    // 유효성 결과 확인 (valid 배지 또는 메시지)
    await expect(page.getByText(/valid/i)).toBeVisible({ timeout: 5000 });
  });
});
```

- [ ] **Step 2: Run E2E test**

Run: `npx playwright test e2e/search-sse.spec.ts`
Expected: PASS (서버가 실행 중이어야 함)

- [ ] **Step 3: Commit**

```bash
git add e2e/search-sse.spec.ts
git commit -m "test(e2e): add search SSE and DSL linter E2E tests"
```

---

## Task 14: Final Integration Verification

**Files:**
- All previously modified files

- [ ] **Step 1: Run all backend tests**

Run: `cd backend && go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Run all frontend tests**

Run: `npx jest --no-coverage`
Expected: ALL PASS

- [ ] **Step 3: Build backend**

Run: `cd backend && make build`
Expected: `bin/api`, `bin/worker`, `bin/mcp-server` 모두 생성

- [ ] **Step 4: Build frontend**

Run: `npx next build`
Expected: build 성공

- [ ] **Step 5: Commit (if any remaining changes)**

```bash
git status
# 변경사항 있으면 커밋
```

---

## Dependency Graph

```
Task 1 (LLM types)
    ├── Task 2 (Config)
    ├── Task 3 (MCP tools) → Task 4 (MCP protocol) → Task 5 (MCP binary)
    ├── Task 6 (Prompts)
    └── Task 7 (ClaudeCLI provider)
         └── Task 8 (Backend SSE integration) ← Task 2, 6
              └── Task 10 (Frontend SSE actions) ← Task 9 (SSE parser)

Task 11 (DSL completions)  ← independent
Task 12 (DSL linter)       ← independent

Task 13 (E2E test)           ← after Tasks 10, 12
Task 14 (Final integration)  ← all above
```

**Parallelizable groups:**
- Group A: Tasks 1, 2, 6 (foundation)
- Group B: Tasks 3, 4, 5 (MCP server) — after Task 1
- Group C: Tasks 9, 11, 12 (frontend) — independent of backend
- Group D: Task 7 (ClaudeCLI) — after Tasks 1, 2, 6
- Group E: Task 8 (backend integration) — after Tasks 1, 2, 7
- Group F: Task 10 (frontend SSE) — after Task 9
- Group G: Task 13 (E2E) — after Tasks 10, 12
- Group H: Task 14 (final) — after all
