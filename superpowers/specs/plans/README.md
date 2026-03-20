# NEXUS Implementation Plans

> Source spec: `superpowers/specs/2026-03-18-nexus-design.md`

## Dependency Graph

```
Plan 1: Infrastructure (DB + API + DSL Engine)
  │
  ├─→ Plan 2: Search (NL/DSL 2-Tab)
  │     └─→ Plan 3: Chart (Stock List Sidebar + Navigation)
  │
  ├─→ Plan 4: Pipeline Builder (Agent Blocks + 3 Sections)
  │     ├─→ Plan 5: Case Management (Timeline 50:50)
  │     │     └─→ Plan 8: Portfolio (Trade Integration)
  │     └─→ Plan 6: Monitoring Engine (Cron + DSL Polling)
  │           └─→ Plan 7: Alert System (In-App + Push + Messenger)
  │
  ├─→ Plan 9: Backtest + Pattern Matching
  └─→ Plan 10: Marketplace
```

## Priority & Order

| Priority | Plans | Description |
|----------|-------|-------------|
| **P0** | 1 → 2 → 3 → 4 → 5 | Core loop: 검색 → 차트 → 파이프라인 → 케이스 |
| **P1** | 6 → 7 | 자동화: 모니터링 → 알림 |
| **P2** | 8, 9 | 확장: 포트폴리오, 백테스트 |
| **P3** | 10 | 생태계: 마켓플레이스 |

## Tech Stack

- **Frontend**: Next.js (App Router) + TypeScript + TailwindCSS — 페이지 렌더링만
- **Backend**: Go (Gin + sqlc + asynq) — 모든 API + 비즈니스 로직
- **Database**: PostgreSQL + sqlc (type-safe SQL)
- **Worker**: Go + asynq (Redis 기반 job queue)
- **AI Agent**: Google ADK (Go SDK 또는 REST API 호출)
- **DSL Engine**: Go custom parser (lexer/parser/evaluator)
- **Real-time**: Server-Sent Events (SSE) — Go handler
- **Auth**: Google OAuth + JWT (Go middleware)
- **Deployment**: Vercel (frontend) + Cloud Run (Go backend)
