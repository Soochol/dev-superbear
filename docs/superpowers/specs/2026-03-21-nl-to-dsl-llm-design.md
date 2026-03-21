# NL-to-DSL LLM Integration Design

**Date**: 2026-03-21
**Status**: Approved

## Overview

Search 페이지의 NL(자연어) 탭에서 실제 LLM을 사용하여 자연어를 DSL로 변환하고, DSL 탭에서는 실시간 자동완성과 에러 힌트를 강화한다.

## Goals

1. `claude -p` subprocess + MCP 도구로 NL→DSL 변환 구현
2. 나중에 Claude Agent SDK, Google ADK로 교체 가능한 인터페이스 설계
3. SSE 스트리밍으로 실시간 진행 상태 전달
4. DSL 에디터에 문맥 인식 자동완성 + 실시간 lint 추가

## Non-Goals

- DSL 문법 확장 (scan where 외 명령어 추가)
- DB 스키마 직접 조회 도구
- 프로덕션 인증/과금 시스템
- `OR` 연산자 지원 (현재 백엔드 파서는 `AND`만 지원, 프론트엔드 자동완성에서도 제외)

---

## Architecture

```
Frontend (Next.js)
    │ SSE (POST /search/nl-to-dsl)
    │ JSON (POST /search/explain, /execute, /validate)
    ▼
Go Backend (Gin)
    ├── SearchHandler
    │      ├── NLToDSL()  → SSE 스트림 (Provider에서 DSL 수신 → Executor로 실행 → done)
    │      ├── Explain()  → 동기 JSON (Provider에서 설명 수신)
    │      ├── Execute()  → 동기 JSON (기존 유지)
    │      └── Validate() → 동기 JSON (기존 유지)
    │
    ├── LLMProvider interface
    │      ├── ClaudeCLIProvider  (subprocess: claude -p --mcp-config)
    │      ├── ClaudeAPIProvider  (나중에)
    │      └── GoogleADKProvider  (나중에)
    │
    ├── Go MCP Server (별도 바이너리, stdio, claude -p 전용)
    │      ├── get_dsl_grammar      (문법 참조)
    │      ├── list_available_fields (필드 목록)
    │      └── validate_dsl         (검증)
    │
    └── DSL Executor (기존 코드 유지)
```

**실행 주체 결정**: DSL 실행은 항상 **Go 핸들러**가 담당한다.
LLM은 DSL 생성 + 검증만 하고, `dsl_ready` 이벤트로 DSL을 반환하면 핸들러가 `Executor.Execute()`를 호출한다.
MCP 서버에 `execute_dsl` 도구는 포함하지 않는다 — LLM에게 직접 DB 접근 권한을 주지 않기 위함.

접근법: Monolith (A) — 단일 Go 백엔드에 모두 통합.

---

## Section 1: LLM Provider Interface

### 타입 정의

```go
// backend/internal/llm/provider.go

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

type Provider interface {
    // NLToDSL converts natural language to DSL, streaming events via channel.
    // The final event before channel close MUST be EventDSLReady (with DSL + Explanation)
    // or EventError. The provider does NOT execute the DSL.
    NLToDSL(ctx context.Context, query string) (<-chan Event, error)

    // Explain converts DSL to natural language explanation (synchronous).
    Explain(ctx context.Context, dsl string) (string, error)

    // Name returns provider identifier ("claude-cli", "claude-api", "google-adk")
    Name() string
}
```

**Note**: `Explain`은 동기 메서드. DSL→NL 설명은 단일 텍스트 응답이므로 스트리밍이 불필요하다.

### Provider 구현 계획

| Provider | 패키지 | 사용 시점 | API 키 필요 |
|----------|--------|----------|------------|
| `ClaudeCLIProvider` | `llm/claudecli` | 1차 구현 | 불필요 (구독 세션) |
| `ClaudeAPIProvider` | `llm/claudeapi` | 나중에 | `ANTHROPIC_API_KEY` |
| `GoogleADKProvider` | `llm/googleadk` | 나중에 | `GOOGLE_API_KEY` |

### ClaudeCLIProvider 동작

1. `claude -p --output-format stream-json --mcp-config <path>` subprocess 실행
2. stdin에 시스템 프롬프트 + 사용자 쿼리를 파이프로 전달
3. stdout에서 JSON 이벤트를 줄 단위로 파싱
4. `Event` 채널로 변환하여 반환
5. ctx 취소 시 `cmd.Process.Kill()`로 subprocess 정리

### Subprocess 생명주기

- **타임아웃**: 60초 (context.WithTimeout). 초과 시 subprocess kill + EventError 전송.
- **클라이언트 연결 해제**: Gin의 `c.Request.Context()`가 취소됨 → Provider에 전파 → subprocess kill.
- **동시성 제한**: `sync.Semaphore` (최대 5개 동시 subprocess). 초과 시 429 Too Many Requests 응답.
- **비정상 종료**: subprocess exit code != 0 → stderr 읽어서 EventError 전송 + 채널 close.

### Config

```go
// backend/internal/config/config.go에 추가

type LLMConfig struct {
    Provider       string // "claude-cli", "claude-api", "google-adk"
    ClaudeCLIPath  string // claude binary 경로 (기본: "claude")
    MCPConfigPath  string // mcp-config.json 경로 (절대 경로 권장)
    AnthropicKey   string // Claude API용 (선택)
    MaxConcurrent  int    // 최대 동시 LLM 호출 수 (기본: 5)
    TimeoutSeconds int    // subprocess 타임아웃 (기본: 60)
}
```

환경변수:
- `LLM_PROVIDER` (기본: `claude-cli`)
- `CLAUDE_CLI_PATH` (기본: `claude`)
- `MCP_CONFIG_PATH` (기본: 절대 경로로 변환된 `./mcp-config.json`)
- `ANTHROPIC_API_KEY` (선택, Claude API 전환 시)
- `LLM_MAX_CONCURRENT` (기본: `5`)
- `LLM_TIMEOUT_SECONDS` (기본: `60`)

### Dependency Injection

```go
// backend/cmd/api/main.go 변경

// 1. LLM Provider 생성
llmProvider := claudecli.New(cfg.LLM)

// 2. 기존 서비스에 Provider 주입
nlSvc := service.NewNLToDSLService(llmProvider)

// 3. SearchHandler에 주입 (기존 구조 유지)
searchHandler := handler.NewSearchHandler(searchSvc, nlSvc)
```

```go
// backend/internal/service/nl_to_dsl_service.go 변경

type NLToDSLService struct {
    provider llm.Provider
}

func NewNLToDSLService(provider llm.Provider) *NLToDSLService {
    return &NLToDSLService{provider: provider}
}

// Stream returns the event channel from the provider.
func (s *NLToDSLService) Stream(ctx context.Context, query string) (<-chan llm.Event, error) {
    return s.provider.NLToDSL(ctx, query)
}

// Explain returns a natural language explanation of the DSL.
func (s *NLToDSLService) Explain(ctx context.Context, dsl string) (string, error) {
    return s.provider.Explain(ctx, dsl)
}
```

기존 `NLToDSLResult` 구조체와 `Convert()` 메서드는 제거한다.

---

## Section 2: Go MCP Server

별도 바이너리 `backend/cmd/mcp-server/main.go`로 빌드.

### 바이너리 부트스트랩

```go
// backend/cmd/mcp-server/main.go

func main() {
    // 1. CLI 플래그 파싱
    dbURL := flag.String("db-url", "", "PostgreSQL connection URL")
    flag.Parse()

    // 2. DB 연결 풀 생성 (선택적 — validate에는 불필요, 향후 확장용)
    var pool *pgxpool.Pool
    if *dbURL != "" {
        pool, _ = pgxpool.New(context.Background(), *dbURL)
        defer pool.Close()
    }

    // 3. DSL Executor 초기화
    executor := dsl.NewExecutor(pool)

    // 4. MCP Server 실행 (stdin/stdout)
    server := mcp.NewServer(executor)
    server.Run(context.Background())
}
```

**`claude -p`의 환경변수 전달**: MCP 설정의 `"args": ["--db-url", "${DATABASE_URL}"]`에서 `${DATABASE_URL}`은 `claude -p`가 셸 확장하지 않는다. 대신 MCP 설정에 `"env"` 필드를 사용하거나, MCP 서버가 직접 `os.Getenv("DATABASE_URL")`을 읽도록 한다.

```json
// backend/mcp-config.json (수정)
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

MCP 서버 내부에서 `os.Getenv("DATABASE_URL")` 사용. `--db-url` 플래그는 override용으로 유지.

### 프로토콜

JSON-RPC 2.0 over stdio (MCP 표준). 구현할 메서드:
- `initialize` — 서버 정보 반환
- `tools/list` — 도구 목록 반환
- `tools/call` — 도구 실행

### 도구 정의 (3개)

#### get_dsl_grammar

```json
{
  "name": "get_dsl_grammar",
  "description": "Returns the complete DSL grammar rules and syntax guide",
  "inputSchema": { "type": "object", "properties": {} }
}
```

응답: DSL 문법 규칙 텍스트. 포함 내용:
- `scan where <conditions> [sort by <field> [asc|desc]] [limit N]`
- 조건: `<field> <op> <value>` (AND로 연결, OR 미지원)
- 연산자: `>`, `<`, `>=`, `<=`, `=`
- 기본 limit: 50, 최대 limit: 500

#### list_available_fields

```json
{
  "name": "list_available_fields",
  "description": "Lists all available fields and operators for DSL queries",
  "inputSchema": { "type": "object", "properties": {} }
}
```

응답: 백엔드 `allowedFields`와 동일한 필드 목록:
- `close` — 종가/현재가 (숫자)
- `open` — 시가 (숫자)
- `high` — 고가 (숫자)
- `low` — 저가 (숫자)
- `volume` — 거래량 (정수)
- `trade_value` — 거래대금 (close × volume, 숫자)
- `change_pct` — 전일 대비 등락률 (%, 숫자)

#### validate_dsl

```json
{
  "name": "validate_dsl",
  "description": "Validates a DSL query string for syntax errors. Returns validation result with error details if invalid.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "dsl": { "type": "string", "description": "DSL query to validate" }
    },
    "required": ["dsl"]
  }
}
```

성공: `{ "valid": true }`
실패: MCP 표준에 따라 `isError: true` + `content: [{ type: "text", text: "error message" }]`

### MCP 서버 내부 구조

```go
// backend/internal/mcp/server.go
type Server struct {
    executor *dsl.Executor
}

func NewServer(executor *dsl.Executor) *Server

// Run reads JSON-RPC requests from stdin, dispatches to tools, writes responses to stdout.
func (s *Server) Run(ctx context.Context) error
```

```go
// backend/internal/mcp/tools.go
// 각 도구의 handler 함수 + tool definition 상수
func (s *Server) handleGetDSLGrammar() string
func (s *Server) handleListAvailableFields() string
func (s *Server) handleValidateDSL(dsl string) (string, error)
```

DSL Executor를 직접 import하여 in-process로 validate.

---

## Section 3: SSE Streaming API

### 엔드포인트 변경

| 엔드포인트 | 기존 | 변경 | 이유 |
|-----------|------|------|------|
| `POST /search/nl-to-dsl` | 동기 JSON | SSE 스트림 | LLM 호출이 수 초 걸림, 단계별 피드백 필요 |
| `POST /search/explain` | 동기 JSON | **유지 (동기)** | 단일 텍스트 응답, 스트리밍 불필요 |
| `POST /search/execute` | 동기 JSON | 유지 | LLM 불필요 |
| `POST /search/validate` | 동기 JSON | 유지 | LLM 불필요 |

**Note**: `POST`로 SSE를 보내는 이유 — 브라우저 `EventSource` API는 GET만 지원하므로 사용할 수 없다. `fetch()` + `ReadableStream`으로 수동 파싱한다.

### SSE 이벤트 포맷

```
event: thinking
data: {"message": "쿼리를 분석하고 있습니다..."}

event: tool_call
data: {"tool": "get_dsl_grammar", "message": "DSL 문법 확인 중..."}

event: tool_result
data: {"tool": "get_dsl_grammar", "message": "문법 로드 완료"}

event: tool_call
data: {"tool": "validate_dsl", "message": "DSL 검증 중..."}

event: tool_result
data: {"tool": "validate_dsl", "message": "검증 통과"}

event: dsl_ready
data: {"dsl": "scan where volume > 1000000 sort by trade_value desc", "explanation": "거래량 100만 이상 종목을 거래대금 순으로 정렬"}

event: done
data: {"results": [...], "count": 23}
```

### 백엔드 핸들러

```go
func (h *SearchHandler) NLToDSL(c *gin.Context) {
    // 1. 요청 파싱 + 입력 길이 검증 (maxDSLLength)
    var req NLToDSLRequest
    if err := c.ShouldBindJSON(&req); err != nil { ... }
    if !validateInputLength(c, req.Query, "query", maxDSLLength) { return }

    // 2. SSE 헤더 설정
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    // 3. Provider에서 이벤트 스트림 수신
    events, err := h.nlSvc.Stream(c.Request.Context(), req.Query)
    if err != nil { ... }

    // 4. 이벤트 → SSE 포맷으로 flush
    var finalDSL string
    for event := range events {
        writeSSE(c, event)
        if event.Type == llm.EventDSLReady {
            finalDSL = event.DSL
        }
    }

    // 5. DSL 실행 (핸들러가 담당)
    if finalDSL != "" {
        results, err := h.searchSvc.Execute(c.Request.Context(), finalDSL)
        writeSSE(c, llm.Event{Type: llm.EventDone, Data: results})
    }
}
```

### 프론트엔드 SSE 타입 정의

```typescript
// features/search/model/types.ts에 추가

type SSEEventType = "thinking" | "tool_call" | "tool_result" | "dsl_ready" | "done" | "error";

type SSEEvent =
  | { type: "thinking"; message: string }
  | { type: "tool_call"; tool: string; message: string }
  | { type: "tool_result"; tool: string; message: string }
  | { type: "dsl_ready"; dsl: string; explanation: string }
  | { type: "done"; results: SearchResult[]; count: number }
  | { type: "error"; message: string };
```

### 프론트엔드 SSE 클라이언트

```typescript
// features/search/api/search-api.ts

async function* nlSearchStream(query: string): AsyncGenerator<SSEEvent> {
  const response = await fetch("/api/v1/search/nl-to-dsl", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query }),
  });

  if (!response.ok) throw new Error(`HTTP ${response.status}`);

  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    // SSE 파싱: "event:" + "data:" 라인 분리 → JSON.parse → yield
    // 빈 줄(\n\n)이 이벤트 구분자
    const events = parseSSEBuffer(buffer);
    buffer = events.remaining;
    for (const event of events.parsed) {
      yield event;
    }
  }
}
```

### use-search-actions.ts 변경

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
          setState({ dslCode: event.dsl, explanation: event.explanation,
                     agentStatus: "scanning", agentMessage: "검색 중..." });
          break;
        case "done":
          setState({ results: event.results, agentStatus: "done",
                     agentMessage: `${event.count}개 종목 발견` });
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

### agentStatus 매핑

| SSE event | agentStatus | agentMessage |
|-----------|-------------|--------------|
| `thinking` | `interpreting` | 서버 메시지 |
| `tool_call` | `building` | 도구명 + 메시지 |
| `tool_result` | `building` | 결과 메시지 |
| `dsl_ready` | `scanning` | "검색 중..." |
| `done` | `done` | "N개 종목 발견" |
| `error` | `error` | 에러 메시지 |

기존 `AgentStatus` 타입과 `AgentStatusIndicator` UI를 그대로 활용.
`AgentStatusIndicator` 변경: `building` 상태일 때 `agentMessage`를 직접 표시 (현재 하드코딩된 "Building DSL..." 대신).

---

## Section 4: DSL Editor Enhancement

### 4-1. 문맥 인식 자동완성

커서 위치의 토큰 스트림을 분석하여 적절한 제안만 표시:

| 문맥 | 제안 |
|------|------|
| 빈 에디터 | `scan` |
| `scan` 뒤 | `where` |
| `where` 뒤 또는 `and` 뒤 | 필드 목록 (백엔드 지원 필드만: close, open, high, low, volume, trade_value, change_pct) |
| 필드 뒤 | 연산자 (`>`, `<`, `>=`, `<=`, `=`) |
| 값 뒤 | `and`, `sort`, `limit` |
| `sort` 뒤 | `by` |
| `sort by` 뒤 | 필드 목록 |
| 정렬 필드 뒤 | `asc`, `desc` |

**필드 목록 정리**: 현재 `dsl-completions.ts`에 있는 `market_cap`, `per`, `pbr`, `roe`, `event_*`, `post_*`, `days_since_event` 및 함수들 (`ma`, `rsi`, `macd`, `bb` 등)은 백엔드 `allowedFields`에 없으므로 **제거**한다. 이 필드들은 향후 DSL 문법 확장 시 다시 추가.

구현: 기존 `shared/lib/dsl/lexer.ts` 토큰 스트림을 활용한 문맥 판단 함수.

변경 파일: `features/search/lib/dsl-completions.ts`

### 4-2. 실시간 Lint

프론트엔드 전용 (백엔드 호출 없음, debounce 300ms):

1. TS lexer로 토큰화
2. 기본 문법 체크:
   - `scan` 키워드 존재 여부
   - `where` 존재 여부
   - 필드명 유효성 (allowedFields에 포함 여부)
   - 연산자 유효성
   - 숫자 값 유효성
   - `or` 사용 시 "OR은 지원되지 않습니다. AND를 사용하세요" 에러
3. 에러 위치에 빨간 밑줄 + 인라인 메시지 표시

CodeMirror `@codemirror/lint` extension의 `linter()` 함수 활용.

신규 파일: `features/search/lib/dsl-linter.ts`
변경 파일: `features/search/ui/DSLEditor.tsx` (lint extension 연결)

"Validate" 버튼은 유지 — 백엔드 정밀 검증 용도.

---

## Section 5: System Prompts

### NL→DSL 프롬프트 (`backend/internal/llm/prompts/nl-to-dsl.txt`)

핵심 지시:
- 한국 주식 시장 검색 DSL 생성 전문가 역할
- 반드시 `get_dsl_grammar` 먼저 호출하여 문법 확인
- DSL 생성 후 `validate_dsl`로 반드시 검증
- 검증 실패 시 수정 후 재검증 (최대 3회)
- 최종 DSL과 한국어 설명을 함께 반환
- 모호한 요청에는 합리적 기본값 사용 (limit 미지정 시 50)
- **OR 연산자는 사용하지 않는다** — 현재 파서는 AND만 지원
- **execute_dsl 도구는 없다** — DSL 생성과 검증만 수행하고 실행은 시스템이 담당

### Explain 프롬프트 (`backend/internal/llm/prompts/explain.txt`)

핵심 지시:
- DSL 쿼리를 한국어로 자연스럽게 설명
- 각 조건의 의미와 전체 쿼리의 목적 설명

---

## File Structure (New/Changed)

```
backend/
├── cmd/
│   ├── api/main.go                          # 변경: LLM provider 초기화, DI 수정
│   └── mcp-server/main.go                   # 신규: MCP 바이너리 (DB pool + Executor 부트스트랩)
├── internal/
│   ├── llm/
│   │   ├── provider.go                      # 신규: Provider interface + Event 타입
│   │   ├── claudecli/
│   │   │   └── provider.go                  # 신규: claude -p subprocess (타임아웃, 취소, 세마포어)
│   │   └── prompts/
│   │       ├── nl-to-dsl.txt                # 신규
│   │       └── explain.txt                  # 신규
│   ├── mcp/
│   │   ├── server.go                        # 신규: JSON-RPC stdio 서버
│   │   └── tools.go                         # 신규: 3개 도구 (get_dsl_grammar, list_available_fields, validate_dsl)
│   ├── config/config.go                     # 변경: LLMConfig 추가
│   ├── handler/search_handler.go            # 변경: NLToDSL → SSE, Explain → Provider 위임
│   └── service/nl_to_dsl_service.go         # 변경: Provider 위임, Convert() 제거 → Stream()/Explain()
├── mcp-config.json                          # 신규: env 기반 DB URL 전달
└── Makefile                                 # 변경: build-mcp 타겟

src/features/search/
├── api/search-api.ts                        # 변경: nlSearchStream() AsyncGenerator 추가, parseSSEBuffer() 유틸
├── lib/
│   ├── dsl-completions.ts                   # 변경: 문맥 인식, 미지원 필드/함수 제거
│   └── dsl-linter.ts                        # 신규: 실시간 lint (OR 미지원 에러 포함)
├── model/
│   ├── use-search-actions.ts                # 변경: runNLSearch → for-await SSE 소비
│   └── types.ts                             # 변경: SSEEvent union 타입 추가
└── ui/
    ├── DSLEditor.tsx                        # 변경: @codemirror/lint extension 추가
    └── AgentStatusIndicator.tsx             # 변경: building 시 agentMessage 직접 표시
```

---

## Build & Run

```bash
# 빌드
make build-mcp    # go build -o bin/mcp-server ./cmd/mcp-server
make build        # api + worker + mcp-server 모두

# 개발 환경
./dev.sh start    # 기존대로 (mcp-server는 claude -p가 자동 실행)

# 환경변수 (선택)
LLM_PROVIDER=claude-cli          # 기본값
CLAUDE_CLI_PATH=claude           # 기본값
MCP_CONFIG_PATH=./mcp-config.json  # 기본값
LLM_MAX_CONCURRENT=5             # 기본값
LLM_TIMEOUT_SECONDS=60           # 기본값
```

---

## Security Considerations

- MCP 서버는 `claude -p`의 subprocess로만 실행 (외부 노출 없음)
- LLM에게 DSL 실행 권한 없음 (execute_dsl 도구 제외, 핸들러가 실행 담당)
- DSL 입력 길이 검증: SSE 엔드포인트 포함 모든 엔드포인트에서 `maxDSLLength` 적용
- `ANTHROPIC_API_KEY`는 프로덕션에서만 필요, 환경변수로 관리
- 동시성 제한: 최대 N개 subprocess (DoS 방지)

## Testing Strategy

- **MCP 서버 도구 테스트**: 각 도구의 입출력 검증 (Go 유닛 테스트)
- **Provider 테스트**: mock MCP 서버로 ClaudeCLIProvider 통합 테스트
- **SSE 핸들러 테스트**: httptest로 SSE 이벤트 스트림 검증
- **프론트엔드 테스트**:
  - 문맥 인식 자동완성 로직 유닛 테스트 (각 문맥별 올바른 제안 검증)
  - lint 로직 유닛 테스트 (유효/무효 DSL에 대한 에러 위치/메시지 검증)
  - SSE 파서 (`parseSSEBuffer`) 유닛 테스트
- **E2E 테스트**: NL 탭 → SSE 스트림 → 결과 표시 flow
