# 마켓플레이스 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 파이프라인, 에이전트 블록, 검색 프리셋, 판단 스크립트를 공유하고 Fork(딥 카피)하여 커스터마이징할 수 있는 마켓플레이스를 구축하며, 사용량/성과 기반 랭킹과 백테스트 검증 뱃지 시스템을 제공한다.
**Architecture:** 4가지 공유 아이템(Pipeline, AgentBlock, SearchPreset, JudgmentScript)을 MarketplaceItem 테이블로 통합 관리한다. Fork 시 원본을 deep copy하고 `forked_from_id`로 출처를 추적한다. 랭킹은 usage_count(사용량)와 backtest_win_rate(성과) 두 축으로 제공하며, 백테스트 결과가 연결된 아이템에는 "Verified" 뱃지를 부여한다.
**Tech Stack:** Go (Gin), sqlc, PostgreSQL tsvector (Full-Text Search), asynq (비동기 통계 갱신)

**Backend Layers (Go):**
- `backend/internal/domain/marketplace/` — 도메인 모델, DTO, enum 정의
- `backend/internal/repository/marketplace_repo.go` — MarketplaceItem sqlc/SQL CRUD
- `backend/internal/service/marketplace_service.go` — 마켓플레이스 비즈니스 로직 (목록, 상세, 게시, 좋아요, 검증)
- `backend/internal/service/fork_service.go` — Deep copy fork 엔진
- `backend/internal/handler/marketplace_handler.go` — Gin HTTP 핸들러 (thin controller)
- `backend/internal/worker/marketplace_stats.go` — asynq 비동기 통계 갱신 워커

**Frontend Layers (FSD, 유지):**
- `src/entities/marketplace-item/` — MarketplaceItem 프론트엔드 타입 정의
- `src/features/marketplace/` — 사용자 대면 기능(목록/검색, Fork, 좋아요, 게시, 검증) UI

**Note:** SearchPreset and JudgmentScript are domain modules defined in earlier plans. This plan assumes they already exist as SQL tables. If they don't, create them first.

---

## 의존성

- **Plan 4 (파이프라인 빌더)**: Pipeline, AgentBlock 테이블 (is_public 필드)
- **Plan 9 (백테스트)**: backtest_jobs 테이블 (Verified 뱃지 판단 기준)

---

## Task 1: SQL 마이그레이션 및 도메인 모델

4가지 공유 아이템을 통합 관리하는 데이터 모델과 Fork 추적, 사용 통계 테이블을 생성한다.

**Files:**
- Create: `backend/db/migrations/007_marketplace.sql`
- Create: `backend/db/queries/marketplace.sql`
- Create: `backend/internal/domain/marketplace/models.go`

**Steps:**

- [ ] SQL 마이그레이션 작성 — marketplace_items, marketplace_likes, marketplace_usage_logs

```sql
-- backend/db/migrations/007_marketplace.sql
CREATE TYPE marketplace_item_type AS ENUM ('PIPELINE', 'AGENT_BLOCK', 'SEARCH_PRESET', 'JUDGMENT_SCRIPT');
CREATE TYPE marketplace_status AS ENUM ('ACTIVE', 'HIDDEN', 'REMOVED');
CREATE TYPE usage_action AS ENUM ('VIEW', 'FORK', 'EXECUTE', 'LIKE');

CREATE TABLE marketplace_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id),
  type marketplace_item_type NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  tags TEXT[] DEFAULT '{}',
  pipeline_id UUID UNIQUE REFERENCES pipelines(id),
  agent_block_id UUID UNIQUE REFERENCES agent_blocks(id),
  search_preset_id UUID UNIQUE REFERENCES search_presets(id),
  judgment_script_id UUID UNIQUE,
  forked_from_id UUID REFERENCES marketplace_items(id),
  fork_count INT NOT NULL DEFAULT 0,
  usage_count INT NOT NULL DEFAULT 0,
  view_count INT NOT NULL DEFAULT 0,
  like_count INT NOT NULL DEFAULT 0,
  verified BOOLEAN NOT NULL DEFAULT false,
  backtest_job_id UUID REFERENCES backtest_jobs(id),
  backtest_win_rate NUMERIC(5,2),
  backtest_avg_return NUMERIC(8,2),
  backtest_total_events INT,
  status marketplace_status NOT NULL DEFAULT 'ACTIVE',
  published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  search_vector TSVECTOR
);

CREATE INDEX idx_marketplace_search ON marketplace_items USING GIN (search_vector);
CREATE INDEX idx_marketplace_items_type_status ON marketplace_items(type, status);
-- + search_vector auto-update trigger, marketplace_likes, marketplace_usage_logs
```

- [ ] sqlc 쿼리 작성 — CRUD, 정렬/필터별 목록, Full-Text Search, 카운터 갱신

```sql
-- backend/db/queries/marketplace.sql
-- name: SearchMarketplaceItems :many
SELECT mi.*, ts_rank(mi.search_vector, plainto_tsquery('simple', $1)) AS rank
FROM marketplace_items mi
JOIN users u ON mi.user_id = u.id
WHERE mi.search_vector @@ plainto_tsquery('simple', $1)
  AND mi.status = 'ACTIVE'
ORDER BY rank DESC
LIMIT $5 OFFSET $6;
```

- [ ] Go 도메인 모델 정의 — MarketplaceItem, MarketplaceLike, MarketplaceUsageLog, DTOs

```go
// backend/internal/domain/marketplace/models.go
package marketplace

type ItemType string
const (
    ItemTypePipeline       ItemType = "PIPELINE"
    ItemTypeAgentBlock     ItemType = "AGENT_BLOCK"
    ItemTypeSearchPreset   ItemType = "SEARCH_PRESET"
    ItemTypeJudgmentScript ItemType = "JUDGMENT_SCRIPT"
)

type MarketplaceItem struct { /* ... full model ... */ }
type ListQuery struct { /* type, sort, search, tags, verifiedOnly, page, limit */ }
type ItemDetail struct { /* API 응답 DTO */ }
type PublishRequest struct { /* 게시 요청 */ }
type ForkResult struct { /* Fork 결과 */ }
type LikeToggleResult struct { /* 좋아요 토글 결과 */ }
type VerificationResult struct { /* 검증 결과 */ }
```

- [ ] 마이그레이션 실행

```bash
psql -f backend/db/migrations/007_marketplace.sql
```

- [ ] 테스트: 모델 CRUD 기본 동작 확인
- [ ] 테스트: MarketplaceLike unique 제약조건 (중복 좋아요 방지) 확인

```bash
git add backend/db/ backend/internal/domain/marketplace/
git commit -m "feat(marketplace): SQL 마이그레이션 및 Go 도메인 모델 정의"
```

---

## Task 2: Repository 및 목록 조회/검색 API

인기순/성과순/최신순/Fork순 정렬, 태그 필터, Full-Text Search를 지원하는 마켓플레이스 목록 API를 구현한다.

**Files:**
- Create: `backend/internal/repository/marketplace_repo.go`
- Create: `backend/internal/service/marketplace_service.go`
- Create: `backend/internal/handler/marketplace_handler.go`

**Steps:**

- [ ] MarketplaceRepo 구현 — DB 접근 캡슐화, SQL 직접 실행

```go
// backend/internal/repository/marketplace_repo.go
func (r *MarketplaceRepo) ListItems(ctx context.Context, q mkt.ListQuery) ([]ItemRow, int64, error) {
    // 정렬/필터 빌드 → SQL 실행 → rows 반환
    // Full-text search: plainto_tsquery('simple', $1) + ts_rank
}

func (r *MarketplaceRepo) GetItemByID(ctx context.Context, id uuid.UUID) (*ItemRow, error) {
    // JOIN users, LEFT JOIN forked_from → single row
}

func (r *MarketplaceRepo) IncrementViewCount(ctx context.Context, id uuid.UUID) error { /* ... */ }
func (r *MarketplaceRepo) HasRecentView(ctx context.Context, userID, itemID uuid.UUID, since time.Time) (bool, error) { /* ... */ }
```

- [ ] MarketplaceService 구현 — 비즈니스 로직

```go
// backend/internal/service/marketplace_service.go
func (s *MarketplaceService) ListItems(ctx, query, currentUserID) (*mkt.ListResponse, error) {
    // repo.ListItems → rowToDetail 변환 → isLikedByMe 체크
}

func (s *MarketplaceService) GetDetail(ctx, itemID, currentUserID) (*mkt.ItemDetail, error) {
    // deduplicated view count → repo.GetItemByID → detail 변환
}
```

- [ ] MarketplaceHandler — Gin 핸들러 (thin controller)

```go
// backend/internal/handler/marketplace_handler.go
func (h *MarketplaceHandler) RegisterRoutes(rg *gin.RouterGroup) {
    g := rg.Group("/marketplace")
    g.GET("", h.List)                    // 목록/검색
    g.GET("/:id", h.GetDetail)           // 상세 조회
    g.POST("/publish", h.Publish)        // 게시
    g.PUT("/:id", h.Update)             // 수정
    g.DELETE("/:id", h.Delete)           // 삭제
    g.POST("/:id/fork", h.Fork)         // Fork
    g.POST("/:id/like", h.ToggleLike)   // 좋아요
    g.POST("/:id/verify", h.Verify)     // 검증
}
```

- [ ] 테스트: 인기순 정렬 → usageCount 내림차순 확인
- [ ] 테스트: 성과순 정렬 → backtestWinRate 내림차순 확인
- [ ] 테스트: 태그 필터 → 일치하는 아이템만 반환 확인
- [ ] 테스트: Full-Text Search → title/description/tags tsvector 매칭 확인
- [ ] 테스트: 조회 시 viewCount 증가 (deduplicated, 60분 윈도우) 확인

```bash
git add backend/internal/repository/ backend/internal/service/ backend/internal/handler/
git commit -m "feat(marketplace): Go 목록 조회 및 검색 API 구현 (정렬/필터/Full-Text Search)"
```

---

## Task 3: Fork (Deep Copy) 엔진

마켓플레이스 아이템을 Fork하여 자신의 리소스로 deep copy하는 엔진을 구현한다.
전체 Fork 프로세스를 단일 SQL 트랜잭션으로 감싸 race condition을 방지한다.

**Files:**
- Create: `backend/internal/service/fork_service.go`

**Steps:**

- [ ] ForkService 구현 — 타입별 deep copy 로직, transactional

```go
// backend/internal/service/fork_service.go
func (s *ForkService) ForkItem(ctx context.Context, userID, itemID uuid.UUID) (*mkt.ForkResult, error) {
    // 1. 원본 조회
    // 2. Begin TX
    // 3. switch item.Type:
    //      PIPELINE:        forkPipeline(tx, pipelineID, userID)
    //                       → pipeline + analysis_stages + blocks + monitors deep copy
    //      AGENT_BLOCK:     forkAgentBlock(tx, blockID, userID)
    //      SEARCH_PRESET:   forkSearchPreset(tx, presetID, userID)
    //      JUDGMENT_SCRIPT: forkJudgmentScript(tx, scriptID, userID)
    // 4. 새 marketplace_item INSERT (forked_from_id 설정)
    // 5. 원본 fork_count + 1
    // 6. usage_log INSERT (action = 'FORK')
    // 7. Commit TX
}

func (s *ForkService) forkPipeline(ctx, tx, originalID, userID) (uuid.UUID, error) {
    // INSERT INTO pipelines ... SELECT ... FROM pipelines WHERE id = $original
    // 각 analysis_stages → 새 stage 생성 → 각 agent_blocks deep copy
    // 각 monitors → 새 block 생성 → 새 monitor 생성
}

func (s *ForkService) forkAgentBlock(ctx, tx, originalID, userID) (uuid.UUID, error) { /* ... */ }
func (s *ForkService) forkSearchPreset(ctx, tx, originalID, userID) (uuid.UUID, error) { /* ... */ }
func (s *ForkService) forkJudgmentScript(ctx, tx, originalID, userID) (uuid.UUID, error) { /* ... */ }
```

- [ ] Fork API 핸들러 — POST /api/marketplace/:id/fork

```go
// handler/marketplace_handler.go → Fork()
// requires auth → forkSvc.ForkItem() → 201 Created { newItemId, newResourceId }
```

- [ ] 테스트: Pipeline Fork → analysis_stages + blocks + monitors 전체 deep copy 확인
- [ ] 테스트: Fork된 리소스의 user_id가 현재 사용자인지 확인
- [ ] 테스트: Fork된 리소스의 is_public = false 확인
- [ ] 테스트: 원본 fork_count 증가 확인
- [ ] 테스트: AgentBlock Fork → 독립 블록 생성 확인
- [ ] 테스트: 자기 자신의 아이템도 Fork 가능 확인

```bash
git add backend/internal/service/fork_service.go
git commit -m "feat(marketplace): Fork (Deep Copy) 엔진 구현 (Go, transactional)"
```

---

## Task 4: 좋아요 및 사용량 추적

좋아요 토글과 사용량 집계 로직을 구현한다.

**Files:**
- Modify: `backend/internal/service/marketplace_service.go` (ToggleLike, TrackUsage)
- Modify: `backend/internal/repository/marketplace_repo.go` (likes, usage logs)

**Steps:**

- [ ] 좋아요 토글 — ToggleLike (IsLiked → 존재하면 삭제+감소, 없으면 생성+증가)

```go
// service/marketplace_service.go
func (s *MarketplaceService) ToggleLike(ctx, userID, itemID uuid.UUID) (*mkt.LikeToggleResult, error) {
    liked := repo.IsLiked(ctx, userID, itemID)
    if liked {
        repo.DeleteLike → repo.DecrementLikeCount
    } else {
        repo.CreateLike → repo.IncrementLikeCount → repo.CreateUsageLog(LIKE)
    }
    return { Liked: !liked, LikeCount: repo.GetItemLikeCount() }
}
```

- [ ] 사용량 추적 — TrackUsage (리소스 ID → marketplace_item 조회 → usage_count 증가)

```go
// service/marketplace_service.go
func (s *MarketplaceService) TrackUsage(ctx, resourceID uuid.UUID, resourceType mkt.ItemType, userID uuid.UUID) error {
    itemID := repo.FindItemIDByResourceID(ctx, resourceID, resourceType)
    if itemID == nil { return nil } // 마켓플레이스에 등록되지 않은 리소스
    repo.IncrementUsageCount → repo.CreateUsageLog(EXECUTE)
}
```

- [ ] Plan 4의 파이프라인 실행 API에 TrackUsage() 호출 연동
- [ ] 테스트: 좋아요 토글 → liked=true → liked=false → likeCount 정확 확인
- [ ] 테스트: 동일 사용자 중복 좋아요 방지 (unique 제약) 확인
- [ ] 테스트: 파이프라인 실행 시 usageCount 증가 확인

```bash
git add backend/internal/service/ backend/internal/repository/
git commit -m "feat(marketplace): 좋아요 토글 및 사용량 추적 구현 (Go)"
```

---

## Task 5: Verified 뱃지 시스템

백테스트 결과를 마켓플레이스 아이템에 연결하여 "Verified" 뱃지를 부여하는 시스템을 구현한다.

**Files:**
- Modify: `backend/internal/service/marketplace_service.go` (VerifyItem)

**Steps:**

- [ ] Verified 뱃지 부여 로직 — VerifyItem

```go
// service/marketplace_service.go
// VerificationCriteria:
//   MinTotalEvents: 10, MinWinRate: 50.0, MinAvgReturn: 0.0

func (s *MarketplaceService) VerifyItem(ctx, itemID, userID uuid.UUID, job BacktestJobInfo) (*mkt.VerificationResult, error) {
    // 1. 소유자 확인 (row.UserID != userID → 거부)
    // 2. 백테스트 상태 확인 (job.Status != "COMPLETED" → 거부)
    // 3. 파이프라인 일치 확인 (PIPELINE 타입일 때)
    // 4. Stats 존재 확인
    // 5. 기준 체크:
    //    - TotalEvents >= 10
    //    - WinRate >= 50%
    //    - AvgReturn >= 0%
    // 6. 통과 시: repo.SetVerification(itemID, jobID, winRate, avgReturn, totalEvents)
    // 7. 결과 반환 { verified, reason, stats }
}
```

- [ ] Verify API 핸들러 — POST /api/marketplace/:id/verify { backtestJobId }

- [ ] 테스트: 백테스트 승률 60%, 이벤트 15건 → Verified=true 확인
- [ ] 테스트: 백테스트 승률 40% → Verified=false, 이유 메시지 확인
- [ ] 테스트: 백테스트 이벤트 5건 → Verified=false (최소 10건) 확인
- [ ] 테스트: 다른 사용자의 아이템 검증 시도 → 거부 확인
- [ ] 테스트: 파이프라인 불일치 시 거부 확인

```bash
git add backend/internal/service/
git commit -m "feat(marketplace): Verified 뱃지 시스템 구현 (Go, 백테스트 기반 검증)"
```

---

## Task 6: 비동기 통계 갱신 워커 (asynq)

인라인 카운터 증가의 drift를 보정하기 위해, 전체/개별 아이템 통계를 소스 테이블에서 재계산하는 asynq 워커를 구현한다.

**Files:**
- Create: `backend/internal/worker/marketplace_stats.go`

**Steps:**

- [ ] MarketplaceStatsWorker 구현

```go
// worker/marketplace_stats.go
const (
    TypeMarketplaceStatsRefresh     = "marketplace:stats:refresh"
    TypeMarketplaceStatsRefreshItem = "marketplace:stats:refresh_item"
)

func (w *MarketplaceStatsWorker) HandleStatsRefresh(ctx, t *asynq.Task) error {
    // repo.ListActiveItemIDs → 각 아이템별 RefreshForkCount, RefreshLikeCount, RefreshUsageCount, RefreshViewCount
}

func (w *MarketplaceStatsWorker) HandleStatsRefreshItem(ctx, t *asynq.Task) error {
    // payload에서 itemID 추출 → 단일 아이템 통계 재계산
}

// Task 생성 헬퍼
func NewMarketplaceStatsRefreshTask() (*asynq.Task, error) { /* ... */ }
func NewMarketplaceStatsRefreshItemTask(itemID uuid.UUID) (*asynq.Task, error) { /* ... */ }

// 핸들러 등록
func (w *MarketplaceStatsWorker) RegisterHandlers(mux *asynq.ServeMux) {
    mux.HandleFunc(TypeMarketplaceStatsRefresh, w.HandleStatsRefresh)
    mux.HandleFunc(TypeMarketplaceStatsRefreshItem, w.HandleStatsRefreshItem)
}
```

- [ ] asynq 스케줄러에 정기 실행 등록 (예: 매 10분)

```go
scheduler.Register("@every 10m", NewMarketplaceStatsRefreshTask())
```

- [ ] 테스트: 전체 통계 갱신 → 소스 테이블 기준 정확한 카운트 확인
- [ ] 테스트: 개별 아이템 통계 갱신 → 정확한 카운트 확인

```bash
git add backend/internal/worker/
git commit -m "feat(marketplace): asynq 비동기 통계 갱신 워커 구현"
```

---

## Task 7: 아이템 게시 API 및 통합 테스트

리소스를 마켓플레이스에 게시하는 API와 전체 통합 테스트를 구현한다.

**Files:**
- Modify: `backend/internal/service/marketplace_service.go` (Publish)
- Modify: `backend/internal/handler/marketplace_handler.go` (Publish endpoint)

**Steps:**

- [ ] 게시 서비스 구현 — Publish

```go
// service/marketplace_service.go
func (s *MarketplaceService) Publish(ctx, userID uuid.UUID, req mkt.PublishRequest) (uuid.UUID, error) {
    // 1. resourceID 파싱
    // 2. TODO: 리소스 소유자 확인 (별도 resource repo)
    // 3. switch req.Type → 적절한 FK 설정
    // 4. repo.CreateItem → item.ID 반환
}
```

- [ ] 게시 API 핸들러 — POST /api/marketplace/publish { type, resourceId, title, description, tags }

- [ ] 통합 테스트 작성

```go
func TestMarketplaceIntegration(t *testing.T) {
    t.Run("파이프라인 게시 → 목록 조회 → Fork → 커스텀", func(t *testing.T) {
        // 1. Pipeline 생성 (비공개)
        // 2. POST /api/marketplace/publish { type: PIPELINE, resourceId, title, tags }
        // 3. GET /api/marketplace?type=PIPELINE → 목록에 표시 확인
        // 4. POST /api/marketplace/:id/fork → 새 Pipeline 생성 확인
        // 5. 원본 fork_count 증가 확인
    })

    t.Run("Verified 뱃지 플로우", func(t *testing.T) {
        // 1. Pipeline 게시
        // 2. 백테스트 실행 (승률 70%, 15건)
        // 3. POST /api/marketplace/:id/verify { backtestJobId }
        // 4. verified=true 확인
        // 5. GET /api/marketplace?verifiedOnly=true → 해당 아이템 포함
    })

    t.Run("인기 랭킹 — 사용량 기반", func(t *testing.T) {
        // 아이템 A: usageCount=100, 아이템 B: usageCount=50
        // GET /api/marketplace?sort=popular → A가 먼저
    })

    t.Run("좋아요 토글 → likeCount 정확", func(t *testing.T) {
        // 좋아요 → likeCount=1 → 다시 좋아요 → likeCount=0
    })

    t.Run("4가지 타입 모두 Fork 가능", func(t *testing.T) {
        // PIPELINE, AGENT_BLOCK, SEARCH_PRESET, JUDGMENT_SCRIPT 각각 Fork 테스트
    })

    t.Run("타인의 리소스 검증 시도 → 거부", func(t *testing.T) {
        // 다른 유저의 아이템 → 거부 메시지
    })
}
```

- [ ] 테스트 실행 및 전체 통과 확인

```bash
go test ./backend/internal/...
git add backend/
git commit -m "feat(marketplace): 아이템 게시 API 및 전체 통합 테스트 완성 (Go)"
```

---

## Task 8: 프론트엔드 API 호출 변경

프론트엔드의 API 호출을 Next.js API Routes에서 Go 백엔드 엔드포인트로 변경한다.

**Files:**
- Modify: `src/features/marketplace/` — API 호출 URL 변경
- Modify: `src/entities/marketplace-item/` — 타입 정의 유지 (Go DTO와 호환)

**Steps:**

- [ ] API base URL 변경 — `/api/marketplace` → Go 백엔드 주소 (환경 변수)
- [ ] 기존 Next.js API Route 파일 제거 (Go로 이관 완료)
- [ ] 프론트엔드 타입이 Go DTO 응답과 호환되는지 확인
- [ ] E2E 테스트: 프론트엔드 → Go 백엔드 전체 플로우 확인

```bash
git add src/
git commit -m "feat(marketplace): 프론트엔드 API 호출을 Go 백엔드로 전환"
```

---

## 백엔드 Go 파일 구조 요약

```
backend/
  db/
    migrations/
      007_marketplace.sql              # DDL: marketplace_items, marketplace_likes, marketplace_usage_logs
    queries/
      marketplace.sql                  # sqlc 쿼리: CRUD, 목록/검색, 카운터, 좋아요, 사용 로그
  internal/
    domain/marketplace/
      models.go                        # 도메인 모델, enum, DTO (ItemType, Status, ListQuery, ItemDetail, etc.)
    repository/
      marketplace_repo.go             # DB 접근 (Create, Get, List, Search, counters, likes, usage logs)
    service/
      marketplace_service.go          # 비즈니스 로직 (List, Detail, Publish, ToggleLike, TrackUsage, VerifyItem)
      fork_service.go                 # Fork 엔진 (Pipeline + stages + blocks deep copy, transactional)
    handler/
      marketplace_handler.go          # Gin 핸들러 (8개 엔드포인트, thin controller)
    worker/
      marketplace_stats.go            # asynq 워커 (전체/개별 통계 재계산, 스케줄 등록)
```

## API 엔드포인트 요약

| Method | Path | Handler | 설명 |
|--------|------|---------|------|
| GET | `/api/marketplace` | List | 목록 조회/검색 (type, sort, search, tags, verifiedOnly, page, limit) |
| GET | `/api/marketplace/:id` | GetDetail | 상세 조회 (deduplicated view count) |
| POST | `/api/marketplace/publish` | Publish | 리소스 게시 |
| PUT | `/api/marketplace/:id` | Update | 제목/설명/태그 수정 |
| DELETE | `/api/marketplace/:id` | Delete | 소프트 삭제 (REMOVED) |
| POST | `/api/marketplace/:id/fork` | Fork | Deep copy fork (transactional) |
| POST | `/api/marketplace/:id/like` | ToggleLike | 좋아요 토글 |
| POST | `/api/marketplace/:id/verify` | Verify | Verified 뱃지 요청 |
