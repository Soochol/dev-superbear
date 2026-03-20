# Case Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 파이프라인 실행 결과를 케이스로 관리하고, 타임라인/수익률 추적/매매 기록/가격 알림을 통합된 50:50 레이아웃으로 제공한다.
**Architecture:** FSD(Feature-Sliced Design) + DDD 아키텍처를 적용한다. UI는 `widgets/`(case-timeline, case-detail-panel, case-tab-bar)로 조합하고, 사용자 인터랙션은 `features/`(manage-trades, manage-alerts)로 분리한다. 백엔드는 Go(Gin + sqlc)로 구현하며, handler/service/repository 3-layer 아키텍처를 따른다. PnL 계산은 Go 도메인 로직(`backend/internal/domain/trade/pnl_calculator.go`)으로, Return Tracking은 Go 도메인 로직(`backend/internal/domain/case/return_tracking.go`)으로 배치한다. PriceAlert는 Case aggregate에 소속된다. 프론트엔드 스토어는 entity/feature별로 분할한다(case.store, trade.store, alert.store). 스토어의 API 호출만 Go 백엔드 엔드포인트로 변경한다.
**Tech Stack:** Next.js (App Router) + TypeScript + TailwindCSS + **Go (Gin)** + **sqlc** + PostgreSQL + Zustand

### FSD Directory Map (Frontend — 유지)

```
src/
  app/
    cases/                      # page entry point only

  features/
    manage-trades/
      ui/                       # TradeHistory, TradeForm
      model/
        trade.store.ts          # trade list + PnL summary state
      api/                      # feature-level API calls → Go backend

    manage-alerts/
      ui/                       # PriceAlertsList, AlertForm
      model/
        alert.store.ts          # pending/triggered alerts state
      api/                      # feature-level API calls → Go backend

  entities/
    case/
      model/
        types.ts                # CaseSummary, CaseDetail, CaseFilters, CaseStatus
        case.store.ts           # entity-level state (cases list, selectedCase, timeline)
      lib/
        return-tracking.ts      # (REMOVED — moved to Go backend)

    trade/
      model/
        types.ts                # Trade, TradeType, PnLSummary, CreateTradeInput
      lib/
        pnl-calculator.ts       # (REMOVED — moved to Go backend)

  widgets/
    case-timeline/
      ui/                       # Timeline, TimelineNode, TimelineDot, TimelineCard, TimelineConnector
      lib/
        timeline-formatter.ts   # event → timeline component props (presentation logic)

    case-detail-panel/
      ui/                       # CaseDetailPanel (composes 4 sections), ConditionProgress, ReturnTrackingTable

    case-tab-bar/
      ui/                       # CaseTabBar, CaseSummaryHeader

  shared/
    lib/logger.ts               # logger (replaces console.log)
    api/                        # shared API utilities (fetch wrapper for Go backend)
```

### Go Backend Directory Map (신규)

```
backend/
  internal/
    handler/
      case_handler.go           # Case CRUD + Timeline endpoints (Gin)
      trade_handler.go          # Trade CRUD endpoints (Gin)
      alert_handler.go          # PriceAlert CRUD endpoints (Gin)
    service/
      case_service.go           # 케이스 비즈니스 로직 (목록/상세/상태전환)
      trade_service.go          # 매매 + 타임라인 이벤트 생성
      alert_service.go          # 알림 생성/조회/삭제/트리거
    repository/
      case_repo.go              # Case sqlc CRUD wrapper
      trade_repo.go             # Trade sqlc CRUD wrapper
      alert_repo.go             # PriceAlert sqlc CRUD wrapper
    domain/
      case/
        return_tracking.go      # D+1/D+7/D+30/Peak/Current return calculations
        return_tracking_test.go
      trade/
        pnl_calculator.go       # FIFO 기반 실현/미실현 P&L
        pnl_calculator_test.go
    router/
      router.go                 # Gin router setup + route registration
  db/
    migrations/
      004_case.sql              # Case, TimelineEvent, Trade, PriceAlert tables
    queries/
      cases.sql                 # sqlc queries for cases
      trades.sql                # sqlc queries for trades
      timeline_events.sql       # sqlc queries for timeline_events
      alerts.sql                # sqlc queries for price_alerts
    sqlc.yaml                   # sqlc configuration
    generated/                  # sqlc generated Go code (models.go, querier.go, *.sql.go)
```

---

## Task 1: SQL 마이그레이션 + sqlc — Case, TimelineEvent, Trade, PriceAlert 모델 정의

케이스 관련 전체 데이터 모델을 SQL 마이그레이션으로 정의하고 sqlc로 타입-세이프한 Go 코드를 생성한다.

**Files:**
- Create: `backend/db/migrations/004_case.sql`
- Create: `backend/db/queries/cases.sql`
- Create: `backend/db/queries/trades.sql`
- Create: `backend/db/queries/timeline_events.sql`
- Create: `backend/db/queries/alerts.sql`
- Modify: `backend/db/sqlc.yaml` (신규 쿼리 파일 등록)
- Create: `src/entities/case/model/types.ts` (프론트엔드 타입 — Prisma 의존성 제거)
- Create: `src/entities/trade/model/types.ts` (프론트엔드 타입 — Prisma 의존성 제거)
- Test: `backend/internal/domain/trade/pnl_calculator_test.go`

**Steps:**

- [ ] 1.1 SQL 마이그레이션 파일에 Case 테이블을 정의한다.

```sql
-- backend/db/migrations/004_case.sql

-- === Enums ===
CREATE TYPE case_status AS ENUM ('LIVE', 'CLOSED_SUCCESS', 'CLOSED_FAILURE', 'BACKTEST');
CREATE TYPE timeline_event_type AS ENUM ('NEWS', 'DISCLOSURE', 'SECTOR', 'PRICE_ALERT', 'TRADE', 'PIPELINE_RESULT', 'MONITOR_RESULT');
CREATE TYPE trade_type AS ENUM ('BUY', 'SELL');

-- === Cases ===
CREATE TABLE cases (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  pipeline_id UUID NOT NULL REFERENCES pipelines(id),
  symbol TEXT NOT NULL,
  symbol_name TEXT NOT NULL DEFAULT '',
  sector TEXT,
  status case_status NOT NULL DEFAULT 'LIVE',
  event_date DATE NOT NULL,
  event_snapshot JSONB NOT NULL,
  success_script TEXT,
  failure_script TEXT,
  closed_at DATE,
  closed_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cases_user_id ON cases(user_id);
CREATE INDEX idx_cases_status ON cases(user_id, status);
CREATE INDEX idx_cases_symbol ON cases(user_id, symbol);

-- === Timeline Events ===
CREATE TABLE timeline_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  day_offset INT NOT NULL DEFAULT 0,
  type timeline_event_type NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  ai_analysis TEXT,
  data JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_timeline_events_case_date ON timeline_events(case_id, date);

-- === Trades ===
CREATE TABLE trades (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id),
  trade_type trade_type NOT NULL,
  price NUMERIC(12,2) NOT NULL,
  quantity INT NOT NULL,
  fee NUMERIC(12,2) NOT NULL DEFAULT 0,
  traded_at TIMESTAMPTZ NOT NULL,
  note TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trades_case_id ON trades(case_id);

-- === Trigger: auto-update updated_at ===
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_cases_updated_at
  BEFORE UPDATE ON cases
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

> **DDD Note:** `pipeline_id`는 FK로만 보유하고, Pipeline aggregate에 대한 navigation 관계를 제거했다 (cross-aggregate 커플링 방지). PriceAlert는 Plan 4에서 이미 정의된 테이블을 참조하며, Case aggregate에 소속된다.

- [ ] 1.2 sqlc 쿼리 파일: `backend/db/queries/cases.sql`

```sql
-- backend/db/queries/cases.sql

-- name: ListCasesByUser :many
SELECT * FROM cases
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListCasesByUserWithFilters :many
SELECT * FROM cases
WHERE user_id = $1
  AND ($2::case_status IS NULL OR status = $2)
  AND ($3::text IS NULL OR symbol = $3)
  AND ($4::text IS NULL OR sector = $4)
ORDER BY created_at DESC;

-- name: GetCaseByID :one
SELECT * FROM cases
WHERE id = $1 AND user_id = $2;

-- name: CreateCase :one
INSERT INTO cases (user_id, pipeline_id, symbol, symbol_name, sector, status, event_date, event_snapshot, success_script, failure_script)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdateCaseStatus :one
UPDATE cases
SET status = $2, closed_at = $3, closed_reason = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCase :exec
DELETE FROM cases WHERE id = $1;

-- name: CountCasesByUser :one
SELECT COUNT(*) FROM cases WHERE user_id = $1;
```

- [ ] 1.3 sqlc 쿼리 파일: `backend/db/queries/timeline_events.sql`

```sql
-- backend/db/queries/timeline_events.sql

-- name: ListTimelineEvents :many
SELECT * FROM timeline_events
WHERE case_id = $1
ORDER BY date ASC, created_at ASC;

-- name: ListTimelineEventsByType :many
SELECT * FROM timeline_events
WHERE case_id = $1 AND type = $2
ORDER BY date ASC, created_at ASC;

-- name: ListTimelineEventsWithPaging :many
SELECT * FROM timeline_events
WHERE case_id = $1
ORDER BY date ASC, created_at ASC
LIMIT $2 OFFSET $3;

-- name: GetRecentTimelineEvents :many
SELECT * FROM timeline_events
WHERE case_id = $1
ORDER BY date DESC, created_at DESC
LIMIT $2;

-- name: CreateTimelineEvent :one
INSERT INTO timeline_events (case_id, date, day_offset, type, title, content, ai_analysis, data)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: DeleteTimelineEventsByCase :exec
DELETE FROM timeline_events WHERE case_id = $1;
```

- [ ] 1.4 sqlc 쿼리 파일: `backend/db/queries/trades.sql`

```sql
-- backend/db/queries/trades.sql

-- name: ListTradesByCase :many
SELECT * FROM trades
WHERE case_id = $1
ORDER BY traded_at ASC;

-- name: CreateTrade :one
INSERT INTO trades (case_id, user_id, trade_type, price, quantity, fee, traded_at, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: DeleteTrade :exec
DELETE FROM trades WHERE id = $1;

-- name: DeleteTradesByCase :exec
DELETE FROM trades WHERE case_id = $1;
```

- [ ] 1.5 sqlc 쿼리 파일: `backend/db/queries/alerts.sql`

```sql
-- backend/db/queries/alerts.sql

-- name: ListAlertsByCase :many
SELECT * FROM price_alerts
WHERE case_id = $1
ORDER BY created_at DESC;

-- name: ListPendingAlertsByCase :many
SELECT * FROM price_alerts
WHERE case_id = $1 AND triggered = false
ORDER BY created_at DESC;

-- name: ListTriggeredAlertsByCase :many
SELECT * FROM price_alerts
WHERE case_id = $1 AND triggered = true
ORDER BY triggered_at DESC;

-- name: CreateAlert :one
INSERT INTO price_alerts (case_id, condition, label)
VALUES ($1, $2, $3)
RETURNING *;

-- name: TriggerAlert :one
UPDATE price_alerts
SET triggered = true, triggered_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteAlert :exec
DELETE FROM price_alerts WHERE id = $1;
```

- [ ] 1.6 `backend/db/sqlc.yaml`에 새 쿼리 파일을 등록한다.

```yaml
# backend/db/sqlc.yaml (append to existing queries list)
version: "2"
sql:
  - engine: "postgresql"
    queries:
      - "queries/cases.sql"
      - "queries/trades.sql"
      - "queries/timeline_events.sql"
      - "queries/alerts.sql"
    schema: "migrations/"
    gen:
      go:
        package: "db"
        out: "generated"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

- [ ] 1.7 `src/entities/case/model/types.ts`에 프론트엔드 Case 도메인 타입을 정의한다 (Prisma 의존성 제거, 순수 TypeScript 타입).

```typescript
// src/entities/case/model/types.ts
// Pure TypeScript types — no Prisma dependency (Go backend)

export type CaseStatus = "LIVE" | "CLOSED_SUCCESS" | "CLOSED_FAILURE" | "BACKTEST";

export type TimelineEventType =
  | "NEWS"
  | "DISCLOSURE"
  | "SECTOR"
  | "PRICE_ALERT"
  | "TRADE"
  | "PIPELINE_RESULT"
  | "MONITOR_RESULT";

export interface CaseRow {
  id: string;
  userId: string;
  pipelineId: string;
  symbol: string;
  symbolName: string;
  sector: string | null;
  status: CaseStatus;
  eventDate: string;
  eventSnapshot: Record<string, unknown>;
  successScript: string | null;
  failureScript: string | null;
  closedAt: string | null;
  closedReason: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CaseWithRelations extends CaseRow {
  timelineEvents: TimelineEventRow[];
  trades: TradeRow[];
  priceAlerts: PriceAlertRow[];
}

export interface TimelineEventRow {
  id: string;
  caseId: string;
  date: string;
  dayOffset: number;
  type: TimelineEventType;
  title: string;
  content: string;
  aiAnalysis: string | null;
  data: Record<string, unknown> | null;
  createdAt: string;
}

export interface TradeRow {
  id: string;
  caseId: string;
  userId: string;
  tradeType: "BUY" | "SELL";
  price: number;
  quantity: number;
  fee: number;
  tradedAt: string;
  note: string | null;
  createdAt: string;
}

export interface PriceAlertRow {
  id: string;
  caseId: string;
  condition: string;
  label: string;
  triggered: boolean;
  triggeredAt: string | null;
  createdAt: string;
}

export interface CaseSummary {
  id: string;
  symbol: string;
  symbolName: string;
  sector: string | null;
  status: CaseStatus;
  eventDate: string;
  dayOffset: number;       // D+N (오늘 - eventDate)
  currentReturn: number;   // 현재 수익률 vs event_close
  peakReturn: number;      // 이벤트 이후 최고 수익률
}

export interface CaseDetail {
  id: string;
  symbol: string;
  symbolName: string;
  sector: string | null;
  status: CaseStatus;
  eventDate: string;
  eventSnapshot: EventSnapshot;
  successScript: string | null;
  failureScript: string | null;
  closedAt: string | null;
  closedReason: string | null;
  recentTimeline: TimelineEventRow[];
}

export interface EventSnapshot {
  close: number;
  volume: number;
  high: number;
  low: number;
  [key: string]: unknown;
}

export interface CaseFilters {
  status?: CaseStatus;
  symbol?: string;
  sector?: string;
}
```

- [ ] 1.8 `src/entities/trade/model/types.ts`에 프론트엔드 Trade 도메인 타입을 정의한다 (Prisma 의존성 제거).

```typescript
// src/entities/trade/model/types.ts
// Pure TypeScript types — no Prisma dependency (Go backend)

export type TradeType = "BUY" | "SELL";

export interface TradeRow {
  id: string;
  caseId: string;
  userId: string;
  tradeType: TradeType;
  price: number;
  quantity: number;
  fee: number;
  tradedAt: string;
  note: string | null;
  createdAt: string;
}

export interface PnLSummary {
  totalBuyQuantity: number;
  totalSellQuantity: number;
  remainingQuantity: number;
  averageBuyPrice: number;
  realizedPnL: number;       // 실현 손익
  realizedReturn: number;    // 실현 수익률
  unrealizedPnL: number;     // 미실현 손익 (현재가 기준)
  unrealizedReturn: number;  // 미실현 수익률
  totalFees: number;
}

export interface CreateTradeInput {
  tradeType: TradeType;
  price: number;
  quantity: number;
  fee?: number;
  tradedAt: string;
  note?: string;
}

export interface TradesResponse {
  trades: TradeRow[];
  summary: PnLSummary;
}
```

- [ ] 1.9 sqlc 코드 생성을 실행한다.

```bash
cd backend && sqlc generate
```

- [ ] 1.10 마이그레이션을 실행한다.

```bash
cd backend && goose -dir db/migrations postgres "$DATABASE_URL" up
```

- [ ] 1.11 변경사항을 커밋한다.

```bash
git add backend/db/ src/entities/case/model/types.ts src/entities/trade/model/types.ts
git commit -m "feat(db): add Case, TimelineEvent, Trade SQL migrations and sqlc queries"
```

---

## Task 2: Go Case Handler + Service + Repository — Case CRUD + Timeline API

케이스 목록/상세 조회, 타임라인 이벤트 조회 API를 Go Gin handler + service + repository 패턴으로 구현한다.

**Files:**
- Create: `backend/internal/repository/case_repo.go`
- Create: `backend/internal/service/case_service.go`
- Create: `backend/internal/handler/case_handler.go`
- Modify: `backend/internal/router/router.go` (케이스 라우트 등록)

**Steps:**

- [ ] 2.1 `backend/internal/repository/case_repo.go`에 데이터 접근 레이어를 구현한다.

```go
// backend/internal/repository/case_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "your-module/backend/db/generated"
)

type CaseRepo struct {
	q  *db.Queries
	pool *pgxpool.Pool
}

func NewCaseRepo(pool *pgxpool.Pool) *CaseRepo {
	return &CaseRepo{q: db.New(pool), pool: pool}
}

func (r *CaseRepo) ListByUser(ctx context.Context, userID uuid.UUID, filters CaseFilters) ([]db.Case, error) {
	return r.q.ListCasesByUserWithFilters(ctx, db.ListCasesByUserWithFiltersParams{
		UserID: userID,
		Status: filters.Status,  // nullable
		Symbol: filters.Symbol,  // nullable
		Sector: filters.Sector,  // nullable
	})
}

func (r *CaseRepo) GetByID(ctx context.Context, id, userID uuid.UUID) (db.Case, error) {
	return r.q.GetCaseByID(ctx, db.GetCaseByIDParams{ID: id, UserID: userID})
}

func (r *CaseRepo) Create(ctx context.Context, params db.CreateCaseParams) (db.Case, error) {
	return r.q.CreateCase(ctx, params)
}

func (r *CaseRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status db.CaseStatus, closedAt *string, closedReason *string) (db.Case, error) {
	return r.q.UpdateCaseStatus(ctx, db.UpdateCaseStatusParams{
		ID:           id,
		Status:       status,
		ClosedAt:     closedAt,
		ClosedReason: closedReason,
	})
}

func (r *CaseRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteCase(ctx, id)
}

// --- Filter types ---

type CaseFilters struct {
	Status *db.CaseStatus
	Symbol *string
	Sector *string
}
```

- [ ] 2.2 `backend/internal/repository/case_repo.go`에 타임라인 조회 메서드를 추가한다.

```go
func (r *CaseRepo) ListTimeline(ctx context.Context, caseID uuid.UUID) ([]db.TimelineEvent, error) {
	return r.q.ListTimelineEvents(ctx, caseID)
}

func (r *CaseRepo) ListTimelineByType(ctx context.Context, caseID uuid.UUID, eventType db.TimelineEventType) ([]db.TimelineEvent, error) {
	return r.q.ListTimelineEventsByType(ctx, db.ListTimelineEventsByTypeParams{
		CaseID: caseID,
		Type:   eventType,
	})
}

func (r *CaseRepo) ListTimelineWithPaging(ctx context.Context, caseID uuid.UUID, limit, offset int32) ([]db.TimelineEvent, error) {
	return r.q.ListTimelineEventsWithPaging(ctx, db.ListTimelineEventsWithPagingParams{
		CaseID: caseID,
		Limit:  limit,
		Offset: offset,
	})
}

func (r *CaseRepo) GetRecentTimeline(ctx context.Context, caseID uuid.UUID, limit int32) ([]db.TimelineEvent, error) {
	return r.q.GetRecentTimelineEvents(ctx, db.GetRecentTimelineEventsParams{
		CaseID: caseID,
		Limit:  limit,
	})
}

func (r *CaseRepo) CreateTimelineEvent(ctx context.Context, params db.CreateTimelineEventParams) (db.TimelineEvent, error) {
	return r.q.CreateTimelineEvent(ctx, params)
}
```

- [ ] 2.3 `backend/internal/service/case_service.go`에 도메인 서비스를 구현한다.

```go
// backend/internal/service/case_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	db "your-module/backend/db/generated"
	"your-module/backend/internal/domain/case"
	"your-module/backend/internal/repository"
)

type CaseService struct {
	repo *repository.CaseRepo
	log  *slog.Logger
}

func NewCaseService(repo *repository.CaseRepo, log *slog.Logger) *CaseService {
	return &CaseService{repo: repo, log: log}
}

// CaseSummaryDTO — API 응답용 DTO
type CaseSummaryDTO struct {
	ID            string  `json:"id"`
	Symbol        string  `json:"symbol"`
	SymbolName    string  `json:"symbolName"`
	Sector        *string `json:"sector"`
	Status        string  `json:"status"`
	EventDate     string  `json:"eventDate"`
	DayOffset     int     `json:"dayOffset"`
	CurrentReturn float64 `json:"currentReturn"`
	PeakReturn    float64 `json:"peakReturn"`
}

// CaseDetailDTO — API 응답용 DTO
type CaseDetailDTO struct {
	ID              string               `json:"id"`
	Symbol          string               `json:"symbol"`
	SymbolName      string               `json:"symbolName"`
	Sector          *string              `json:"sector"`
	Status          string               `json:"status"`
	EventDate       string               `json:"eventDate"`
	EventSnapshot   map[string]any       `json:"eventSnapshot"`
	SuccessScript   *string              `json:"successScript"`
	FailureScript   *string              `json:"failureScript"`
	ClosedAt        *string              `json:"closedAt"`
	ClosedReason    *string              `json:"closedReason"`
	RecentTimeline  []TimelineEventDTO   `json:"recentTimeline"`
}

type TimelineEventDTO struct {
	ID         string         `json:"id"`
	CaseID     string         `json:"caseId"`
	Date       string         `json:"date"`
	DayOffset  int            `json:"dayOffset"`
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	AIAnalysis *string        `json:"aiAnalysis"`
	Data       map[string]any `json:"data"`
	CreatedAt  string         `json:"createdAt"`
}

// ListCases — 케이스 목록 (필터 + 요약 정보)
func (s *CaseService) ListCases(ctx context.Context, userID uuid.UUID, filters repository.CaseFilters) ([]CaseSummaryDTO, error) {
	cases, err := s.repo.ListByUser(ctx, userID, filters)
	if err != nil {
		return nil, fmt.Errorf("list cases: %w", err)
	}

	summaries := make([]CaseSummaryDTO, 0, len(cases))
	for _, c := range cases {
		summary := s.toCaseSummary(c)
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

// GetCase — 케이스 상세 (eventSnapshot, success/failure scripts, 최근 타임라인 5건)
func (s *CaseService) GetCase(ctx context.Context, id, userID uuid.UUID) (*CaseDetailDTO, error) {
	c, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get case: %w", err)
	}

	recent, err := s.repo.GetRecentTimeline(ctx, id, 5)
	if err != nil {
		s.log.Warn("failed to get recent timeline", "caseID", id, "error", err)
		recent = []db.TimelineEvent{}
	}

	detail := s.toCaseDetail(c, recent)
	return &detail, nil
}

// DeleteCase — 케이스 삭제 (cascade: timeline, trades)
func (s *CaseService) DeleteCase(ctx context.Context, id, userID uuid.UUID) error {
	// Verify ownership
	_, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("case not found: %w", err)
	}
	s.log.Info("deleting case", "id", id)
	return s.repo.Delete(ctx, id)
}

// CloseCase — 케이스 상태 전환 (LIVE → CLOSED_SUCCESS / CLOSED_FAILURE)
func (s *CaseService) CloseCase(ctx context.Context, id uuid.UUID, status db.CaseStatus, reason string) (*db.Case, error) {
	now := time.Now().Format("2006-01-02")
	c, err := s.repo.UpdateStatus(ctx, id, status, &now, &reason)
	if err != nil {
		return nil, fmt.Errorf("close case: %w", err)
	}

	// 종료 타임라인 이벤트 추가
	_, _ = s.repo.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
		CaseID:    id,
		Date:      time.Now(),
		DayOffset: casedomain.CalculateDayOffset(c.EventDate, time.Now()),
		Type:      db.TimelineEventTypePIPELINERESULT,
		Title:     fmt.Sprintf("Case closed - %s", status),
		Content:   reason,
	})

	s.log.Info("case closed", "id", id, "status", status, "reason", reason)
	return &c, nil
}

// GetTimeline — 타임라인 이벤트 목록 (필터 + 페이징)
func (s *CaseService) GetTimeline(ctx context.Context, caseID uuid.UUID, eventType *string, limit, offset int32) ([]TimelineEventDTO, error) {
	var events []db.TimelineEvent
	var err error

	if eventType != nil {
		events, err = s.repo.ListTimelineByType(ctx, caseID, db.TimelineEventType(*eventType))
	} else if limit > 0 {
		events, err = s.repo.ListTimelineWithPaging(ctx, caseID, limit, offset)
	} else {
		events, err = s.repo.ListTimeline(ctx, caseID)
	}
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}

	dtos := make([]TimelineEventDTO, 0, len(events))
	for _, e := range events {
		dtos = append(dtos, toTimelineEventDTO(e))
	}
	return dtos, nil
}

// GetReturnTracking — D+1/D+7/D+30/Peak/Current 수익률 추적
func (s *CaseService) GetReturnTracking(ctx context.Context, id, userID uuid.UUID) (*casedomain.ReturnTrackingData, error) {
	c, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get case for return tracking: %w", err)
	}
	return casedomain.GetReturnTracking(c)
}

// --- private helpers ---

func (s *CaseService) toCaseSummary(c db.Case) CaseSummaryDTO {
	dayOffset := casedomain.CalculateDayOffset(c.EventDate, time.Now())
	return CaseSummaryDTO{
		ID:            c.ID.String(),
		Symbol:        c.Symbol,
		SymbolName:    c.SymbolName,
		Sector:        c.Sector,
		Status:        string(c.Status),
		EventDate:     c.EventDate.Format("2006-01-02"),
		DayOffset:     dayOffset,
		CurrentReturn: 0, // TODO: integrate with price API
		PeakReturn:    0, // TODO: integrate with price API
	}
}

func (s *CaseService) toCaseDetail(c db.Case, timeline []db.TimelineEvent) CaseDetailDTO {
	events := make([]TimelineEventDTO, 0, len(timeline))
	for _, e := range timeline {
		events = append(events, toTimelineEventDTO(e))
	}

	var closedAt, closedReason *string
	if c.ClosedAt != nil {
		s := c.ClosedAt.Format("2006-01-02")
		closedAt = &s
	}
	closedReason = c.ClosedReason

	return CaseDetailDTO{
		ID:             c.ID.String(),
		Symbol:         c.Symbol,
		SymbolName:     c.SymbolName,
		Sector:         c.Sector,
		Status:         string(c.Status),
		EventDate:      c.EventDate.Format("2006-01-02"),
		EventSnapshot:  c.EventSnapshot,
		SuccessScript:  c.SuccessScript,
		FailureScript:  c.FailureScript,
		ClosedAt:       closedAt,
		ClosedReason:   closedReason,
		RecentTimeline: events,
	}
}

func toTimelineEventDTO(e db.TimelineEvent) TimelineEventDTO {
	var data map[string]any
	if e.Data != nil {
		data = e.Data
	}
	var aiAnalysis *string
	if e.AiAnalysis != nil {
		aiAnalysis = e.AiAnalysis
	}
	return TimelineEventDTO{
		ID:         e.ID.String(),
		CaseID:     e.CaseID.String(),
		Date:       e.Date.Format("2006-01-02"),
		DayOffset:  int(e.DayOffset),
		Type:       string(e.Type),
		Title:      e.Title,
		Content:    e.Content,
		AIAnalysis: aiAnalysis,
		Data:       data,
		CreatedAt:  e.CreatedAt.Format(time.RFC3339),
	}
}
```

- [ ] 2.4 `backend/internal/handler/case_handler.go`에 Gin 핸들러를 구현한다 — thin controller 패턴.

```go
// backend/internal/handler/case_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"your-module/backend/internal/repository"
	"your-module/backend/internal/service"
)

type CaseHandler struct {
	svc *service.CaseService
}

func NewCaseHandler(svc *service.CaseService) *CaseHandler {
	return &CaseHandler{svc: svc}
}

// GET /api/cases — 케이스 목록 (status, symbol, sector 필터)
func (h *CaseHandler) ListCases(c *gin.Context) {
	userID := mustGetUserID(c)

	filters := repository.CaseFilters{}
	if s := c.Query("status"); s != "" {
		filters.Status = &s
	}
	if s := c.Query("symbol"); s != "" {
		filters.Symbol = &s
	}
	if s := c.Query("sector"); s != "" {
		filters.Sector = &s
	}

	cases, err := h.svc.ListCases(c.Request.Context(), userID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cases)
}

// GET /api/cases/:id — 케이스 상세 조회
func (h *CaseHandler) GetCase(c *gin.Context) {
	userID := mustGetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	detail, err := h.svc.GetCase(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "case not found"})
		return
	}
	c.JSON(http.StatusOK, detail)
}

// DELETE /api/cases/:id — 케이스 삭제
func (h *CaseHandler) DeleteCase(c *gin.Context) {
	userID := mustGetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	if err := h.svc.DeleteCase(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "case not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// POST /api/cases/:id/close — 케이스 상태 전환
func (h *CaseHandler) CloseCase(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.CloseCase(c.Request.Context(), id, body.Status, body.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GET /api/cases/:id/timeline — 타임라인 이벤트 목록
func (h *CaseHandler) GetTimeline(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	var eventType *string
	if t := c.Query("type"); t != "" {
		eventType = &t
	}

	limit := int32(0)
	offset := int32(0)
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = int32(v)
		}
	}
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil {
			offset = int32(v)
		}
	}

	events, err := h.svc.GetTimeline(c.Request.Context(), caseID, eventType, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, events)
}

// GET /api/cases/:id/return-tracking — 수익률 추적 데이터
func (h *CaseHandler) GetReturnTracking(c *gin.Context) {
	userID := mustGetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	data, err := h.svc.GetReturnTracking(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

// --- helper ---
func mustGetUserID(c *gin.Context) uuid.UUID {
	userIDStr, _ := c.Get("userID")
	uid, _ := uuid.Parse(userIDStr.(string))
	return uid
}
```

- [ ] 2.5 `backend/internal/router/router.go`에 케이스 라우트를 등록한다.

```go
// backend/internal/router/router.go (append to existing routes)

func RegisterCaseRoutes(r *gin.RouterGroup, h *handler.CaseHandler) {
	cases := r.Group("/cases")
	{
		cases.GET("", h.ListCases)
		cases.GET("/:id", h.GetCase)
		cases.DELETE("/:id", h.DeleteCase)
		cases.POST("/:id/close", h.CloseCase)
		cases.GET("/:id/timeline", h.GetTimeline)
		cases.GET("/:id/return-tracking", h.GetReturnTracking)
	}
}
```

- [ ] 2.6 변경사항을 커밋한다.

```bash
git add backend/internal/handler/case_handler.go backend/internal/service/case_service.go backend/internal/repository/case_repo.go backend/internal/router/router.go
git commit -m "feat(api): implement Case CRUD + Timeline Go API with handler/service/repo pattern"
```

---

## Task 3: Go Trade Handler + Service + Repository + PnL Calculator

매수/매도 기록 CRUD와 FIFO 기반 실현/미실현 손익 계산 로직을 Go로 구현한다.

**Files:**
- Create: `backend/internal/domain/trade/pnl_calculator.go`
- Create: `backend/internal/domain/trade/pnl_calculator_test.go`
- Create: `backend/internal/repository/trade_repo.go`
- Create: `backend/internal/service/trade_service.go`
- Create: `backend/internal/handler/trade_handler.go`
- Modify: `backend/internal/router/router.go` (매매 라우트 등록)

**Steps:**

- [ ] 3.1 TDD: `backend/internal/domain/trade/pnl_calculator_test.go`에 P&L Calculator 테스트를 먼저 작성한다.

```go
// backend/internal/domain/trade/pnl_calculator_test.go
package trade_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"your-module/backend/internal/domain/trade"
)

func TestCalculatePnL_BuySellScenario(t *testing.T) {
	// BUY 100주 @ 50,000 → BUY 50주 @ 48,000 → SELL 80주 @ 55,000
	trades := []trade.TradeInput{
		{Type: "BUY", Price: 50000, Quantity: 100, Fee: 5000, TradedAt: time.Now().Add(-72 * time.Hour)},
		{Type: "BUY", Price: 48000, Quantity: 50, Fee: 2400, TradedAt: time.Now().Add(-48 * time.Hour)},
		{Type: "SELL", Price: 55000, Quantity: 80, Fee: 4400, TradedAt: time.Now().Add(-24 * time.Hour)},
	}

	result := trade.CalculatePnL(trades, 52000)

	assert.Equal(t, 150, result.TotalBuyQuantity)
	assert.Equal(t, 80, result.TotalSellQuantity)
	assert.Equal(t, 70, result.RemainingQuantity)
	assert.InDelta(t, 49333.33, result.AverageBuyPrice, 1.0) // (100*50000 + 50*48000) / 150
	assert.Greater(t, result.RealizedPnL, 0.0)                // 이익 실현
	assert.NotZero(t, result.TotalFees)
}

func TestCalculatePnL_OnlyBuys(t *testing.T) {
	trades := []trade.TradeInput{
		{Type: "BUY", Price: 50000, Quantity: 100, Fee: 5000, TradedAt: time.Now()},
	}

	result := trade.CalculatePnL(trades, 55000)

	assert.Equal(t, 100, result.TotalBuyQuantity)
	assert.Equal(t, 0, result.TotalSellQuantity)
	assert.Equal(t, 100, result.RemainingQuantity)
	assert.Equal(t, 50000.0, result.AverageBuyPrice)
	assert.Equal(t, 0.0, result.RealizedPnL)
	assert.InDelta(t, 500000.0, result.UnrealizedPnL, 1.0) // (55000-50000) * 100
}

func TestCalculatePnL_EmptyTrades(t *testing.T) {
	result := trade.CalculatePnL([]trade.TradeInput{}, 50000)

	assert.Equal(t, 0, result.RemainingQuantity)
	assert.Equal(t, 0.0, result.RealizedPnL)
	assert.Equal(t, 0.0, result.UnrealizedPnL)
}
```

- [ ] 3.2 `backend/internal/domain/trade/pnl_calculator.go`를 구현한다.

```go
// backend/internal/domain/trade/pnl_calculator.go
package trade

import (
	"sort"
	"time"
)

type TradeInput struct {
	Type     string    // "BUY" or "SELL"
	Price    float64
	Quantity int
	Fee      float64
	TradedAt time.Time
}

type PnLSummary struct {
	TotalBuyQuantity  int     `json:"totalBuyQuantity"`
	TotalSellQuantity int     `json:"totalSellQuantity"`
	RemainingQuantity int     `json:"remainingQuantity"`
	AverageBuyPrice   float64 `json:"averageBuyPrice"`
	RealizedPnL       float64 `json:"realizedPnL"`
	RealizedReturn    float64 `json:"realizedReturn"`
	UnrealizedPnL     float64 `json:"unrealizedPnL"`
	UnrealizedReturn  float64 `json:"unrealizedReturn"`
	TotalFees         float64 `json:"totalFees"`
}

// CalculatePnL — FIFO 기반 실현/미실현 P&L 계산
func CalculatePnL(trades []TradeInput, currentPrice float64) PnLSummary {
	if len(trades) == 0 {
		return PnLSummary{}
	}

	// 시간순 정렬
	sorted := make([]TradeInput, len(trades))
	copy(sorted, trades)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TradedAt.Before(sorted[j].TradedAt)
	})

	// FIFO 큐: (price, quantity) 쌍
	type lot struct {
		price    float64
		quantity int
	}
	var buyQueue []lot

	var totalBuyQty, totalSellQty int
	var totalBuyCost, totalFees, realizedPnL float64

	for _, t := range sorted {
		totalFees += t.Fee

		if t.Type == "BUY" {
			buyQueue = append(buyQueue, lot{price: t.Price, quantity: t.Quantity})
			totalBuyQty += t.Quantity
			totalBuyCost += t.Price * float64(t.Quantity)
		} else { // SELL
			totalSellQty += t.Quantity
			remaining := t.Quantity

			for remaining > 0 && len(buyQueue) > 0 {
				front := &buyQueue[0]
				matched := min(remaining, front.quantity)

				realizedPnL += float64(matched) * (t.Price - front.price)
				front.quantity -= matched
				remaining -= matched

				if front.quantity == 0 {
					buyQueue = buyQueue[1:]
				}
			}
		}
	}

	// 잔여 수량 및 평균 매수가
	remainingQty := 0
	remainingCost := 0.0
	for _, lot := range buyQueue {
		remainingQty += lot.quantity
		remainingCost += lot.price * float64(lot.quantity)
	}

	avgBuyPrice := 0.0
	if totalBuyQty > 0 {
		avgBuyPrice = totalBuyCost / float64(totalBuyQty)
	}

	// 미실현 P&L
	unrealizedPnL := 0.0
	if remainingQty > 0 {
		avgRemaining := remainingCost / float64(remainingQty)
		unrealizedPnL = float64(remainingQty) * (currentPrice - avgRemaining)
	}

	// 수익률 계산
	realizedReturn := 0.0
	if totalSellQty > 0 && avgBuyPrice > 0 {
		realizedReturn = realizedPnL / (avgBuyPrice * float64(totalSellQty)) * 100
	}
	unrealizedReturn := 0.0
	if remainingQty > 0 && remainingCost > 0 {
		unrealizedReturn = unrealizedPnL / remainingCost * 100
	}

	return PnLSummary{
		TotalBuyQuantity:  totalBuyQty,
		TotalSellQuantity: totalSellQty,
		RemainingQuantity: remainingQty,
		AverageBuyPrice:   avgBuyPrice,
		RealizedPnL:       realizedPnL,
		RealizedReturn:    realizedReturn,
		UnrealizedPnL:     unrealizedPnL,
		UnrealizedReturn:  unrealizedReturn,
		TotalFees:         totalFees,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] 3.3 P&L Calculator 테스트를 실행한다.

```bash
cd backend && go test ./internal/domain/trade/... -v
```

- [ ] 3.4 `backend/internal/repository/trade_repo.go`에 데이터 접근 레이어를 구현한다.

```go
// backend/internal/repository/trade_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "your-module/backend/db/generated"
)

type TradeRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewTradeRepo(pool *pgxpool.Pool) *TradeRepo {
	return &TradeRepo{q: db.New(pool), pool: pool}
}

func (r *TradeRepo) ListByCaseID(ctx context.Context, caseID uuid.UUID) ([]db.Trade, error) {
	return r.q.ListTradesByCase(ctx, caseID)
}

func (r *TradeRepo) Create(ctx context.Context, params db.CreateTradeParams) (db.Trade, error) {
	return r.q.CreateTrade(ctx, params)
}

func (r *TradeRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteTrade(ctx, id)
}
```

- [ ] 3.5 `backend/internal/service/trade_service.go`에 도메인 서비스를 구현한다 — Trade 생성 시 TimelineEvent(TRADE 유형)도 함께 생성.

```go
// backend/internal/service/trade_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	db "your-module/backend/db/generated"
	tradedomain "your-module/backend/internal/domain/trade"
	"your-module/backend/internal/repository"
)

type TradeService struct {
	tradeRepo *repository.TradeRepo
	caseRepo  *repository.CaseRepo
	log       *slog.Logger
}

func NewTradeService(tradeRepo *repository.TradeRepo, caseRepo *repository.CaseRepo, log *slog.Logger) *TradeService {
	return &TradeService{tradeRepo: tradeRepo, caseRepo: caseRepo, log: log}
}

type CreateTradeRequest struct {
	TradeType string  `json:"tradeType" binding:"required,oneof=BUY SELL"`
	Price     float64 `json:"price" binding:"required,gt=0"`
	Quantity  int     `json:"quantity" binding:"required,gt=0"`
	Fee       float64 `json:"fee"`
	TradedAt  string  `json:"tradedAt" binding:"required"`
	Note      string  `json:"note"`
}

type TradeDTO struct {
	ID        string  `json:"id"`
	CaseID    string  `json:"caseId"`
	UserID    string  `json:"userId"`
	TradeType string  `json:"tradeType"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Fee       float64 `json:"fee"`
	TradedAt  string  `json:"tradedAt"`
	Note      *string `json:"note"`
	CreatedAt string  `json:"createdAt"`
}

type TradesResponseDTO struct {
	Trades  []TradeDTO            `json:"trades"`
	Summary tradedomain.PnLSummary `json:"summary"`
}

// CreateTrade — 매매 기록 추가 + 타임라인 이벤트 생성
func (s *TradeService) CreateTrade(ctx context.Context, caseID, userID uuid.UUID, req CreateTradeRequest) (*TradeDTO, error) {
	// SELL인 경우 잔여 수량 확인
	if req.TradeType == "SELL" {
		existing, err := s.tradeRepo.ListByCaseID(ctx, caseID)
		if err != nil {
			return nil, fmt.Errorf("list trades: %w", err)
		}
		remaining := calculateRemainingQty(existing)
		if req.Quantity > remaining {
			return nil, fmt.Errorf("sell quantity (%d) exceeds remaining (%d)", req.Quantity, remaining)
		}
	}

	tradedAt, err := time.Parse(time.RFC3339, req.TradedAt)
	if err != nil {
		tradedAt, err = time.Parse("2006-01-02", req.TradedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid tradedAt: %w", err)
		}
	}

	var note *string
	if req.Note != "" {
		note = &req.Note
	}

	trade, err := s.tradeRepo.Create(ctx, db.CreateTradeParams{
		CaseID:    caseID,
		UserID:    userID,
		TradeType: db.TradeType(req.TradeType),
		Price:     req.Price,
		Quantity:  int32(req.Quantity),
		Fee:       req.Fee,
		TradedAt:  tradedAt,
		Note:      note,
	})
	if err != nil {
		return nil, fmt.Errorf("create trade: %w", err)
	}

	// 타임라인 이벤트 생성
	caseRow, _ := s.caseRepo.GetByID(ctx, caseID, userID)
	dayOffset := 0
	if !caseRow.EventDate.IsZero() {
		dayOffset = int(tradedAt.Sub(caseRow.EventDate).Hours() / 24)
	}

	_, _ = s.caseRepo.CreateTimelineEvent(ctx, db.CreateTimelineEventParams{
		CaseID:    caseID,
		Date:      tradedAt,
		DayOffset: int32(dayOffset),
		Type:      db.TimelineEventTypeTRADE,
		Title:     fmt.Sprintf("%s %d shares @ %.0f", req.TradeType, req.Quantity, req.Price),
		Content:   req.Note,
	})

	s.log.Info("trade created", "caseID", caseID, "type", req.TradeType, "qty", req.Quantity, "price", req.Price)
	dto := toTradeDTO(trade)
	return &dto, nil
}

// GetTradesWithSummary — 매매 히스토리 + P&L 요약
func (s *TradeService) GetTradesWithSummary(ctx context.Context, caseID uuid.UUID, currentPrice float64) (*TradesResponseDTO, error) {
	trades, err := s.tradeRepo.ListByCaseID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("list trades: %w", err)
	}

	// db.Trade → domain TradeInput 변환
	inputs := make([]tradedomain.TradeInput, 0, len(trades))
	for _, t := range trades {
		inputs = append(inputs, tradedomain.TradeInput{
			Type:     string(t.TradeType),
			Price:    t.Price,
			Quantity: int(t.Quantity),
			Fee:      t.Fee,
			TradedAt: t.TradedAt,
		})
	}

	summary := tradedomain.CalculatePnL(inputs, currentPrice)

	dtos := make([]TradeDTO, 0, len(trades))
	for _, t := range trades {
		dtos = append(dtos, toTradeDTO(t))
	}

	return &TradesResponseDTO{
		Trades:  dtos,
		Summary: summary,
	}, nil
}

// --- helpers ---

func calculateRemainingQty(trades []db.Trade) int {
	total := 0
	for _, t := range trades {
		if t.TradeType == db.TradeTypeBUY {
			total += int(t.Quantity)
		} else {
			total -= int(t.Quantity)
		}
	}
	return total
}

func toTradeDTO(t db.Trade) TradeDTO {
	return TradeDTO{
		ID:        t.ID.String(),
		CaseID:    t.CaseID.String(),
		UserID:    t.UserID.String(),
		TradeType: string(t.TradeType),
		Price:     t.Price,
		Quantity:  int(t.Quantity),
		Fee:       t.Fee,
		TradedAt:  t.TradedAt.Format(time.RFC3339),
		Note:      t.Note,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
}
```

- [ ] 3.6 `backend/internal/handler/trade_handler.go`에 Gin 핸들러를 구현한다 — thin controller.

```go
// backend/internal/handler/trade_handler.go
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"your-module/backend/internal/service"
)

type TradeHandler struct {
	svc *service.TradeService
}

func NewTradeHandler(svc *service.TradeService) *TradeHandler {
	return &TradeHandler{svc: svc}
}

// POST /api/cases/:id/trades — 매매 기록 추가
func (h *TradeHandler) CreateTrade(c *gin.Context) {
	userID := mustGetUserID(c)
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	var req service.CreateTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trade, err := h.svc.CreateTrade(c.Request.Context(), caseID, userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, trade)
}

// GET /api/cases/:id/trades — 매매 히스토리 + P&L 요약
func (h *TradeHandler) ListTrades(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	// currentPrice는 쿼리 파라미터로 전달 (프론트엔드에서 현재가 제공)
	currentPrice := 0.0
	if p := c.Query("currentPrice"); p != "" {
		if v, err := strconv.ParseFloat(p, 64); err == nil {
			currentPrice = v
		}
	}

	result, err := h.svc.GetTradesWithSummary(c.Request.Context(), caseID, currentPrice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
```

- [ ] 3.7 `backend/internal/router/router.go`에 매매 라우트를 등록한다.

```go
// backend/internal/router/router.go (append)

func RegisterTradeRoutes(r *gin.RouterGroup, h *handler.TradeHandler) {
	r.POST("/cases/:id/trades", h.CreateTrade)
	r.GET("/cases/:id/trades", h.ListTrades)
}
```

- [ ] 3.8 테스트 실행 및 커밋.

```bash
cd backend && go test ./internal/domain/trade/... -v
git add backend/internal/domain/trade/ backend/internal/repository/trade_repo.go backend/internal/service/trade_service.go backend/internal/handler/trade_handler.go backend/internal/router/
git commit -m "feat(api): implement Trade CRUD with FIFO P&L calculator in Go domain"
```

---

## Task 4: Go Alert Handler + Service + Repository — Case Aggregate 소속

케이스에 연결된 가격 알림의 생성/조회/삭제 API를 Go로 구현한다. PriceAlert는 Case aggregate에 소속된다.

**Files:**
- Create: `backend/internal/repository/alert_repo.go`
- Create: `backend/internal/service/alert_service.go`
- Create: `backend/internal/handler/alert_handler.go`
- Modify: `backend/internal/router/router.go` (알림 라우트 등록)

**Steps:**

- [ ] 4.1 `backend/internal/repository/alert_repo.go`에 데이터 접근 레이어를 구현한다.

```go
// backend/internal/repository/alert_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	db "your-module/backend/db/generated"
)

type AlertRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewAlertRepo(pool *pgxpool.Pool) *AlertRepo {
	return &AlertRepo{q: db.New(pool), pool: pool}
}

func (r *AlertRepo) ListByCase(ctx context.Context, caseID uuid.UUID) ([]db.PriceAlert, error) {
	return r.q.ListAlertsByCase(ctx, caseID)
}

func (r *AlertRepo) ListPendingByCase(ctx context.Context, caseID uuid.UUID) ([]db.PriceAlert, error) {
	return r.q.ListPendingAlertsByCase(ctx, caseID)
}

func (r *AlertRepo) ListTriggeredByCase(ctx context.Context, caseID uuid.UUID) ([]db.PriceAlert, error) {
	return r.q.ListTriggeredAlertsByCase(ctx, caseID)
}

func (r *AlertRepo) Create(ctx context.Context, params db.CreateAlertParams) (db.PriceAlert, error) {
	return r.q.CreateAlert(ctx, params)
}

func (r *AlertRepo) Trigger(ctx context.Context, id uuid.UUID) (db.PriceAlert, error) {
	return r.q.TriggerAlert(ctx, id)
}

func (r *AlertRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteAlert(ctx, id)
}
```

- [ ] 4.2 `backend/internal/service/alert_service.go`에 서비스 함수를 구현한다.

```go
// backend/internal/service/alert_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	db "your-module/backend/db/generated"
	"your-module/backend/internal/repository"
)

type AlertService struct {
	alertRepo *repository.AlertRepo
	log       *slog.Logger
}

func NewAlertService(alertRepo *repository.AlertRepo, log *slog.Logger) *AlertService {
	return &AlertService{alertRepo: alertRepo, log: log}
}

type AlertDTO struct {
	ID          string  `json:"id"`
	CaseID      string  `json:"caseId"`
	Condition   string  `json:"condition"`
	Label       string  `json:"label"`
	Triggered   bool    `json:"triggered"`
	TriggeredAt *string `json:"triggeredAt"`
	CreatedAt   string  `json:"createdAt"`
}

type AlertsResponseDTO struct {
	Pending   []AlertDTO `json:"pending"`
	Triggered []AlertDTO `json:"triggered"`
}

// ListAlerts — pending/triggered 분리하여 반환
func (s *AlertService) ListAlerts(ctx context.Context, caseID uuid.UUID) (*AlertsResponseDTO, error) {
	pending, err := s.alertRepo.ListPendingByCase(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("list pending alerts: %w", err)
	}
	triggered, err := s.alertRepo.ListTriggeredByCase(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("list triggered alerts: %w", err)
	}

	return &AlertsResponseDTO{
		Pending:   toAlertDTOs(pending),
		Triggered: toAlertDTOs(triggered),
	}, nil
}

// CreateAlert — 가격 알림 생성 (DSL condition + label)
func (s *AlertService) CreateAlert(ctx context.Context, caseID uuid.UUID, condition, label string) (*AlertDTO, error) {
	alert, err := s.alertRepo.Create(ctx, db.CreateAlertParams{
		CaseID:    caseID,
		Condition: condition,
		Label:     label,
	})
	if err != nil {
		return nil, fmt.Errorf("create alert: %w", err)
	}

	s.log.Info("alert created", "caseID", caseID, "condition", condition, "label", label)
	dto := toAlertDTO(alert)
	return &dto, nil
}

// TriggerAlert — 알림 트리거 상태 변경
func (s *AlertService) TriggerAlert(ctx context.Context, alertID uuid.UUID) (*AlertDTO, error) {
	alert, err := s.alertRepo.Trigger(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("trigger alert: %w", err)
	}
	s.log.Info("alert triggered", "alertID", alertID)
	dto := toAlertDTO(alert)
	return &dto, nil
}

// DeleteAlert — 가격 알림 삭제 (pending 상태만)
func (s *AlertService) DeleteAlert(ctx context.Context, alertID uuid.UUID) error {
	s.log.Info("deleting alert", "id", alertID)
	return s.alertRepo.Delete(ctx, alertID)
}

// --- helpers ---

func toAlertDTO(a db.PriceAlert) AlertDTO {
	var triggeredAt *string
	if a.TriggeredAt != nil {
		s := a.TriggeredAt.Format("2006-01-02T15:04:05Z")
		triggeredAt = &s
	}
	return AlertDTO{
		ID:          a.ID.String(),
		CaseID:      a.CaseID.String(),
		Condition:   a.Condition,
		Label:       a.Label,
		Triggered:   a.Triggered,
		TriggeredAt: triggeredAt,
		CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toAlertDTOs(alerts []db.PriceAlert) []AlertDTO {
	dtos := make([]AlertDTO, 0, len(alerts))
	for _, a := range alerts {
		dtos = append(dtos, toAlertDTO(a))
	}
	return dtos
}
```

- [ ] 4.3 `backend/internal/handler/alert_handler.go`에 Gin 핸들러를 구현한다 — thin controller.

```go
// backend/internal/handler/alert_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"your-module/backend/internal/service"
)

type AlertHandler struct {
	svc *service.AlertService
}

func NewAlertHandler(svc *service.AlertService) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// GET /api/cases/:id/alerts — 가격 알림 목록 (pending/triggered 분리)
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	alerts, err := h.svc.ListAlerts(c.Request.Context(), caseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

// POST /api/cases/:id/alerts — 가격 알림 생성
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	caseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid case id"})
		return
	}

	var body struct {
		Condition string `json:"condition" binding:"required"`
		Label     string `json:"label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert, err := h.svc.CreateAlert(c.Request.Context(), caseID, body.Condition, body.Label)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, alert)
}

// DELETE /api/cases/:id/alerts/:alertId — 가격 알림 삭제
func (h *AlertHandler) DeleteAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("alertId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	if err := h.svc.DeleteAlert(c.Request.Context(), alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
```

- [ ] 4.4 `backend/internal/router/router.go`에 알림 라우트를 등록한다.

```go
// backend/internal/router/router.go (append)

func RegisterAlertRoutes(r *gin.RouterGroup, h *handler.AlertHandler) {
	r.GET("/cases/:id/alerts", h.ListAlerts)
	r.POST("/cases/:id/alerts", h.CreateAlert)
	r.DELETE("/cases/:id/alerts/:alertId", h.DeleteAlert)
}
```

- [ ] 4.5 변경사항을 커밋한다.

```bash
git add backend/internal/repository/alert_repo.go backend/internal/service/alert_service.go backend/internal/handler/alert_handler.go backend/internal/router/
git commit -m "feat(api): implement PriceAlert CRUD within Case aggregate in Go"
```

---

## Task 5: Go Return Tracking 도메인 로직

D+1/D+7/D+30/Peak/Current 시점의 수익률과 KOSPI/섹터 대비 수익률 계산 로직을 Go 도메인에 구현한다.

**Files:**
- Create: `backend/internal/domain/case/return_tracking.go`
- Create: `backend/internal/domain/case/return_tracking_test.go`

**Steps:**

- [ ] 5.1 TDD: `backend/internal/domain/case/return_tracking_test.go`에 테스트를 작성한다.

```go
// backend/internal/domain/case/return_tracking_test.go
package casedomain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	casedomain "your-module/backend/internal/domain/case"
)

func TestCalculateDayOffset(t *testing.T) {
	eventDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

	offset := casedomain.CalculateDayOffset(eventDate, now)
	assert.Equal(t, 127, offset) // D+127
}

func TestCalculateReturnPct(t *testing.T) {
	basePrice := 50000.0
	currentPrice := 55000.0

	ret := casedomain.CalculateReturnPct(basePrice, currentPrice)
	assert.InDelta(t, 10.0, ret, 0.01) // +10%
}

func TestCalculateReturnPct_Negative(t *testing.T) {
	basePrice := 50000.0
	currentPrice := 40000.0

	ret := casedomain.CalculateReturnPct(basePrice, currentPrice)
	assert.InDelta(t, -20.0, ret, 0.01) // -20%
}
```

- [ ] 5.2 `backend/internal/domain/case/return_tracking.go`를 구현한다.

```go
// backend/internal/domain/case/return_tracking.go
package casedomain

import (
	"time"
)

// ReturnPeriod — 특정 시점의 수익률 데이터
type ReturnPeriod struct {
	Label     string  `json:"label"`      // "D+1", "D+7", "D+30", "Peak", "Current"
	ReturnPct float64 `json:"returnPct"`  // 수익률 (%)
	VsKospi   float64 `json:"vsKospi"`    // KOSPI 대비 초과수익률 (%)
	VsSector  float64 `json:"vsSector"`   // 섹터 대비 초과수익률 (%)
	DayOffset int     `json:"dayOffset"`  // D+N
}

// ReturnTrackingData — 전체 수익률 추적 데이터
type ReturnTrackingData struct {
	Periods []ReturnPeriod `json:"periods"`
}

// CalculateDayOffset — eventDate 기준 D+N 계산
func CalculateDayOffset(eventDate, now time.Time) int {
	eventDay := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, time.UTC)
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return int(nowDay.Sub(eventDay).Hours() / 24)
}

// CalculateReturnPct — 수익률 계산 (%)
func CalculateReturnPct(basePrice, currentPrice float64) float64 {
	if basePrice == 0 {
		return 0
	}
	return (currentPrice - basePrice) / basePrice * 100
}

// GetReturnTracking — D+1/D+7/D+30/Peak/Current 수익률 추적 데이터 생성
// eventSnapshot의 close 가격을 기준으로 각 시점 수익률을 계산한다.
// priceProvider는 외부에서 주입 (KIS API 등)
func GetReturnTracking(eventClose float64, eventDate time.Time, priceHistory []PricePoint) *ReturnTrackingData {
	periods := make([]ReturnPeriod, 0, 5)

	targetOffsets := []struct {
		label  string
		offset int
	}{
		{"D+1", 1},
		{"D+7", 7},
		{"D+30", 30},
	}

	for _, target := range targetOffsets {
		targetDate := eventDate.AddDate(0, 0, target.offset)
		price := findClosestPrice(priceHistory, targetDate)
		if price > 0 {
			periods = append(periods, ReturnPeriod{
				Label:     target.label,
				ReturnPct: CalculateReturnPct(eventClose, price),
				DayOffset: target.offset,
			})
		}
	}

	// Peak 수익률
	if peakPrice, peakOffset := findPeakPrice(priceHistory, eventDate); peakPrice > 0 {
		periods = append(periods, ReturnPeriod{
			Label:     "Peak",
			ReturnPct: CalculateReturnPct(eventClose, peakPrice),
			DayOffset: peakOffset,
		})
	}

	// Current 수익률
	if len(priceHistory) > 0 {
		latest := priceHistory[len(priceHistory)-1]
		periods = append(periods, ReturnPeriod{
			Label:     "Current",
			ReturnPct: CalculateReturnPct(eventClose, latest.Close),
			DayOffset: CalculateDayOffset(eventDate, latest.Date),
		})
	}

	return &ReturnTrackingData{Periods: periods}
}

// PricePoint — 날짜별 가격 데이터
type PricePoint struct {
	Date  time.Time
	Close float64
}

// findClosestPrice — 특정 날짜에 가장 가까운 가격을 찾는다
func findClosestPrice(history []PricePoint, targetDate time.Time) float64 {
	target := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
	bestDiff := time.Duration(1<<63 - 1)
	bestPrice := 0.0

	for _, p := range history {
		d := time.Date(p.Date.Year(), p.Date.Month(), p.Date.Day(), 0, 0, 0, 0, time.UTC)
		diff := d.Sub(target)
		if diff < 0 {
			diff = -diff
		}
		// 3일 이내의 가장 가까운 데이터만 사용
		if diff < bestDiff && diff <= 3*24*time.Hour {
			bestDiff = diff
			bestPrice = p.Close
		}
	}
	return bestPrice
}

// findPeakPrice — 이벤트 이후 최고가와 해당 D+N을 찾는다
func findPeakPrice(history []PricePoint, eventDate time.Time) (float64, int) {
	peakPrice := 0.0
	peakOffset := 0

	for _, p := range history {
		if p.Date.After(eventDate) && p.Close > peakPrice {
			peakPrice = p.Close
			peakOffset = CalculateDayOffset(eventDate, p.Date)
		}
	}
	return peakPrice, peakOffset
}
```

- [ ] 5.3 테스트 실행 및 커밋.

```bash
cd backend && go test ./internal/domain/case/... -v
git add backend/internal/domain/case/
git commit -m "feat(domain): implement Return Tracking D+1/D+7/D+30/Peak/Current in Go"
```

---

## Task 6: Case 페이지 레이아웃 — 위젯 + 분할 스토어 (프론트엔드 — 유지)

케이스 관리 페이지의 전체 레이아웃을 구현한다. UI는 widgets로 조합하고, 스토어는 entity/feature별로 분할한다. API 호출만 Go 백엔드로 변경한다.

**Files:**
- Create: `src/app/cases/page.tsx` (page entry point only)
- Create: `src/app/cases/layout.tsx`
- Create: `src/widgets/case-tab-bar/ui/CaseTabBar.tsx`
- Create: `src/widgets/case-tab-bar/ui/CaseSummaryHeader.tsx`
- Create: `src/widgets/case-detail-panel/ui/CaseDetailLayout.tsx`
- Create: `src/entities/case/model/case.store.ts` (entity state — API → Go backend)
- Create: `src/features/manage-trades/model/trade.store.ts` (trade state — API → Go backend)
- Create: `src/features/manage-alerts/model/alert.store.ts` (alert state — API → Go backend)
- Test: `src/widgets/case-tab-bar/__tests__/CaseLayout.test.tsx`

**Steps:**

- [ ] 6.1 TDD: 케이스 레이아웃 렌더링 테스트를 작성한다 — 탭 바에 케이스 목록 표시, 탭 클릭 시 선택 변경, 요약 헤더에 symbol/status/D+N/return/peak/sector 표시, 50:50 분할 영역 렌더링.

- [ ] 6.2 `src/entities/case/model/case.store.ts`에 엔티티 스토어를 정의한다 — API 호출은 Go 백엔드로.

```typescript
import { create } from "zustand";
import type { CaseSummary, CaseDetail, TimelineEventRow } from "./types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

interface CaseStore {
  cases: CaseSummary[];
  selectedCaseId: string | null;
  selectedCase: CaseDetail | null;
  timelineEvents: TimelineEventRow[];

  fetchCases: () => Promise<void>;
  selectCase: (id: string) => Promise<void>;
  fetchTimeline: (caseId: string) => Promise<void>;
}

export const useCaseStore = create<CaseStore>()((set) => ({
  cases: [],
  selectedCaseId: null,
  selectedCase: null,
  timelineEvents: [],

  fetchCases: async () => {
    const res = await fetch(`${API_BASE}/cases`, { credentials: "include" });
    const cases = await res.json();
    set({ cases });
  },

  selectCase: async (id) => {
    set({ selectedCaseId: id });
    const res = await fetch(`${API_BASE}/cases/${id}`, { credentials: "include" });
    const selectedCase = await res.json();
    set({ selectedCase });
  },

  fetchTimeline: async (caseId) => {
    const res = await fetch(`${API_BASE}/cases/${caseId}/timeline`, { credentials: "include" });
    const timelineEvents = await res.json();
    set({ timelineEvents });
  },
}));
```

- [ ] 6.3 `src/features/manage-trades/model/trade.store.ts`에 feature-scoped 스토어를 정의한다 — API → Go 백엔드.

```typescript
import { create } from "zustand";
import type { TradeRow, PnLSummary, CreateTradeInput } from "@/entities/trade/model/types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

interface TradeStore {
  trades: TradeRow[];
  tradeSummary: PnLSummary | null;
  fetchTrades: (caseId: string, currentPrice?: number) => Promise<void>;
  addTrade: (caseId: string, trade: CreateTradeInput) => Promise<void>;
}

export const useTradeStore = create<TradeStore>()((set) => ({
  trades: [],
  tradeSummary: null,

  fetchTrades: async (caseId, currentPrice = 0) => {
    const res = await fetch(
      `${API_BASE}/cases/${caseId}/trades?currentPrice=${currentPrice}`,
      { credentials: "include" }
    );
    const { trades, summary } = await res.json();
    set({ trades, tradeSummary: summary });
  },

  addTrade: async (caseId, trade) => {
    await fetch(`${API_BASE}/cases/${caseId}/trades`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(trade),
    });
  },
}));
```

- [ ] 6.4 `src/features/manage-alerts/model/alert.store.ts`에 feature-scoped 스토어를 정의한다 — API → Go 백엔드.

```typescript
import { create } from "zustand";
import type { PriceAlertRow } from "@/entities/case/model/types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api";

interface AlertStore {
  pendingAlerts: PriceAlertRow[];
  triggeredAlerts: PriceAlertRow[];
  fetchAlerts: (caseId: string) => Promise<void>;
  addAlert: (caseId: string, condition: string, label: string) => Promise<void>;
  deleteAlert: (caseId: string, alertId: string) => Promise<void>;
}

export const useAlertStore = create<AlertStore>()((set) => ({
  pendingAlerts: [],
  triggeredAlerts: [],

  fetchAlerts: async (caseId) => {
    const res = await fetch(`${API_BASE}/cases/${caseId}/alerts`, { credentials: "include" });
    const { pending, triggered } = await res.json();
    set({ pendingAlerts: pending, triggeredAlerts: triggered });
  },

  addAlert: async (caseId, condition, label) => {
    await fetch(`${API_BASE}/cases/${caseId}/alerts`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ condition, label }),
    });
  },

  deleteAlert: async (_caseId, alertId) => {
    await fetch(`${API_BASE}/cases/${_caseId}/alerts/${alertId}`, {
      method: "DELETE",
      credentials: "include",
    });
  },
}));
```

- [ ] 6.5 `src/widgets/case-tab-bar/ui/CaseTabBar.tsx`를 구현한다 — 가로 스크롤 가능한 탭 바. 각 탭: `[{symbol} {symbolName} {status badge}]`. 활성 탭 하이라이트. 스크롤 화살표 버튼.

```
┌────────────────────────────────────────────────────────────┐
│ ◀ [247540 ecoprobm LIVE] [005930 Samsung LIVE] [000660 SK..] ▶ │
└────────────────────────────────────────────────────────────┘
```

- [ ] 6.6 `src/widgets/case-tab-bar/ui/CaseSummaryHeader.tsx`를 구현한다 — 선택된 케이스의 요약 정보를 한 줄로 표시.

```
┌────────────────────────────────────────────────────────────┐
│ 247540 에코프로비엠  ● LIVE  D+127  -18.4%  Peak +42.1%  2차전지 │
└────────────────────────────────────────────────────────────┘
```

상태 배지 색상: LIVE=초록, CLOSED_SUCCESS=파랑, CLOSED_FAILURE=빨강.

- [ ] 6.7 `src/widgets/case-detail-panel/ui/CaseDetailLayout.tsx`를 구현한다 — 좌측 50% (Timeline 슬롯) + 우측 50% (Detail 슬롯)의 반응형 분할.

- [ ] 6.8 전체 페이지 조립 — `/cases` 경로에서 TabBar + SummaryHeader + DetailLayout을 조합.

- [ ] 6.9 테스트 실행 및 커밋.

```bash
npx jest src/widgets/case-tab-bar/__tests__/CaseLayout.test.tsx
git add src/app/cases/ src/widgets/case-tab-bar/ src/widgets/case-detail-panel/ src/entities/case/model/case.store.ts src/features/manage-trades/model/trade.store.ts src/features/manage-alerts/model/alert.store.ts
git commit -m "feat(ui): implement Case page layout with widgets, split stores, and Go backend API calls"
```

---

## Task 7: 좌측 50% — Timeline 위젯 (tl-node / tl-dot / tl-card) (프론트엔드 — 유지)

D-Day 기반 타임라인을 세로 축으로 렌더링한다. 각 이벤트를 유형별 컴포넌트(tl-node, tl-dot, tl-card)로 표시한다.

**Files:**
- Create: `src/widgets/case-timeline/ui/Timeline.tsx`
- Create: `src/widgets/case-timeline/ui/TimelineNode.tsx` (tl-node: 주요 이벤트)
- Create: `src/widgets/case-timeline/ui/TimelineDot.tsx` (tl-dot: 간단한 이벤트)
- Create: `src/widgets/case-timeline/ui/TimelineCard.tsx` (tl-card: AI 분석 결과)
- Create: `src/widgets/case-timeline/ui/TimelineConnector.tsx` (세로 연결선)
- Create: `src/widgets/case-timeline/lib/timeline-formatter.ts`
- Test: `src/widgets/case-timeline/__tests__/Timeline.test.tsx`

**Steps:**

- [ ] 7.1 TDD: Timeline 렌더링 테스트를 작성한다 — 다양한 이벤트 유형(NEWS, TRADE, PIPELINE_RESULT, MONITOR_RESULT, PRICE_ALERT)이 올바른 컴포넌트(tl-node/tl-dot/tl-card)로 렌더링되고, D+N 라벨이 정확한지 검증.

- [ ] 7.2 `src/widgets/case-timeline/lib/timeline-formatter.ts`에 이벤트 데이터를 타임라인 컴포넌트 props로 변환하는 프레젠테이션 유틸리티를 구현한다.

```typescript
import type { TimelineEventRow } from "@/entities/case/model/types";

export interface TimelineItem {
  id: string;
  label: string;
  date: Date;
  type: string;
  variant: "node" | "dot" | "card";
  title: string;
  content: string;
  aiAnalysis: string | null;
  icon: string;
  color: string;
}

export function formatTimelineEvent(event: TimelineEventRow, eventDate: Date): TimelineItem {
  const dayOffset = Math.floor((new Date(event.date).getTime() - eventDate.getTime()) / (1000 * 60 * 60 * 24));
  const label = dayOffset === 0 ? "D-Day" : `D+${dayOffset}`;

  return {
    id: event.id,
    label,
    date: new Date(event.date),
    type: event.type,
    variant: getVariant(event.type), // "node" | "dot" | "card"
    title: event.title,
    content: event.content,
    aiAnalysis: event.aiAnalysis,
    icon: getIcon(event.type),
    color: getColor(event.type),
  };
}

// 매핑 규칙:
// PIPELINE_RESULT, TRADE → tl-node (주요 이벤트, 큰 노드)
// PRICE_ALERT, SECTOR → tl-dot (간단한 이벤트, 작은 점)
// NEWS, DISCLOSURE, MONITOR_RESULT → tl-card (AI 분석 포함, 카드형)

function getVariant(type: string): "node" | "dot" | "card" {
  switch (type) {
    case "PIPELINE_RESULT":
    case "TRADE":
      return "node";
    case "PRICE_ALERT":
    case "SECTOR":
      return "dot";
    case "NEWS":
    case "DISCLOSURE":
    case "MONITOR_RESULT":
    default:
      return "card";
  }
}

function getIcon(type: string): string {
  const icons: Record<string, string> = {
    NEWS: "newspaper",
    DISCLOSURE: "file-text",
    SECTOR: "layers",
    PRICE_ALERT: "bell",
    TRADE: "trending-up",
    PIPELINE_RESULT: "play-circle",
    MONITOR_RESULT: "activity",
  };
  return icons[type] || "circle";
}

function getColor(type: string): string {
  const colors: Record<string, string> = {
    NEWS: "blue",
    DISCLOSURE: "purple",
    SECTOR: "yellow",
    PRICE_ALERT: "orange",
    TRADE: "green",
    PIPELINE_RESULT: "cyan",
    MONITOR_RESULT: "pink",
  };
  return colors[type] || "gray";
}
```

> **DDD Note:** timeline-formatter는 `widgets/case-timeline/lib/`에 배치한다. 프레젠테이션 로직이므로 entity가 아닌 widget 내부에 위치한다.

- [ ] 7.3 `src/widgets/case-timeline/ui/TimelineNode.tsx` (tl-node)를 구현한다 — 큰 원형 노드 + 제목 + 날짜. 주요 이벤트(D-Day, 매수/매도, Peak 등)에 사용.

```
  ●── D-Day: 2yr Max Volume
  │   2026-01-15  Vol: 28.4M
```

- [ ] 7.4 `src/widgets/case-timeline/ui/TimelineDot.tsx` (tl-dot)를 구현한다 — 작은 점 + 한 줄 설명. 가격 알림, 섹터 이벤트 등.

```
  ·── D+34: Peak +42.1%
```

- [ ] 7.5 `src/widgets/case-timeline/ui/TimelineCard.tsx` (tl-card)를 구현한다 — 카드형 레이아웃. AI 분석 결과, 뉴스 요약 등. 접기/펼치기 가능.

```
  ●── D+7: 관련 뉴스 분석
  │   ┌─────────────────────────────┐
  │   │ IRA 보조금 최종 확정         │
  │   │ Impact: 8/10  Sentiment: +  │
  │   │ [자세히 보기]                │
  │   └─────────────────────────────┘
```

- [ ] 7.6 `src/widgets/case-timeline/ui/TimelineConnector.tsx`를 구현한다 — tl-node/tl-dot/tl-card 사이를 잇는 세로 연결선. 현재 시점 표시 마커.

- [ ] 7.7 `src/widgets/case-timeline/ui/Timeline.tsx`에 전체 타임라인을 조합한다 — 스크롤 가능한 세로 레이아웃. 최신 이벤트가 하단. "현재" 마커 표시.

- [ ] 7.8 테스트 실행 및 커밋.

```bash
npx jest src/widgets/case-timeline/__tests__/Timeline.test.tsx
git add src/widgets/case-timeline/
git commit -m "feat(ui): implement Case Timeline widget with tl-node, tl-dot, tl-card components"
```

---

## Task 8: 우측 50% — Detail Panel 위젯 + Feature UI (프론트엔드 — 유지)

케이스 상세 우측 패널의 4개 섹션을 구현한다. ConditionProgress/ReturnTrackingTable은 widget UI, TradeHistory/PriceAlertsList는 feature UI이다.

**Files:**
- Create: `src/widgets/case-detail-panel/ui/ConditionProgress.tsx`
- Create: `src/widgets/case-detail-panel/ui/ReturnTrackingTable.tsx`
- Create: `src/widgets/case-detail-panel/ui/CaseDetailPanel.tsx` (4개 섹션 조합)
- Create: `src/features/manage-trades/ui/TradeHistory.tsx`
- Create: `src/features/manage-trades/ui/TradeForm.tsx`
- Create: `src/features/manage-alerts/ui/PriceAlertsList.tsx`
- Create: `src/features/manage-alerts/ui/AlertForm.tsx`
- Test: `src/widgets/case-detail-panel/__tests__/CaseDetailPanel.test.tsx`

**Steps:**

- [ ] 8.1 TDD: CaseDetailPanel 통합 테스트를 작성한다 — 4개 섹션 렌더링, 성공/실패 진행률 바 표시, Return Tracking 테이블 데이터, Trade History 목록, Price Alerts pending/triggered 분리.

- [ ] 8.2 `src/widgets/case-detail-panel/ui/ConditionProgress.tsx`를 구현한다 — 성공/실패 조건의 현재 진행률을 프로그레스 바로 표시.

```
┌─ Success / Failure Conditions ──────────────┐
│ ✓ Success: close >= event_high * 2.0        │
│   Target: 304,000  Current: 248,600         │
│   ████████████░░░░░░ 81.8%                  │
│                                              │
│ ✗ Failure: close < pre_event_ma(120)        │
│   Threshold: 135,200  Current: 248,600      │
│   Safe margin: +83.9%                        │
│   ░░░░░░░░░░░░░░░░░░ 0%                    │
└──────────────────────────────────────────────┘
```

진행률 계산: `(현재가 - 기준가) / (목표가 - 기준가) * 100`.

- [ ] 8.3 `src/widgets/case-detail-panel/ui/ReturnTrackingTable.tsx`를 구현한다 — D+1, D+7, D+30, Peak, Current 시점의 수익률을 KOSPI, 섹터 대비 비교. 데이터는 Go 백엔드의 `/api/cases/:id/return-tracking`에서 가져온다.

```
┌─ Return Tracking ─────────────────────────────────────┐
│ Period    │ Return  │ vs KOSPI │ vs Sector │ Status    │
│───────────┼─────────┼──────────┼───────────┼──────────│
│ D+1       │ +3.2%   │ +2.8%   │ +1.5%     │ ✓        │
│ D+7       │ +8.7%   │ +7.1%   │ +5.2%     │ ✓        │
│ D+30      │ +15.4%  │ +14.0%  │ +10.8%    │ ✓        │
│ Peak      │ +42.1%  │ +40.5%  │ +35.3%    │ D+34     │
│ Current   │ -18.4%  │ -17.2%  │ -12.1%    │ D+127    │
└───────────────────────────────────────────────────────┘
```

수익률 양수=초록, 음수=빨강 텍스트 색상.

- [ ] 8.4 `src/features/manage-trades/ui/TradeHistory.tsx`를 구현한다 — 매수/매도 기록 목록 + P&L 요약. `useTradeStore`에서 상태를 읽는다.

```
┌─ Trade History ─────────────────────────────────────┐
│ Date       │ Type │ Price    │ Qty  │ Amount        │
│────────────┼──────┼──────────┼──────┼──────────────│
│ 2026-01-22 │ BUY  │ 155,000  │ 200  │ 31,000,000   │
│ 2026-02-10 │ BUY  │ 148,000  │ 100  │ 14,800,000   │
│ 2026-02-28 │ SELL │ 185,000  │ 150  │ 27,750,000   │
├────────────────────────────────────────────────────│
│ Avg Buy: 152,667  Remaining: 150주                  │
│ Realized P&L: +4,800,000 (+12.2%)                   │
│ Unrealized P&L: -1,300,000 (-5.7%)                  │
│ [+ Add Trade]                                       │
└─────────────────────────────────────────────────────┘
```

- [ ] 8.5 `src/features/manage-trades/ui/TradeForm.tsx`를 구현한다 — 매수/매도 입력 폼. BUY/SELL 토글, 가격, 수량, 수수료, 날짜, 메모 필드. SELL 시 잔여 수량 초과 방지 유효성 검증.

- [ ] 8.6 `src/features/manage-alerts/ui/PriceAlertsList.tsx`를 구현한다 — pending 알림(대기 중)과 triggered 알림(발동됨)을 구분하여 표시. `useAlertStore`에서 상태를 읽는다.

```
┌─ Price Alerts ──────────────────────────────────┐
│ Pending:                                         │
│   ○ close >= 200,000  "목표가 도달"        [x]   │
│   ○ rsi(14) < 30      "RSI 과매도"         [x]   │
│                                                  │
│ Triggered:                                       │
│   ● close >= 180,000  "1차 저항선"  2026-02-15   │
│                                                  │
│ [+ Add Alert]                                    │
└──────────────────────────────────────────────────┘
```

- [ ] 8.7 `src/features/manage-alerts/ui/AlertForm.tsx`를 구현한다 — 가격 알림 추가 폼. DSL condition 입력 + label 입력.

- [ ] 8.8 `src/widgets/case-detail-panel/ui/CaseDetailPanel.tsx`에 4개 섹션을 세로로 조합한다 — ConditionProgress -> ReturnTrackingTable -> TradeHistory -> PriceAlertsList. 각 섹션은 접기/펼치기 가능.

> **FSD Note:** CaseDetailPanel(widget)은 features/manage-trades와 features/manage-alerts의 UI 컴포넌트를 import한다. Widget은 feature를 조합할 수 있다 (widget > feature > entity 계층 구조).

- [ ] 8.9 테스트 실행 및 커밋.

```bash
npx jest src/widgets/case-detail-panel/__tests__/CaseDetailPanel.test.tsx
git add src/widgets/case-detail-panel/ src/features/manage-trades/ui/ src/features/manage-alerts/ui/
git commit -m "feat(ui): implement Case detail panel widget with trade/alert feature UIs"
```

---

## Task 9: 전체 통합 — 상태 전환 + 데이터 플로우 + UI 연결

케이스 상태 전환(LIVE -> CLOSED_SUCCESS / CLOSED_FAILURE) 로직과 페이지 전체 데이터 플로우를 통합한다.

**Files:**
- Modify: `src/app/cases/page.tsx` (데이터 페칭 + 전체 조립)
- Modify: `src/entities/case/model/case.store.ts` (상태 전환 액션 추가)
- Modify: `src/widgets/case-tab-bar/ui/CaseSummaryHeader.tsx` (상태 배지 동적 업데이트)
- Test: `src/entities/case/__tests__/case-integration.test.tsx`

**Steps:**

- [ ] 9.1 TDD: 케이스 상태 전환 테스트를 작성한다 — LIVE 케이스가 성공 조건 도달 시 CLOSED_SUCCESS로, 실패 조건 도달 시 CLOSED_FAILURE로 전환. closedAt, closedReason 필드 업데이트 검증.

- [ ] 9.2 `src/entities/case/model/case.store.ts`에 상태 전환 액션을 추가한다.

```typescript
// case.store.ts에 추가
closeCase: async (id: string, status: "CLOSED_SUCCESS" | "CLOSED_FAILURE", reason: string) => {
  await fetch(`${API_BASE}/cases/${id}/close`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ status, reason }),
  });
  // 목록 갱신
  get().fetchCases();
},
```

- [ ] 9.3 `/cases` 페이지의 전체 데이터 플로우를 연결한다 — 페이지 로드 시 `useCaseStore.fetchCases()`, 탭 클릭 시 `selectCase()` -> `fetchTimeline()` + `useTradeStore.fetchTrades()` + `useAlertStore.fetchAlerts()`.

- [ ] 9.4 케이스 선택 시 좌측 Timeline(widget)과 우측 DetailPanel(widget)이 동시에 업데이트되는 반응형 데이터 플로우를 검증한다.

- [ ] 9.5 CLOSED 상태의 케이스는 매매 추가/알림 추가 UI를 비활성화하고, 상태 배지를 CLOSED_SUCCESS(파랑) 또는 CLOSED_FAILURE(빨강)로 표시한다.

- [ ] 9.6 통합 테스트 실행 및 커밋.

```bash
npx jest src/entities/case/__tests__/case-integration.test.tsx
git add src/entities/case/ src/app/cases/ src/widgets/case-tab-bar/
git commit -m "feat(case): implement status transitions and full page integration with Go backend"
```

---

## API Route Summary (Go Gin)

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/api/cases` | `CaseHandler.ListCases` | 케이스 목록 (필터: status, symbol, sector) |
| GET | `/api/cases/:id` | `CaseHandler.GetCase` | 케이스 상세 (eventSnapshot + 최근 타임라인 5건) |
| DELETE | `/api/cases/:id` | `CaseHandler.DeleteCase` | 케이스 삭제 (cascade) |
| POST | `/api/cases/:id/close` | `CaseHandler.CloseCase` | 케이스 상태 전환 |
| GET | `/api/cases/:id/timeline` | `CaseHandler.GetTimeline` | 타임라인 이벤트 목록 (type, limit, offset) |
| GET | `/api/cases/:id/return-tracking` | `CaseHandler.GetReturnTracking` | D+1/D+7/D+30/Peak/Current 수익률 |
| POST | `/api/cases/:id/trades` | `TradeHandler.CreateTrade` | 매매 기록 추가 |
| GET | `/api/cases/:id/trades` | `TradeHandler.ListTrades` | 매매 히스토리 + P&L 요약 |
| GET | `/api/cases/:id/alerts` | `AlertHandler.ListAlerts` | 가격 알림 목록 (pending/triggered) |
| POST | `/api/cases/:id/alerts` | `AlertHandler.CreateAlert` | 가격 알림 생성 |
| DELETE | `/api/cases/:id/alerts/:alertId` | `AlertHandler.DeleteAlert` | 가격 알림 삭제 |

## Migration from Original Plan

### Removed (Prisma/Next.js API Routes)
- `src/entities/case/api/case.repository.ts` — replaced by `backend/internal/repository/case_repo.go`
- `src/entities/case/api/case.service.ts` — replaced by `backend/internal/service/case_service.go`
- `src/entities/trade/api/trade.repository.ts` — replaced by `backend/internal/repository/trade_repo.go`
- `src/entities/trade/api/trade.service.ts` — replaced by `backend/internal/service/trade_service.go`
- `src/entities/case/api/alert.service.ts` — replaced by `backend/internal/service/alert_service.go`
- `src/entities/case/lib/return-tracking.ts` — replaced by `backend/internal/domain/case/return_tracking.go`
- `src/entities/trade/lib/pnl-calculator.ts` — replaced by `backend/internal/domain/trade/pnl_calculator.go`
- `src/app/api/cases/` — all API routes replaced by Go Gin handlers
- `prisma/schema.prisma` (Case models) — replaced by SQL migration + sqlc

### Added (Go Backend)
- `backend/db/migrations/004_case.sql` — SQL migration for all case tables
- `backend/db/queries/cases.sql` — sqlc queries for cases
- `backend/db/queries/trades.sql` — sqlc queries for trades
- `backend/db/queries/timeline_events.sql` — sqlc queries for timeline events
- `backend/db/queries/alerts.sql` — sqlc queries for price alerts
- `backend/internal/handler/case_handler.go` — Case CRUD + Timeline Gin handler
- `backend/internal/handler/trade_handler.go` — Trade CRUD Gin handler
- `backend/internal/handler/alert_handler.go` — Alert CRUD Gin handler
- `backend/internal/service/case_service.go` — Case business logic
- `backend/internal/service/trade_service.go` — Trade + PnL business logic
- `backend/internal/service/alert_service.go` — Alert business logic
- `backend/internal/repository/case_repo.go` — Case sqlc wrapper
- `backend/internal/repository/trade_repo.go` — Trade sqlc wrapper
- `backend/internal/repository/alert_repo.go` — Alert sqlc wrapper
- `backend/internal/domain/case/return_tracking.go` — Return tracking domain logic
- `backend/internal/domain/trade/pnl_calculator.go` — FIFO PnL calculator domain logic
- `backend/internal/router/router.go` — Gin route registration

### Modified (Frontend API Calls Only)
- `src/entities/case/model/case.store.ts` — API calls changed from Next.js API routes to Go backend
- `src/features/manage-trades/model/trade.store.ts` — API calls changed to Go backend
- `src/features/manage-alerts/model/alert.store.ts` — API calls changed to Go backend
