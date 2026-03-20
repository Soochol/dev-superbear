# Pipeline Builder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 에이전트 블록 기반 파이프라인 빌더를 구축하여 종목 분석 → 모니터링 → 판단까지 하나의 파이프라인으로 정의하고 실행할 수 있게 한다.
**Architecture:** FSD(Feature-Sliced Design) + DDD 아키텍처를 적용한다. 프론트엔드는 `features/pipeline-builder/ui/`(3-섹션 캔버스, Topbar, NodePalette), `features/agent-block-editor/ui/`(AgentBlockPrompt 구조화 에디터), `features/pipeline-generator/ui/`(AI 자연어 파이프라인 생성)로 구성한다. 백엔드는 Go(Gin + sqlc + Google ADK Go)로 구현하며, `backend/internal/` 아래 handler/service/repository/agent/domain 레이어로 분리한다. Repository 패턴으로 데이터 접근을 추상화한다. 파이프라인 스토어는 3개 슬라이스(analysis, monitor, judgment)로 분할하여 기능별 관심사를 분리한다. 팔레트의 블록은 AgentBlockTemplate(재사용 가능 원본)이며, 캔버스에 드래그 시 복사본(AgentBlock)으로 생성된다.
**Tech Stack:** Next.js (App Router) + TypeScript + TailwindCSS (Frontend) | Go + Gin + sqlc + PostgreSQL + Google ADK Go (Backend)

### FSD Directory Map

```
src/                                    # Frontend (유지)
  app/
    pipeline/                           # page entry point only

  features/
    pipeline-builder/
      ui/                               # PipelineCanvas, PipelineTopbar, NodePalette
        sections/                       # AnalysisSection, MonitoringSection, JudgmentSection
      model/
        analysis.slice.ts               # analysis stages state
        monitor.slice.ts                # monitor blocks state
        judgment.slice.ts               # success/failure scripts, price alerts state
        pipeline.store.ts               # composed store (merges 3 slices)
      lib/
        useRegisterAndRun.ts            # extracted Register & Run hook
        usePipelineDragDrop.ts          # drag-and-drop hook
      api/                              # feature-level API calls → Go backend

    agent-block-editor/
      ui/                               # AgentBlockEditor modal, ToolSelector
      model/                            # editor-local state if needed

    pipeline-generator/
      ui/                               # AIGenerateModal
      api/                              # generate endpoint call → Go backend

  entities/
    pipeline/
      model/
        types.ts                        # Pipeline, Stage 프론트엔드 타입 (API 응답 기반)
    agent-block/
      model/
        types.ts                        # AgentBlock, MonitorBlock 프론트엔드 타입
      ui/
        BlockCard.tsx                   # reusable block card display
        MonitorCard.tsx                 # reusable monitor card display
      lib/
        agent-tools.ts                  # AGENT_TOOLS constant

  shared/
    lib/logger.ts                       # logger (replaces console.log)
    api/                                # shared API utilities (fetch wrapper for Go backend)

backend/                                # Go Backend (신규)
  internal/
    handler/
      pipeline_handler.go              # Pipeline CRUD + Execute (Gin handlers)
      block_handler.go                 # AgentBlock CRUD (Gin handlers)
    service/
      pipeline_service.go              # 파이프라인 비즈니스 로직
      pipeline_orchestrator.go         # 파이프라인 실행 오케스트레이션 (순차/병렬)
      pipeline_generator.go            # NL → Pipeline (ADK Go)
    repository/
      pipeline_repo.go                 # Pipeline sqlc CRUD
      block_repo.go                    # AgentBlock/MonitorBlock sqlc CRUD
    agent/
      runner.go                        # AgentRunner (ADK Go wrapper)
    domain/
      pipeline.go                      # Pipeline, Stage, PipelineJob 도메인 모델
      block.go                         # AgentBlock, MonitorBlock, PriceAlert 도메인 모델
      input_output.go                  # AgentInput, AgentOutput, PipelineExecutionContext
  db/
    migrations/
      003_pipeline.sql                 # Pipeline, Stage, AgentBlock, MonitorBlock, PriceAlert, PipelineJob
    queries/
      pipelines.sql                    # sqlc queries for Pipeline, Stage, PipelineJob
      blocks.sql                       # sqlc queries for AgentBlock, MonitorBlock, PriceAlert
```

---

## Task 1: SQL 마이그레이션 + sqlc — Pipeline, Stage, AgentBlock, MonitorBlock, PriceAlert, PipelineJob 모델 정의

파이프라인 관련 전체 데이터 모델을 SQL 마이그레이션과 sqlc 쿼리로 정의한다.

**Files:**
- Create: `backend/db/migrations/003_pipeline.sql`
- Create: `backend/db/queries/pipelines.sql`
- Create: `backend/db/queries/blocks.sql`
- Create: `backend/internal/domain/pipeline.go`
- Create: `backend/internal/domain/block.go`
- Create: `backend/internal/domain/input_output.go`
- Create: `src/entities/pipeline/model/types.ts` (프론트엔드 타입)
- Create: `src/entities/agent-block/model/types.ts` (프론트엔드 타입)
- Test: `backend/internal/domain/pipeline_test.go`

**Steps:**

- [ ] 1.1 SQL 마이그레이션 파일을 작성한다 — Pipeline 테이블.

```sql
-- backend/db/migrations/003_pipeline.sql

-- Pipeline: 에이전트 블록 기반 분석 파이프라인
CREATE TABLE pipelines (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id),
  name            TEXT NOT NULL,
  description     TEXT NOT NULL DEFAULT '',
  success_script  TEXT,
  failure_script  TEXT,
  is_public       BOOLEAN NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipelines_user_id ON pipelines(user_id);
```

> **DDD Note:** `cases Case[]` 관계를 제거했다. Pipeline->Case는 cross-aggregate 커플링이므로, Case 쪽에서 pipeline_id로 단방향 참조한다.

- [ ] 1.2 Stage 테이블을 추가한다 (같은 order_index = 병렬 실행).

```sql
-- Stage: 파이프라인 내 실행 단계 (같은 order_index = 병렬, 다른 order_index = 순차)
CREATE TABLE stages (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
  section         TEXT NOT NULL CHECK (section IN ('analysis', 'monitoring', 'judgment')),
  order_index     INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_stages_pipeline_id ON stages(pipeline_id);
```

- [ ] 1.3 AgentBlock 테이블을 추가한다 (AgentBlockPrompt 구조 포함).

```sql
-- AgentBlock: 에이전트 실행 단위 (AgentBlockPrompt 구조화 필드 포함)
CREATE TABLE agent_blocks (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id),
  stage_id        UUID REFERENCES stages(id) ON DELETE CASCADE,
  name            TEXT NOT NULL,
  objective       TEXT NOT NULL DEFAULT '',
  input_desc      TEXT NOT NULL DEFAULT '',
  tools           TEXT[] DEFAULT '{}',
  output_format   TEXT NOT NULL DEFAULT '',
  constraints     TEXT,
  examples        TEXT,
  instruction     TEXT NOT NULL DEFAULT '',
  system_prompt   TEXT,
  allowed_tools   TEXT[] DEFAULT '{}',
  output_schema   JSONB,
  is_public       BOOLEAN NOT NULL DEFAULT false,
  is_template     BOOLEAN NOT NULL DEFAULT false,
  template_id     UUID,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_blocks_user_id ON agent_blocks(user_id);
CREATE INDEX idx_agent_blocks_stage_id ON agent_blocks(stage_id);
CREATE INDEX idx_agent_blocks_template ON agent_blocks(is_template) WHERE is_template = true;
```

> **DDD Note:** `is_template`/`template_id` 필드를 추가했다. 팔레트의 블록은 Template(is_template=true)이고, 캔버스에 드래그 시 template_id를 참조하는 복사본(is_template=false)을 생성한다.

- [ ] 1.4 MonitorBlock 테이블을 추가한다 (cron 스케줄 + 활성화 제어).

```sql
-- MonitorBlock: cron 기반 반복 실행 모니터링 블록
CREATE TABLE monitor_blocks (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
  block_id        UUID NOT NULL UNIQUE REFERENCES agent_blocks(id) ON DELETE CASCADE,
  cron            TEXT NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitor_blocks_pipeline_id ON monitor_blocks(pipeline_id);
```

- [ ] 1.5 PriceAlert 테이블을 추가한다.

```sql
-- PriceAlert: DSL 조건 기반 가격 알림
CREATE TABLE price_alerts (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID REFERENCES pipelines(id) ON DELETE CASCADE,
  case_id         UUID REFERENCES cases(id) ON DELETE CASCADE,
  condition       TEXT NOT NULL,
  label           TEXT NOT NULL,
  triggered       BOOLEAN NOT NULL DEFAULT false,
  triggered_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_alerts_pipeline_id ON price_alerts(pipeline_id);
CREATE INDEX idx_price_alerts_case_id ON price_alerts(case_id);
```

> **DDD Note:** PriceAlert는 Pipeline aggregate 또는 Case aggregate에 소속된다. 두 FK 중 하나만 설정된다.

- [ ] 1.6 PipelineJob 테이블을 추가한다 (실행 상태 추적).

```sql
-- PipelineJob: 파이프라인 실행 인스턴스 (상태 추적)
CREATE TABLE pipeline_jobs (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
  symbol          TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED')),
  result          JSONB,
  error           TEXT,
  started_at      TIMESTAMPTZ,
  completed_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipeline_jobs_pipeline_id ON pipeline_jobs(pipeline_id);
CREATE INDEX idx_pipeline_jobs_status ON pipeline_jobs(status);
```

- [ ] 1.7 `backend/internal/domain/pipeline.go`에 Pipeline 도메인 모델을 정의한다.

```go
// backend/internal/domain/pipeline.go
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Pipeline struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"userId"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	SuccessScript *string    `json:"successScript"`
	FailureScript *string    `json:"failureScript"`
	IsPublic      bool       `json:"isPublic"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`

	// Relations (populated by repository)
	Stages      []Stage      `json:"stages,omitempty"`
	Monitors    []MonitorBlock `json:"monitors,omitempty"`
	PriceAlerts []PriceAlert `json:"priceAlerts,omitempty"`
}

type Stage struct {
	ID         uuid.UUID `json:"id"`
	PipelineID uuid.UUID `json:"pipelineId"`
	Section    string    `json:"section"`
	OrderIndex int       `json:"order"`
	CreatedAt  time.Time `json:"createdAt"`

	// Relations
	Blocks []AgentBlock `json:"blocks,omitempty"`
}

type PipelineJob struct {
	ID          uuid.UUID        `json:"id"`
	PipelineID  uuid.UUID        `json:"pipelineId"`
	Symbol      string           `json:"symbol"`
	Status      string           `json:"status"`
	Result      *json.RawMessage `json:"result,omitempty"`
	Error       *string          `json:"error,omitempty"`
	StartedAt   *time.Time       `json:"startedAt,omitempty"`
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
}

// PipelineJob status constants
const (
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
)

// Stage section constants
const (
	SectionAnalysis   = "analysis"
	SectionMonitoring = "monitoring"
	SectionJudgment   = "judgment"
)
```

- [ ] 1.8 `backend/internal/domain/block.go`에 AgentBlock 도메인 모델을 정의한다.

```go
// backend/internal/domain/block.go
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AgentBlock struct {
	ID           uuid.UUID        `json:"id"`
	UserID       uuid.UUID        `json:"userId"`
	StageID      *uuid.UUID       `json:"stageId,omitempty"`
	Name         string           `json:"name"`
	Objective    string           `json:"objective"`
	InputDesc    string           `json:"inputDesc"`
	Tools        []string         `json:"tools"`
	OutputFormat string           `json:"outputFormat"`
	Constraints  *string          `json:"constraints,omitempty"`
	Examples     *string          `json:"examples,omitempty"`
	Instruction  string           `json:"instruction"`
	SystemPrompt *string          `json:"systemPrompt,omitempty"`
	AllowedTools []string         `json:"allowedTools"`
	OutputSchema *json.RawMessage `json:"outputSchema,omitempty"`
	IsPublic     bool             `json:"isPublic"`
	IsTemplate   bool             `json:"isTemplate"`
	TemplateID   *uuid.UUID       `json:"templateId,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`

	// Relations
	MonitorBlock *MonitorBlock `json:"monitorBlock,omitempty"`
}

type MonitorBlock struct {
	ID         uuid.UUID `json:"id"`
	PipelineID uuid.UUID `json:"pipelineId"`
	BlockID    uuid.UUID `json:"blockId"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	// Relations
	Block *AgentBlock `json:"block,omitempty"`
}

type PriceAlert struct {
	ID          uuid.UUID  `json:"id"`
	PipelineID  *uuid.UUID `json:"pipelineId,omitempty"`
	CaseID      *uuid.UUID `json:"caseId,omitempty"`
	Condition   string     `json:"condition"`
	Label       string     `json:"label"`
	Triggered   bool       `json:"triggered"`
	TriggeredAt *time.Time `json:"triggeredAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// AgentBlockTemplate is a read-only view for palette display.
// Template: reusable block in palette (IsTemplate=true). Drag creates a copy.
type AgentBlockTemplate struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Objective    string    `json:"objective"`
	InputDesc    string    `json:"inputDesc"`
	Tools        []string  `json:"tools"`
	OutputFormat string    `json:"outputFormat"`
	Constraints  *string   `json:"constraints,omitempty"`
	Examples     *string   `json:"examples,omitempty"`
	IsPublic     bool      `json:"isPublic"`
}
```

- [ ] 1.9 `backend/internal/domain/input_output.go`에 실행 컨텍스트 도메인 타입을 정의한다.

```go
// backend/internal/domain/input_output.go
package domain

type AgentInput struct {
	Instruction string            `json:"instruction"`
	Context     AgentInputContext `json:"context"`
}

type AgentInputContext struct {
	Symbol          string           `json:"symbol"`
	SymbolName      string           `json:"symbolName"`
	EventDate       string           `json:"eventDate"`
	EventSnapshot   *EventSnapshot   `json:"eventSnapshot,omitempty"`
	PreviousResults []PreviousResult `json:"previousResults"`
}

type AgentOutput struct {
	Summary    string                 `json:"summary"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Confidence *float64               `json:"confidence,omitempty"`
}

type PreviousResult struct {
	BlockName string                 `json:"blockName"`
	Summary   string                 `json:"summary"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type EventSnapshot struct {
	High       float64            `json:"high"`
	Low        float64            `json:"low"`
	Close      float64            `json:"close"`
	Volume     float64            `json:"volume"`
	TradeValue float64            `json:"tradeValue"`
	PreMa      map[int]float64    `json:"preMa"`
}

type PipelineExecutionContext struct {
	Symbol          string           `json:"symbol"`
	PreviousResults []PreviousResult `json:"previousResults"`
}
```

- [ ] 1.10 `src/entities/pipeline/model/types.ts`에 프론트엔드 Pipeline 타입을 정의한다 (Go API 응답 기반).

```typescript
// src/entities/pipeline/model/types.ts
// Frontend types — mirrors Go backend domain models (API response shape)

export interface Pipeline {
  id: string;
  userId: string;
  name: string;
  description: string;
  successScript: string | null;
  failureScript: string | null;
  isPublic: boolean;
  createdAt: string;
  updatedAt: string;
  stages?: Stage[];
  monitors?: MonitorBlock[];
  priceAlerts?: PriceAlert[];
}

export interface Stage {
  id: string;
  pipelineId: string;
  section: "analysis" | "monitoring" | "judgment";
  order: number;
  createdAt: string;
  blocks?: AgentBlock[];
}

export interface PipelineJob {
  id: string;
  pipelineId: string;
  symbol: string;
  status: "PENDING" | "RUNNING" | "COMPLETED" | "FAILED";
  result?: Record<string, unknown>;
  error?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
}

export interface AgentInput {
  instruction: string;
  context: {
    symbol: string;
    symbolName: string;
    eventDate: string;
    eventSnapshot?: EventSnapshot;
    previousResults: PreviousResult[];
  };
}

export interface AgentOutput {
  summary: string;
  data?: Record<string, unknown>;
  confidence?: number;
}

export interface PreviousResult {
  blockName: string;
  summary: string;
  data?: Record<string, unknown>;
}

export interface EventSnapshot {
  high: number;
  low: number;
  close: number;
  volume: number;
  tradeValue: number;
  preMa: Record<number, number>;
}

export interface PipelineExecutionContext {
  symbol: string;
  previousResults: PreviousResult[];
}

// Re-export block types for convenience
export type { AgentBlock, MonitorBlock, PriceAlert } from "@/entities/agent-block/model/types";
```

- [ ] 1.11 `src/entities/agent-block/model/types.ts`에 프론트엔드 AgentBlock 타입을 정의한다.

```typescript
// src/entities/agent-block/model/types.ts
// Frontend types — mirrors Go backend domain models

export interface AgentBlock {
  id: string;
  userId: string;
  stageId?: string;
  name: string;
  objective: string;
  inputDesc: string;
  tools: string[];
  outputFormat: string;
  constraints: string | null;
  examples: string | null;
  instruction: string;
  systemPrompt: string | null;
  allowedTools: string[];
  outputSchema?: Record<string, unknown>;
  isPublic: boolean;
  isTemplate: boolean;
  templateId?: string;
  createdAt: string;
  updatedAt: string;
  monitorBlock?: MonitorBlock;
}

export interface MonitorBlock {
  id: string;
  pipelineId: string;
  blockId: string;
  cron: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
  block?: AgentBlock;
}

export interface PriceAlert {
  id: string;
  pipelineId?: string;
  caseId?: string;
  condition: string;
  label: string;
  triggered: boolean;
  triggeredAt?: string;
  createdAt: string;
}

/** Template: reusable block in palette (isTemplate=true). Drag creates a copy. */
export interface AgentBlockTemplate {
  id: string;
  name: string;
  objective: string;
  inputDesc: string;
  tools: string[];
  outputFormat: string;
  constraints: string | null;
  examples: string | null;
  isPublic: boolean;
}
```

- [ ] 1.12 sqlc 쿼리를 생성하고 Go 코드를 생성한다.

```bash
cd backend && sqlc generate
```

- [ ] 1.13 도메인 모델 단위 테스트를 작성한다 — 상수 값, 기본 구조체 생성 검증.

```bash
cd backend && go test ./internal/domain/...
```

- [ ] 1.14 변경사항을 커밋한다.

```bash
git add backend/db/migrations/003_pipeline.sql backend/db/queries/ backend/internal/domain/ src/entities/pipeline/model/types.ts src/entities/agent-block/model/types.ts
git commit -m "feat(db): add Pipeline, Stage, AgentBlock, MonitorBlock, PriceAlert models with Go domain types and sqlc queries"
```

---

## Task 2: Pipeline CRUD API — Go Repository + Service + Gin Handlers

파이프라인의 생성/조회/수정/삭제를 Go Repository 패턴으로 구현하고, Gin Handler는 thin controller로 구성한다.

**Files:**
- Create: `backend/internal/repository/pipeline_repo.go`
- Create: `backend/internal/service/pipeline_service.go`
- Create: `backend/internal/handler/pipeline_handler.go`
- Create: `backend/db/queries/pipelines.sql`
- Test: `backend/internal/handler/pipeline_handler_test.go`

**Steps:**

- [ ] 2.1 TDD: Pipeline CRUD 테스트를 먼저 작성한다 — 파이프라인 생성(stages + blocks + monitors 포함), 목록 조회, 상세 조회, 수정, 삭제 시나리오.

- [ ] 2.2 `backend/db/queries/pipelines.sql`에 sqlc 쿼리를 정의한다.

```sql
-- backend/db/queries/pipelines.sql

-- name: ListPipelinesByUser :many
SELECT * FROM pipelines WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetPipelineByID :one
SELECT * FROM pipelines WHERE id = $1 AND user_id = $2;

-- name: CreatePipeline :one
INSERT INTO pipelines (user_id, name, description, success_script, failure_script, is_public)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdatePipeline :one
UPDATE pipelines
SET name = $3, description = $4, success_script = $5, failure_script = $6, is_public = $7, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeletePipeline :exec
DELETE FROM pipelines WHERE id = $1 AND user_id = $2;

-- name: CreateStage :one
INSERT INTO stages (pipeline_id, section, order_index)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListStagesByPipeline :many
SELECT * FROM stages WHERE pipeline_id = $1 ORDER BY order_index;

-- name: DeleteStagesByPipeline :exec
DELETE FROM stages WHERE pipeline_id = $1;

-- name: CreatePipelineJob :one
INSERT INTO pipeline_jobs (pipeline_id, symbol, status)
VALUES ($1, $2, 'PENDING')
RETURNING *;

-- name: GetPipelineJob :one
SELECT * FROM pipeline_jobs WHERE id = $1;

-- name: UpdatePipelineJobStatus :one
UPDATE pipeline_jobs
SET status = $2, result = $3, error = $4, started_at = $5, completed_at = $6
WHERE id = $1
RETURNING *;
```

- [ ] 2.3 `backend/internal/repository/pipeline_repo.go`에 데이터 접근 레이어를 구현한다.

```go
// backend/internal/repository/pipeline_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"superbear/backend/internal/domain"
	db "superbear/backend/internal/sqlc"
)

type PipelineRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPipelineRepository(pool *pgxpool.Pool) *PipelineRepository {
	return &PipelineRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (r *PipelineRepository) FindMany(ctx context.Context, userID uuid.UUID) ([]domain.Pipeline, error) {
	// List pipelines, then load relations for each
	rows, err := r.queries.ListPipelinesByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	pipelines := make([]domain.Pipeline, 0, len(rows))
	for _, row := range rows {
		p := toDomainPipeline(row)
		pipelines = append(pipelines, p)
	}
	return pipelines, nil
}

func (r *PipelineRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Pipeline, error) {
	row, err := r.queries.GetPipelineByID(ctx, db.GetPipelineByIDParams{ID: id, UserID: userID})
	if err != nil {
		return nil, err
	}
	p := toDomainPipeline(row)
	// Load stages with blocks
	stages, err := r.loadStagesWithBlocks(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Stages = stages
	return &p, nil
}

func (r *PipelineRepository) Create(ctx context.Context, p *domain.Pipeline) (*domain.Pipeline, error) {
	row, err := r.queries.CreatePipeline(ctx, db.CreatePipelineParams{
		UserID:        p.UserID,
		Name:          p.Name,
		Description:   p.Description,
		SuccessScript: p.SuccessScript,
		FailureScript: p.FailureScript,
		IsPublic:      p.IsPublic,
	})
	if err != nil {
		return nil, err
	}
	result := toDomainPipeline(row)
	return &result, nil
}

func (r *PipelineRepository) Update(ctx context.Context, id, userID uuid.UUID, p *domain.Pipeline) (*domain.Pipeline, error) {
	row, err := r.queries.UpdatePipeline(ctx, db.UpdatePipelineParams{
		ID:            id,
		UserID:        userID,
		Name:          p.Name,
		Description:   p.Description,
		SuccessScript: p.SuccessScript,
		FailureScript: p.FailureScript,
		IsPublic:      p.IsPublic,
	})
	if err != nil {
		return nil, err
	}
	result := toDomainPipeline(row)
	return &result, nil
}

func (r *PipelineRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.queries.DeletePipeline(ctx, db.DeletePipelineParams{ID: id, UserID: userID})
}
```

- [ ] 2.4 `backend/internal/service/pipeline_service.go`에 도메인 서비스를 구현한다 — `Create` (중첩 생성: stages -> blocks, monitors -> block), `List`, `GetByID`, `Update`, `Delete`, `StartExecution`. Repository를 호출하여 데이터 접근을 위임.

```go
// backend/internal/service/pipeline_service.go
package service

import (
	"context"

	"github.com/google/uuid"

	"superbear/backend/internal/domain"
	"superbear/backend/internal/repository"
)

type PipelineService struct {
	pipelineRepo *repository.PipelineRepository
	blockRepo    *repository.BlockRepository
}

func NewPipelineService(pr *repository.PipelineRepository, br *repository.BlockRepository) *PipelineService {
	return &PipelineService{pipelineRepo: pr, blockRepo: br}
}

func (s *PipelineService) Create(ctx context.Context, userID string, req *CreatePipelineRequest) (*domain.Pipeline, error) {
	uid, _ := uuid.Parse(userID)
	p := &domain.Pipeline{
		UserID:        uid,
		Name:          req.Name,
		Description:   req.Description,
		SuccessScript: req.SuccessScript,
		FailureScript: req.FailureScript,
		IsPublic:      req.IsPublic,
	}
	pipeline, err := s.pipelineRepo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	// Create nested stages with blocks
	for _, stageReq := range req.Stages {
		stage, err := s.pipelineRepo.CreateStage(ctx, pipeline.ID, stageReq.Section, stageReq.Order)
		if err != nil {
			return nil, err
		}
		for _, blockReq := range stageReq.Blocks {
			_, err := s.blockRepo.Create(ctx, uid, &stage.ID, &blockReq)
			if err != nil {
				return nil, err
			}
		}
	}
	// Reload with relations
	return s.pipelineRepo.FindByID(ctx, pipeline.ID, uid)
}

func (s *PipelineService) List(ctx context.Context, userID string) ([]domain.Pipeline, error) {
	uid, _ := uuid.Parse(userID)
	return s.pipelineRepo.FindMany(ctx, uid)
}

func (s *PipelineService) GetByID(ctx context.Context, userID, id string) (*domain.Pipeline, error) {
	uid, _ := uuid.Parse(userID)
	pid, _ := uuid.Parse(id)
	return s.pipelineRepo.FindByID(ctx, pid, uid)
}

func (s *PipelineService) Update(ctx context.Context, userID, id string, req *UpdatePipelineRequest) (*domain.Pipeline, error) {
	uid, _ := uuid.Parse(userID)
	pid, _ := uuid.Parse(id)
	p := &domain.Pipeline{
		Name:          req.Name,
		Description:   req.Description,
		SuccessScript: req.SuccessScript,
		FailureScript: req.FailureScript,
		IsPublic:      req.IsPublic,
	}
	// Delete existing stages and recreate (full replace strategy)
	if err := s.pipelineRepo.DeleteStages(ctx, pid); err != nil {
		return nil, err
	}
	updated, err := s.pipelineRepo.Update(ctx, pid, uid, p)
	if err != nil {
		return nil, err
	}
	for _, stageReq := range req.Stages {
		stage, err := s.pipelineRepo.CreateStage(ctx, pid, stageReq.Section, stageReq.Order)
		if err != nil {
			return nil, err
		}
		for _, blockReq := range stageReq.Blocks {
			_, err := s.blockRepo.Create(ctx, uid, &stage.ID, &blockReq)
			if err != nil {
				return nil, err
			}
		}
	}
	_ = updated
	return s.pipelineRepo.FindByID(ctx, pid, uid)
}

func (s *PipelineService) Delete(ctx context.Context, userID, id string) error {
	uid, _ := uuid.Parse(userID)
	pid, _ := uuid.Parse(id)
	return s.pipelineRepo.Delete(ctx, pid, uid)
}

func (s *PipelineService) StartExecution(ctx context.Context, userID, pipelineID, symbol string) (*domain.PipelineJob, error) {
	pid, _ := uuid.Parse(pipelineID)
	return s.pipelineRepo.CreateJob(ctx, pid, symbol)
}
```

- [ ] 2.5 `backend/internal/handler/pipeline_handler.go` — **thin Gin handler**. 요청 파싱 + 인증 확인 후 `PipelineService`에 위임. 비즈니스 로직 없음.

```go
// backend/internal/handler/pipeline_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"superbear/backend/internal/service"
)

type PipelineHandler struct {
	svc *service.PipelineService
}

func NewPipelineHandler(svc *service.PipelineService) *PipelineHandler {
	return &PipelineHandler{svc: svc}
}

// POST /api/pipelines
func (h *PipelineHandler) Create(c *gin.Context) {
	var req service.CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("userId")
	pipeline, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pipeline)
}

// GET /api/pipelines
func (h *PipelineHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	pipelines, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pipelines)
}

// GET /api/pipelines/:id
func (h *PipelineHandler) GetByID(c *gin.Context) {
	userID := c.GetString("userId")
	id := c.Param("id")
	pipeline, err := h.svc.GetByID(c.Request.Context(), userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
		return
	}
	c.JSON(http.StatusOK, pipeline)
}

// PUT /api/pipelines/:id
func (h *PipelineHandler) Update(c *gin.Context) {
	var req service.UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("userId")
	id := c.Param("id")
	pipeline, err := h.svc.Update(c.Request.Context(), userID, id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pipeline)
}

// DELETE /api/pipelines/:id
func (h *PipelineHandler) Delete(c *gin.Context) {
	userID := c.GetString("userId")
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// POST /api/pipelines/:id/execute
func (h *PipelineHandler) Execute(c *gin.Context) {
	var req struct {
		Symbol string `json:"symbol" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("userId")
	id := c.Param("id")
	job, err := h.svc.StartExecution(c.Request.Context(), userID, id, req.Symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"jobId": job.ID})
}

// GET /api/pipelines/:id/jobs/:jobId
func (h *PipelineHandler) GetJob(c *gin.Context) {
	jobID := c.Param("jobId")
	job, err := h.svc.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

// RegisterRoutes registers all pipeline routes on the Gin router group.
func (h *PipelineHandler) RegisterRoutes(rg *gin.RouterGroup) {
	pipelines := rg.Group("/pipelines")
	{
		pipelines.GET("", h.List)
		pipelines.POST("", h.Create)
		pipelines.GET("/:id", h.GetByID)
		pipelines.PUT("/:id", h.Update)
		pipelines.DELETE("/:id", h.Delete)
		pipelines.POST("/:id/execute", h.Execute)
		pipelines.GET("/:id/jobs/:jobId", h.GetJob)
	}
}
```

- [ ] 2.6 Request/Response DTO를 service 패키지에 정의한다.

```go
// backend/internal/service/pipeline_dto.go
package service

type CreatePipelineRequest struct {
	Name          string              `json:"name" binding:"required,min=1,max=200"`
	Description   string              `json:"description"`
	Stages        []StageRequest      `json:"stages"`
	Monitors      []MonitorRequest    `json:"monitors"`
	SuccessScript *string             `json:"successScript"`
	FailureScript *string             `json:"failureScript"`
	PriceAlerts   []PriceAlertRequest `json:"priceAlerts"`
	IsPublic      bool                `json:"isPublic"`
}

type UpdatePipelineRequest struct {
	Name          string              `json:"name" binding:"required,min=1,max=200"`
	Description   string              `json:"description"`
	Stages        []StageRequest      `json:"stages"`
	Monitors      []MonitorRequest    `json:"monitors"`
	SuccessScript *string             `json:"successScript"`
	FailureScript *string             `json:"failureScript"`
	PriceAlerts   []PriceAlertRequest `json:"priceAlerts"`
	IsPublic      bool                `json:"isPublic"`
}

type StageRequest struct {
	Section string              `json:"section" binding:"required"`
	Order   int                 `json:"order"`
	Blocks  []AgentBlockRequest `json:"blocks" binding:"required,min=1"`
}

type AgentBlockRequest struct {
	Name         string   `json:"name" binding:"required,min=1"`
	Objective    string   `json:"objective"`
	InputDesc    string   `json:"inputDesc"`
	Tools        []string `json:"tools"`
	OutputFormat string   `json:"outputFormat"`
	Constraints  *string  `json:"constraints"`
	Examples     *string  `json:"examples"`
}

type MonitorRequest struct {
	Block   AgentBlockRequest `json:"block"`
	Cron    string            `json:"cron" binding:"required,min=1"`
	Enabled bool              `json:"enabled"`
}

type PriceAlertRequest struct {
	Condition string `json:"condition" binding:"required,min=1"`
	Label     string `json:"label" binding:"required,min=1"`
}
```

- [ ] 2.7 테스트 실행 및 커밋.

```bash
cd backend && go test ./internal/handler/... ./internal/service/... ./internal/repository/...
git add backend/internal/handler/pipeline_handler.go backend/internal/service/ backend/internal/repository/pipeline_repo.go backend/db/queries/pipelines.sql
git commit -m "feat(api): implement Pipeline CRUD with Go repository pattern and Gin thin handlers"
```

---

## Task 3: AgentBlock CRUD API — Go 독립 블록 + Template 패턴

독립 에이전트 블록의 CRUD API를 Go로 구현한다. 팔레트 노드는 AgentBlockTemplate(is_template=true)이며, 캔버스에 드래그하면 template_id를 참조하는 복사본을 생성한다.

**Files:**
- Create: `backend/internal/repository/block_repo.go`
- Create: `backend/internal/service/block_service.go`
- Create: `backend/internal/handler/block_handler.go`
- Create: `backend/db/queries/blocks.sql`
- Test: `backend/internal/handler/block_handler_test.go`

**Steps:**

- [ ] 3.1 TDD: AgentBlock CRUD 테스트를 먼저 작성한다 — AgentBlockPrompt 전체 필드로 블록 생성, template 블록 생성, template에서 복사본 생성, 목록 조회, 수정, 삭제.

- [ ] 3.2 `backend/db/queries/blocks.sql`에 sqlc 쿼리를 정의한다.

```sql
-- backend/db/queries/blocks.sql

-- name: ListBlocksByUser :many
SELECT * FROM agent_blocks
WHERE user_id = $1 AND stage_id IS NULL
ORDER BY created_at DESC;

-- name: ListTemplates :many
SELECT * FROM agent_blocks
WHERE (user_id = $1 OR is_public = true) AND is_template = true
ORDER BY name;

-- name: GetBlockByID :one
SELECT * FROM agent_blocks WHERE id = $1;

-- name: CreateBlock :one
INSERT INTO agent_blocks (
  user_id, stage_id, name, objective, input_desc, tools, output_format,
  constraints, examples, instruction, system_prompt, allowed_tools,
  output_schema, is_public, is_template, template_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
RETURNING *;

-- name: UpdateBlock :one
UPDATE agent_blocks
SET name = $2, objective = $3, input_desc = $4, tools = $5, output_format = $6,
    constraints = $7, examples = $8, instruction = $9, system_prompt = $10,
    allowed_tools = $11, output_schema = $12, is_public = $13, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteBlock :exec
DELETE FROM agent_blocks WHERE id = $1 AND user_id = $2;

-- name: ListBlocksByStage :many
SELECT * FROM agent_blocks WHERE stage_id = $1 ORDER BY created_at;

-- name: CreateMonitorBlock :one
INSERT INTO monitor_blocks (pipeline_id, block_id, cron, enabled)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListMonitorsByPipeline :many
SELECT mb.*, ab.name as block_name
FROM monitor_blocks mb
JOIN agent_blocks ab ON ab.id = mb.block_id
WHERE mb.pipeline_id = $1;

-- name: UpdateMonitorBlock :one
UPDATE monitor_blocks SET cron = $2, enabled = $3, updated_at = now() WHERE id = $1
RETURNING *;

-- name: DeleteMonitorBlock :exec
DELETE FROM monitor_blocks WHERE id = $1;

-- name: CreatePriceAlert :one
INSERT INTO price_alerts (pipeline_id, case_id, condition, label)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListPriceAlertsByPipeline :many
SELECT * FROM price_alerts WHERE pipeline_id = $1 ORDER BY created_at;

-- name: DeletePriceAlert :exec
DELETE FROM price_alerts WHERE id = $1;
```

- [ ] 3.3 `backend/internal/repository/block_repo.go`에 데이터 접근 레이어를 구현한다.

```go
// backend/internal/repository/block_repo.go
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"superbear/backend/internal/domain"
	db "superbear/backend/internal/sqlc"
)

type BlockRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{pool: pool, queries: db.New(pool)}
}

func (r *BlockRepository) FindMany(ctx context.Context, userID uuid.UUID) ([]domain.AgentBlock, error) {
	rows, err := r.queries.ListBlocksByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDomainBlocks(rows), nil
}

func (r *BlockRepository) FindTemplates(ctx context.Context, userID uuid.UUID) ([]domain.AgentBlock, error) {
	rows, err := r.queries.ListTemplates(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toDomainBlocks(rows), nil
}

func (r *BlockRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AgentBlock, error) {
	row, err := r.queries.GetBlockByID(ctx, id)
	if err != nil {
		return nil, err
	}
	block := toDomainBlock(row)
	return &block, nil
}

func (r *BlockRepository) Create(ctx context.Context, userID uuid.UUID, stageID *uuid.UUID, req *service.AgentBlockRequest) (*domain.AgentBlock, error) {
	row, err := r.queries.CreateBlock(ctx, db.CreateBlockParams{
		UserID:      userID,
		StageID:     stageID,
		Name:        req.Name,
		Objective:   req.Objective,
		InputDesc:   req.InputDesc,
		Tools:       req.Tools,
		OutputFormat: req.OutputFormat,
		Constraints: req.Constraints,
		Examples:    req.Examples,
	})
	if err != nil {
		return nil, err
	}
	block := toDomainBlock(row)
	return &block, nil
}

func (r *BlockRepository) CreateFromTemplate(ctx context.Context, templateID, userID, stageID uuid.UUID) (*domain.AgentBlock, error) {
	// Fetch template, then create a copy with template_id reference
	tmpl, err := r.queries.GetBlockByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateBlock(ctx, db.CreateBlockParams{
		UserID:      userID,
		StageID:     &stageID,
		Name:        tmpl.Name,
		Objective:   tmpl.Objective,
		InputDesc:   tmpl.InputDesc,
		Tools:       tmpl.Tools,
		OutputFormat: tmpl.OutputFormat,
		Constraints: tmpl.Constraints,
		Examples:    tmpl.Examples,
		IsTemplate:  false,
		TemplateID:  &templateID,
	})
	if err != nil {
		return nil, err
	}
	block := toDomainBlock(row)
	return &block, nil
}

func (r *BlockRepository) Update(ctx context.Context, id uuid.UUID, req *service.UpdateBlockRequest) (*domain.AgentBlock, error) {
	row, err := r.queries.UpdateBlock(ctx, db.UpdateBlockParams{
		ID:          id,
		Name:        req.Name,
		Objective:   req.Objective,
		InputDesc:   req.InputDesc,
		Tools:       req.Tools,
		OutputFormat: req.OutputFormat,
		Constraints: req.Constraints,
		Examples:    req.Examples,
		IsPublic:    req.IsPublic,
	})
	if err != nil {
		return nil, err
	}
	block := toDomainBlock(row)
	return &block, nil
}

func (r *BlockRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.queries.DeleteBlock(ctx, db.DeleteBlockParams{ID: id, UserID: userID})
}
```

- [ ] 3.4 `backend/internal/service/block_service.go`에 도메인 서비스를 구현한다 — `CreateBlock`, `CreateTemplate`, `CopyFromTemplate`, `ListBlocks`, `GetBlock`, `UpdateBlock`, `DeleteBlock`.

```go
// backend/internal/service/block_service.go
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"superbear/backend/internal/domain"
	"superbear/backend/internal/repository"
)

type BlockService struct {
	repo *repository.BlockRepository
}

func NewBlockService(repo *repository.BlockRepository) *BlockService {
	return &BlockService{repo: repo}
}

func (s *BlockService) ListBlocks(ctx context.Context, userID string) ([]domain.AgentBlock, error) {
	uid, _ := uuid.Parse(userID)
	return s.repo.FindMany(ctx, uid)
}

func (s *BlockService) ListTemplates(ctx context.Context, userID string) ([]domain.AgentBlock, error) {
	uid, _ := uuid.Parse(userID)
	return s.repo.FindTemplates(ctx, uid)
}

func (s *BlockService) GetBlock(ctx context.Context, id string) (*domain.AgentBlock, error) {
	bid, _ := uuid.Parse(id)
	return s.repo.FindByID(ctx, bid)
}

func (s *BlockService) CreateBlock(ctx context.Context, userID string, req *CreateBlockRequest) (*domain.AgentBlock, error) {
	uid, _ := uuid.Parse(userID)
	return s.repo.Create(ctx, uid, nil, &req.AgentBlockRequest)
}

func (s *BlockService) CreateTemplate(ctx context.Context, userID string, req *CreateBlockRequest) (*domain.AgentBlock, error) {
	uid, _ := uuid.Parse(userID)
	req.IsTemplate = true
	return s.repo.Create(ctx, uid, nil, &req.AgentBlockRequest)
}

func (s *BlockService) CopyFromTemplate(ctx context.Context, userID, templateID, stageID string) (*domain.AgentBlock, error) {
	uid, _ := uuid.Parse(userID)
	tid, _ := uuid.Parse(templateID)
	sid, _ := uuid.Parse(stageID)
	return s.repo.CreateFromTemplate(ctx, tid, uid, sid)
}

func (s *BlockService) UpdateBlock(ctx context.Context, userID, id string, req *UpdateBlockRequest) (*domain.AgentBlock, error) {
	bid, _ := uuid.Parse(id)
	// Verify ownership
	block, err := s.repo.FindByID(ctx, bid)
	if err != nil {
		return nil, err
	}
	if block.UserID.String() != userID {
		return nil, fmt.Errorf("forbidden: not the owner of this block")
	}
	return s.repo.Update(ctx, bid, req)
}

func (s *BlockService) DeleteBlock(ctx context.Context, userID, id string) error {
	uid, _ := uuid.Parse(userID)
	bid, _ := uuid.Parse(id)
	return s.repo.Delete(ctx, bid, uid)
}
```

- [ ] 3.5 `backend/internal/handler/block_handler.go` — **thin Gin handler**.

```go
// backend/internal/handler/block_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"superbear/backend/internal/service"
)

type BlockHandler struct {
	svc *service.BlockService
}

func NewBlockHandler(svc *service.BlockService) *BlockHandler {
	return &BlockHandler{svc: svc}
}

// GET /api/blocks
func (h *BlockHandler) List(c *gin.Context) {
	userID := c.GetString("userId")
	blocks, err := h.svc.ListBlocks(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, blocks)
}

// GET /api/blocks/templates
func (h *BlockHandler) ListTemplates(c *gin.Context) {
	userID := c.GetString("userId")
	templates, err := h.svc.ListTemplates(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, templates)
}

// GET /api/blocks/:id
func (h *BlockHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	block, err := h.svc.GetBlock(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "block not found"})
		return
	}
	c.JSON(http.StatusOK, block)
}

// POST /api/blocks
func (h *BlockHandler) Create(c *gin.Context) {
	var req service.CreateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("userId")
	block, err := h.svc.CreateBlock(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, block)
}

// POST /api/blocks/from-template
func (h *BlockHandler) CopyFromTemplate(c *gin.Context) {
	var req struct {
		TemplateID string `json:"templateId" binding:"required"`
		StageID    string `json:"stageId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("userId")
	block, err := h.svc.CopyFromTemplate(c.Request.Context(), userID, req.TemplateID, req.StageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, block)
}

// PUT /api/blocks/:id
func (h *BlockHandler) Update(c *gin.Context) {
	var req service.UpdateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetString("userId")
	id := c.Param("id")
	block, err := h.svc.UpdateBlock(c.Request.Context(), userID, id, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, block)
}

// DELETE /api/blocks/:id
func (h *BlockHandler) Delete(c *gin.Context) {
	userID := c.GetString("userId")
	id := c.Param("id")
	if err := h.svc.DeleteBlock(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// RegisterRoutes registers all block routes on the Gin router group.
func (h *BlockHandler) RegisterRoutes(rg *gin.RouterGroup) {
	blocks := rg.Group("/blocks")
	{
		blocks.GET("", h.List)
		blocks.GET("/templates", h.ListTemplates)
		blocks.GET("/:id", h.GetByID)
		blocks.POST("", h.Create)
		blocks.POST("/from-template", h.CopyFromTemplate)
		blocks.PUT("/:id", h.Update)
		blocks.DELETE("/:id", h.Delete)
	}
}
```

- [ ] 3.6 Block DTO를 service 패키지에 정의한다.

```go
// backend/internal/service/block_dto.go
package service

type CreateBlockRequest struct {
	AgentBlockRequest
	IsTemplate bool `json:"isTemplate"`
}

type UpdateBlockRequest struct {
	Name         string   `json:"name" binding:"required,min=1"`
	Objective    string   `json:"objective"`
	InputDesc    string   `json:"inputDesc"`
	Tools        []string `json:"tools"`
	OutputFormat string   `json:"outputFormat"`
	Constraints  *string  `json:"constraints"`
	Examples     *string  `json:"examples"`
	Instruction  string   `json:"instruction"`
	SystemPrompt *string  `json:"systemPrompt"`
	AllowedTools []string `json:"allowedTools"`
	IsPublic     bool     `json:"isPublic"`
}
```

- [ ] 3.7 테스트 실행 및 커밋.

```bash
cd backend && go test ./internal/handler/... ./internal/service/... ./internal/repository/...
git add backend/internal/handler/block_handler.go backend/internal/service/block_service.go backend/internal/service/block_dto.go backend/internal/repository/block_repo.go backend/db/queries/blocks.sql
git commit -m "feat(api): implement AgentBlock CRUD with Go template pattern and repository"
```

---

## Task 4: PipelineOrchestrator — Go 순차/병렬 에이전트 실행 엔진

파이프라인 실행 엔진을 Go로 구현한다. 분석 섹션의 Stage를 order 순으로 실행하되, 같은 order의 블록은 goroutine으로 병렬 실행한다.

**Files:**
- Create: `backend/internal/service/pipeline_orchestrator.go`
- Create: `backend/internal/agent/runner.go`
- Test: `backend/internal/service/pipeline_orchestrator_test.go`

> **DDD Note:** Domain types (AgentInput, AgentOutput, PipelineExecutionContext) live in `backend/internal/domain/input_output.go`. Orchestrator is a service-level use case that composes repository and agent layers.

**Steps:**

- [ ] 4.1 도메인 타입은 Task 1에서 `backend/internal/domain/input_output.go`에 이미 정의됨 (AgentInput, AgentOutput, PipelineExecutionContext, EventSnapshot, PreviousResult).

- [ ] 4.2 TDD: Orchestrator 테스트를 작성한다 — 3-stage 파이프라인(1순차 -> 3병렬 -> 1순차)에서 올바른 실행 순서와 데이터 전달을 검증. AgentRunner를 모킹.

- [ ] 4.3 `backend/internal/agent/runner.go`에 Google ADK Go 에이전트 실행 래퍼를 구현한다 — AgentInput을 받아 ADK 에이전트를 호출하고 AgentOutput을 반환.

```go
// backend/internal/agent/runner.go
package agent

import (
	"context"
	"fmt"
	"log/slog"

	"superbear/backend/internal/domain"
)

// Runner defines the interface for agent execution (mockable for tests).
type Runner interface {
	Execute(ctx context.Context, block *domain.AgentBlock, input *domain.AgentInput) (*domain.AgentOutput, error)
}

// ADKRunner implements Runner using Google ADK Go SDK.
type ADKRunner struct {
	// ADK client configuration
}

func NewADKRunner() *ADKRunner {
	return &ADKRunner{}
}

func (r *ADKRunner) Execute(ctx context.Context, block *domain.AgentBlock, input *domain.AgentInput) (*domain.AgentOutput, error) {
	slog.Info("AgentRunner: executing block", "blockId", block.ID, "blockName", block.Name)

	// 1. block의 AgentBlockPrompt 필드로 시스템 프롬프트 구성
	systemPrompt := buildSystemPrompt(block)

	// 2. input.Context를 사용자 메시지에 포함
	userMessage := buildUserMessage(input)

	// 3. block.Tools로 허용 도구 제한
	tools := block.Tools
	if len(tools) == 0 {
		tools = block.AllowedTools
	}

	// 4. ADK 에이전트 호출
	_ = systemPrompt
	_ = userMessage
	_ = tools
	// TODO: Integrate with Google ADK Go SDK
	// response, err := adkClient.Generate(ctx, systemPrompt, userMessage, tools)

	// 5. 응답을 AgentOutput으로 파싱
	return &domain.AgentOutput{
		Summary: fmt.Sprintf("Result from %s", block.Name),
	}, nil
}

func buildSystemPrompt(block *domain.AgentBlock) string {
	prompt := fmt.Sprintf("You are an agent named '%s'.\n", block.Name)
	if block.Objective != "" {
		prompt += fmt.Sprintf("Objective: %s\n", block.Objective)
	}
	if block.OutputFormat != "" {
		prompt += fmt.Sprintf("Output Format: %s\n", block.OutputFormat)
	}
	if block.Constraints != nil {
		prompt += fmt.Sprintf("Constraints: %s\n", *block.Constraints)
	}
	if block.Examples != nil {
		prompt += fmt.Sprintf("Examples: %s\n", *block.Examples)
	}
	return prompt
}

func buildUserMessage(input *domain.AgentInput) string {
	msg := input.Instruction + "\n"
	msg += fmt.Sprintf("Symbol: %s (%s)\n", input.Context.Symbol, input.Context.SymbolName)
	msg += fmt.Sprintf("Event Date: %s\n", input.Context.EventDate)
	if len(input.Context.PreviousResults) > 0 {
		msg += "Previous Results:\n"
		for _, pr := range input.Context.PreviousResults {
			msg += fmt.Sprintf("- %s: %s\n", pr.BlockName, pr.Summary)
		}
	}
	return msg
}
```

- [ ] 4.4 `backend/internal/service/pipeline_orchestrator.go`에 PipelineOrchestrator를 구현한다.

```go
// backend/internal/service/pipeline_orchestrator.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"superbear/backend/internal/agent"
	"superbear/backend/internal/domain"
)

type PipelineOrchestrator struct {
	agentRunner agent.Runner
}

func NewPipelineOrchestrator(runner agent.Runner) *PipelineOrchestrator {
	return &PipelineOrchestrator{agentRunner: runner}
}

func (o *PipelineOrchestrator) Execute(ctx context.Context, pipeline *domain.Pipeline, symbol string) (*domain.PipelineExecutionContext, error) {
	execCtx := &domain.PipelineExecutionContext{
		Symbol:          symbol,
		PreviousResults: make([]domain.PreviousResult, 0),
	}

	// Group stages by order_index
	stageGroups := groupStagesByOrder(pipeline.Stages)
	sortedOrders := sortedKeys(stageGroups)

	for _, order := range sortedOrders {
		stages := stageGroups[order]
		blocks := collectBlocks(stages)

		if len(blocks) == 0 {
			continue
		}

		// Same order blocks execute in parallel using goroutines
		type blockResult struct {
			index  int
			output *domain.AgentOutput
			err    error
		}

		results := make([]blockResult, len(blocks))
		var wg sync.WaitGroup

		for i, block := range blocks {
			wg.Add(1)
			go func(idx int, b domain.AgentBlock) {
				defer wg.Done()
				input := o.buildInput(&b, execCtx)
				output, err := o.agentRunner.Execute(ctx, &b, input)
				results[idx] = blockResult{index: idx, output: output, err: err}
			}(i, block)
		}

		wg.Wait()

		// Collect results into context.PreviousResults (for next stage input)
		for i, r := range results {
			if r.err != nil {
				slog.Error("AgentBlock execution failed",
					"blockName", blocks[i].Name,
					"error", r.err,
				)
				continue
			}
			if r.output != nil {
				execCtx.PreviousResults = append(execCtx.PreviousResults, domain.PreviousResult{
					BlockName: blocks[i].Name,
					Summary:   r.output.Summary,
					Data:      r.output.Data,
				})
			}
		}
	}

	return execCtx, nil
}

func (o *PipelineOrchestrator) buildInput(block *domain.AgentBlock, ctx *domain.PipelineExecutionContext) *domain.AgentInput {
	instruction := block.Instruction
	if instruction == "" {
		instruction = block.Objective
	}
	return &domain.AgentInput{
		Instruction: instruction,
		Context: domain.AgentInputContext{
			Symbol:          ctx.Symbol,
			PreviousResults: ctx.PreviousResults,
		},
	}
}

// groupStagesByOrder groups stages by their OrderIndex.
func groupStagesByOrder(stages []domain.Stage) map[int][]domain.Stage {
	groups := make(map[int][]domain.Stage)
	for _, s := range stages {
		groups[s.OrderIndex] = append(groups[s.OrderIndex], s)
	}
	return groups
}

// sortedKeys returns sorted keys of the map.
func sortedKeys(m map[int][]domain.Stage) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// collectBlocks flattens all blocks from a slice of stages.
func collectBlocks(stages []domain.Stage) []domain.AgentBlock {
	var blocks []domain.AgentBlock
	for _, s := range stages {
		blocks = append(blocks, s.Blocks...)
	}
	return blocks
}
```

- [ ] 4.5 Orchestrator를 PipelineService의 StartExecution에 연결한다 — PipelineJob을 생성하고 goroutine으로 Orchestrator를 비동기 실행. 즉시 jobId를 반환.

```go
// Add to pipeline_service.go

func (s *PipelineService) ExecuteAsync(ctx context.Context, userID, pipelineID, symbol string) (*domain.PipelineJob, error) {
	uid, _ := uuid.Parse(userID)
	pid, _ := uuid.Parse(pipelineID)

	// Create job record
	job, err := s.pipelineRepo.CreateJob(ctx, pid, symbol)
	if err != nil {
		return nil, err
	}

	// Load full pipeline with relations
	pipeline, err := s.pipelineRepo.FindByID(ctx, pid, uid)
	if err != nil {
		return nil, err
	}

	// Execute asynchronously
	go func() {
		bgCtx := context.Background()
		s.pipelineRepo.UpdateJobStatus(bgCtx, job.ID, domain.JobStatusRunning, nil, nil)

		result, err := s.orchestrator.Execute(bgCtx, pipeline, symbol)
		if err != nil {
			errStr := err.Error()
			s.pipelineRepo.UpdateJobStatus(bgCtx, job.ID, domain.JobStatusFailed, nil, &errStr)
			return
		}

		s.pipelineRepo.UpdateJobStatus(bgCtx, job.ID, domain.JobStatusCompleted, result, nil)
	}()

	return job, nil
}

func (s *PipelineService) GetJob(ctx context.Context, jobID string) (*domain.PipelineJob, error) {
	jid, _ := uuid.Parse(jobID)
	return s.pipelineRepo.GetJob(ctx, jid)
}
```

- [ ] 4.6 실행 완료 후 케이스 자동 생성 로직 추가 — Orchestrator 실행 완료 시 Case 레코드와 초기 TimelineEvent(PIPELINE_RESULT)를 생성.

- [ ] 4.7 테스트 실행 및 커밋.

```bash
cd backend && go test ./internal/service/... ./internal/agent/...
git add backend/internal/service/pipeline_orchestrator.go backend/internal/agent/runner.go
git commit -m "feat(pipeline): implement Go PipelineOrchestrator with goroutine-based parallel agent execution"
```

---

## Task 5: MonitorBlock 스케줄링 — Plan 6로 연기 (Go cron worker)

> **Deferred:** MonitorBlock의 cron 기반 반복 실행은 Plan 6에서 Go 기반 작업 큐(Asynq 또는 River)로 구현한다. 싱글턴 cron 스케줄러는 서버 재시작 시 상태 유실, 멀티 프로세스 환경 미지원 등의 문제가 있으므로 Go job queue가 적절하다.
>
> 이 Task에서는 MonitorBlock SQL 스키마(Task 1에서 정의)와 CRUD만 존재하며, 실제 스케줄링 실행은 Plan 6에서 구현한다.

---

## Task 6: Pipeline Builder 페이지 레이아웃 — Topbar + 3-Section Canvas + Left Palette

파이프라인 빌더 페이지의 기본 레이아웃을 구현한다.

**Files:**
- Create: `src/app/pipeline/page.tsx` (page entry point only)
- Create: `src/app/pipeline/layout.tsx`
- Create: `src/features/pipeline-builder/ui/PipelineTopbar.tsx`
- Create: `src/features/pipeline-builder/ui/NodePalette.tsx`
- Create: `src/features/pipeline-builder/ui/PipelineCanvas.tsx`
- Create: `src/features/pipeline-builder/ui/sections/AnalysisSection.tsx`
- Create: `src/features/pipeline-builder/ui/sections/MonitoringSection.tsx`
- Create: `src/features/pipeline-builder/ui/sections/JudgmentSection.tsx`
- Test: `src/features/pipeline-builder/__tests__/PipelineCanvas.test.tsx`

**Steps:**

- [ ] 6.1 TDD: PipelineCanvas 렌더링 테스트를 작성한다 — 3개 섹션(분석/모니터링/판단)이 렌더링되고, Topbar에 종목 선택기와 Register & Run 버튼이 표시되는지 검증.

- [ ] 6.2 `src/features/pipeline-builder/ui/PipelineTopbar.tsx`를 구현한다 — 종목 선택기(1개 종목, 검색 + 선택), 파이프라인 드롭다운(기존 파이프라인 불러오기), Register & Run 버튼.

```typescript
interface PipelineTopbarProps {
  selectedSymbol: string | null;
  onSymbolChange: (symbol: string) => void;
  pipelines: Pipeline[];
  selectedPipelineId: string | null;
  onPipelineSelect: (id: string) => void;
  onRegisterAndRun: () => void;
  isRunning: boolean;
}
```

- [ ] 6.3 `src/features/pipeline-builder/ui/NodePalette.tsx`를 구현한다 — 왼쪽 사이드바에 3개 카테고리(Agent Nodes / DSL Nodes / Output Nodes)로 AgentBlockTemplate 목록 표시. 드래그 시작 이벤트 핸들러 포함. 드래그 시 template 복사본 생성.

```
┌─ Node Palette ──────┐
│ ▾ Agent Nodes        │
│   ▪ 뉴스 분석        │  ← AgentBlockTemplate (isTemplate=true)
│   ▪ 섹터 비교        │
│   ▪ 재무 분석        │
│   ▪ Custom Agent     │
│ ▾ DSL Nodes          │
│   ▪ DSL 조건 평가    │
│   ▪ 가격 알림        │
│ ▾ Output Nodes       │
│   ▪ 케이스 생성      │
│   ▪ 알림 전송        │
└─────────────────────┘
```

- [ ] 6.4 `src/features/pipeline-builder/ui/PipelineCanvas.tsx`를 구현한다 — 3개 섹션을 세로로 배치. 각 섹션은 드롭 존으로 동작하여 노드를 받을 수 있다.

```
┌─────────────────────────────────────────────────┐
│ [Analysis Section] — 1회 실행                     │
│  드래그된 에이전트 블록들이 여기에 배치              │
│  order가 같으면 가로 배치(병렬), 다르면 세로(순차)    │
├─────────────────────────────────────────────────┤
│ [Monitoring Section] — cron 반복                  │
│  모니터링 블록 + cron 설정 카드들                    │
├─────────────────────────────────────────────────┤
│ [Judgment Section] — DSL 경량 폴링                │
│  성공/실패 조건 DSL 에디터 + 가격 알림 목록          │
└─────────────────────────────────────────────────┘
```

- [ ] 6.5 각 섹션 컴포넌트(`AnalysisSection`, `MonitoringSection`, `JudgmentSection`)의 기본 구조를 구현한다.

- [ ] 6.6 페이지 전체 레이아웃 조립 — `/pipeline` 경로에서 Topbar + (좌측 Palette | 우측 Canvas) 레이아웃 완성.

- [ ] 6.7 테스트 실행 및 커밋.

```bash
npx jest src/features/pipeline-builder/__tests__/PipelineCanvas.test.tsx
git add src/app/pipeline/ src/features/pipeline-builder/ui/
git commit -m "feat(ui): implement Pipeline Builder page layout with 3-section canvas and node palette"
```

---

## Task 7: 드래그 앤 드롭 — 노드 배치 및 섹션 간 이동 + Pipeline Store (3 slices)

NodePalette에서 Canvas 섹션으로 AgentBlockTemplate를 드래그하여 복사본을 배치하고, 섹션 내에서 순서를 변경하는 인터랙션을 구현한다. Pipeline 스토어를 3개 슬라이스로 분할한다.

**Files:**
- Create: `src/features/pipeline-builder/model/analysis.slice.ts`
- Create: `src/features/pipeline-builder/model/monitor.slice.ts`
- Create: `src/features/pipeline-builder/model/judgment.slice.ts`
- Create: `src/features/pipeline-builder/model/pipeline.store.ts` (composed)
- Create: `src/features/pipeline-builder/lib/usePipelineDragDrop.ts`
- Create: `src/entities/agent-block/ui/BlockCard.tsx`
- Create: `src/entities/agent-block/ui/MonitorCard.tsx`
- Modify: `src/features/pipeline-builder/ui/PipelineCanvas.tsx`
- Modify: `src/features/pipeline-builder/ui/sections/AnalysisSection.tsx`
- Modify: `src/features/pipeline-builder/ui/sections/MonitoringSection.tsx`
- Test: `src/features/pipeline-builder/__tests__/usePipelineDragDrop.test.ts`

**Steps:**

- [ ] 7.1 `src/features/pipeline-builder/model/analysis.slice.ts`를 구현한다 — analysisStages 상태 관리.

```typescript
import type { StateCreator } from "zustand";

export interface AnalysisSlice {
  analysisStages: StageState[];  // [{order, blocks: BlockState[]}]
  addBlockToStage: (order: number, block: BlockState) => void;
  removeBlock: (blockId: string) => void;
  reorderStages: (fromOrder: number, toOrder: number) => void;
}

export const createAnalysisSlice: StateCreator<AnalysisSlice> = (set) => ({
  analysisStages: [],
  addBlockToStage: (order, block) => set((state) => { /* ... */ }),
  removeBlock: (blockId) => set((state) => { /* ... */ }),
  reorderStages: (from, to) => set((state) => { /* ... */ }),
});
```

- [ ] 7.2 `src/features/pipeline-builder/model/monitor.slice.ts`를 구현한다 — monitorBlocks 상태 관리.

```typescript
export interface MonitorSlice {
  monitorBlocks: MonitorBlockState[];
  addMonitorBlock: (block: MonitorBlockState) => void;
  removeMonitorBlock: (id: string) => void;
  updateMonitorCron: (id: string, cron: string) => void;
}
```

- [ ] 7.3 `src/features/pipeline-builder/model/judgment.slice.ts`를 구현한다 — 성공/실패 스크립트 + 가격 알림 상태 관리.

```typescript
export interface JudgmentSlice {
  successScript: string;
  failureScript: string;
  priceAlerts: PriceAlertState[];
  setSuccessScript: (script: string) => void;
  setFailureScript: (script: string) => void;
  addPriceAlert: (alert: PriceAlertState) => void;
  removePriceAlert: (id: string) => void;
}
```

- [ ] 7.4 `src/features/pipeline-builder/model/pipeline.store.ts`에 3개 슬라이스를 조합한 스토어를 생성한다.

```typescript
import { create } from "zustand";
import { createAnalysisSlice, type AnalysisSlice } from "./analysis.slice";
import { createMonitorSlice, type MonitorSlice } from "./monitor.slice";
import { createJudgmentSlice, type JudgmentSlice } from "./judgment.slice";

type PipelineStore = AnalysisSlice & MonitorSlice & JudgmentSlice;

export const usePipelineStore = create<PipelineStore>()((...a) => ({
  ...createAnalysisSlice(...a),
  ...createMonitorSlice(...a),
  ...createJudgmentSlice(...a),
}));
```

- [ ] 7.5 TDD: 드래그 앤 드롭 훅 테스트를 작성한다 — 팔레트에서 분석 섹션으로 드롭(template -> 복사본 생성), 같은 order에 드롭(병렬), 다른 order에 드롭(순차), 모니터링 섹션으로 드롭.

- [ ] 7.6 `src/features/pipeline-builder/lib/usePipelineDragDrop.ts` 훅을 구현한다 — HTML5 Drag & Drop API 기반. 드래그 소스(팔레트 template), 드롭 타겟(섹션), 드롭 시 Go backend의 `POST /api/blocks/from-template`를 호출하여 복사본 생성.

- [ ] 7.7 `src/entities/agent-block/ui/BlockCard.tsx`를 구현한다 — 분석 섹션에 배치된 에이전트 블록 카드. 이름, 목표 요약, 도구 아이콘, 편집/삭제 버튼.

- [ ] 7.8 `src/entities/agent-block/ui/MonitorCard.tsx`를 구현한다 — 모니터링 섹션에 배치된 블록 카드. BlockCard + cron 스케줄 입력 + 활성화/비활성화 토글.

- [ ] 7.9 AnalysisSection에 순차/병렬 시각화를 구현한다 — 같은 order의 블록은 가로 배열, 화살표(->)로 order 간 순차 연결 표시.

- [ ] 7.10 테스트 실행 및 커밋.

```bash
npx jest src/features/pipeline-builder/__tests__/usePipelineDragDrop.test.ts
git add src/features/pipeline-builder/model/ src/features/pipeline-builder/lib/usePipelineDragDrop.ts src/entities/agent-block/ui/
git commit -m "feat(ui): implement pipeline store (3 slices) and drag-and-drop with template copy"
```

---

## Task 8: AgentBlockPrompt 구조화 에디터

에이전트 블록의 프롬프트를 구조화된 필드(name, objective, input_desc, tools, output_format, constraints, examples)로 편집하는 모달 에디터를 구현한다.

**Files:**
- Create: `src/features/agent-block-editor/ui/AgentBlockEditor.tsx`
- Create: `src/features/agent-block-editor/ui/ToolSelector.tsx`
- Create: `src/entities/agent-block/lib/agent-tools.ts`
- Test: `src/features/agent-block-editor/__tests__/AgentBlockEditor.test.tsx`

**Steps:**

- [ ] 8.1 TDD: AgentBlockEditor 테스트를 작성한다 — 모든 필드 렌더링, 필드 입력, tools 멀티셀렉트, 저장 버튼 클릭 시 올바른 AgentBlockPrompt 반환.

- [ ] 8.2 `src/entities/agent-block/lib/agent-tools.ts`에 사용 가능한 도구 목록을 상수로 정의한다.

```typescript
export const AGENT_TOOLS = [
  { name: "get_candles", category: "price", description: "캔들 데이터 조회" },
  { name: "get_price", category: "price", description: "현재가 조회" },
  { name: "scan_stocks", category: "price", description: "조건 기반 종목 스캐닝" },
  { name: "get_financials", category: "fundamental", description: "재무제표 조회" },
  { name: "get_disclosures", category: "fundamental", description: "공시 목록 조회" },
  { name: "get_valuation", category: "fundamental", description: "밸류에이션 지표" },
  { name: "search_news", category: "news", description: "뉴스 검색 및 분석" },
  { name: "get_sector_stocks", category: "sector", description: "동일 섹터 종목 목록" },
  { name: "compare_sector", category: "sector", description: "섹터 내 상대 비교" },
  { name: "get_fund_flow", category: "sector", description: "외국인/기관 매매 동향" },
  { name: "dsl_evaluate", category: "dsl", description: "DSL 표현식 평가" },
] as const;
```

- [ ] 8.3 `src/features/agent-block-editor/ui/ToolSelector.tsx`를 구현한다 — 카테고리별로 그룹화된 도구 멀티셀렉트. 체크박스 기반. "전체 허용" 옵션.

- [ ] 8.4 `src/features/agent-block-editor/ui/AgentBlockEditor.tsx`를 모달 컴포넌트로 구현한다.

```
┌─ Agent Block Editor ──────────────────────────┐
│ Name:       [뉴스 임팩트 분석              ]   │
│                                                │
│ Objective:                                     │
│ ┌──────────────────────────────────────────┐   │
│ │ 이 종목 관련 최근 30일 뉴스를 검색하고,  │   │
│ │ 가장 중요한 촉매를 식별한 뒤...          │   │
│ └──────────────────────────────────────────┘   │
│                                                │
│ Input Description:                             │
│ [종목 코드, 종목명, 이벤트 날짜            ]   │
│                                                │
│ Tools: ☑ search_news  ☑ get_sector_stocks      │
│        ☐ get_candles  ☐ get_financials  ...    │
│                                                │
│ Output Format:                                 │
│ ┌──────────────────────────────────────────┐   │
│ │ catalyst_type: 정책 | 실적 | M&A        │   │
│ │ impact_score: 1~10                      │   │
│ └──────────────────────────────────────────┘   │
│                                                │
│ Constraints (optional):                        │
│ [뉴스 원문에 없는 내용을 추측하지 말 것    ]   │
│                                                │
│ Examples (optional):                           │
│ ┌──────────────────────────────────────────┐   │
│ │                                          │   │
│ └──────────────────────────────────────────┘   │
│                                                │
│              [Cancel]  [Save Block]            │
└────────────────────────────────────────────────┘
```

- [ ] 8.5 BlockCard 클릭 시 AgentBlockEditor 모달이 열리도록 연결한다. 저장 시 Go backend의 `PUT /api/blocks/:id`를 호출.

- [ ] 8.6 테스트 실행 및 커밋.

```bash
npx jest src/features/agent-block-editor/__tests__/AgentBlockEditor.test.tsx
git add src/features/agent-block-editor/ src/entities/agent-block/lib/agent-tools.ts
git commit -m "feat(ui): implement AgentBlockPrompt structured editor modal with tool selector"
```

---

## Task 9: AI Generate — Go 서비스로 자연어 파이프라인 자동 생성

사용자가 자연어로 파이프라인을 설명하면 Go 백엔드의 AI 서비스가 전체 파이프라인 구조(분석 stages + 모니터링 + 판단 조건)를 자동 생성하는 기능을 구현한다.

**Files:**
- Create: `backend/internal/service/pipeline_generator.go`
- Create: `src/features/pipeline-generator/lib/pipeline-generator-api.ts` (Go backend 호출)
- Create: `src/features/pipeline-generator/ui/AIGenerateModal.tsx`
- Test: `backend/internal/service/pipeline_generator_test.go`
- Test: `src/features/pipeline-generator/__tests__/pipeline-generator.test.ts`

**Steps:**

- [ ] 9.1 TDD: Pipeline Generator 테스트를 작성한다 — 자연어 입력("2년 최대거래량 종목을 찾아서 뉴스, 재무, 섹터 분석 후 모니터링")에 대해 올바른 파이프라인 구조가 생성되는지 검증 (LLM 응답 모킹).

- [ ] 9.2 `backend/internal/service/pipeline_generator.go`에 Go 서비스로 자연어 -> 파이프라인 구조 변환 로직을 구현한다.

```go
// backend/internal/service/pipeline_generator.go
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

type PipelineGenerator struct {
	// LLM client (ADK Go or direct API)
}

func NewPipelineGenerator() *PipelineGenerator {
	return &PipelineGenerator{}
}

type GeneratedPipeline struct {
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Stages        []StageRequest      `json:"stages"`
	Monitors      []MonitorRequest    `json:"monitors"`
	SuccessScript *string             `json:"successScript"`
	FailureScript *string             `json:"failureScript"`
	PriceAlerts   []PriceAlertRequest `json:"priceAlerts"`
}

type GenerateRequest struct {
	Description string `json:"description" binding:"required,min=1"`
}

func (g *PipelineGenerator) Generate(ctx context.Context, description string) (*GeneratedPipeline, error) {
	slog.Info("PipelineGenerator: generating from description", "description", description)

	// 1. System prompt: pipeline structure (stages, monitors, judgment) specification
	systemPrompt := buildGeneratorSystemPrompt()

	// 2. User input: natural language description
	// 3. Call LLM via ADK Go SDK
	// 4. Parse LLM response into GeneratedPipeline struct
	// 5. Validate and return

	_ = systemPrompt
	// TODO: Integrate with Google ADK Go SDK for LLM call

	return &GeneratedPipeline{
		Name:        fmt.Sprintf("Generated: %s", truncate(description, 50)),
		Description: description,
	}, nil
}

func buildGeneratorSystemPrompt() string {
	return `You are a pipeline structure generator. Given a natural language description of a stock analysis workflow,
generate a structured pipeline with:
- stages: ordered analysis steps (each with agent blocks)
- monitors: cron-based monitoring blocks
- successScript/failureScript: DSL conditions
- priceAlerts: price condition alerts

Each agent block should have: name, objective, inputDesc, tools (from available tools), outputFormat.
Available tools: get_candles, get_price, scan_stocks, get_financials, get_disclosures, get_valuation, search_news, get_sector_stocks, compare_sector, get_fund_flow, dsl_evaluate.

Respond in JSON format matching the CreatePipelineRequest schema.`
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
```

- [ ] 9.3 Pipeline Generator를 PipelineHandler에 연결한다 — `POST /api/pipelines/generate` 엔드포인트 추가.

```go
// Add to pipeline_handler.go

// POST /api/pipelines/generate
func (h *PipelineHandler) Generate(c *gin.Context) {
	var req service.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	generator := service.NewPipelineGenerator()
	pipeline, err := generator.Generate(c.Request.Context(), req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pipeline": pipeline})
}

// Add route registration:
// pipelines.POST("/generate", h.Generate)
```

- [ ] 9.4 `src/features/pipeline-generator/lib/pipeline-generator-api.ts`에 Go backend 호출 래퍼를 구현한다.

```typescript
// src/features/pipeline-generator/lib/pipeline-generator-api.ts
import type { CreatePipelineRequest } from "@/entities/pipeline/model/types";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export async function generatePipeline(description: string): Promise<{ pipeline: CreatePipelineRequest }> {
  const res = await fetch(`${API_BASE}/api/pipelines/generate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ description }),
  });
  if (!res.ok) throw new Error("Failed to generate pipeline");
  return res.json();
}
```

- [ ] 9.5 `src/features/pipeline-generator/ui/AIGenerateModal.tsx`를 구현한다 — 자연어 입력 텍스트 영역 + "Generate" 버튼 + 생성된 파이프라인 미리보기 + "Apply to Canvas" 버튼.

- [ ] 9.6 "Apply to Canvas" 클릭 시 `usePipelineStore`에 생성된 구조를 반영하여 캔버스에 블록들이 자동 배치되도록 연결한다.

- [ ] 9.7 테스트 실행 및 커밋.

```bash
cd backend && go test ./internal/service/...
npx jest src/features/pipeline-generator/__tests__/pipeline-generator.test.ts
git add backend/internal/service/pipeline_generator.go src/features/pipeline-generator/
git commit -m "feat(pipeline): implement Go NL-to-pipeline AI generation service with frontend modal"
```

---

## Task 10: 판단 섹션 UI + Register & Run 통합

판단 섹션(성공/실패 DSL 에디터 + 가격 알림)과 Register & Run 전체 플로우를 구현한다. API 호출은 Go 백엔드를 타겟한다.

**Files:**
- Create: `src/features/pipeline-builder/ui/sections/JudgmentEditor.tsx`
- Create: `src/features/pipeline-builder/ui/DSLConditionEditor.tsx`
- Create: `src/features/pipeline-builder/ui/PriceAlertEditor.tsx`
- Create: `src/features/pipeline-builder/lib/useRegisterAndRun.ts`
- Modify: `src/features/pipeline-builder/ui/PipelineTopbar.tsx` (Register & Run 연결)
- Modify: `src/features/pipeline-builder/model/pipeline.store.ts` (save/load 액션 추가)
- Test: `src/features/pipeline-builder/__tests__/JudgmentEditor.test.tsx`
- Test: `src/features/pipeline-builder/__tests__/RegisterAndRun.test.tsx`

**Steps:**

- [ ] 10.1 TDD: JudgmentEditor 테스트를 작성한다 — 성공/실패 DSL 입력, 가격 알림 추가/삭제, DSL 문법 검증 표시.

- [ ] 10.2 `src/features/pipeline-builder/ui/DSLConditionEditor.tsx`를 구현한다 — 코드 에디터 스타일(monospace, 다크 배경)의 DSL 입력. 이벤트 상대 변수(event_high, pre_event_ma 등) 자동완성 힌트.

```
┌─ Success Condition ─────────────────────┐
│ close >= event_high * 2.0               │
│                     ✓ Valid             │
└─────────────────────────────────────────┘
┌─ Failure Condition ─────────────────────┐
│ close < pre_event_ma(120)               │
│                     ✓ Valid             │
└─────────────────────────────────────────┘
```

- [ ] 10.3 `src/features/pipeline-builder/ui/PriceAlertEditor.tsx`를 구현한다 — 가격 알림 목록 + 추가 폼(condition DSL + label).

```
┌─ Price Alerts ──────────────────────────┐
│ close >= 65000  "목표가 도달"    [x]     │
│ rsi(14) < 30    "RSI 과매도"    [x]     │
│ [+ Add Alert]                           │
└─────────────────────────────────────────┘
```

- [ ] 10.4 `src/features/pipeline-builder/ui/sections/JudgmentEditor.tsx`에 DSLConditionEditor 2개(성공/실패) + PriceAlertEditor를 조합한다.

- [ ] 10.5 TDD: RegisterAndRun 통합 테스트를 작성한다 — 종목 선택 + 파이프라인 캔버스 상태에서 Register & Run 클릭 시: (1) Go 백엔드 파이프라인 저장 API 호출, (2) 실행 API 호출, (3) 실행 상태 폴링, (4) 완료 시 케이스 페이지로 이동.

- [ ] 10.6 `src/features/pipeline-builder/lib/useRegisterAndRun.ts`에 Register & Run 플로우를 추출한다 — Go 백엔드 API를 호출.

```typescript
import { usePipelineStore } from "../model/pipeline.store";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export function useRegisterAndRun() {
  const store = usePipelineStore();

  async function handleRegisterAndRun(symbol: string) {
    // 1. pipeline.store 상태를 CreatePipelineRequest로 직렬화
    const payload = serializeStoreToRequest(store);

    // 2. POST /api/pipelines (파이프라인 저장 → Go backend)
    const pipelineRes = await fetch(`${API_BASE}/api/pipelines`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(payload),
    });
    const pipeline = await pipelineRes.json();

    // 3. POST /api/pipelines/:id/execute (실행 시작 → Go backend)
    const execRes = await fetch(`${API_BASE}/api/pipelines/${pipeline.id}/execute`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ symbol }),
    });
    const { jobId } = await execRes.json();

    // 4. GET /api/pipelines/:id/jobs/:jobId 폴링 (실행 상태 → Go backend)
    // 5. 실행 완료 → router.push(`/cases/${caseId}`)
  }

  return { handleRegisterAndRun, isRunning: false };
}
```

- [ ] 10.7 실행 중 상태 표시 UI — Topbar에 실행 진행 상태(에이전트별) 표시. 각 블록 카드에 상태 아이콘(대기/실행 중/완료/실패).

- [ ] 10.8 테스트 실행 및 커밋.

```bash
npx jest src/features/pipeline-builder/__tests__/JudgmentEditor.test.tsx src/features/pipeline-builder/__tests__/RegisterAndRun.test.tsx
git add src/features/pipeline-builder/
git commit -m "feat(pipeline): implement judgment section editors and Register & Run hook with Go backend integration"
```
