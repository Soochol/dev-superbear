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

---

## Architecture

```
Frontend (Next.js)
    │ SSE (POST)
    ▼
Go Backend (Gin)
    ├── /search/nl-to-dsl  (SSE endpoint)
    │      │
    │      ▼
    │   LLMProvider interface
    │      ├── ClaudeCLIProvider  (subprocess: claude -p --mcp-config)
    │      ├── ClaudeAPIProvider  (나중에)
    │      └── GoogleADKProvider  (나중에)
    │
    ├── Go MCP Server (별도 바이너리, stdio)
    │      ├── get_dsl_grammar
    │      ├── list_available_fields
    │      ├── validate_dsl
    │      └── execute_dsl
    │
    └── DSL Executor (기존 코드 유지)
```

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
    Type    EventType `json:"type"`
    Message string    `json:"message"`
    DSL     string    `json:"dsl,omitempty"`
    Data    any       `json:"data,omitempty"`
}

type Provider interface {
    NLToDSL(ctx context.Context, query string) (<-chan Event, error)
    Explain(ctx context.Context, dsl string) (<-chan Event, error)
    Name() string
}
```

### Provider 구현 계획

| Provider | 패키지 | 사용 시점 | API 키 필요 |
|----------|--------|----------|------------|
| `ClaudeCLIProvider` | `llm/claudecli` | 1차 구현 | 불필요 (구독 세션) |
| `ClaudeAPIProvider` | `llm/claudeapi` | 나중에 | `ANTHROPIC_API_KEY` |
| `GoogleADKProvider` | `llm/googleadk` | 나중에 | `GOOGLE_API_KEY` |

### ClaudeCLIProvider 동작

1. `claude -p --output-format stream-json --mcp-config <path>` subprocess 실행
2. stdin에 시스템 프롬프트 + 사용자 쿼리 전달
3. stdout에서 JSON 이벤트 파싱
4. `Event` 채널로 변환하여 반환

### Config

```go
// backend/internal/config/config.go에 추가

type LLMConfig struct {
    Provider       string // "claude-cli", "claude-api", "google-adk"
    ClaudeCLIPath  string // claude binary 경로 (기본: "claude")
    MCPConfigPath  string // mcp-config.json 경로
    AnthropicKey   string // Claude API용 (선택)
}
```

환경변수:
- `LLM_PROVIDER` (기본: `claude-cli`)
- `CLAUDE_CLI_PATH` (기본: `claude`)
- `MCP_CONFIG_PATH` (기본: `./mcp-config.json`)
- `ANTHROPIC_API_KEY` (선택, Claude API 전환 시)

---

## Section 2: Go MCP Server

별도 바이너리 `backend/cmd/mcp-server/main.go`로 빌드.

### 프로토콜

JSON-RPC 2.0 over stdio (MCP 표준). 구현할 메서드:
- `initialize` — 서버 정보 반환
- `tools/list` — 도구 목록 반환
- `tools/call` — 도구 실행

### 도구 정의

#### get_dsl_grammar

```json
{
  "name": "get_dsl_grammar",
  "description": "Returns the complete DSL grammar rules and syntax guide",
  "inputSchema": { "type": "object", "properties": {} }
}
```

응답: DSL 문법 규칙 텍스트 (scan where 구문, 연산자, sort/limit 등)

#### list_available_fields

```json
{
  "name": "list_available_fields",
  "description": "Lists all available fields, operators, and functions for DSL queries",
  "inputSchema": { "type": "object", "properties": {} }
}
```

응답: 필드 목록 (close, open, high, low, volume, trade_value, change_pct) + 설명 + 타입

#### validate_dsl

```json
{
  "name": "validate_dsl",
  "description": "Validates a DSL query string for syntax errors",
  "inputSchema": {
    "type": "object",
    "properties": {
      "dsl": { "type": "string", "description": "DSL query to validate" }
    },
    "required": ["dsl"]
  }
}
```

응답: `{ "valid": true }` 또는 `{ "valid": false, "error": "..." }`

#### execute_dsl

```json
{
  "name": "execute_dsl",
  "description": "Executes a validated DSL query and returns matching stocks",
  "inputSchema": {
    "type": "object",
    "properties": {
      "dsl": { "type": "string", "description": "DSL query to execute" }
    },
    "required": ["dsl"]
  }
}
```

응답: `{ "results": [...], "count": N }`

### MCP 서버 내부 구조

```go
// backend/internal/mcp/server.go
type Server struct {
    executor *dsl.Executor
}

// stdin에서 JSON-RPC 요청 읽기 → 처리 → stdout으로 응답
func (s *Server) Run(ctx context.Context) error
```

DSL Executor를 직접 import하여 in-process로 validate/execute.

### MCP 설정 파일

```json
// backend/mcp-config.json
{
  "mcpServers": {
    "nexus-dsl": {
      "command": "./bin/mcp-server",
      "args": ["--db-url", "${DATABASE_URL}"]
    }
  }
}
```

---

## Section 3: SSE Streaming API

### 엔드포인트 변경

| 엔드포인트 | 기존 | 변경 |
|-----------|------|------|
| `POST /search/nl-to-dsl` | 동기 JSON | SSE 스트림 |
| `POST /search/explain` | 동기 JSON | SSE 스트림 |
| `POST /search/execute` | 동기 JSON | 유지 (LLM 불필요) |
| `POST /search/validate` | 동기 JSON | 유지 (LLM 불필요) |

### SSE 이벤트 포맷

```
event: thinking
data: {"message": "쿼리를 분석하고 있습니다..."}

event: tool_call
data: {"tool": "get_dsl_grammar", "message": "DSL 문법 확인 중..."}

event: tool_result
data: {"tool": "get_dsl_grammar", "message": "문법 로드 완료"}

event: tool_call
data: {"tool": "validate_dsl", "input": {"dsl": "scan where volume > 1000000"}}

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
    // 1. 요청 파싱
    // 2. SSE 헤더 설정 (Content-Type: text/event-stream)
    // 3. Provider.NLToDSL() 채널에서 이벤트 수신
    // 4. 각 이벤트를 SSE 포맷으로 flush
    // 5. dsl_ready 이벤트 수신 시 → executor.Execute() 호출
    // 6. done 이벤트 전송
}
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
  const reader = response.body!.getReader();
  const decoder = new TextDecoder();
  // SSE 파싱 → yield events
}
```

### agentStatus 매핑

| SSE event | agentStatus | agentMessage |
|-----------|-------------|--------------|
| `thinking` | `interpreting` | 서버 메시지 |
| `tool_call` | `building` | 도구명 + 메시지 |
| `tool_result` | `building` | 결과 메시지 |
| `dsl_ready` | `scanning` | "DSL 생성 완료, 검색 중..." |
| `done` | `done` | "N개 종목 발견" |
| `error` | `error` | 에러 메시지 |

기존 `AgentStatus` 타입과 `AgentStatusIndicator` UI를 그대로 활용.

---

## Section 4: DSL Editor Enhancement

### 4-1. 문맥 인식 자동완성

커서 위치의 토큰 스트림을 분석하여 적절한 제안만 표시:

| 문맥 | 제안 |
|------|------|
| 빈 에디터 | `scan` |
| `scan` 뒤 | `where` |
| `where` 뒤 또는 `and` 뒤 | 필드 목록 |
| 필드 뒤 | 연산자 (`>`, `<`, `>=`, `<=`, `=`) |
| 값 뒤 | `and`, `sort`, `limit` |
| `sort` 뒤 | `by` |
| `sort by` 뒤 | 필드 목록 |
| 정렬 필드 뒤 | `asc`, `desc` |

구현: 기존 `shared/lib/dsl/lexer.ts` 토큰 스트림을 활용한 문맥 판단 함수.

변경 파일: `features/search/lib/dsl-completions.ts`

### 4-2. 실시간 Lint

프론트엔드 전용 (백엔드 호출 없음, debounce 300ms):

1. TS lexer로 토큰화
2. 기본 문법 체크:
   - `scan` 키워드 존재 여부
   - `where` 존재 여부
   - 필드명 유효성
   - 연산자 유효성
   - 숫자 값 유효성
3. 에러 위치에 빨간 밑줄 + 인라인 메시지 표시

CodeMirror `linter` extension 활용.

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
- execute_dsl은 사용자가 명시적으로 실행 요청한 경우에만 호출

### Explain 프롬프트 (`backend/internal/llm/prompts/explain.txt`)

핵심 지시:
- DSL 쿼리를 한국어로 자연스럽게 설명
- 각 조건의 의미와 전체 쿼리의 목적 설명

---

## File Structure (New/Changed)

```
backend/
├── cmd/
│   ├── api/main.go                          # 변경: LLM provider 초기화
│   └── mcp-server/main.go                   # 신규
├── internal/
│   ├── llm/
│   │   ├── provider.go                      # 신규: interface + Event
│   │   ├── claudecli/
│   │   │   └── provider.go                  # 신규: claude -p subprocess
│   │   └── prompts/
│   │       ├── nl-to-dsl.txt                # 신규
│   │       └── explain.txt                  # 신규
│   ├── mcp/
│   │   ├── server.go                        # 신규: JSON-RPC stdio
│   │   └── tools.go                         # 신규: 4 tools
│   ├── config/config.go                     # 변경: LLMConfig 추가
│   ├── handler/search_handler.go            # 변경: SSE 응답
│   └── service/nl_to_dsl_service.go         # 변경: Provider 위임
├── mcp-config.json                          # 신규
└── Makefile                                 # 변경: build-mcp 타겟

src/features/search/
├── api/search-api.ts                        # 변경: SSE 클라이언트
├── lib/
│   ├── dsl-completions.ts                   # 변경: 문맥 인식
│   └── dsl-linter.ts                        # 신규
├── model/
│   ├── use-search-actions.ts                # 변경: SSE 소비
│   └── types.ts                             # 변경: SSE 타입
└── ui/
    ├── DSLEditor.tsx                        # 변경: lint extension
    └── AgentStatusIndicator.tsx             # 변경: tool_call 표시
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
LLM_PROVIDER=claude-cli    # 기본값
CLAUDE_CLI_PATH=claude     # 기본값
MCP_CONFIG_PATH=./mcp-config.json  # 기본값
```

---

## Security Considerations

- MCP 서버는 `claude -p`의 subprocess로만 실행 (외부 노출 없음)
- `execute_dsl` 도구는 파라미터화된 쿼리 사용 (SQL injection 방지, 기존 executor 로직)
- `ANTHROPIC_API_KEY`는 프로덕션에서만 필요, 환경변수로 관리
- DSL 입력 길이 제한 유지 (maxDSLLength: 10000)

## Testing Strategy

- **MCP 서버 도구 테스트**: 각 도구의 입출력 검증 (Go 유닛 테스트)
- **Provider 테스트**: mock MCP 서버로 ClaudeCLIProvider 통합 테스트
- **SSE 핸들러 테스트**: httptest로 SSE 이벤트 스트림 검증
- **프론트엔드 테스트**: 문맥 인식 자동완성 로직 유닛 테스트, lint 로직 유닛 테스트
- **E2E 테스트**: NL 탭 → SSE 스트림 → 결과 표시 flow
