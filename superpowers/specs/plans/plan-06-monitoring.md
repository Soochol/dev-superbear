# 모니터링 엔진 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** LIVE 케이스를 대상으로 에이전트 블록(cron)과 가격 폴링(DSL)을 자동 실행하여 타임라인에 결과를 축적하는 백엔드 모니터링 엔진을 구축한다.
**Architecture:** asynq(Go) 기반 작업 큐를 사용하여 에이전트 모니터링 블록의 cron 스케줄 실행과 DSL 가격 폴링 핸들러를 분리 관리한다. 가격 폴링은 1분 간격으로 KIS API를 배치 호출하며, 에이전트 블록은 개별 cron 표현식에 따라 asynq scheduler에 등록된다. 모든 실행 결과는 TimelineEvent 레코드로 저장되고, 성공/실패 조건 도달 시 케이스 상태를 자동 전환한다.
**Tech Stack:** Go, asynq + Redis, robfig/cron/v3, sqlc (PostgreSQL, pgx/v5), KIS API, DSL Engine (Plan 1), Agent Runtime (Plan 4)

**Go 패키지 구조:**
```
backend/
  cmd/worker/main.go                          # asynq 워커 서버 + 스케줄러 엔트리포인트
  internal/
    worker/
      task_types.go                            # asynq task type 상수 및 payload 정의
      monitor_agent.go                         # 에이전트 블록 실행 핸들러
      dsl_poller.go                            # 가격 폴링 핸들러
      lifecycle.go                             # 케이스 상태 전환 핸들러
      scheduler.go                             # cron 스케줄 등록 (asynq scheduler)
    service/
      monitoring_service.go                    # 모니터링 서비스 (큐 등록/해제, 중단/재개)
      metrics.go                               # 메트릭 수집
    handler/
      monitoring_handler.go                    # HTTP 핸들러 (중단/재개, 헬스체크)
    infra/
      redis.go                                 # Redis 연결 설정
      market_time.go                           # 장중 시간 유틸리티
      kis/
        batch_client.go                        # KIS API 배치 클라이언트
    config/
      config.go                                # 환경변수 로드
```

---

## 의존성

- **Plan 4 (파이프라인 빌더)**: Pipeline, AgentBlock, MonitorBlock, PriceAlert 모델 및 에이전트 실행 런타임
- **Plan 5 (케이스 관리)**: Case, TimelineEvent 모델 및 타임라인 CRUD API

---

## Task 1: asynq 인프라 및 태스크 타입 설정

모니터링 엔진의 기반이 되는 asynq 작업 큐, Redis 연결, 태스크 타입을 구성한다.

**Files:**
- Create: `backend/internal/infra/redis.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/worker/task_types.go`
- Modify: `docker-compose.yml` (Redis 서비스 추가 또는 기존 파일 수정)

**Steps:**

- [ ] Redis 연결 설정 및 환경 변수 정의

```go
// backend/internal/infra/redis.go
package infra

import (
	"github.com/hibiken/asynq"
)

// NewRedisClientOpt returns the asynq Redis connection option
// from the given address string (e.g. "localhost:6379").
func NewRedisClientOpt(addr, password string) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     addr,
		Password: password,
	}
}
```

```go
// backend/internal/config/config.go
package config

import "os"

type Config struct {
	RedisAddr     string
	RedisPassword string
}

func Load() *Config {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return &Config{
		RedisAddr:     addr,
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
	}
}
```

- [ ] asynq 태스크 타입 상수 및 payload 구조체 정의 — 모니터링 도메인 전용

```go
// backend/internal/worker/task_types.go
package worker

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// ── Task type constants ─────────────────────────────────────────
const (
	TypeMonitorAgent     = "monitor:agent"
	TypeDSLPoller        = "monitor:dsl-poll"
	TypeMonitorLifecycle = "monitor:lifecycle"
)

// ── Payloads ────────────────────────────────────────────────────

type MonitorAgentPayload struct {
	CaseID           string   `json:"case_id"`
	MonitorBlockID   string   `json:"monitor_block_id"`
	PipelineID       string   `json:"pipeline_id"`
	Symbol           string   `json:"symbol"`
	BlockInstruction string   `json:"block_instruction"`
	AllowedTools     []string `json:"allowed_tools"`
}

type DSLPollingPayload struct {
	CaseID        string `json:"case_id"`
	Symbol        string `json:"symbol"`
	SuccessScript string `json:"success_script"`
	FailureScript string `json:"failure_script"`
	PriceAlerts   []struct {
		ID        string `json:"id"`
		Condition string `json:"condition"`
		Label     string `json:"label"`
	} `json:"price_alerts"`
	EventSnapshot map[string]interface{} `json:"event_snapshot"`
}

type LifecyclePayload struct {
	CaseID  string `json:"case_id"`
	Action  string `json:"action"` // "CLOSE_SUCCESS" | "CLOSE_FAILURE" | "TRIGGER_ALERT"
	Reason  string `json:"reason"`
	AlertID string `json:"alert_id,omitempty"`
}

// DSLContext built from a Case + live price snapshot
type DSLContext struct {
	Close          float64 `json:"close"`
	High           float64 `json:"high"`
	Low            float64 `json:"low"`
	Volume         float64 `json:"volume"`
	EventHigh      float64 `json:"event_high"`
	EventLow       float64 `json:"event_low"`
	EventClose     float64 `json:"event_close"`
	EventVolume    float64 `json:"event_volume"`
	PreEventMA5    float64 `json:"pre_event_ma_5"`
	PreEventMA20   float64 `json:"pre_event_ma_20"`
	PreEventMA60   float64 `json:"pre_event_ma_60"`
	PreEventMA120  float64 `json:"pre_event_ma_120"`
	PreEventMA200  float64 `json:"pre_event_ma_200"`
	PreEventClose  float64 `json:"pre_event_close"`
}

// MonitoringMetrics collected from queue inspection and DB counts.
type MonitoringMetrics struct {
	DSLPollingDurationMs    int64 `json:"dsl_polling_duration_ms"`
	AgentExecutionDurationMs int64 `json:"agent_execution_duration_ms"`
	ActiveCases             int   `json:"active_cases"`
	ActiveMonitorBlocks     int   `json:"active_monitor_blocks"`
	QueueDepth              struct {
		Agent     int `json:"agent"`
		DSL       int `json:"dsl"`
		Lifecycle int `json:"lifecycle"`
	} `json:"queue_depth"`
}

// ── Task constructors ───────────────────────────────────────────

func NewMonitorAgentTask(p MonitorAgentPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal MonitorAgentPayload: %w", err)
	}
	return asynq.NewTask(TypeMonitorAgent, data, asynq.MaxRetry(3)), nil
}

func NewDSLPollerTask() *asynq.Task {
	return asynq.NewTask(TypeDSLPoller, nil, asynq.MaxRetry(2))
}

func NewLifecycleTask(p LifecyclePayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal LifecyclePayload: %w", err)
	}
	return asynq.NewTask(TypeMonitorLifecycle, data), nil
}
```

- [ ] docker-compose.yml에 Redis 서비스 추가

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - '6379:6379'
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes
volumes:
  redis-data:
```

- [ ] `.env.example`에 `REDIS_ADDR`, `REDIS_PASSWORD` 추가

```bash
go get github.com/hibiken/asynq
git add backend/internal/infra/redis.go backend/internal/config/config.go backend/internal/worker/task_types.go docker-compose.yml .env.example
git commit -m "feat(monitoring): asynq 인프라 및 태스크 타입 설정"
```

---

## Task 2: 에이전트 모니터링 핸들러 (cron 기반)

MonitorBlock의 cron 스케줄에 따라 에이전트 블록을 실행하고 결과를 타임라인에 기록하는 asynq 핸들러를 구현한다. asynq scheduler를 통해 cron 표현식으로 주기적 태스크를 등록한다.

**Files:**
- Create: `backend/internal/worker/monitor_agent.go`
- Create: `backend/internal/worker/scheduler.go`
- Create: `backend/db/queries/monitoring.sql` (sqlc 쿼리)

**Steps:**

- [ ] sqlc 쿼리 정의 (`backend/db/queries/monitoring.sql`) — 모니터링 엔진에서 사용하는 모든 쿼리

```sql
-- backend/db/queries/monitoring.sql

-- name: ListActiveMonitorBlocks :many
-- 스케줄러 동기화: LIVE 케이스의 enabled MonitorBlock 조회
SELECT mb.id AS monitor_block_id,
       c.id  AS case_id,
       c.pipeline_id,
       c.symbol,
       mb.cron,
       ab.instruction AS block_instruction,
       ab.allowed_tools
FROM monitor_blocks mb
JOIN cases c ON c.id = mb.case_id
JOIN agent_blocks ab ON ab.id = mb.block_id
WHERE mb.enabled = true AND c.status = 'LIVE';

-- name: GetCaseEventSnapshot :one
-- 에이전트 핸들러: 케이스 이벤트 스냅샷 조회
SELECT id, event_snapshot
FROM cases
WHERE id = $1;

-- name: ListLiveCases :many
-- DSL 폴러: LIVE 상태 케이스 목록
SELECT id, symbol, success_script, failure_script, event_snapshot
FROM cases
WHERE status = 'LIVE';

-- name: ListUntriggeredAlertsByCase :many
-- DSL 폴러: 미트리거 가격 알림 조회
SELECT id, condition, label
FROM price_alerts
WHERE case_id = $1 AND triggered = false;

-- name: DisableMonitorBlocksByCase :exec
-- 라이프사이클: 케이스의 모든 모니터 블록 비활성화
UPDATE monitor_blocks SET enabled = false WHERE case_id = $1;

-- name: TriggerPriceAlert :exec
-- 라이프사이클: 가격 알림 트리거 상태 업데이트
UPDATE price_alerts SET triggered = true, triggered_at = $2 WHERE id = $1;

-- name: UpdateMonitorBlockEnabled :exec
-- 서비스: 개별 모니터 블록 활성/비활성
UPDATE monitor_blocks SET enabled = $2 WHERE id = $1;

-- name: UpdateMonitorBlocksEnabledByCase :exec
-- 서비스: 케이스 전체 모니터 블록 활성/비활성
UPDATE monitor_blocks SET enabled = $2 WHERE case_id = $1;

-- name: UpdateCaseDSLPollingEnabled :exec
-- 서비스: DSL 폴링 활성/비활성 토글
UPDATE cases SET dsl_polling_enabled = $2 WHERE id = $1;

-- name: ListMonitorBlocksByCase :many
-- 서비스: 케이스별 모니터 블록 목록 조회
SELECT mb.id, mb.enabled, mb.cron, mb.last_executed_at, ab.instruction
FROM monitor_blocks mb
JOIN agent_blocks ab ON ab.id = mb.block_id
WHERE mb.case_id = $1;
```

- [ ] `backend/db/sqlc.yaml`에 `monitoring.sql` 쿼리 파일 등록

- [ ] asynq scheduler 구현 — LIVE 케이스의 활성 MonitorBlock을 조회하고 asynq scheduler에 cron 태스크로 등록

```go
// backend/internal/worker/scheduler.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	db "your-module/backend/db/generated"
)

// SchedulerManager manages asynq scheduler entries for active monitor blocks.
type SchedulerManager struct {
	queries   *db.Queries
	pool      *pgxpool.Pool
	redisOpt  asynq.RedisClientOpt
	scheduler *asynq.Scheduler
}

func NewSchedulerManager(pool *pgxpool.Pool, redisOpt asynq.RedisClientOpt) *SchedulerManager {
	scheduler := asynq.NewScheduler(redisOpt, nil)
	return &SchedulerManager{
		queries:   db.New(pool),
		pool:      pool,
		redisOpt:  redisOpt,
		scheduler: scheduler,
	}
}

// SyncMonitorSchedules reads all active MonitorBlocks from the DB and registers
// them as periodic tasks with the asynq scheduler. It rebuilds the scheduler
// from scratch each time to handle additions, removals, and cron changes.
func (sm *SchedulerManager) SyncMonitorSchedules() error {
	// 1. LIVE 케이스의 enabled MonitorBlock 조회 (sqlc 생성 쿼리 사용)
	rows, err := sm.queries.ListActiveMonitorBlocks(context.Background())
	if err != nil {
		return fmt.Errorf("query active monitors: %w", err)
	}

	// 2. 새 스케줄러 인스턴스를 구성 (기존 스케줄 전체 교체)
	newScheduler := asynq.NewScheduler(sm.redisOpt, nil)

	// DSL 가격 폴링: 장중 1분마다
	_, err = newScheduler.Register("*/1 9-15 * * 1-5", NewDSLPollerTask())
	if err != nil {
		slog.Error("failed to register DSL poller schedule", "error", err)
	}

	// 각 MonitorBlock에 대해 cron 태스크 등록
	for _, row := range rows {
		var tools []string
		if row.AllowedTools != "" {
			_ = json.Unmarshal([]byte(row.AllowedTools), &tools)
		}
		payload := MonitorAgentPayload{
			CaseID:           row.CaseID,
			MonitorBlockID:   row.MonitorBlockID,
			PipelineID:       row.PipelineID,
			Symbol:           row.Symbol,
			BlockInstruction: row.BlockInstruction,
			AllowedTools:     tools,
		}
		task, err := NewMonitorAgentTask(payload)
		if err != nil {
			slog.Error("failed to create agent task", "monitor_block_id", row.MonitorBlockID, "error", err)
			continue
		}
		_, err = newScheduler.Register(row.Cron, task)
		if err != nil {
			slog.Error("failed to register agent schedule", "monitor_block_id", row.MonitorBlockID, "cron", row.Cron, "error", err)
		}
	}

	// 기존 스케줄러를 교체
	sm.scheduler = newScheduler
	slog.Info("monitor schedules synced", "count", len(rows))
	return nil
}

// Run starts the underlying asynq scheduler (blocking).
func (sm *SchedulerManager) Run() error {
	return sm.scheduler.Run()
}

// Shutdown stops the scheduler gracefully.
func (sm *SchedulerManager) Shutdown() {
	sm.scheduler.Shutdown()
}
```

- [ ] 에이전트 모니터링 asynq 핸들러 구현 — 에이전트 런타임(Plan 4)을 호출하고 결과를 TimelineEvent로 변환

```go
// backend/internal/worker/monitor_agent.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	db "your-module/backend/db/generated"
)

// AgentHandler handles monitor:agent tasks.
type AgentHandler struct {
	queries *db.Queries
}

func NewAgentHandler(pool *pgxpool.Pool) *AgentHandler {
	return &AgentHandler{queries: db.New(pool)}
}

func (h *AgentHandler) HandleMonitorAgent(ctx context.Context, t *asynq.Task) error {
	var payload MonitorAgentPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal MonitorAgentPayload: %w", err)
	}

	slog.Info("executing monitor agent",
		"case_id", payload.CaseID,
		"monitor_block_id", payload.MonitorBlockID,
	)

	// 1. 케이스 조회 (sqlc 생성 쿼리)
	caseID, _ := uuid.Parse(payload.CaseID)
	caseRecord, err := h.queries.GetCaseEventSnapshot(ctx, caseID)
	if err != nil {
		return fmt.Errorf("query case %s: %w", payload.CaseID, err)
	}

	// 2. 에이전트 실행 (Plan 4 런타임 호출)
	agentResult, err := executeAgentBlock(ctx, payload, caseRecord.EventSnapshot)
	if err != nil {
		return fmt.Errorf("execute agent block: %w", err)
	}

	// 3. TimelineEvent 생성 (sqlc 생성 쿼리)
	title := fmt.Sprintf("[모니터링] %.80s", agentResult.Summary)
	now := time.Now()
	_, err = h.queries.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
		CaseID:     caseID,
		Date:       now,
		Type:       "PIPELINE_RESULT",
		Title:      title,
		Content:    &agentResult.Summary,
		AiAnalysis: &agentResult.Summary,
		Data:       agentResult.Data,
	})
	if err != nil {
		return fmt.Errorf("insert timeline event: %w", err)
	}

	slog.Info("monitor agent completed", "case_id", payload.CaseID)
	return nil
}

// ── agent runtime stub (Plan 4 제공) ────────────────────────────

type agentBlockResult struct {
	Summary string
	Data    json.RawMessage
}

func executeAgentBlock(
	ctx context.Context,
	payload MonitorAgentPayload,
	eventSnapshot json.RawMessage,
) (*agentBlockResult, error) {
	// TODO: Plan 4 에이전트 런타임 연동
	return &agentBlockResult{
		Summary: "stub agent result",
		Data:    json.RawMessage(`{}`),
	}, nil
}
```

- [ ] 테스트: mock 에이전트 런타임으로 핸들러 실행 -> TimelineEvent 생성 확인
- [ ] 테스트: 스케줄러가 enabled=true인 MonitorBlock만 등록하는지 확인
- [ ] 테스트: 케이스 상태가 LIVE가 아닌 경우 스케줄 제거 확인

```bash
git add backend/internal/worker/monitor_agent.go backend/internal/worker/scheduler.go
git commit -m "feat(monitoring): 에이전트 모니터링 핸들러 및 asynq 스케줄러 구현"
```

---

## Task 3: DSL 가격 폴링 핸들러

1분 간격으로 LIVE 케이스의 성공/실패 조건과 가격 알림을 DSL 엔진으로 평가하는 경량 폴링 핸들러를 구현한다. KIS 배치 클라이언트는 `internal/infra/kis/`에, 장중 시간 판단은 `internal/infra/market_time.go`에 배치한다.

**Files:**
- Create: `backend/internal/worker/dsl_poller.go`
- Create: `backend/internal/infra/kis/batch_client.go`
- Create: `backend/internal/infra/market_time.go`

**Steps:**

- [ ] KIS API 배치 클라이언트 구현 — 여러 종목의 현재가를 배치로 조회하여 rate limit 준수

```go
// backend/internal/infra/kis/batch_client.go
package kis

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	batchSize    = 20   // KIS API rate limit 고려
	batchDelayMs = 1000 // 배치 간 간격
)

// PriceSnapshot holds a single symbol's latest market data.
type PriceSnapshot struct {
	Symbol    string
	Close     float64
	High      float64
	Low       float64
	Volume    float64
	Timestamp time.Time
}

// FetchPricesBatch retrieves current prices for the given symbols in batches
// to respect KIS API rate limits. Returns a map keyed by symbol.
func FetchPricesBatch(symbols []string) (map[string]*PriceSnapshot, error) {
	results := make(map[string]*PriceSnapshot)
	var mu sync.Mutex

	batches := chunk(symbols, batchSize)
	for i, batch := range batches {
		var wg sync.WaitGroup
		for _, sym := range batch {
			wg.Add(1)
			go func(symbol string) {
				defer wg.Done()
				price, err := getPrice(symbol)
				if err != nil {
					slog.Error("failed to fetch price", "symbol", symbol, "error", err)
					return
				}
				mu.Lock()
				results[symbol] = price
				mu.Unlock()
			}(sym)
		}
		wg.Wait()
		if i < len(batches)-1 {
			time.Sleep(time.Duration(batchDelayMs) * time.Millisecond)
		}
	}
	return results, nil
}

// getPrice fetches the current price for a single symbol from KIS API.
func getPrice(symbol string) (*PriceSnapshot, error) {
	// TODO: KIS API 클라이언트 연동 (Plan 1 shared infra)
	return nil, fmt.Errorf("KIS API client not yet implemented for symbol %s", symbol)
}

func chunk(items []string, size int) [][]string {
	var chunks [][]string
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}
```

- [ ] 장중 시간 유틸리티

```go
// backend/internal/infra/market_time.go
package infra

import "time"

const (
	marketOpenHour    = 9
	marketCloseHour   = 15
	marketCloseMinute = 30
)

var kstLocation = time.FixedZone("KST", 9*60*60)

// IsMarketHours checks whether the given time (or now) falls within
// Korean stock market hours (09:00 ~ 15:30 KST).
// Does NOT account for holidays or half-day schedules.
func IsMarketHours(now time.Time) bool {
	kst := now.In(kstLocation)
	hour := kst.Hour()
	minute := kst.Minute()

	if hour < marketOpenHour {
		return false
	}
	if hour > marketCloseHour {
		return false
	}
	if hour == marketCloseHour && minute > marketCloseMinute {
		return false
	}
	return true
}
```

- [ ] DSL 폴링 asynq 핸들러 구현 — `RunDSLPollingCycle`을 핸들러 내부에서 호출

```go
// backend/internal/worker/dsl_poller.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	db "your-module/backend/db/generated"
	"github.com/dev-superbear/nexus-backend/internal/infra"
	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
)

// DSLPollerHandler handles monitor:dsl-poll tasks.
type DSLPollerHandler struct {
	queries  *db.Queries
	enqueuer *asynq.Client
}

func NewDSLPollerHandler(pool *pgxpool.Pool, enqueuer *asynq.Client) *DSLPollerHandler {
	return &DSLPollerHandler{queries: db.New(pool), enqueuer: enqueuer}
}

func (h *DSLPollerHandler) HandleDSLPoller(ctx context.Context, t *asynq.Task) error {
	// 장외 시간이면 스킵
	if !infra.IsMarketHours(time.Now()) {
		slog.Debug("outside market hours, skipping DSL poll")
		return nil
	}

	slog.Info("running DSL polling cycle")
	if err := h.runDSLPollingCycle(ctx); err != nil {
		return fmt.Errorf("DSL polling cycle: %w", err)
	}
	slog.Info("DSL polling cycle complete")
	return nil
}

func (h *DSLPollerHandler) runDSLPollingCycle(ctx context.Context) error {
	// 1. LIVE 케이스 조회
	liveCases, err := h.fetchLiveCases()
	if err != nil {
		return err
	}
	if len(liveCases) == 0 {
		return nil
	}

	// 2. 고유 심볼 추출 및 배치 가격 조회
	symbolSet := make(map[string]struct{})
	for _, c := range liveCases {
		symbolSet[c.Symbol] = struct{}{}
	}
	symbols := make([]string, 0, len(symbolSet))
	for s := range symbolSet {
		symbols = append(symbols, s)
	}
	prices, err := kis.FetchPricesBatch(symbols)
	if err != nil {
		return fmt.Errorf("fetch prices batch: %w", err)
	}

	// 3. 각 케이스의 조건 평가
	for _, c := range liveCases {
		price, ok := prices[c.Symbol]
		if !ok {
			continue
		}
		if err := h.evaluateCaseConditions(ctx, c, price); err != nil {
			slog.Error("evaluate case conditions failed", "case_id", c.ID, "error", err)
		}
	}
	return nil
}

// ── internal helpers ────────────────────────────────────────────

type liveCaseRow struct {
	ID            string
	Symbol        string
	SuccessScript string
	FailureScript string
	EventSnapshot json.RawMessage
	PriceAlerts   []priceAlertRow
}

type priceAlertRow struct {
	ID        string
	Condition string
	Label     string
}

func (h *DSLPollerHandler) fetchLiveCases() ([]liveCaseRow, error) {
	ctx := context.Background()

	// sqlc 생성 쿼리로 LIVE 케이스 조회
	cases, err := h.queries.ListLiveCases(ctx)
	if err != nil {
		return nil, fmt.Errorf("query live cases: %w", err)
	}

	result := make([]liveCaseRow, 0, len(cases))
	for _, c := range cases {
		// sqlc 생성 쿼리로 미트리거 알림 조회
		alerts, err := h.queries.ListUntriggeredAlertsByCase(ctx, c.ID)
		if err != nil {
			slog.Error("query untriggered alerts", "case_id", c.ID, "error", err)
			continue
		}
		priceAlerts := make([]priceAlertRow, len(alerts))
		for i, a := range alerts {
			priceAlerts[i] = priceAlertRow{
				ID:        a.ID.String(),
				Condition: a.Condition,
				Label:     a.Label,
			}
		}

		result = append(result, liveCaseRow{
			ID:            c.ID.String(),
			Symbol:        c.Symbol,
			SuccessScript: c.SuccessScript,
			FailureScript: c.FailureScript,
			EventSnapshot: c.EventSnapshot,
			PriceAlerts:   priceAlerts,
		})
	}
	return result, nil
}

// BuildDSLContext creates a typed DSL context from a case event snapshot and a live price.
func BuildDSLContext(eventSnapshot json.RawMessage, price *kis.PriceSnapshot) DSLContext {
	var snapshot map[string]interface{}
	_ = json.Unmarshal(eventSnapshot, &snapshot)

	floatVal := func(key string) float64 {
		if v, ok := snapshot[key]; ok {
			if f, ok := v.(float64); ok {
				return f
			}
		}
		return 0
	}
	maVal := func(period int) float64 {
		if preMa, ok := snapshot["preMa"]; ok {
			if m, ok := preMa.(map[string]interface{}); ok {
				key := fmt.Sprintf("%d", period)
				if v, ok := m[key]; ok {
					if f, ok := v.(float64); ok {
						return f
					}
				}
			}
		}
		return 0
	}

	return DSLContext{
		Close:         price.Close,
		High:          price.High,
		Low:           price.Low,
		Volume:        price.Volume,
		EventHigh:     floatVal("high"),
		EventLow:      floatVal("low"),
		EventClose:    floatVal("close"),
		EventVolume:   floatVal("volume"),
		PreEventMA5:   maVal(5),
		PreEventMA20:  maVal(20),
		PreEventMA60:  maVal(60),
		PreEventMA120: maVal(120),
		PreEventMA200: maVal(200),
		PreEventClose: floatVal("preClose"),
	}
}

func (h *DSLPollerHandler) evaluateCaseConditions(
	ctx context.Context,
	caseRow liveCaseRow,
	price *kis.PriceSnapshot,
) error {
	dslCtx := BuildDSLContext(caseRow.EventSnapshot, price)

	// 성공 조건 체크
	if caseRow.SuccessScript != "" {
		if evaluateDSL(caseRow.SuccessScript, dslCtx) {
			payload := LifecyclePayload{
				CaseID: caseRow.ID,
				Action: "CLOSE_SUCCESS",
				Reason: fmt.Sprintf("성공 조건 도달: %s (close=%.2f)", caseRow.SuccessScript, price.Close),
			}
			task, err := NewLifecycleTask(payload)
			if err != nil {
				return err
			}
			if _, err := h.enqueuer.EnqueueContext(ctx, task); err != nil {
				return fmt.Errorf("enqueue lifecycle CLOSE_SUCCESS: %w", err)
			}
			return nil // 성공 시 실패 체크 스킵
		}
	}

	// 실패 조건 체크
	if caseRow.FailureScript != "" {
		if evaluateDSL(caseRow.FailureScript, dslCtx) {
			payload := LifecyclePayload{
				CaseID: caseRow.ID,
				Action: "CLOSE_FAILURE",
				Reason: fmt.Sprintf("실패 조건 도달: %s (close=%.2f)", caseRow.FailureScript, price.Close),
			}
			task, err := NewLifecycleTask(payload)
			if err != nil {
				return err
			}
			if _, err := h.enqueuer.EnqueueContext(ctx, task); err != nil {
				return fmt.Errorf("enqueue lifecycle CLOSE_FAILURE: %w", err)
			}
			return nil
		}
	}

	// 가격 알림 체크
	for _, alert := range caseRow.PriceAlerts {
		if evaluateDSL(alert.Condition, dslCtx) {
			payload := LifecyclePayload{
				CaseID:  caseRow.ID,
				Action:  "TRIGGER_ALERT",
				Reason:  fmt.Sprintf("가격 알림 도달: %s", alert.Label),
				AlertID: alert.ID,
			}
			task, err := NewLifecycleTask(payload)
			if err != nil {
				slog.Error("create lifecycle task for alert", "alert_id", alert.ID, "error", err)
				continue
			}
			if _, err := h.enqueuer.EnqueueContext(ctx, task); err != nil {
				slog.Error("enqueue lifecycle TRIGGER_ALERT", "alert_id", alert.ID, "error", err)
			}
		}
	}

	return nil
}

// evaluateDSL is a stub for the DSL engine (Plan 1).
func evaluateDSL(script string, ctx DSLContext) bool {
	// TODO: Plan 1 DSL 엔진 연동
	return false
}
```

- [ ] 테스트: mock KIS 응답으로 성공 조건(`close >= event_high * 2.0`) 도달 시 lifecycle 태스크 enqueue 확인
- [ ] 테스트: 실패 조건(`close < pre_event_ma(120)`) 도달 시 lifecycle 태스크 enqueue 확인
- [ ] 테스트: 가격 알림 트리거 시 `triggered` 플래그 업데이트 확인
- [ ] 테스트: 배치 클라이언트가 batchSize 단위로 분할 요청하는지 확인

```bash
git add backend/internal/worker/dsl_poller.go backend/internal/infra/kis/batch_client.go backend/internal/infra/market_time.go
git commit -m "feat(monitoring): DSL 가격 폴링 핸들러 구현 (asynq, 1분 간격, 배치 KIS API)"
```

---

## Task 4: 케이스 라이프사이클 핸들러

성공/실패 조건 도달 또는 가격 알림 트리거 시 케이스 상태를 전환하고, 타임라인 이벤트를 생성하며, 알림 시스템(Plan 7)에 이벤트를 전파하는 asynq 핸들러를 구현한다.

**Files:**
- Create: `backend/internal/worker/lifecycle.go`

**Steps:**

- [ ] 라이프사이클 asynq 핸들러 구현 — 케이스 상태 전환, 모니터링 스케줄 해제, 타임라인 기록

```go
// backend/internal/worker/lifecycle.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	db "your-module/backend/db/generated"
)

// LifecycleHandler handles monitor:lifecycle tasks.
type LifecycleHandler struct {
	queries          *db.Queries
	schedulerManager *SchedulerManager
}

func NewLifecycleHandler(pool *pgxpool.Pool, sm *SchedulerManager) *LifecycleHandler {
	return &LifecycleHandler{queries: db.New(pool), schedulerManager: sm}
}

func (h *LifecycleHandler) HandleLifecycle(ctx context.Context, t *asynq.Task) error {
	var payload LifecyclePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal LifecyclePayload: %w", err)
	}

	slog.Info("processing lifecycle event",
		"case_id", payload.CaseID,
		"action", payload.Action,
	)

	switch payload.Action {
	case "CLOSE_SUCCESS":
		return h.closeCase(payload.CaseID, "CLOSED_SUCCESS", payload.Reason)
	case "CLOSE_FAILURE":
		return h.closeCase(payload.CaseID, "CLOSED_FAILURE", payload.Reason)
	case "TRIGGER_ALERT":
		return h.triggerAlert(payload.CaseID, payload.AlertID, payload.Reason)
	default:
		return fmt.Errorf("unknown lifecycle action: %s", payload.Action)
	}
}

func (h *LifecycleHandler) closeCase(caseID, status, reason string) error {
	ctx := context.Background()
	now := time.Now()
	id, _ := uuid.Parse(caseID)
	closedAt := now.Format(time.RFC3339)

	// 1. 케이스 상태 업데이트 (sqlc 생성 쿼리)
	_, err := h.queries.UpdateCaseStatus(ctx, db.UpdateCaseStatusParams{
		ID:           id,
		Status:       db.CaseStatus(status),
		ClosedAt:     &closedAt,
		ClosedReason: &reason,
	})
	if err != nil {
		return fmt.Errorf("update case status: %w", err)
	}

	// 2. 타임라인 이벤트 생성 (sqlc 생성 쿼리)
	title := "성공 조건 도달"
	if status == "CLOSED_FAILURE" {
		title = "실패 조건 도달"
	}
	_, err = h.queries.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
		CaseID:  id,
		Date:    now,
		Type:    "PRICE_ALERT",
		Title:   title,
		Content: &reason,
	})
	if err != nil {
		return fmt.Errorf("insert timeline event: %w", err)
	}

	// 3. 해당 케이스의 모니터링 스케줄 모두 해제 (sqlc 생성 쿼리)
	err = h.queries.DisableMonitorBlocksByCase(ctx, id)
	if err != nil {
		return fmt.Errorf("disable monitor blocks: %w", err)
	}

	// 스케줄 재동기화
	if err := h.schedulerManager.SyncMonitorSchedules(); err != nil {
		slog.Error("failed to sync schedules after case close", "case_id", caseID, "error", err)
	}

	// 4. 알림 전파 (Plan 7)
	// TODO: emitNotification(NotificationEvent{Type: "CASE_CLOSED", CaseID: caseID, Status: status, Reason: reason})

	slog.Info("case closed", "case_id", caseID, "status", status)
	return nil
}

func (h *LifecycleHandler) triggerAlert(caseID, alertID, reason string) error {
	ctx := context.Background()
	now := time.Now()
	caseUUID, _ := uuid.Parse(caseID)
	alertUUID, _ := uuid.Parse(alertID)

	// 1. PriceAlert 트리거 상태 업데이트 (sqlc 생성 쿼리)
	err := h.queries.TriggerPriceAlert(ctx, db.TriggerPriceAlertParams{
		ID:          alertUUID,
		TriggeredAt: &now,
	})
	if err != nil {
		return fmt.Errorf("update price alert: %w", err)
	}

	// 2. 타임라인 이벤트 생성 (sqlc 생성 쿼리)
	_, err = h.queries.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
		CaseID:  caseUUID,
		Date:    now,
		Type:    "PRICE_ALERT",
		Title:   "가격 알림 도달",
		Content: &reason,
	})
	if err != nil {
		return fmt.Errorf("insert timeline event: %w", err)
	}

	// 3. 알림 전파 (Plan 7)
	// TODO: emitNotification(NotificationEvent{Type: "PRICE_ALERT", CaseID: caseID, AlertID: alertID, Reason: reason})

	slog.Info("price alert triggered", "case_id", caseID, "alert_id", alertID)
	return nil
}
```

- [ ] 테스트: CLOSE_SUCCESS 시 케이스 상태 전환 + 타임라인 생성 + 스케줄 해제 확인
- [ ] 테스트: CLOSE_FAILURE 시 동일 플로우 확인
- [ ] 테스트: TRIGGER_ALERT 시 PriceAlert.triggered=true 업데이트 확인
- [ ] 테스트: 이미 CLOSED 상태인 케이스에 중복 이벤트 방지 확인

```bash
git add backend/internal/worker/lifecycle.go
git commit -m "feat(monitoring): 케이스 라이프사이클 핸들러 구현 (상태 전환 + 알림 전파)"
```

---

## Task 5: 블록 단위 / 케이스 전체 중단·재개 서비스 및 HTTP 핸들러

개별 모니터링 블록 또는 케이스 전체의 모니터링을 일시 중단/재개하는 서비스 계층과 HTTP 핸들러를 구현한다.

**Files:**
- Create: `backend/internal/service/monitoring_service.go`
- Create: `backend/internal/handler/monitoring_handler.go`

**Steps:**

- [ ] 중단/재개 서비스 구현

```go
// backend/internal/service/monitoring_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "your-module/backend/db/generated"
	"github.com/dev-superbear/nexus-backend/internal/worker"
)

// MonitoringService manages pause/resume of individual blocks and entire cases.
type MonitoringService struct {
	queries          *db.Queries
	schedulerManager *worker.SchedulerManager
}

func NewMonitoringService(pool *pgxpool.Pool, sm *worker.SchedulerManager) *MonitoringService {
	return &MonitoringService{queries: db.New(pool), schedulerManager: sm}
}

// ToggleMonitorBlock enables or disables a single monitor block
// and re-syncs the asynq scheduler.
func (s *MonitoringService) ToggleMonitorBlock(monitorBlockID string, enabled bool) error {
	ctx := context.Background()
	id, _ := uuid.Parse(monitorBlockID)
	err := s.queries.UpdateMonitorBlockEnabled(ctx, db.UpdateMonitorBlockEnabledParams{
		ID:      id,
		Enabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("update monitor block: %w", err)
	}

	if err := s.schedulerManager.SyncMonitorSchedules(); err != nil {
		slog.Error("sync schedules after toggle block", "error", err)
	}
	return nil
}

// ToggleCaseMonitoring enables or disables all monitor blocks for a case.
// If keepDSLPolling is true and enabled is false, DSL polling remains active.
func (s *MonitoringService) ToggleCaseMonitoring(caseID string, enabled bool, keepDSLPolling bool) error {
	ctx := context.Background()
	id, _ := uuid.Parse(caseID)

	// 모든 MonitorBlock의 enabled 상태 변경
	err := s.queries.UpdateMonitorBlocksEnabledByCase(ctx, db.UpdateMonitorBlocksEnabledByCaseParams{
		CaseID:  id,
		Enabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("update monitor blocks: %w", err)
	}

	// DSL 폴링은 선택적으로 유지 가능
	if !keepDSLPolling && !enabled {
		err = s.queries.UpdateCaseDSLPollingEnabled(ctx, db.UpdateCaseDSLPollingEnabledParams{
			ID:                id,
			DslPollingEnabled: false,
		})
		if err != nil {
			return fmt.Errorf("disable dsl polling: %w", err)
		}
	} else if enabled {
		err = s.queries.UpdateCaseDSLPollingEnabled(ctx, db.UpdateCaseDSLPollingEnabledParams{
			ID:                id,
			DslPollingEnabled: true,
		})
		if err != nil {
			return fmt.Errorf("enable dsl polling: %w", err)
		}
	}

	if err := s.schedulerManager.SyncMonitorSchedules(); err != nil {
		slog.Error("sync schedules after toggle case", "error", err)
	}
	return nil
}

// ListMonitorBlocks returns all monitor blocks for a case.
func (s *MonitoringService) ListMonitorBlocks(caseID string) ([]MonitorBlockInfo, error) {
	ctx := context.Background()
	id, _ := uuid.Parse(caseID)
	rows, err := s.queries.ListMonitorBlocksByCase(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("query monitor blocks: %w", err)
	}
	blocks := make([]MonitorBlockInfo, len(rows))
	for i, r := range rows {
		blocks[i] = MonitorBlockInfo{
			ID:             r.ID.String(),
			Enabled:        r.Enabled,
			Cron:           r.Cron,
			LastExecutedAt: r.LastExecutedAt,
			Instruction:    r.Instruction,
		}
	}
	return blocks, nil
}

// MonitorBlockInfo is a read DTO for monitor block listing.
type MonitorBlockInfo struct {
	ID             string  `json:"id"`
	Enabled        bool    `json:"enabled"`
	Cron           string  `json:"cron"`
	LastExecutedAt *string `json:"last_executed_at"`
	Instruction    string  `json:"instruction"`
}
```

- [ ] HTTP 핸들러 구현 — thin controller

```go
// backend/internal/handler/monitoring_handler.go
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dev-superbear/nexus-backend/internal/service"
)

// MonitoringHandler exposes HTTP endpoints for monitoring control.
type MonitoringHandler struct {
	svc *service.MonitoringService
}

func NewMonitoringHandler(svc *service.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{svc: svc}
}

// PATCH /api/cases/{id}/monitors/{monitorId}
// Body: { "enabled": bool }
func (h *MonitoringHandler) ToggleBlock(w http.ResponseWriter, r *http.Request) {
	monitorID := r.PathValue("monitorId")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.svc.ToggleMonitorBlock(monitorID, body.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// PATCH /api/cases/{id}/monitoring-status
// Body: { "enabled": bool, "keep_dsl_polling": bool }
func (h *MonitoringHandler) ToggleCaseMonitoring(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("id")
	var body struct {
		Enabled        bool `json:"enabled"`
		KeepDSLPolling bool `json:"keep_dsl_polling"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.svc.ToggleCaseMonitoring(caseID, body.Enabled, body.KeepDSLPolling); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GET /api/cases/{id}/monitors
func (h *MonitoringHandler) ListMonitors(w http.ResponseWriter, r *http.Request) {
	caseID := r.PathValue("id")
	blocks, err := h.svc.ListMonitorBlocks(caseID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}
```

- [ ] DB 스키마에 `dsl_polling_enabled` 컬럼 추가 (Case 테이블)

```sql
ALTER TABLE cases ADD COLUMN dsl_polling_enabled BOOLEAN NOT NULL DEFAULT true;
```

- [ ] 테스트: 개별 블록 중단 후 asynq scheduler에서 해당 태스크 제거 확인
- [ ] 테스트: 케이스 전체 중단 시 모든 블록 중단 + keepDSLPolling=true 시 가격 폴링 유지 확인
- [ ] 테스트: 재개 시 모든 블록 재등록 확인

```bash
git add backend/internal/service/monitoring_service.go backend/internal/handler/monitoring_handler.go
git commit -m "feat(monitoring): 블록 단위 / 케이스 전체 중단·재개 서비스 및 HTTP 핸들러 구현"
```

---

## Task 6: 워커 서버 엔트리포인트 및 헬스체크

모든 모니터링 핸들러를 asynq 서버에 등록하고, 스케줄러를 시작하며, 헬스체크 엔드포인트와 graceful shutdown을 지원하는 엔트리포인트를 구현한다.

**Files:**
- Create: `backend/cmd/worker/main.go`
- Create: `backend/internal/service/metrics.go`

**Steps:**

- [ ] asynq 워커 서버 엔트리포인트 구현

```go
// backend/cmd/worker/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/handler"
	"github.com/dev-superbear/nexus-backend/internal/infra"
	"github.com/dev-superbear/nexus-backend/internal/service"
	"github.com/dev-superbear/nexus-backend/internal/worker"
)

func main() {
	cfg := config.Load()
	redisOpt := infra.NewRedisClientOpt(cfg.RedisAddr, cfg.RedisPassword)

	// ── DB (pgxpool + sqlc) ─────────────────────────────────────
	pool := infra.NewPool(cfg) // pgxpool.Pool 초기화 (Plan 1 인프라 제공)
	defer pool.Close()

	// ── asynq Client (for enqueuing tasks from within handlers) ─
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	// ── Handlers ────────────────────────────────────────────────
	agentHandler := worker.NewAgentHandler(pool)
	dslHandler := worker.NewDSLPollerHandler(pool, client)
	schedulerMgr := worker.NewSchedulerManager(pool, redisOpt)
	lifecycleHandler := worker.NewLifecycleHandler(pool, schedulerMgr)

	// ── asynq Server (task processing) ──────────────────────────
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.Error("task failed", "type", task.Type(), "error", err)
			}),
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TypeMonitorAgent, agentHandler.HandleMonitorAgent)
	mux.HandleFunc(worker.TypeDSLPoller, dslHandler.HandleDSLPoller)
	mux.HandleFunc(worker.TypeMonitorLifecycle, lifecycleHandler.HandleLifecycle)

	// ── Initial schedule sync ───────────────────────────────────
	if err := schedulerMgr.SyncMonitorSchedules(); err != nil {
		log.Fatalf("initial schedule sync failed: %v", err)
	}
	slog.Info("initial schedule sync completed")

	// ── Start asynq scheduler in a goroutine ────────────────────
	go func() {
		if err := schedulerMgr.Run(); err != nil {
			slog.Error("scheduler stopped", "error", err)
		}
	}()

	// ── Health check HTTP server ────────────────────────────────
	metricsSvc := service.NewMetricsService(redisOpt)
	monitoringSvc := service.NewMonitoringService(pool, schedulerMgr)
	monitoringHandler := handler.NewMonitoringHandler(monitoringSvc)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("GET /api/health/workers", func(w http.ResponseWriter, r *http.Request) {
		metrics, err := metricsSvc.CollectMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"metrics":   metrics,
			"timestamp": fmt.Sprintf("%v", metrics),
		})
	})
	httpMux.HandleFunc("GET /api/cases/{id}/monitors", monitoringHandler.ListMonitors)
	httpMux.HandleFunc("PATCH /api/cases/{id}/monitors/{monitorId}", monitoringHandler.ToggleBlock)
	httpMux.HandleFunc("PATCH /api/cases/{id}/monitoring-status", monitoringHandler.ToggleCaseMonitoring)

	go func() {
		addr := ":8081"
		slog.Info("health/monitoring HTTP server starting", "addr", addr)
		if err := http.ListenAndServe(addr, httpMux); err != nil {
			slog.Error("HTTP server stopped", "error", err)
		}
	}()

	// ── Start asynq worker server (blocking) ────────────────────
	slog.Info("starting asynq worker server")
	go func() {
		if err := srv.Run(mux); err != nil {
			log.Fatalf("asynq server failed: %v", err)
		}
	}()

	// ── Graceful shutdown ───────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("received shutdown signal", "signal", sig)

	srv.Shutdown()
	schedulerMgr.Shutdown()
	client.Close()
	slog.Info("all workers shut down gracefully")
}
```

- [ ] 메트릭 수집 서비스

```go
// backend/internal/service/metrics.go
package service

import (
	"fmt"

	"github.com/hibiken/asynq"

	"github.com/dev-superbear/nexus-backend/internal/worker"
)

// MetricsService collects monitoring metrics from asynq queues.
type MetricsService struct {
	inspector *asynq.Inspector
}

func NewMetricsService(redisOpt asynq.RedisClientOpt) *MetricsService {
	return &MetricsService{
		inspector: asynq.NewInspector(redisOpt),
	}
}

func (s *MetricsService) CollectMetrics() (*worker.MonitoringMetrics, error) {
	agentInfo, err := s.inspector.GetQueueInfo("default")
	if err != nil {
		return nil, fmt.Errorf("get queue info: %w", err)
	}

	m := &worker.MonitoringMetrics{}
	m.QueueDepth.Agent = agentInfo.Pending + agentInfo.Active
	m.QueueDepth.DSL = 0       // asynq scheduler-driven, no persistent queue depth
	m.QueueDepth.Lifecycle = 0 // transient tasks
	return m, nil
}
```

- [ ] 테스트: 엔트리포인트가 모든 핸들러를 asynq 서버에 등록하는지 확인
- [ ] 테스트: 헬스체크 엔드포인트가 메트릭을 올바르게 반환하는지 확인
- [ ] 테스트: SIGTERM 시 graceful shutdown 확인

```bash
git add backend/cmd/worker/main.go backend/internal/service/metrics.go
git commit -m "feat(monitoring): asynq 워커 서버 엔트리포인트 및 헬스체크 구현"
```

---

## Task 7: 스케줄 동기화 cron 및 통합 테스트

5분마다 DB 상태와 asynq 스케줄을 동기화하는 안전장치를 추가하고, 전체 모니터링 파이프라인의 통합 테스트를 작성한다.

**Files:**
- Modify: `backend/cmd/worker/main.go`
- Create: `backend/internal/worker/scheduler_sync.go`

**Steps:**

- [ ] 5분 주기 스케줄 동기화 cron 추가 — DB에서 활성 모니터를 다시 읽어 asynq 스케줄과 차이를 보정. robfig/cron 사용.

```go
// backend/internal/worker/scheduler_sync.go
package worker

import (
	"log/slog"

	"github.com/robfig/cron/v3"
)

// StartScheduleSync launches a background cron job that re-syncs
// the asynq scheduler with the DB every 5 minutes as a safety net.
func StartScheduleSync(sm *SchedulerManager) *cron.Cron {
	c := cron.New()
	c.AddFunc("@every 5m", func() {
		if err := sm.SyncMonitorSchedules(); err != nil {
			slog.Error("periodic schedule sync failed", "error", err)
			return
		}
		slog.Info("periodic schedule sync completed")
	})
	c.Start()
	return c
}
```

- [ ] `cmd/worker/main.go`에 sync cron 등록 추가

```go
// main() 함수 내, 스케줄러 시작 직후에 추가:
syncCron := worker.StartScheduleSync(schedulerMgr)
defer syncCron.Stop()
```

- [ ] 통합 테스트 작성 — 전체 플로우 검증

```go
// backend/internal/worker/integration_test.go
package worker_test

import (
	"testing"
)

func TestMonitoringIntegration(t *testing.T) {
	t.Run("케이스 생성 → 모니터링 등록 → 가격 폴링 → 성공 조건 도달 → 케이스 종료", func(t *testing.T) {
		// 1. 테스트 케이스 생성 (LIVE, successScript 포함)
		// 2. MonitorBlock 생성 (enabled=true, cron='*/6 * * * *')
		// 3. SyncMonitorSchedules() 호출
		// 4. asynq scheduler에 태스크 등록 확인
		// 5. mock KIS API로 성공 조건 가격 반환
		// 6. HandleDSLPoller() 호출
		// 7. lifecycle 태스크 enqueue 확인
		// 8. HandleLifecycle() 호출
		// 9. 케이스 상태 CLOSED_SUCCESS 확인
		// 10. 타임라인 이벤트 생성 확인
	})

	t.Run("블록 중단 시 해당 cron 태스크만 제거", func(t *testing.T) {
		// ...
	})

	t.Run("케이스 전체 중단 시 모든 블록 중단 + DSL 폴링 선택적 유지", func(t *testing.T) {
		// ...
	})

	t.Run("장외 시간에는 DSL 폴링 스킵", func(t *testing.T) {
		// ...
	})
}
```

- [ ] 테스트 실행 및 전체 통과 확인

```bash
go get github.com/robfig/cron/v3
go test ./backend/internal/worker/...
git add backend/internal/worker/scheduler_sync.go backend/internal/worker/integration_test.go backend/cmd/worker/main.go
git commit -m "feat(monitoring): 스케줄 동기화 안전장치 및 통합 테스트 완성"
```
