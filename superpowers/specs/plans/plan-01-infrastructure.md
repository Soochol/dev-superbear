# Infrastructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Next.js (프론트엔드) + Go/Gin/sqlc + PostgreSQL 기반 NEXUS 플랫폼의 기반 인프라를 구축한다 (프론트엔드 FSD 프로젝트 구조, Go 백엔드 DDD 구조, SQL 마이그레이션, DSL 엔진, API 핸들러, 인증, 공통 유틸리티).
**Architecture:** 프론트엔드는 Feature-Sliced Design (FSD) 아키텍처 기반 App Router Next.js 프로젝트(API Routes 없음)로 구성하고, 백엔드는 Go/Gin 기반 DDD 구조로 분리한다. sqlc로 PostgreSQL을 연동하고, 6개 핵심 테이블을 SQL 마이그레이션한다. DSL 엔진(lexer/parser/evaluator)을 `backend/internal/dsl/`에 Go로 구현하여 검색/성공조건/실패조건/가격알림에 공통 사용한다. Google OAuth + JWT(httpOnly cookie) 인증 미들웨어로 모든 API를 보호한다. API 핸들러는 thin controller 패턴으로 구현하여 service/repository 레이어에 위임한다.
**Tech Stack:** Next.js 15 (App Router, 프론트엔드 전용), TypeScript, TailwindCSS 4, Go 1.22+, Gin, sqlc, pgx/v5, PostgreSQL, golang-jwt, slog (logging), Google OAuth2

---

## Task 1: 프론트엔드 FSD 프로젝트 스캐폴딩 + Next.js 초기 구조

FSD(Feature-Sliced Design) 아키텍처 기반 프론트엔드 프로젝트 구조를 생성하고 다크 테마 기반 레이아웃을 구성한다. API Routes 디렉토리는 생성하지 않는다.

**Files:**
- Create: `package.json`, `tsconfig.json`, `next.config.ts`, `tailwind.config.ts`, `postcss.config.mjs`
- Create: `src/app/layout.tsx`, `src/app/page.tsx`, `src/app/globals.css`
- Create: `src/shared/config/constants.ts`
- Create: `src/shared/api/client.ts` (Go API 호출 클라이언트)
- Create: FSD layer directories (see below)

### Steps

- [ ] Next.js 프로젝트 생성

```bash
cd /home/dev/code/dev-superbear
npx create-next-app@latest nexus --typescript --tailwind --eslint --app --src-dir --import-alias "@/*" --use-npm
# 생성된 파일들을 프로젝트 루트로 이동
cp -r nexus/* nexus/.* . 2>/dev/null; rm -rf nexus
```

- [ ] FSD 디렉토리 구조 생성 — Feature-Sliced Design 레이어별 폴더를 생성 (API Routes 디렉토리 없음)

```bash
cd /home/dev/code/dev-superbear

# FSD app layer (Next.js App Router — 페이지만, API Routes 없음)
mkdir -p src/app/\(pages\)

# FSD features layer — user-facing interactions
mkdir -p src/features

# FSD entities layer — business entities (프론트엔드 모델 + UI만)
mkdir -p src/entities/case/model
mkdir -p src/entities/case/ui
mkdir -p src/entities/pipeline/model
mkdir -p src/entities/pipeline/ui
mkdir -p src/entities/agent-block/model
mkdir -p src/entities/agent-block/ui
mkdir -p src/entities/trade/model
mkdir -p src/entities/trade/ui
mkdir -p src/entities/timeline-event/model
mkdir -p src/entities/timeline-event/ui
mkdir -p src/entities/price-alert/model
mkdir -p src/entities/price-alert/ui
mkdir -p src/entities/user/model
mkdir -p src/entities/user/ui

# FSD widgets layer — composed UI sections
mkdir -p src/widgets

# FSD shared layer
mkdir -p src/shared/api
mkdir -p src/shared/lib/logger
mkdir -p src/shared/ui
mkdir -p src/shared/config
```

- [ ] TailwindCSS 다크 테마 설정 — `tailwind.config.ts`에 다크 모드를 `class`로 설정하고 커스텀 색상 팔레트 추가

```typescript
// tailwind.config.ts
import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: "class",
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        nexus: {
          bg: "#0a0a0f",
          surface: "#12121a",
          border: "#1e1e2e",
          accent: "#6366f1",
          success: "#22c55e",
          failure: "#ef4444",
          warning: "#f59e0b",
          text: {
            primary: "#e2e8f0",
            secondary: "#94a3b8",
            muted: "#64748b",
          },
        },
      },
      fontFamily: {
        mono: ["JetBrains Mono", "Fira Code", "monospace"],
      },
    },
  },
  plugins: [],
};

export default config;
```

- [ ] 루트 레이아웃 (`src/app/layout.tsx`) — 다크 테마 기본 적용, 한글 폰트(Pretendard) + 모노스페이스 폰트 로드

```typescript
// src/app/layout.tsx
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "NEXUS — AI-Native Investment Intelligence",
  description: "Agent-based investment analysis platform",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ko" className="dark">
      <body className="bg-nexus-bg text-nexus-text-primary min-h-screen antialiased">
        <div className="flex flex-col min-h-screen">
          {children}
        </div>
      </body>
    </html>
  );
}
```

- [ ] `globals.css` — TailwindCSS 디렉티브 + 스크롤바/선택 스타일

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  ::-webkit-scrollbar { width: 6px; height: 6px; }
  ::-webkit-scrollbar-track { background: #0a0a0f; }
  ::-webkit-scrollbar-thumb { background: #1e1e2e; border-radius: 3px; }
  ::selection { background: rgba(99, 102, 241, 0.3); }
}
```

- [ ] 상수 파일 생성 (`src/shared/config/constants.ts`) — 앱 전역 상수 (네비게이션 항목, 상태 enum 등)

- [ ] 프론트엔드 Logger 모듈 생성 (`src/shared/lib/logger/index.ts`) — `console.log` 대신 사용하는 구조화된 로거

```typescript
// src/shared/lib/logger/index.ts
type LogLevel = "debug" | "info" | "warn" | "error";

const LOG_LEVELS: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

const currentLevel: LogLevel =
  (process.env.LOG_LEVEL as LogLevel) ?? (process.env.NODE_ENV === "production" ? "info" : "debug");

function shouldLog(level: LogLevel): boolean {
  return LOG_LEVELS[level] >= LOG_LEVELS[currentLevel];
}

function formatMessage(level: LogLevel, message: string, context?: Record<string, unknown>): string {
  const timestamp = new Date().toISOString();
  const ctx = context ? ` ${JSON.stringify(context)}` : "";
  return `[${timestamp}] [${level.toUpperCase()}] ${message}${ctx}`;
}

export const logger = {
  debug(message: string, context?: Record<string, unknown>) {
    if (shouldLog("debug")) console.debug(formatMessage("debug", message, context));
  },
  info(message: string, context?: Record<string, unknown>) {
    if (shouldLog("info")) console.info(formatMessage("info", message, context));
  },
  warn(message: string, context?: Record<string, unknown>) {
    if (shouldLog("warn")) console.warn(formatMessage("warn", message, context));
  },
  error(message: string, context?: Record<string, unknown>) {
    if (shouldLog("error")) console.error(formatMessage("error", message, context));
  },
};
```

- [ ] Go API 클라이언트 생성 (`src/shared/api/client.ts`) — Go 백엔드 서버 호출용 fetch wrapper

```typescript
// src/shared/api/client.ts
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiError extends Error {
  constructor(public status: number, public body: string) {
    super(`API Error ${status}: ${body}`);
    this.name = 'ApiError';
  }
}

export async function apiClient<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...init?.headers,
    },
  });
  if (!res.ok) {
    throw new ApiError(res.status, await res.text());
  }
  return res.json();
}

/** GET 요청 헬퍼 */
export function apiGet<T>(path: string): Promise<T> {
  return apiClient<T>(path, { method: 'GET' });
}

/** POST 요청 헬퍼 */
export function apiPost<T>(path: string, body: unknown): Promise<T> {
  return apiClient<T>(path, {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

/** PUT 요청 헬퍼 */
export function apiPut<T>(path: string, body: unknown): Promise<T> {
  return apiClient<T>(path, {
    method: 'PUT',
    body: JSON.stringify(body),
  });
}

/** DELETE 요청 헬퍼 */
export function apiDelete<T>(path: string): Promise<T> {
  return apiClient<T>(path, { method: 'DELETE' });
}
```

- [ ] 개발 서버 기동 확인

```bash
cd /home/dev/code/dev-superbear
npm run dev -- --port 3000
# 브라우저에서 http://localhost:3000 접속하여 다크 배경 확인
```

- [ ] 커밋

```bash
git add -A && git commit -m "scaffold: FSD frontend structure with Next.js, dark theme, logger, and Go API client"
```

---

## Task 2: Go 백엔드 프로젝트 스캐폴딩

Go 백엔드 프로젝트를 DDD 구조로 생성한다. `cmd/`, `internal/`, `db/` 디렉토리 구조를 만들고 go.mod을 초기화한다.

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/api/main.go`
- Create: `backend/cmd/worker/main.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/middleware/cors.go`
- Create: `backend/internal/middleware/logger.go`
- Create: DDD layer directories

### Steps

- [ ] Go 프로젝트 초기화 및 DDD 디렉토리 구조 생성

```bash
cd /home/dev/code/dev-superbear
mkdir -p backend && cd backend
go mod init github.com/dev-superbear/nexus-backend

# DDD 구조
mkdir -p cmd/api
mkdir -p cmd/worker
mkdir -p internal/domain
mkdir -p internal/handler
mkdir -p internal/service
mkdir -p internal/repository/sqlc
mkdir -p internal/worker
mkdir -p internal/dsl
mkdir -p internal/agent
mkdir -p internal/infra
mkdir -p internal/middleware
mkdir -p internal/config
mkdir -p db/migrations
mkdir -p db/queries
```

- [ ] Go 의존성 설치

```bash
cd /home/dev/code/dev-superbear/backend
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v5
go get github.com/golang-jwt/jwt/v5
go get github.com/hibiken/asynq
go get golang.org/x/oauth2
```

- [ ] 설정 모듈 생성 (`backend/internal/config/config.go`) — 환경 변수 로드

```go
// backend/internal/config/config.go
package config

import (
	"os"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	GoogleClientID   string
	GoogleClientSecret string
	AllowedOrigins   []string
	Env              string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgresql://nexus:nexus@localhost:5432/nexus?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		AllowedOrigins:     []string{getEnv("ALLOWED_ORIGIN", "http://localhost:3000")},
		Env:                getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] CORS 미들웨어 생성 (`backend/internal/middleware/cors.go`)

```go
// backend/internal/middleware/cors.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
```

- [ ] 로깅 미들웨어 생성 (`backend/internal/middleware/logger.go`) — slog 기반 구조화된 HTTP 요청 로깅

```go
// backend/internal/middleware/logger.go
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		slog.Info("HTTP request",
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
```

- [ ] API 서버 엔트리포인트 생성 (`backend/cmd/api/main.go`)

```go
// backend/cmd/api/main.go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

func main() {
	// Logger 설정
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	cfg := config.Load()

	// DB 연결
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	queries := sqlc.New(pool)

	// Gin 서버 설정
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// API v1 라우트 그룹
	api := r.Group("/api/v1")

	// 헬스 체크 (인증 불필요)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 인증 필요 라우트
	auth := api.Group("")
	auth.Use(middleware.AuthRequired(cfg.JWTSecret))

	// 핸들러 등록
	registerRoutes(auth, queries)

	slog.Info("starting server", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func registerRoutes(rg *gin.RouterGroup, queries *sqlc.Queries) {
	// TODO: Task 6에서 핸들러 등록
	_ = queries
}
```

- [ ] Worker 엔트리포인트 생성 (`backend/cmd/worker/main.go`)

```go
// backend/cmd/worker/main.go
package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	slog.Info("worker starting (placeholder)")
	// TODO: asynq 워커 구현
	os.Exit(0)
}
```

- [ ] 빌드 확인

```bash
cd /home/dev/code/dev-superbear/backend
go build ./cmd/api/...
go build ./cmd/worker/...
```

- [ ] 커밋

```bash
git add backend/
git commit -m "scaffold: Go backend DDD project structure with Gin, config, CORS, and logger middleware"
```

---

## Task 3: SQL 마이그레이션 + sqlc 설정 + 도메인 모델

스펙의 Data Models 섹션에 정의된 6개 테이블을 SQL 마이그레이션으로 정의하고 sqlc를 설정한다. Go 도메인 모델과 repository 패턴을 적용한다.

**Files:**
- Create: `backend/db/migrations/001_initial.sql`
- Create: `backend/db/queries/users.sql`
- Create: `backend/db/queries/cases.sql`
- Create: `backend/db/queries/pipelines.sql`
- Create: `backend/db/queries/trades.sql`
- Create: `backend/db/queries/timeline_events.sql`
- Create: `backend/db/queries/agent_blocks.sql`
- Create: `backend/db/queries/price_alerts.sql`
- Create: `backend/sqlc.yaml`
- Create: `backend/internal/domain/models.go`
- Create: `src/entities/case/model/types.ts` (프론트엔드 도메인 타입)
- Create: `src/entities/pipeline/model/types.ts`
- Create: `src/entities/user/model/types.ts`
- Test: `backend/internal/repository/sqlc_test.go`

### Steps

- [ ] sqlc 설정 파일 생성 (`backend/sqlc.yaml`)

```yaml
# backend/sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries"
    schema: "db/migrations"
    gen:
      go:
        package: "sqlc"
        out: "internal/repository/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_empty_slices: true
```

- [ ] `.env` 파일에 DB 접속 정보 설정

```bash
# backend/.env
DATABASE_URL="postgresql://nexus:nexus@localhost:5432/nexus?sslmode=disable"
JWT_SECRET="dev-secret-change-in-production"
GOOGLE_CLIENT_ID=""
GOOGLE_CLIENT_SECRET=""
ALLOWED_ORIGIN="http://localhost:3000"
PORT="8080"
```

- [ ] SQL 마이그레이션 작성 (`backend/db/migrations/001_initial.sql`) — 스펙 Data Models 섹션의 6개 테이블을 정확히 반영

```sql
-- backend/db/migrations/001_initial.sql

-- ────────────────────────────────────────
-- Enum types
-- ────────────────────────────────────────

CREATE TYPE case_status AS ENUM ('LIVE', 'CLOSED_SUCCESS', 'CLOSED_FAILURE', 'BACKTEST');
CREATE TYPE timeline_event_type AS ENUM ('NEWS', 'DISCLOSURE', 'SECTOR', 'PRICE_ALERT', 'TRADE', 'PIPELINE_RESULT');
CREATE TYPE trade_type AS ENUM ('BUY', 'SELL');

-- ────────────────────────────────────────
-- Auth
-- ────────────────────────────────────────

CREATE TABLE users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email      TEXT NOT NULL UNIQUE,
  name       TEXT,
  image      TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ────────────────────────────────────────
-- Pipeline (파이프라인)
-- ────────────────────────────────────────

CREATE TABLE pipelines (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id),
  name             TEXT NOT NULL,
  description      TEXT NOT NULL DEFAULT '',

  -- 분석 섹션 — 순서가 있는 단계 목록, 각 단계에 에이전트 블록 ID 배열
  -- [{order: 1, blockIds: ["uuid1"]}, {order: 2, blockIds: ["uuid2","uuid3"]}]
  analysis_stages  JSONB NOT NULL DEFAULT '[]',

  -- 모니터링 섹션 — [{blockId, cron, enabled}]
  monitors         JSONB NOT NULL DEFAULT '[]',

  -- 판단 섹션
  success_script   TEXT NOT NULL DEFAULT '',
  failure_script   TEXT NOT NULL DEFAULT '',

  is_public        BOOLEAN NOT NULL DEFAULT false,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipelines_user_id ON pipelines(user_id);

-- ────────────────────────────────────────
-- Case (케이스)
-- ────────────────────────────────────────

CREATE TABLE cases (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id),
  pipeline_id     UUID NOT NULL REFERENCES pipelines(id),
  symbol          TEXT NOT NULL,                -- 종목 코드 (e.g. "005930")
  status          case_status NOT NULL DEFAULT 'LIVE',
  event_date      DATE NOT NULL,
  event_snapshot  JSONB NOT NULL,
  -- EventSnapshot: { high, low, close, volume, trade_value, pre_ma: {5,20,60,120,200} }
  success_script  TEXT NOT NULL,                -- 성공 조건 DSL
  failure_script  TEXT NOT NULL,                -- 실패 조건 DSL
  closed_at       DATE,
  closed_reason   TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cases_user_id ON cases(user_id);
CREATE INDEX idx_cases_symbol ON cases(symbol);
CREATE INDEX idx_cases_status ON cases(status);

-- ────────────────────────────────────────
-- Timeline Event (타임라인 이벤트)
-- ────────────────────────────────────────

CREATE TABLE timeline_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id     UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  date        DATE NOT NULL,
  type        timeline_event_type NOT NULL,
  title       TEXT NOT NULL,
  content     TEXT NOT NULL,
  ai_analysis TEXT,
  data        JSONB,                            -- 이벤트 유형별 구조화 데이터
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_timeline_events_case_id ON timeline_events(case_id);
CREATE INDEX idx_timeline_events_date ON timeline_events(date);

-- ────────────────────────────────────────
-- Trade (매수/매도 기록)
-- ────────────────────────────────────────

CREATE TABLE trades (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id    UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id),
  type       trade_type NOT NULL,
  price      DOUBLE PRECISION NOT NULL,
  quantity   INTEGER NOT NULL,
  fee        DOUBLE PRECISION NOT NULL DEFAULT 0,
  date       DATE NOT NULL,
  note       TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trades_case_id ON trades(case_id);
CREATE INDEX idx_trades_user_id ON trades(user_id);

-- ────────────────────────────────────────
-- Agent Block (에이전트 블록)
-- ────────────────────────────────────────

CREATE TABLE agent_blocks (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id       UUID NOT NULL REFERENCES users(id),
  name          TEXT NOT NULL,
  instruction   TEXT NOT NULL,                  -- 자연어 지시문
  system_prompt TEXT,
  allowed_tools JSONB,                          -- string[] | null
  output_schema JSONB,
  is_public     BOOLEAN NOT NULL DEFAULT false,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_agent_blocks_user_id ON agent_blocks(user_id);

-- ────────────────────────────────────────
-- Price Alert (가격 알림)
-- ────────────────────────────────────────

CREATE TABLE price_alerts (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id      UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  pipeline_id  UUID REFERENCES pipelines(id),
  condition    TEXT NOT NULL,                   -- DSL 표현식 (e.g. "close >= 75000")
  label        TEXT NOT NULL,                   -- 사용자 메모 (e.g. "목표가 도달")
  triggered    BOOLEAN NOT NULL DEFAULT false,
  triggered_at DATE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_price_alerts_case_id ON price_alerts(case_id);

-- ────────────────────────────────────────
-- updated_at 트리거 함수
-- ────────────────────────────────────────

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_users BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_updated_at_pipelines BEFORE UPDATE ON pipelines FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
CREATE TRIGGER set_updated_at_cases BEFORE UPDATE ON cases FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
```

- [ ] sqlc 쿼리 파일 작성 — Users

```sql
-- backend/db/queries/users.sql

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email, name, image)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET name = $2, image = $3 WHERE id = $1
RETURNING *;

-- name: UpsertUser :one
INSERT INTO users (email, name, image)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE SET name = $2, image = $3
RETURNING *;
```

- [ ] sqlc 쿼리 파일 작성 — Cases

```sql
-- backend/db/queries/cases.sql

-- name: ListCasesByUser :many
SELECT * FROM cases
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountCasesByUser :one
SELECT count(*) FROM cases WHERE user_id = $1;

-- name: ListCasesByUserAndStatus :many
SELECT * FROM cases
WHERE user_id = $1 AND status = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountCasesByUserAndStatus :one
SELECT count(*) FROM cases WHERE user_id = $1 AND status = $2;

-- name: GetCase :one
SELECT * FROM cases WHERE id = $1 AND user_id = $2;

-- name: CreateCase :one
INSERT INTO cases (user_id, pipeline_id, symbol, event_date, event_snapshot, success_script, failure_script)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateCaseStatus :one
UPDATE cases SET status = $2, closed_at = $3, closed_reason = $4
WHERE id = $1
RETURNING *;

-- name: DeleteCase :exec
DELETE FROM cases WHERE id = $1 AND user_id = $2;
```

- [ ] sqlc 쿼리 파일 작성 — Pipelines

```sql
-- backend/db/queries/pipelines.sql

-- name: ListPipelinesByUser :many
SELECT * FROM pipelines
WHERE user_id = $1
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPipelinesByUser :one
SELECT count(*) FROM pipelines WHERE user_id = $1;

-- name: GetPipeline :one
SELECT * FROM pipelines WHERE id = $1 AND user_id = $2;

-- name: CreatePipeline :one
INSERT INTO pipelines (user_id, name, description, analysis_stages, monitors, success_script, failure_script)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdatePipeline :one
UPDATE pipelines
SET name = $2, description = $3, analysis_stages = $4, monitors = $5, success_script = $6, failure_script = $7
WHERE id = $1
RETURNING *;

-- name: DeletePipeline :exec
DELETE FROM pipelines WHERE id = $1 AND user_id = $2;
```

- [ ] sqlc 쿼리 파일 작성 — Trades

```sql
-- backend/db/queries/trades.sql

-- name: ListTradesByCase :many
SELECT * FROM trades
WHERE case_id = $1
ORDER BY date DESC;

-- name: CreateTrade :one
INSERT INTO trades (case_id, user_id, type, price, quantity, fee, date, note)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
```

- [ ] sqlc 쿼리 파일 작성 — Timeline Events

```sql
-- backend/db/queries/timeline_events.sql

-- name: ListTimelineEventsByCase :many
SELECT * FROM timeline_events
WHERE case_id = $1
ORDER BY date DESC;

-- name: CreateTimelineEvent :one
INSERT INTO timeline_events (case_id, date, type, title, content, ai_analysis, data)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;
```

- [ ] sqlc 쿼리 파일 작성 — Agent Blocks

```sql
-- backend/db/queries/agent_blocks.sql

-- name: ListAgentBlocksByUser :many
SELECT * FROM agent_blocks
WHERE user_id = $1 OR is_public = true
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountAgentBlocksByUser :one
SELECT count(*) FROM agent_blocks WHERE user_id = $1 OR is_public = true;

-- name: GetAgentBlock :one
SELECT * FROM agent_blocks WHERE id = $1;

-- name: CreateAgentBlock :one
INSERT INTO agent_blocks (user_id, name, instruction, system_prompt, allowed_tools, output_schema, is_public)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateAgentBlock :one
UPDATE agent_blocks
SET name = $2, instruction = $3, system_prompt = $4, allowed_tools = $5, output_schema = $6, is_public = $7
WHERE id = $1
RETURNING *;

-- name: DeleteAgentBlock :exec
DELETE FROM agent_blocks WHERE id = $1 AND user_id = $2;
```

- [ ] sqlc 쿼리 파일 작성 — Price Alerts

```sql
-- backend/db/queries/price_alerts.sql

-- name: ListPriceAlertsByCase :many
SELECT * FROM price_alerts
WHERE case_id = $1
ORDER BY created_at DESC;

-- name: CreatePriceAlert :one
INSERT INTO price_alerts (case_id, pipeline_id, condition, label)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: TriggerPriceAlert :one
UPDATE price_alerts SET triggered = true, triggered_at = $2
WHERE id = $1
RETURNING *;
```

- [ ] Go 도메인 모델 정의 (`backend/internal/domain/models.go`)

```go
// backend/internal/domain/models.go
package domain

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ────────────────────────────────────────
// Enums
// ────────────────────────────────────────

type CaseStatus string

const (
	CaseStatusLive          CaseStatus = "LIVE"
	CaseStatusClosedSuccess CaseStatus = "CLOSED_SUCCESS"
	CaseStatusClosedFailure CaseStatus = "CLOSED_FAILURE"
	CaseStatusBacktest      CaseStatus = "BACKTEST"
)

type TimelineEventType string

const (
	TimelineEventTypeNews           TimelineEventType = "NEWS"
	TimelineEventTypeDisclosure     TimelineEventType = "DISCLOSURE"
	TimelineEventTypeSector         TimelineEventType = "SECTOR"
	TimelineEventTypePriceAlert     TimelineEventType = "PRICE_ALERT"
	TimelineEventTypeTrade          TimelineEventType = "TRADE"
	TimelineEventTypePipelineResult TimelineEventType = "PIPELINE_RESULT"
)

type TradeType string

const (
	TradeTypeBuy  TradeType = "BUY"
	TradeTypeSell TradeType = "SELL"
)

// ────────────────────────────────────────
// Domain Models
// ────────────────────────────────────────

type User struct {
	ID        pgtype.UUID        `json:"id"`
	Email     string             `json:"email"`
	Name      *string            `json:"name"`
	Image     *string            `json:"image"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

type Case struct {
	ID             pgtype.UUID        `json:"id"`
	UserID         pgtype.UUID        `json:"user_id"`
	PipelineID     pgtype.UUID        `json:"pipeline_id"`
	Symbol         string             `json:"symbol"`
	Status         CaseStatus         `json:"status"`
	EventDate      pgtype.Date        `json:"event_date"`
	EventSnapshot  json.RawMessage    `json:"event_snapshot"`
	SuccessScript  string             `json:"success_script"`
	FailureScript  string             `json:"failure_script"`
	ClosedAt       pgtype.Date        `json:"closed_at,omitempty"`
	ClosedReason   *string            `json:"closed_reason,omitempty"`
	CreatedAt      pgtype.Timestamptz `json:"created_at"`
	UpdatedAt      pgtype.Timestamptz `json:"updated_at"`
}

type TimelineEvent struct {
	ID         pgtype.UUID        `json:"id"`
	CaseID     pgtype.UUID        `json:"case_id"`
	Date       pgtype.Date        `json:"date"`
	Type       TimelineEventType  `json:"type"`
	Title      string             `json:"title"`
	Content    string             `json:"content"`
	AIAnalysis *string            `json:"ai_analysis,omitempty"`
	Data       json.RawMessage    `json:"data,omitempty"`
	CreatedAt  pgtype.Timestamptz `json:"created_at"`
}

type Trade struct {
	ID        pgtype.UUID        `json:"id"`
	CaseID    pgtype.UUID        `json:"case_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Type      TradeType          `json:"type"`
	Price     float64            `json:"price"`
	Quantity  int32              `json:"quantity"`
	Fee       float64            `json:"fee"`
	Date      pgtype.Date        `json:"date"`
	Note      *string            `json:"note,omitempty"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

type Pipeline struct {
	ID             pgtype.UUID        `json:"id"`
	UserID         pgtype.UUID        `json:"user_id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	AnalysisStages json.RawMessage    `json:"analysis_stages"`
	Monitors       json.RawMessage    `json:"monitors"`
	SuccessScript  string             `json:"success_script"`
	FailureScript  string             `json:"failure_script"`
	IsPublic       bool               `json:"is_public"`
	CreatedAt      pgtype.Timestamptz `json:"created_at"`
	UpdatedAt      pgtype.Timestamptz `json:"updated_at"`
}

type AgentBlock struct {
	ID           pgtype.UUID        `json:"id"`
	UserID       pgtype.UUID        `json:"user_id"`
	Name         string             `json:"name"`
	Instruction  string             `json:"instruction"`
	SystemPrompt *string            `json:"system_prompt,omitempty"`
	AllowedTools json.RawMessage    `json:"allowed_tools,omitempty"`
	OutputSchema json.RawMessage    `json:"output_schema,omitempty"`
	IsPublic     bool               `json:"is_public"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
}

type PriceAlert struct {
	ID          pgtype.UUID        `json:"id"`
	CaseID      pgtype.UUID        `json:"case_id"`
	PipelineID  pgtype.UUID        `json:"pipeline_id,omitempty"`
	Condition   string             `json:"condition"`
	Label       string             `json:"label"`
	Triggered   bool               `json:"triggered"`
	TriggeredAt pgtype.Date        `json:"triggered_at,omitempty"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
}

// ────────────────────────────────────────
// JSON sub-types
// ────────────────────────────────────────

type EventSnapshot struct {
	High       float64            `json:"high"`
	Low        float64            `json:"low"`
	Close      float64            `json:"close"`
	Volume     float64            `json:"volume"`
	TradeValue float64            `json:"trade_value"`
	PreMA      map[int]float64    `json:"pre_ma"`
}

type AnalysisStage struct {
	Order    int      `json:"order"`
	BlockIDs []string `json:"blockIds"`
}

type MonitorConfig struct {
	BlockID string `json:"blockId"`
	Cron    string `json:"cron"`
	Enabled bool   `json:"enabled"`
}
```

- [ ] 프론트엔드 도메인 타입 정의 — Go 백엔드 JSON 응답에 맞는 TypeScript 타입

```typescript
// src/entities/case/model/types.ts

export type CaseStatus = 'LIVE' | 'CLOSED_SUCCESS' | 'CLOSED_FAILURE' | 'BACKTEST';

export interface Case {
  id: string;
  user_id: string;
  pipeline_id: string;
  symbol: string;
  status: CaseStatus;
  event_date: string;
  event_snapshot: EventSnapshot;
  success_script: string;
  failure_script: string;
  closed_at?: string;
  closed_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface CaseWithRelations extends Case {
  pipeline?: Pipeline;
  timeline_events?: TimelineEvent[];
  trades?: Trade[];
  price_alerts?: PriceAlert[];
}

export interface EventSnapshot {
  high: number;
  low: number;
  close: number;
  volume: number;
  trade_value: number;
  pre_ma: Record<number, number>;
}

// Re-export from other entities for convenience
import type { Pipeline } from '@/entities/pipeline/model/types';
import type { TimelineEvent } from '@/entities/timeline-event/model/types';
import type { Trade } from '@/entities/trade/model/types';
import type { PriceAlert } from '@/entities/price-alert/model/types';
```

```typescript
// src/entities/pipeline/model/types.ts

export interface Pipeline {
  id: string;
  user_id: string;
  name: string;
  description: string;
  analysis_stages: AnalysisStage[];
  monitors: MonitorConfig[];
  success_script: string;
  failure_script: string;
  is_public: boolean;
  created_at: string;
  updated_at: string;
}

export interface AnalysisStage {
  order: number;
  blockIds: string[];
}

export interface MonitorConfig {
  blockId: string;
  cron: string;
  enabled: boolean;
}
```

```typescript
// src/entities/user/model/types.ts

export interface User {
  id: string;
  email: string;
  name?: string;
  image?: string;
  created_at: string;
  updated_at: string;
}
```

- [ ] PostgreSQL 데이터베이스 생성 및 마이그레이션 실행 (이미 실행 중이라고 가정)

```bash
# DB 생성 (필요시)
createdb -U nexus nexus 2>/dev/null || true

# 마이그레이션 실행
cd /home/dev/code/dev-superbear/backend
psql "$DATABASE_URL" -f db/migrations/001_initial.sql
```

- [ ] sqlc 코드 생성

```bash
cd /home/dev/code/dev-superbear/backend
# sqlc 설치 (필요시)
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
# 코드 생성
sqlc generate
```

- [ ] 테스트 — DB 연결 확인 (`backend/internal/repository/sqlc_test.go`)

```go
// backend/internal/repository/sqlc_test.go
package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDatabaseConnection(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://nexus:nexus@localhost:5432/nexus?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	var result int
	err = pool.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
}
```

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/repository/ -v
```

- [ ] 커밋

```bash
git add backend/db/ backend/sqlc.yaml backend/internal/domain/ backend/internal/repository/ src/entities/
git commit -m "feat: SQL migration with 6 core tables, sqlc setup, Go domain models, and frontend types"
```

---

## Task 4: DSL 엔진 (Go) — Lexer + Parser + Evaluator

종목 검색(`scan/where/sort`)과 성공/실패 조건 평가를 위한 DSL 엔진을 Go로 구현한다. lexer, parser(Pratt parsing), evaluator를 `backend/internal/dsl/`에 작성한다.

**Files:**
- Create: `backend/internal/dsl/token.go`
- Create: `backend/internal/dsl/lexer.go`
- Create: `backend/internal/dsl/ast.go`
- Create: `backend/internal/dsl/parser.go`
- Create: `backend/internal/dsl/evaluator.go`
- Create: `backend/internal/dsl/context.go`
- Create: `backend/internal/dsl/dsl.go` (public API)
- Test: `backend/internal/dsl/lexer_test.go`
- Test: `backend/internal/dsl/parser_test.go`
- Test: `backend/internal/dsl/evaluator_test.go`

### Steps

- [ ] 토큰 타입 정의 (`backend/internal/dsl/token.go`)

```go
// backend/internal/dsl/token.go
package dsl

type TokenType int

const (
	// Literals
	TOKEN_NUMBER TokenType = iota
	TOKEN_STRING
	TOKEN_IDENTIFIER

	// Keywords
	TOKEN_SCAN
	TOKEN_WHERE
	TOKEN_SORT
	TOKEN_BY
	TOKEN_ASC
	TOKEN_DESC
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
	TOKEN_TRUE
	TOKEN_FALSE
	TOKEN_LIMIT

	// Operators
	TOKEN_PLUS    // +
	TOKEN_MINUS   // -
	TOKEN_STAR    // *
	TOKEN_SLASH   // /
	TOKEN_EQ      // ==
	TOKEN_NEQ     // !=
	TOKEN_LT      // <
	TOKEN_GT      // >
	TOKEN_LTE     // <=
	TOKEN_GTE     // >=
	TOKEN_ASSIGN  // =

	// Delimiters
	TOKEN_LPAREN  // (
	TOKEN_RPAREN  // )
	TOKEN_COMMA   // ,
	TOKEN_DOT     // .

	// Special
	TOKEN_EOF
	TOKEN_NEWLINE
)

type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

var keywords = map[string]TokenType{
	"scan":  TOKEN_SCAN,
	"where": TOKEN_WHERE,
	"sort":  TOKEN_SORT,
	"by":    TOKEN_BY,
	"asc":   TOKEN_ASC,
	"desc":  TOKEN_DESC,
	"and":   TOKEN_AND,
	"or":    TOKEN_OR,
	"not":   TOKEN_NOT,
	"true":  TOKEN_TRUE,
	"false": TOKEN_FALSE,
	"limit": TOKEN_LIMIT,
}
```

- [ ] 렉서 테스트 먼저 작성 (`backend/internal/dsl/lexer_test.go`)

```go
// backend/internal/dsl/lexer_test.go
package dsl

import (
	"testing"
)

func TestLexer_SimpleScanQuery(t *testing.T) {
	tokens, err := Tokenize("scan where volume > 1000000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []TokenType{
		TOKEN_SCAN, TOKEN_WHERE,
		TOKEN_IDENTIFIER, TOKEN_GT, TOKEN_NUMBER,
		TOKEN_EOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %v, got %v", i, expected[i], tok.Type)
		}
	}
}

func TestLexer_ComparisonOperators(t *testing.T) {
	tokens, err := Tokenize("close >= 50000 and rsi <= 30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[1].Type != TOKEN_GTE {
		t.Errorf("expected GTE, got %v", tokens[1].Type)
	}
	if tokens[5].Type != TOKEN_LTE {
		t.Errorf("expected LTE, got %v", tokens[5].Type)
	}
}

func TestLexer_AssignVsEquality(t *testing.T) {
	tokens, err := Tokenize("success = close >= event_high * 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[1].Type != TOKEN_ASSIGN {
		t.Errorf("expected ASSIGN, got %v", tokens[1].Type)
	}
	if tokens[3].Type != TOKEN_GTE {
		t.Errorf("expected GTE, got %v", tokens[3].Type)
	}
}

func TestLexer_FunctionCall(t *testing.T) {
	tokens, err := Tokenize("pre_event_ma(120)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []TokenType{
		TOKEN_IDENTIFIER, TOKEN_LPAREN,
		TOKEN_NUMBER, TOKEN_RPAREN,
		TOKEN_EOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	if tokens[0].Value != "pre_event_ma" {
		t.Errorf("expected 'pre_event_ma', got '%s'", tokens[0].Value)
	}
}

func TestLexer_FloatNumbers(t *testing.T) {
	tokens, err := Tokenize("event_high * 2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[2].Type != TOKEN_NUMBER {
		t.Errorf("expected NUMBER, got %v", tokens[2].Type)
	}
	if tokens[2].Value != "2.0" {
		t.Errorf("expected '2.0', got '%s'", tokens[2].Value)
	}
}

func TestLexer_SortByClause(t *testing.T) {
	tokens, err := Tokenize("sort by volume desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []TokenType{
		TOKEN_SORT, TOKEN_BY,
		TOKEN_IDENTIFIER, TOKEN_DESC,
		TOKEN_EOF,
	}
	for i, tok := range tokens {
		if tok.Type != expected[i] {
			t.Errorf("token[%d]: expected %v, got %v", i, expected[i], tok.Type)
		}
	}
}

func TestLexer_LineAndColumn(t *testing.T) {
	tokens, err := Tokenize("scan\nwhere x > 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Line != 1 || tokens[0].Column != 1 {
		t.Errorf("scan: expected line=1,col=1, got line=%d,col=%d", tokens[0].Line, tokens[0].Column)
	}
	// 'where' is on line 2 after NEWLINE
	whereIdx := -1
	for i, tok := range tokens {
		if tok.Type == TOKEN_WHERE {
			whereIdx = i
			break
		}
	}
	if whereIdx >= 0 && tokens[whereIdx].Line != 2 {
		t.Errorf("where: expected line=2, got line=%d", tokens[whereIdx].Line)
	}
}

func TestLexer_UnexpectedCharacter(t *testing.T) {
	_, err := Tokenize("scan @ where")
	if err == nil {
		t.Error("expected error for unexpected character")
	}
}
```

- [ ] 렉서 구현 (`backend/internal/dsl/lexer.go`) — 문자열을 순회하며 토큰 목록 생성, 키워드 vs 식별자 구분

- [ ] AST 노드 타입 정의 (`backend/internal/dsl/ast.go`)

```go
// backend/internal/dsl/ast.go
package dsl

type Node interface {
	nodeType() string
}

type ScanQuery struct {
	Where  Node
	SortBy *SortClause
	Limit  *int
}

func (s *ScanQuery) nodeType() string { return "ScanQuery" }

type SortClause struct {
	Field     string
	Direction string // "asc" or "desc"
}

type AssignmentExpr struct {
	Name  string
	Value Node
}

func (a *AssignmentExpr) nodeType() string { return "AssignmentExpr" }

type BinaryExpr struct {
	Operator string
	Left     Node
	Right    Node
}

func (b *BinaryExpr) nodeType() string { return "BinaryExpr" }

type UnaryExpr struct {
	Operator string
	Operand  Node
}

func (u *UnaryExpr) nodeType() string { return "UnaryExpr" }

type FunctionCall struct {
	Name string
	Args []Node
}

func (f *FunctionCall) nodeType() string { return "FunctionCall" }

type Identifier struct {
	Name string
}

func (i *Identifier) nodeType() string { return "Identifier" }

type NumberLiteral struct {
	Value float64
}

func (n *NumberLiteral) nodeType() string { return "NumberLiteral" }

type BooleanLiteral struct {
	Value bool
}

func (b *BooleanLiteral) nodeType() string { return "BooleanLiteral" }

type StringLiteral struct {
	Value string
}

func (s *StringLiteral) nodeType() string { return "StringLiteral" }
```

- [ ] 파서 테스트 먼저 작성 (`backend/internal/dsl/parser_test.go`)

```go
// backend/internal/dsl/parser_test.go
package dsl

import (
	"testing"
)

func parseInput(t *testing.T, input string) Node {
	t.Helper()
	tokens, err := Tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	node, err := Parse(tokens)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return node
}

func TestParser_ScanWhereQuery(t *testing.T) {
	ast := parseInput(t, "scan where volume > 1000000")
	scan, ok := ast.(*ScanQuery)
	if !ok {
		t.Fatalf("expected ScanQuery, got %T", ast)
	}
	if scan.Where == nil {
		t.Fatal("expected Where clause")
	}
	bin, ok := scan.Where.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", scan.Where)
	}
	if bin.Operator != ">" {
		t.Errorf("expected '>', got '%s'", bin.Operator)
	}
}

func TestParser_ScanWhereSortLimit(t *testing.T) {
	ast := parseInput(t, "scan where volume > 1000000 sort by trade_value desc limit 50")
	scan, ok := ast.(*ScanQuery)
	if !ok {
		t.Fatalf("expected ScanQuery, got %T", ast)
	}
	if scan.SortBy == nil {
		t.Fatal("expected SortBy")
	}
	if scan.SortBy.Field != "trade_value" || scan.SortBy.Direction != "desc" {
		t.Errorf("unexpected sort: %+v", scan.SortBy)
	}
	if scan.Limit == nil || *scan.Limit != 50 {
		t.Errorf("expected limit=50, got %v", scan.Limit)
	}
}

func TestParser_AssignmentExpr(t *testing.T) {
	ast := parseInput(t, "success = close >= event_high * 2.0")
	assign, ok := ast.(*AssignmentExpr)
	if !ok {
		t.Fatalf("expected AssignmentExpr, got %T", ast)
	}
	if assign.Name != "success" {
		t.Errorf("expected name='success', got '%s'", assign.Name)
	}
}

func TestParser_OperatorPrecedence(t *testing.T) {
	ast := parseInput(t, "success = close >= event_high * 2.0")
	assign := ast.(*AssignmentExpr)
	bin, ok := assign.Value.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", assign.Value)
	}
	if bin.Operator != ">=" {
		t.Errorf("expected '>=', got '%s'", bin.Operator)
	}
	rightBin, ok := bin.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected right BinaryExpr, got %T", bin.Right)
	}
	if rightBin.Operator != "*" {
		t.Errorf("expected '*', got '%s'", rightBin.Operator)
	}
}

func TestParser_FunctionCall(t *testing.T) {
	ast := parseInput(t, "failure = close < pre_event_ma(120)")
	assign := ast.(*AssignmentExpr)
	bin := assign.Value.(*BinaryExpr)
	fn, ok := bin.Right.(*FunctionCall)
	if !ok {
		t.Fatalf("expected FunctionCall, got %T", bin.Right)
	}
	if fn.Name != "pre_event_ma" {
		t.Errorf("expected 'pre_event_ma', got '%s'", fn.Name)
	}
	num, ok := fn.Args[0].(*NumberLiteral)
	if !ok {
		t.Fatalf("expected NumberLiteral, got %T", fn.Args[0])
	}
	if num.Value != 120 {
		t.Errorf("expected 120, got %f", num.Value)
	}
}

func TestParser_AndOrLogic(t *testing.T) {
	ast := parseInput(t, "scan where volume > 1000000 and close > 50000")
	scan := ast.(*ScanQuery)
	bin, ok := scan.Where.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected BinaryExpr, got %T", scan.Where)
	}
	if bin.Operator != "and" {
		t.Errorf("expected 'and', got '%s'", bin.Operator)
	}
}

func TestParser_SyntaxError(t *testing.T) {
	tokens, err := Tokenize("scan where >")
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	_, err = Parse(tokens)
	if err == nil {
		t.Error("expected syntax error")
	}
}
```

- [ ] Pratt parser(재귀 하강) 구현 (`backend/internal/dsl/parser.go`) — 연산자 우선순위: `or` < `and` < 비교(`>=` 등) < 가감(`+-`) < 승제(`*/`) < 단항(`-`, `not`) < 함수호출

- [ ] 평가 컨텍스트 정의 (`backend/internal/dsl/context.go`)

```go
// backend/internal/dsl/context.go
package dsl

import "math"

// EventContext 는 성공/실패 조건 평가 시 주입되는 이벤트 컨텍스트이다.
type EventContext struct {
	Close          float64 // 현재가
	High           float64 // 현재 고가
	Low            float64 // 현재 저가
	Volume         float64 // 현재 거래량
	TradeValue     float64 // 현재 거래대금
	EventHigh      float64 // 이벤트일 고가
	EventLow       float64 // 이벤트일 저가
	EventClose     float64 // 이벤트일 종가
	EventVolume    float64 // 이벤트일 거래량
	PreEventClose  float64 // 이벤트 전일 종가
	PostHigh       float64 // 이벤트 이후 최고가
	PostLow        float64 // 이벤트 이후 최저가
	DaysSinceEvent float64 // 이벤트 이후 경과일
}

// FunctionRegistry 는 DSL 내장 함수 레지스트리이다.
type FunctionRegistry map[string]func(args ...float64) (float64, error)

// EvalContext 는 DSL 평가 시 사용되는 전체 컨텍스트이다.
type EvalContext struct {
	Variables map[string]interface{} // float64 or bool
	Functions FunctionRegistry
}

// NewEventEvalContext 는 기본 이벤트 컨텍스트로 EvalContext를 생성한다.
func NewEventEvalContext(event EventContext, preEventMA map[int]float64) *EvalContext {
	vars := map[string]interface{}{
		"close":            event.Close,
		"high":             event.High,
		"low":              event.Low,
		"volume":           event.Volume,
		"trade_value":      event.TradeValue,
		"event_high":       event.EventHigh,
		"event_low":        event.EventLow,
		"event_close":      event.EventClose,
		"event_volume":     event.EventVolume,
		"pre_event_close":  event.PreEventClose,
		"post_high":        event.PostHigh,
		"post_low":         event.PostLow,
		"days_since_event": event.DaysSinceEvent,
	}

	funcs := FunctionRegistry{
		"pre_event_ma": func(args ...float64) (float64, error) {
			if len(args) != 1 {
				return 0, fmt.Errorf("pre_event_ma expects 1 argument, got %d", len(args))
			}
			n := int(args[0])
			val, ok := preEventMA[n]
			if !ok {
				return 0, fmt.Errorf("pre_event_ma(%d) not available", n)
			}
			return val, nil
		},
		"max": func(args ...float64) (float64, error) {
			if len(args) != 2 {
				return 0, fmt.Errorf("max expects 2 arguments")
			}
			return math.Max(args[0], args[1]), nil
		},
		"min": func(args ...float64) (float64, error) {
			if len(args) != 2 {
				return 0, fmt.Errorf("min expects 2 arguments")
			}
			return math.Min(args[0], args[1]), nil
		},
		"abs": func(args ...float64) (float64, error) {
			if len(args) != 1 {
				return 0, fmt.Errorf("abs expects 1 argument")
			}
			return math.Abs(args[0]), nil
		},
	}

	return &EvalContext{
		Variables: vars,
		Functions: funcs,
	}
}
```

- [ ] 평가기 테스트 먼저 작성 (`backend/internal/dsl/evaluator_test.go`)

```go
// backend/internal/dsl/evaluator_test.go
package dsl

import (
	"testing"
)

func testCtx() *EvalContext {
	return &EvalContext{
		Variables: map[string]interface{}{
			"close":            78000.0,
			"event_high":       40000.0,
			"event_close":      38000.0,
			"event_volume":     28400000.0,
			"pre_event_close":  37500.0,
			"post_high":        85000.0,
			"post_low":         35000.0,
			"days_since_event": 127.0,
		},
		Functions: FunctionRegistry{
			"pre_event_ma": func(args ...float64) (float64, error) {
				mas := map[int]float64{5: 38200, 20: 37800, 60: 36000, 120: 34000}
				n := int(args[0])
				v, ok := mas[n]
				if !ok {
					return 0, fmt.Errorf("pre_event_ma(%d) not available", n)
				}
				return v, nil
			},
		},
	}
}

func TestEvaluator_SuccessCondition(t *testing.T) {
	// 78000 >= 40000 * 2.0 = 80000 -> false
	result, err := EvaluateDSL("success = close >= event_high * 2.0", testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["success"] != false {
		t.Errorf("expected success=false, got %v", m["success"])
	}
}

func TestEvaluator_FailureCondition(t *testing.T) {
	// 78000 < 34000 -> false
	result, err := EvaluateDSL("failure = close < pre_event_ma(120)", testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["failure"] != false {
		t.Errorf("expected failure=false, got %v", m["failure"])
	}
}

func TestEvaluator_Arithmetic(t *testing.T) {
	result, err := EvaluateDSL("result = event_high * 2.0 + 1000", testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["result"] != 81000.0 {
		t.Errorf("expected 81000, got %v", m["result"])
	}
}

func TestEvaluator_BooleanStandalone(t *testing.T) {
	result, err := EvaluateDSL("close > 70000", testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestEvaluator_AndOrLogic(t *testing.T) {
	result, err := EvaluateDSL("close > 70000 and days_since_event > 100", testCtx())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestEvaluator_UndefinedVariable(t *testing.T) {
	_, err := EvaluateDSL("undefined_var > 100", testCtx())
	if err == nil {
		t.Error("expected error for undefined variable")
	}
}
```

- [ ] 평가기 구현 (`backend/internal/dsl/evaluator.go`) — AST를 재귀적으로 순회하며 값 계산. 변수 참조는 컨텍스트에서 조회

- [ ] 공개 API 생성 (`backend/internal/dsl/dsl.go`)

```go
// backend/internal/dsl/dsl.go
package dsl

import "fmt"

// ParseDSL 은 DSL 문자열을 파싱하여 AST를 반환한다.
func ParseDSL(input string) (Node, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, fmt.Errorf("lexer error: %w", err)
	}
	return Parse(tokens)
}

// EvaluateDSL 은 DSL 문자열을 파싱 + 평가하여 결과를 반환한다.
func EvaluateDSL(input string, ctx *EvalContext) (interface{}, error) {
	ast, err := ParseDSL(input)
	if err != nil {
		return nil, err
	}
	return Evaluate(ast, ctx)
}

// ValidateDSL 은 DSL 문법을 검증한다 (에러 메시지 반환, 유효하면 nil).
func ValidateDSL(input string) error {
	_, err := ParseDSL(input)
	return err
}
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/dsl/ -v
```

- [ ] 커밋

```bash
git add backend/internal/dsl/
git commit -m "feat: DSL engine in Go with lexer, Pratt parser, evaluator, and event context"
```

---

## Task 5: Go Auth 미들웨어 (Google OAuth + JWT)

Google OAuth 로그인과 JWT httpOnly cookie 기반 세션 관리 미들웨어를 Go로 구현한다.

**Files:**
- Create: `backend/internal/middleware/auth.go`
- Create: `backend/internal/handler/auth_handler.go`
- Test: `backend/internal/middleware/auth_test.go`

### Steps

- [ ] JWT Auth 미들웨어 구현 (`backend/internal/middleware/auth.go`)

```go
// backend/internal/middleware/auth.go
package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func AuthRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := extractToken(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		claims, err := validateJWT(token, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set("userId", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}

func extractToken(c *gin.Context) (string, error) {
	// 1. httpOnly cookie
	if cookie, err := c.Cookie("nexus_token"); err == nil && cookie != "" {
		return cookie, nil
	}

	// 2. Authorization header (Bearer)
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer "), nil
	}

	return "", errors.New("no token found")
}

func validateJWT(tokenStr, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// GenerateJWT 는 사용자 정보로 JWT 토큰을 생성한다.
func GenerateJWT(userID, email, secret string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GetUserID 는 Gin 컨텍스트에서 현재 인증된 사용자 ID를 반환한다.
func GetUserID(c *gin.Context) string {
	id, _ := c.Get("userId")
	s, _ := id.(string)
	return s
}
```

- [ ] Auth 핸들러 구현 (`backend/internal/handler/auth_handler.go`) — Google OAuth 콜백 + JWT 발급

```go
// backend/internal/handler/auth_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

type AuthHandler struct {
	queries   *sqlc.Queries
	jwtSecret string
}

func NewAuthHandler(queries *sqlc.Queries, jwtSecret string) *AuthHandler {
	return &AuthHandler{queries: queries, jwtSecret: jwtSecret}
}

// GoogleCallback 은 Google OAuth 콜백을 처리한다.
// 실제 구현에서는 Google의 authorization code로 access token을 교환하고
// userinfo를 조회하여 DB upsert 후 JWT를 발급한다.
func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	// TODO: Google OAuth authorization code -> access token -> userinfo
	// 1. code := c.Query("code")
	// 2. exchange code for access token
	// 3. get userinfo (email, name, image)
	// 4. upsert user in DB
	// 5. generate JWT and set httpOnly cookie

	c.JSON(http.StatusOK, gin.H{"message": "OAuth callback placeholder"})
}

// Me 는 현재 인증된 사용자 정보를 반환한다.
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// TODO: fetch user from DB
	c.JSON(http.StatusOK, gin.H{"userId": userID})
}

// Logout 은 JWT 쿠키를 삭제한다.
func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("nexus_token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
```

- [ ] Auth 미들웨어 테스트 (`backend/internal/middleware/auth_test.go`)

```go
// backend/internal/middleware/auth_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthRequired_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired("test-secret"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	token, err := GenerateJWT("user-123", "test@example.com", secret)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	r := gin.New()
	r.Use(AuthRequired(secret))
	r.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.JSON(200, gin.H{"userId": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired("test-secret"))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_CookieToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	token, err := GenerateJWT("user-456", "cookie@example.com", secret)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	r := gin.New()
	r.Use(AuthRequired(secret))
	r.GET("/test", func(c *gin.Context) {
		userID := GetUserID(c)
		c.JSON(200, gin.H{"userId": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "nexus_token", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT("user-123", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	claims, err := validateJWT(token, "secret")
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected 'user-123', got '%s'", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got '%s'", claims.Email)
	}
}
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/middleware/ -v
```

- [ ] 커밋

```bash
git add backend/internal/middleware/ backend/internal/handler/auth_handler.go
git commit -m "feat: Go JWT auth middleware with Google OAuth handler and httpOnly cookie support"
```

---

## Task 6: Go API 서버 (Gin) + 핸들러 + 라우터 등록

스펙의 API Endpoints 섹션에 정의된 핸들러들의 스캐폴드를 생성한다. 각 핸들러는 thin controller 패턴으로 service/repository에 위임한다. Go struct tags로 요청 바디를 검증한다.

**Files:**
- Create: `backend/internal/handler/case_handler.go`
- Create: `backend/internal/handler/pipeline_handler.go`
- Create: `backend/internal/handler/block_handler.go`
- Create: `backend/internal/handler/search_handler.go`
- Create: `backend/internal/handler/helpers.go` (공통 응답/페이지네이션)
- Create: `backend/internal/service/case_service.go`
- Create: `backend/internal/service/pipeline_service.go`
- Modify: `backend/cmd/api/main.go` (핸들러 등록)
- Test: `backend/internal/handler/helpers_test.go`

### Steps

- [ ] 공통 응답 헬퍼 생성 (`backend/internal/handler/helpers.go`) — JSON 응답 빌더, 페이지네이션 파싱

```go
// backend/internal/handler/helpers.go
package handler

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PaginatedResponse struct {
	Data       interface{}        `json:"data"`
	Pagination PaginationMetadata `json:"pagination"`
}

type PaginationMetadata struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func Error(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func Paginated(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	c.JSON(http.StatusOK, PaginatedResponse{
		Data: data,
		Pagination: PaginationMetadata{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	})
}

type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

func GetPagination(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}
```

- [ ] 요청 바인딩 구조체 정의 — Go struct tags (binding:"required") 로 Zod 대체

```go
// backend/internal/handler/requests.go
package handler

type CreateCaseRequest struct {
	PipelineID    string                 `json:"pipeline_id" binding:"required,uuid"`
	Symbol        string                 `json:"symbol" binding:"required,min=1,max=10"`
	EventDate     string                 `json:"event_date" binding:"required"`
	EventSnapshot map[string]interface{} `json:"event_snapshot" binding:"required"`
	SuccessScript string                 `json:"success_script" binding:"required,min=1"`
	FailureScript string                 `json:"failure_script" binding:"required,min=1"`
}

type CreatePipelineRequest struct {
	Name           string        `json:"name" binding:"required,min=1,max=200"`
	Description    string        `json:"description"`
	AnalysisStages []interface{} `json:"analysis_stages"`
	Monitors       []interface{} `json:"monitors"`
	SuccessScript  string        `json:"success_script"`
	FailureScript  string        `json:"failure_script"`
}

type UpdatePipelineRequest struct {
	Name           string        `json:"name" binding:"required,min=1,max=200"`
	Description    string        `json:"description"`
	AnalysisStages []interface{} `json:"analysis_stages"`
	Monitors       []interface{} `json:"monitors"`
	SuccessScript  string        `json:"success_script"`
	FailureScript  string        `json:"failure_script"`
}

type CreateBlockRequest struct {
	Name         string      `json:"name" binding:"required,min=1,max=200"`
	Instruction  string      `json:"instruction" binding:"required,min=1"`
	SystemPrompt *string     `json:"system_prompt"`
	AllowedTools []string    `json:"allowed_tools"`
	OutputSchema interface{} `json:"output_schema"`
	IsPublic     bool        `json:"is_public"`
}

type CreateTradeRequest struct {
	Type     string  `json:"type" binding:"required,oneof=BUY SELL"`
	Price    float64 `json:"price" binding:"required,gt=0"`
	Quantity int     `json:"quantity" binding:"required,gt=0"`
	Fee      float64 `json:"fee" binding:"min=0"`
	Date     string  `json:"date" binding:"required"`
	Note     *string `json:"note"`
}

type CreateAlertRequest struct {
	Condition  string  `json:"condition" binding:"required,min=1"`
	Label      string  `json:"label" binding:"required,min=1"`
	PipelineID *string `json:"pipeline_id" binding:"omitempty,uuid"`
}

type ScanRequest struct {
	Query string `json:"query" binding:"required,min=1"`
}
```

- [ ] Cases 핸들러 생성 (`backend/internal/handler/case_handler.go`) — thin controller 패턴

```go
// backend/internal/handler/case_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

type CaseHandler struct {
	queries *sqlc.Queries
}

func NewCaseHandler(queries *sqlc.Queries) *CaseHandler {
	return &CaseHandler{queries: queries}
}

func (h *CaseHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	p := GetPagination(c)

	cases, err := h.queries.ListCasesByUser(c.Request.Context(), sqlc.ListCasesByUserParams{
		UserID: parseUUID(userID),
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	count, err := h.queries.CountCasesByUser(c.Request.Context(), parseUUID(userID))
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Paginated(c, cases, count, p.Page, p.PageSize)
}

func (h *CaseHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	cs, err := h.queries.GetCase(c.Request.Context(), sqlc.GetCaseParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusNotFound, "case not found")
		return
	}

	Success(c, cs)
}

func (h *CaseHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req CreateCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: create case via repository
	_ = userID
	Created(c, gin.H{"message": "case created (placeholder)"})
}

func (h *CaseHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	err := h.queries.DeleteCase(c.Request.Context(), sqlc.DeleteCaseParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
```

- [ ] Pipelines 핸들러 생성 (`backend/internal/handler/pipeline_handler.go`) — GET (목록), POST (생성), PUT (수정), DELETE (삭제)

```go
// backend/internal/handler/pipeline_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

type PipelineHandler struct {
	queries *sqlc.Queries
}

func NewPipelineHandler(queries *sqlc.Queries) *PipelineHandler {
	return &PipelineHandler{queries: queries}
}

func (h *PipelineHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	p := GetPagination(c)

	pipelines, err := h.queries.ListPipelinesByUser(c.Request.Context(), sqlc.ListPipelinesByUserParams{
		UserID: parseUUID(userID),
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	count, err := h.queries.CountPipelinesByUser(c.Request.Context(), parseUUID(userID))
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Paginated(c, pipelines, count, p.Page, p.PageSize)
}

func (h *PipelineHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	pipeline, err := h.queries.GetPipeline(c.Request.Context(), sqlc.GetPipelineParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusNotFound, "pipeline not found")
		return
	}

	Success(c, pipeline)
}

func (h *PipelineHandler) Create(c *gin.Context) {
	var req CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: create pipeline via repository
	Created(c, gin.H{"message": "pipeline created (placeholder)"})
}

func (h *PipelineHandler) Update(c *gin.Context) {
	var req UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: update pipeline via repository
	Success(c, gin.H{"message": "pipeline updated (placeholder)"})
}

func (h *PipelineHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	err := h.queries.DeletePipeline(c.Request.Context(), sqlc.DeletePipelineParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
```

- [ ] Blocks 핸들러 생성 (`backend/internal/handler/block_handler.go`) — GET (목록), POST (생성), PUT (수정), DELETE (삭제)

- [ ] Search 핸들러 생성 (`backend/internal/handler/search_handler.go`) — POST (DSL 실행 placeholder)

```go
// backend/internal/handler/search_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

type SearchHandler struct{}

func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

func (h *SearchHandler) Scan(c *gin.Context) {
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// DSL 문법 검증
	if err := dsl.ValidateDSL(req.Query); err != nil {
		Error(c, http.StatusBadRequest, "DSL syntax error: "+err.Error())
		return
	}

	// TODO: DSL 실행 및 종목 검색
	Success(c, gin.H{
		"query":   req.Query,
		"results": []interface{}{},
		"message": "scan placeholder — DSL validated successfully",
	})
}
```

- [ ] UUID 파싱 헬퍼 추가 (`backend/internal/handler/helpers.go`에 추가)

```go
// helpers.go에 추가
import "github.com/jackc/pgx/v5/pgtype"

func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	u.Scan(s)
	return u
}
```

- [ ] `backend/cmd/api/main.go` 에서 핸들러 등록 — registerRoutes 구현

```go
// backend/cmd/api/main.go 의 registerRoutes 함수 구현
func registerRoutes(rg *gin.RouterGroup, queries *sqlc.Queries, cfg *config.Config) {
	// Auth 핸들러 (인증 불필요 라우트)
	authH := handler.NewAuthHandler(queries, cfg.JWTSecret)

	// Auth routes (인증 필요 없음)
	// 이 부분은 auth 미들웨어 밖에 등록해야 함

	// Cases
	caseH := handler.NewCaseHandler(queries)
	rg.GET("/cases", caseH.List)
	rg.POST("/cases", caseH.Create)
	rg.GET("/cases/:id", caseH.Get)
	rg.DELETE("/cases/:id", caseH.Delete)

	// Pipelines
	pipeH := handler.NewPipelineHandler(queries)
	rg.GET("/pipelines", pipeH.List)
	rg.POST("/pipelines", pipeH.Create)
	rg.GET("/pipelines/:id", pipeH.Get)
	rg.PUT("/pipelines/:id", pipeH.Update)
	rg.DELETE("/pipelines/:id", pipeH.Delete)

	// Search
	searchH := handler.NewSearchHandler()
	rg.POST("/search/scan", searchH.Scan)

	_ = authH
}
```

- [ ] 헬퍼 테스트 작성 (`backend/internal/handler/helpers_test.go`)

```go
// backend/internal/handler/helpers_test.go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Success(c, map[string]string{"id": "1"})

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data wrapper")
	}
	if data["id"] != "1" {
		t.Errorf("expected id=1, got %v", data["id"])
	}
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Error(c, http.StatusNotFound, "Not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"] != "Not found" {
		t.Errorf("expected 'Not found', got %v", body["error"])
	}
}

func TestPaginated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	Paginated(c, []int{1, 2, 3}, 10, 1, 3)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body PaginatedResponse
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Pagination.Total != 10 {
		t.Errorf("expected total=10, got %d", body.Pagination.Total)
	}
	if body.Pagination.TotalPages != 4 {
		t.Errorf("expected totalPages=4, got %d", body.Pagination.TotalPages)
	}
}

func TestGetPagination_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=2&pageSize=10", nil)

	p := GetPagination(c)
	if p.Page != 2 {
		t.Errorf("expected page=2, got %d", p.Page)
	}
	if p.PageSize != 10 {
		t.Errorf("expected pageSize=10, got %d", p.PageSize)
	}
	if p.Offset != 10 {
		t.Errorf("expected offset=10, got %d", p.Offset)
	}
}
```

- [ ] 테스트 실행

```bash
cd /home/dev/code/dev-superbear/backend
go test ./internal/handler/ -v
```

- [ ] 빌드 + 서버 기동 확인

```bash
cd /home/dev/code/dev-superbear/backend
go build ./cmd/api/...
# 서버 기동 테스트
# ./api &
# curl http://localhost:8080/api/v1/health
```

- [ ] 커밋

```bash
git add backend/internal/handler/ backend/cmd/api/main.go
git commit -m "feat: Go Gin API handlers with thin controller pattern, validation, and router registration"
```

---

## Task 7: Jest 설정 + 프론트엔드 테스트 인프라 + Go 테스트 구성

프론트엔드 Jest 설정을 완료하고, Go 백엔드 테스트 구성을 정리한다.

**Files:**
- Create: `jest.config.ts`
- Create: `jest.setup.ts`
- Modify: `package.json` (scripts 추가)
- Modify: `tsconfig.json` (paths 설정)
- Create: `backend/Makefile` (Go 빌드/테스트 명령)

### Steps

- [ ] Jest 및 테스트 도구 설치

```bash
cd /home/dev/code/dev-superbear
npm install -D jest @jest/types ts-jest @types/jest
npm install -D @testing-library/react @testing-library/jest-dom
npm install -D @testing-library/user-event jest-environment-jsdom
```

- [ ] Jest 설정 (`jest.config.ts`)

```typescript
// jest.config.ts
import type { Config } from "@jest/types";

const config: Config.InitialOptions = {
  preset: "ts-jest",
  testEnvironment: "node",
  roots: ["<rootDir>/src"],
  moduleNameMapper: {
    "^@/(.*)$": "<rootDir>/src/$1",
  },
  transform: {
    "^.+\\.tsx?$": ["ts-jest", { tsconfig: "tsconfig.json" }],
  },
  setupFilesAfterSetup: ["<rootDir>/jest.setup.ts"],
  testMatch: ["**/__tests__/**/*.test.ts", "**/__tests__/**/*.test.tsx"],
  coverageDirectory: "coverage",
  collectCoverageFrom: [
    "src/**/*.{ts,tsx}",
    "!src/**/*.d.ts",
    "!src/**/index.ts",
  ],
};

export default config;
```

- [ ] Jest 셋업 파일 (`jest.setup.ts`)

```typescript
// jest.setup.ts
import "@testing-library/jest-dom";
```

- [ ] `package.json`에 스크립트 추가

```json
{
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage",
    "test:ci": "jest --ci --coverage --forceExit"
  }
}
```

- [ ] Go 백엔드 Makefile 생성 (`backend/Makefile`)

```makefile
# backend/Makefile

.PHONY: build run test lint migrate sqlc

# 빌드
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

# 개발 서버 실행
run:
	go run ./cmd/api

# 전체 테스트
test:
	go test ./... -v

# 테스트 (커버리지)
test-coverage:
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# 린트
lint:
	golangci-lint run ./...

# DB 마이그레이션
migrate:
	psql "$$DATABASE_URL" -f db/migrations/001_initial.sql

# sqlc 코드 생성
sqlc:
	sqlc generate

# 전체 초기화
setup: migrate sqlc build
```

- [ ] 프론트엔드 API 클라이언트 테스트 작성

```typescript
// src/shared/api/__tests__/client.test.ts
import { ApiError } from "../client";

describe("API Client", () => {
  it("ApiError includes status and body", () => {
    const err = new ApiError(404, "Not found");
    expect(err.status).toBe(404);
    expect(err.body).toBe("Not found");
    expect(err.message).toContain("404");
  });
});
```

- [ ] 전체 테스트 실행 확인

```bash
# 프론트엔드 테스트
cd /home/dev/code/dev-superbear
npm test -- --passWithNoTests

# Go 백엔드 테스트
cd /home/dev/code/dev-superbear/backend
go test ./... -v
```

- [ ] `.gitignore` 확인/수정

```
# .gitignore
node_modules/
.next/
coverage/
.env.local
.env

# Go
backend/bin/
backend/coverage.*
backend/internal/repository/sqlc/*.go
!backend/internal/repository/sqlc/.gitkeep
```

- [ ] 커밋

```bash
git add jest.config.ts jest.setup.ts package.json tsconfig.json backend/Makefile .gitignore src/shared/api/__tests__/
git commit -m "feat: Jest frontend test infrastructure, Go test setup, and Makefile"
```
