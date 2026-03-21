# E2E 테스트 Worktree 격리 자동화 디자인 스펙

> 날짜: 2026-03-21
> 상태: Draft

## 개요

여러 git worktree에서 동시에 E2E 테스트를 실행할 수 있도록, Docker 서비스 라이프사이클을 자동 관리하는 `/e2e` 스킬을 만든다.

## 문제

- worktree마다 `docker-compose.test.yml`을 복사하고 포트를 수동 변경해야 함
- 동시 실행 시 포트 충돌 발생
- `playwright.config.ts`의 포트가 하드코딩되어 worktree별 대응 불가
- 서버 띄우기 → 테스트 → 정리 과정이 수동
- 프론트엔드가 호스트에서 직접 실행되어 포트 관리 대상이 늘어남

## 접근법

헬퍼 스크립트 + 스킬 (접근법 B). 셸 스크립트가 Docker 라이프사이클을 관리하고, 스킬이 오케스트레이션한다. 모든 Docker 관련 파일은 스킬 디렉토리에서 관리하며, 프로젝트 루트에 복사하지 않는다.

---

## 아키텍처

### 서비스 분류

| 서비스 | 관리 방식 | 호스트 포트 노출 |
|--------|-----------|-----------------|
| PostgreSQL | **공용** (1개, 최초 실행 시 기동) | 불필요 (Docker 내부만) |
| Redis | **공용** (1개, 최초 실행 시 기동) | 불필요 (Docker 내부만) |
| API | **worktree별** 격리 | **1개 노출** (monitoring-api 테스트용) |
| Worker | **worktree별** 격리 | **1개 노출** (worker health 테스트용) |
| Root App | **worktree별** 격리 | **1개 노출** (landing, chart, case 테스트) |
| Frontend App | **worktree별** 격리 | **1개 노출** (search 테스트) |

### 네트워크 구조

```
[호스트]                            [Docker]
                                     ┌──────── 공용 네트워크 (superbear-infra) ────────┐
                                     │  postgres:5432                                  │
                                     │  redis:6379                                     │
                                     └─────────────────────────────────────────────────┘
                                            ↑                    ↑
Playwright → localhost:E2E_PORT_ROOT  → root-app:3000  →  api:8080  ← localhost:E2E_PORT_API
Playwright → localhost:E2E_PORT_FRONT → frontend:3001      ↑
                                        └─── worktree별 네트워크 (superbear-e2e-XX) ───┘
```

worktree당 호스트 노출 포트 **4개**:

| 환경변수 | 기본값 (오프셋 0) | 용도 |
|----------|------------------|------|
| `E2E_PORT_ROOT` | 3100 | 루트 앱 (landing, chart, monitoring-visual 테스트) |
| `E2E_PORT_FRONT` | 3200 | frontend 앱 (search 테스트) |
| `E2E_PORT_API` | 3300 | API 직접 접근 (monitoring-api 테스트) |
| `E2E_PORT_WORKER` | 3400 | Worker 직접 접근 (worker health 테스트) |

---

## 설계

### 파일 구조

```
.claude/skills/e2e/
  SKILL.md                          # 스킬 정의 (워크플로우 오케스트레이션)
  scripts/
    e2e-server.sh                   # Docker 라이프사이클 관리 (단독 사용 가능)
  templates/
    docker-compose.infra.yml        # 공용 인프라 (postgres, redis)
    docker-compose.test.yml         # worktree별 서비스 (api, worker, root-app, frontend)
    Dockerfile.frontend             # 루트 앱용
    Dockerfile.frontend-app         # frontend/ 앱용
```

모든 Docker 관련 파일은 스킬 디렉토리의 `templates/`에서 관리. 프로젝트 루트에 복사하지 않고, `--project-directory`와 `-f` 플래그로 직접 참조한다.

### Docker Compose 템플릿

#### `templates/docker-compose.infra.yml`

공용 인프라 서비스. 최초 `up` 시 한 번 기동되면 모든 worktree가 공유:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: nexus
      POSTGRES_PASSWORD: nexus
      POSTGRES_DB: nexus_test
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nexus -d nexus_test"]
      interval: 2s
      timeout: 5s
      retries: 10
    volumes:
      - ./backend/db/migrations:/migrations
      - ./backend/db/init-test-db.sh:/docker-entrypoint-initdb.d/init-test-db.sh
    tmpfs:
      - /var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 2s
      timeout: 5s
      retries: 10
    tmpfs:
      - /data

networks:
  default:
    name: superbear-infra
```

`networks.default.name`으로 네트워크 이름을 고정하여, worktree별 compose에서 참조 가능.

#### `templates/docker-compose.test.yml`

worktree별 서비스. 공용 인프라 네트워크에 연결:

```yaml
services:
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "${E2E_PORT_API:-3300}:8080"
    environment:
      DATABASE_URL: postgresql://nexus:nexus@postgres:5432/nexus_test?sslmode=disable
      PORT: "8080"
      JWT_SECRET: test-secret
      ALLOWED_ORIGINS: http://localhost:${E2E_PORT_ROOT:-3100},http://localhost:${E2E_PORT_FRONT:-3200}
      APP_ENV: development
      REDIS_ADDR: redis:6379
      KIS_APP_KEY: ${KIS_APP_KEY:-}
      KIS_APP_SECRET: ${KIS_APP_SECRET:-}
    networks:
      - default
      - infra

  worker:
    build:
      context: ./backend
      dockerfile: Dockerfile
    command: ["/worker"]
    environment:
      DATABASE_URL: postgresql://nexus:nexus@postgres:5432/nexus_test?sslmode=disable
      REDIS_ADDR: redis:6379
      APP_ENV: development
    networks:
      - default
      - infra

  root-app:
    build:
      context: .
      dockerfile: .claude/skills/e2e/templates/Dockerfile.frontend
    ports:
      - "${E2E_PORT_ROOT:-3100}:3000"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:${E2E_PORT_API:-3300}
      API_URL: http://api:8080
    networks:
      - default

  frontend:
    build:
      context: ./frontend
      dockerfile: ../.claude/skills/e2e/templates/Dockerfile.frontend-app
    ports:
      - "${E2E_PORT_FRONT:-3200}:3001"
    environment:
      NEXT_PUBLIC_API_URL: http://localhost:${E2E_PORT_API:-3300}
      API_URL: http://api:8080
    networks:
      - default

networks:
  default:
    name: superbear-e2e-${WORKTREE_NAME:-main}
  infra:
    name: superbear-infra
    external: true
```

서비스에 `networks: [default, infra]`를 명시하여 공용 인프라(postgres, redis)에 접근 가능.

### `e2e-server.sh` — Docker 라이프사이클 스크립트

#### 명령어

| 명령 | 동작 |
|------|------|
| `e2e-server.sh up` | 공용 인프라 확인/기동 → worktree명 감지 → 포트 할당 → `.env.e2e` 생성 → worktree별 서비스 기동 → health check 대기 |
| `e2e-server.sh down` | worktree별 서비스만 종료 (공용 인프라는 유지) |
| `e2e-server.sh down --all` | worktree별 서비스 + 공용 인프라 모두 종료 |
| `e2e-server.sh status` | 공용 인프라 + 현재 worktree 서비스 상태 출력 |

#### 템플릿 참조 방식

스크립트는 자신의 위치를 기준으로 templates 디렉토리를 찾고, `--project-directory`로 빌드 컨텍스트를 프로젝트 루트로 지정:

```bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}/../templates"
PROJECT_DIR="$(git rev-parse --show-toplevel)"

# 공용 인프라
docker compose \
  --project-directory "${PROJECT_DIR}" \
  -f "${TEMPLATE_DIR}/docker-compose.infra.yml" \
  -p superbear-infra \
  up -d

# worktree별 서비스
docker compose \
  --project-directory "${PROJECT_DIR}" \
  -f "${TEMPLATE_DIR}/docker-compose.test.yml" \
  -p "superbear-e2e-${WORKTREE_NAME}" \
  --env-file "${PROJECT_DIR}/.env.e2e" \
  up -d --build
```

프로젝트 루트에 Docker 파일을 복사하지 않는다. `.env.e2e`만 프로젝트 루트에 생성.

#### worktree 감지

```bash
GIT_DIR="$(git rev-parse --git-dir)"
if [[ "$GIT_DIR" == *"/worktrees/"* ]]; then
  WORKTREE_NAME="$(basename "$GIT_DIR")"
else
  WORKTREE_NAME="main"
fi
```

#### 포트 할당 로직

worktree당 호스트 포트 3개를 할당:

- main repo: 오프셋 0 → `E2E_PORT_ROOT=3100`, `E2E_PORT_FRONT=3200`, `E2E_PORT_API=3300`
- worktree: 디렉토리명의 `cksum` 해시로 오프셋(1~9) 계산
- 충돌 시 `ss`로 검증 후 오프셋 증가 (최대 20까지 탐색, 초과 시 에러)

#### `.env.e2e` 생성

`.env.e2e`가 이미 존재하면 기존 포트 재사용. 없으면 새로 생성:

```env
E2E_PORT_ROOT=3101
E2E_PORT_FRONT=3201
E2E_PORT_API=3301
WORKTREE_NAME=chart-search-modal
```

#### Health Check

1. 공용 인프라: `docker compose -p superbear-infra ps`로 healthy 상태 확인
2. API: `curl -sf http://localhost:${E2E_PORT_API}/health`
3. Root App: `curl -sf http://localhost:${E2E_PORT_ROOT}`
4. Frontend: `curl -sf http://localhost:${E2E_PORT_FRONT}`

최대 60초 대기, 실패 시 자동으로 `down` 실행 후 에러 리턴.

### `playwright.config.ts` 수정

환경변수로 포트 참조. `webServer` 설정은 제거 (Docker가 서버를 관리):

```typescript
export default defineConfig({
  testDir: "./e2e",
  timeout: 30000,
  expect: { timeout: 5000 },
  retries: 0,
  use: {
    headless: true,
    screenshot: "only-on-failure",
  },
  // webServer 제거 — Docker가 서버 라이프사이클 관리
  projects: [
    {
      name: "root-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}`,
      },
      testMatch: /landing\.spec\.ts/,
    },
    {
      name: "frontend-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_FRONT ?? 3200}`,
      },
      testMatch: /search.*\.spec\.ts/,
    },
    {
      name: "chart-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}`,
      },
      testMatch: /chart.*\.spec\.ts/,
    },
    {
      name: "monitoring-api",
      use: {
        baseURL: `http://localhost:${process.env.E2E_PORT_API ?? 3300}`,
      },
      testMatch: /monitoring-api\.spec\.ts/,
    },
    {
      name: "case-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}`,
      },
      testMatch: /monitoring-visual\.spec\.ts/,
    },
  ],
});
```

### `SKILL.md` — 워크플로우 오케스트레이션

```
/e2e 실행:
  Phase 1: 서버 기동
    - scripts/e2e-server.sh up 실행
    - .env.e2e에서 할당된 포트 확인 및 출력

  Phase 2: E2E 테스트
    - .env.e2e의 포트를 환경변수로 export
    - Skill("playwright-best-practices") 호출
    - 테스트 범위/실행 방법은 해당 스킬에 위임

  Phase 3: 정리 (테스트 성공/실패 무관)
    - scripts/e2e-server.sh down 실행
    - 결과 요약 출력
```

### `.gitignore` 추가

```
.env.e2e
```

---

## 기존 파일 정리

이 설계 적용 후 불필요해지는 파일:

- `.worktrees/chart-search-modal/docker-compose.e2e.yml`
- `.worktrees/*/docker-compose.test.yml` (스킬 templates로 대체)
- `playwright.config.ts`의 `webServer` 섹션

기존 `docker-compose.test.yml`은 삭제하지 않고 유지해도 무방. `/e2e` 스킬은 templates 디렉토리의 파일만 사용하므로 충돌 없음.

---

## 제약사항

- 공용 DB는 test DB(tmpfs)이므로 마이그레이션 충돌 시 `e2e-server.sh down --all` → `up`으로 재생성
- worktree별 프론트엔드 Docker 빌드 시간 추가 (layer 캐시로 완화)
- `cksum` 해시 오프셋 범위 1~9, 충돌 시 최대 20까지 탐색 후 에러
- `KIS_APP_KEY`, `KIS_APP_SECRET`은 환경변수 통과 (`${KIS_APP_KEY:-}`) — 미설정 시 관련 테스트 스킵
