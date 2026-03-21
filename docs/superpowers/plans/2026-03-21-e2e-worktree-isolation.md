# E2E Worktree 격리 자동화 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/e2e` 스킬로 worktree별 Docker 서비스 라이프사이클을 자동 관리하고, E2E 테스트를 격리 실행한다.

**Architecture:** 공용 인프라(postgres, redis)와 worktree별 서비스(api, worker, root-app, frontend)를 분리. 셸 스크립트가 Docker 라이프사이클을 관리하고, 스킬이 오케스트레이션한다. 모든 Docker 파일은 `.claude/skills/e2e/templates/`에서 관리하며, `--project-directory` 플래그로 프로젝트 루트를 참조한다.

**Tech Stack:** Docker Compose, Bash, Playwright, Next.js 16, Go

**Spec:** `docs/superpowers/specs/2026-03-21-e2e-worktree-isolation-design.md`

---

## File Map

### 신규 생성

| 파일 | 역할 |
|------|------|
| `.claude/skills/e2e/SKILL.md` | `/e2e` 스킬 정의 (워크플로우 오케스트레이션) |
| `.claude/skills/e2e/scripts/e2e-server.sh` | Docker 라이프사이클 관리 (up/down/status) |
| `.claude/skills/e2e/templates/docker-compose.infra.yml` | 공용 인프라 (postgres, redis) |
| `.claude/skills/e2e/templates/docker-compose.test.yml` | worktree별 서비스 (api, worker, root-app, frontend) |
| `.claude/skills/e2e/templates/Dockerfile.frontend` | 루트 앱 컨테이너 빌드 |
| `.claude/skills/e2e/templates/Dockerfile.frontend-app` | frontend/ 앱 컨테이너 빌드 |

### 수정

| 파일 | 변경 내용 |
|------|----------|
| `playwright.config.ts` | 포트 환경변수화, `webServer` 제거 |
| `.gitignore` | `.env.e2e` 추가 |
| `e2e/chart-api.spec.ts` | 하드코딩 URL → 환경변수 |
| `e2e/chart-watchlist.spec.ts` | 하드코딩 URL → 환경변수 |
| `e2e/search-sse.spec.ts` | 하드코딩 URL → 환경변수 |
| `e2e/monitoring-api.spec.ts` | 하드코딩 worker URL → 환경변수 |

---

## Task 1: Docker Compose 템플릿 생성

**Files:**
- Create: `.claude/skills/e2e/templates/docker-compose.infra.yml`
- Create: `.claude/skills/e2e/templates/docker-compose.test.yml`

- [ ] **Step 1: 디렉토리 구조 생성**

```bash
mkdir -p .claude/skills/e2e/templates .claude/skills/e2e/scripts
```

- [ ] **Step 2: `docker-compose.infra.yml` 작성**

공용 인프라 (postgres + redis). 네트워크 이름을 `superbear-infra`로 고정:

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

- [ ] **Step 3: `docker-compose.test.yml` 작성**

worktree별 서비스. `infra` 네트워크를 external로 참조:

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
    ports:
      - "${E2E_PORT_WORKER:-3400}:8081"
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

- [ ] **Step 4: 커밋**

```bash
git add .claude/skills/e2e/templates/docker-compose.infra.yml .claude/skills/e2e/templates/docker-compose.test.yml
git commit -m "feat(e2e): add Docker Compose templates for infra and worktree services"
```

---

## Task 2: Frontend Dockerfile 템플릿 생성

**Files:**
- Create: `.claude/skills/e2e/templates/Dockerfile.frontend`
- Create: `.claude/skills/e2e/templates/Dockerfile.frontend-app`

- [ ] **Step 1: 루트 앱 Dockerfile 작성**

루트 앱은 프로젝트 루트에서 빌드. Next.js 16 dev 서버를 포트 3000으로 실행. `node_modules/next/dist/docs/`에서 Next.js 16의 올바른 dev 서버 실행 방법을 확인한 후 작성:

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
EXPOSE 3000
CMD ["npx", "next", "dev", "--port", "3000"]
```

주의: Next.js 16의 실행 방법이 다를 수 있으므로, 반드시 `node_modules/next/dist/docs/`의 문서를 확인하고 CMD를 조정할 것.

- [ ] **Step 2: frontend/ 앱 Dockerfile 작성**

frontend 앱은 `frontend/` 디렉토리에서 빌드. 포트 3001:

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
EXPOSE 3001
CMD ["npx", "next", "dev", "--port", "3001"]
```

- [ ] **Step 3: 로컬에서 빌드 테스트**

```bash
# 루트 앱 Dockerfile 빌드 테스트
docker build -f .claude/skills/e2e/templates/Dockerfile.frontend -t superbear-root-app-test .
# 성공 시 이미지가 생성됨. 컨테이너 실행은 불필요 (compose에서 할 예정)

# frontend 앱 Dockerfile 빌드 테스트
docker build -f .claude/skills/e2e/templates/Dockerfile.frontend-app -t superbear-frontend-test ./frontend
```

Expected: 두 빌드 모두 성공

- [ ] **Step 4: 커밋**

```bash
git add .claude/skills/e2e/templates/Dockerfile.frontend .claude/skills/e2e/templates/Dockerfile.frontend-app
git commit -m "feat(e2e): add Dockerfiles for root-app and frontend-app"
```

---

## Task 3: `e2e-server.sh` 스크립트 작성

**Files:**
- Create: `.claude/skills/e2e/scripts/e2e-server.sh`

- [ ] **Step 1: 스크립트 뼈대 작성**

worktree 감지, 포트 할당, `.env.e2e` 생성 로직:

```bash
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}/../templates"
PROJECT_DIR="$(git rev-parse --show-toplevel)"

# --- worktree 감지 ---
detect_worktree() {
  local git_dir
  git_dir="$(git rev-parse --git-dir)"
  if [[ "$git_dir" == *"/worktrees/"* ]]; then
    basename "$git_dir"
  else
    echo "main"
  fi
}

# --- 포트 할당 ---
allocate_ports() {
  local name="$1"
  local offset=0

  if [[ "$name" != "main" ]]; then
    offset=$(( $(echo -n "$name" | cksum | awk '{print $1}') % 9 + 1 ))
  fi

  # 포트 충돌 검증 + fallback
  local max_attempts=20
  while (( offset < max_attempts )); do
    local port_root=$(( 3100 + offset ))
    local port_front=$(( 3200 + offset ))
    local port_api=$(( 3300 + offset ))
    local port_worker=$(( 3400 + offset ))

    # 4개 포트 모두 사용 가능한지 확인
    if ! ss -tlnp 2>/dev/null | grep -qE ":($port_root|$port_front|$port_api|$port_worker) " ; then
      echo "${port_root} ${port_front} ${port_api} ${port_worker}"
      return 0
    fi
    offset=$(( offset + 1 ))
  done

  echo "ERROR: No available ports found after $max_attempts attempts" >&2
  return 1
}

# --- .env.e2e 생성 ---
write_env() {
  local worktree_name="$1"
  local port_root="$2"
  local port_front="$3"
  local port_api="$4"
  local port_worker="$5"
  local env_file="${PROJECT_DIR}/.env.e2e"

  cat > "$env_file" <<EOF
E2E_PORT_ROOT=${port_root}
E2E_PORT_FRONT=${port_front}
E2E_PORT_API=${port_api}
E2E_PORT_WORKER=${port_worker}
WORKTREE_NAME=${worktree_name}
EOF
  echo "Generated ${env_file}"
}

# --- docker compose 헬퍼 ---
compose_infra() {
  docker compose \
    --project-directory "${PROJECT_DIR}" \
    -f "${TEMPLATE_DIR}/docker-compose.infra.yml" \
    -p superbear-infra \
    "$@"
}

compose_worktree() {
  local worktree_name="$1"
  shift
  docker compose \
    --project-directory "${PROJECT_DIR}" \
    -f "${TEMPLATE_DIR}/docker-compose.test.yml" \
    -p "superbear-e2e-${worktree_name}" \
    --env-file "${PROJECT_DIR}/.env.e2e" \
    "$@"
}

# --- health check ---
wait_for_healthy() {
  local url="$1"
  local label="$2"
  local timeout=60
  local elapsed=0

  echo -n "Waiting for ${label}..."
  while (( elapsed < timeout )); do
    if curl -sf "$url" > /dev/null 2>&1; then
      echo " ready"
      return 0
    fi
    sleep 2
    elapsed=$(( elapsed + 2 ))
    echo -n "."
  done

  echo " TIMEOUT"
  return 1
}

# --- 명령어 ---
cmd_up() {
  local worktree_name
  worktree_name="$(detect_worktree)"
  echo "Worktree: ${worktree_name}"

  # .env.e2e가 있으면 기존 포트 재사용
  local env_file="${PROJECT_DIR}/.env.e2e"
  if [[ -f "$env_file" ]]; then
    echo "Reusing existing .env.e2e"
    source "$env_file"
  else
    local ports
    ports="$(allocate_ports "$worktree_name")"
    read -r port_root port_front port_api port_worker <<< "$ports"
    write_env "$worktree_name" "$port_root" "$port_front" "$port_api" "$port_worker"
    E2E_PORT_ROOT="$port_root"
    E2E_PORT_FRONT="$port_front"
    E2E_PORT_API="$port_api"
    E2E_PORT_WORKER="$port_worker"
  fi

  # 1. 공용 인프라 (이미 떠있으면 skip)
  if ! docker compose -p superbear-infra ps --status running 2>/dev/null | grep -q "postgres"; then
    echo "Starting shared infra..."
    compose_infra up -d
    compose_infra exec postgres sh -c 'until pg_isready -U nexus -d nexus_test; do sleep 1; done'
    echo "Shared infra ready"
  else
    echo "Shared infra already running"
  fi

  # 2. worktree별 서비스
  echo "Starting worktree services (ports: root=${E2E_PORT_ROOT}, front=${E2E_PORT_FRONT}, api=${E2E_PORT_API}, worker=${E2E_PORT_WORKER})..."
  compose_worktree "$worktree_name" up -d --build

  # 3. health check
  wait_for_healthy "http://localhost:${E2E_PORT_API}/health" "API" || {
    echo "Health check failed, tearing down..."
    cmd_down
    exit 1
  }
  wait_for_healthy "http://localhost:${E2E_PORT_ROOT}" "Root App" || {
    echo "Health check failed, tearing down..."
    cmd_down
    exit 1
  }
  wait_for_healthy "http://localhost:${E2E_PORT_FRONT}" "Frontend" || {
    echo "Health check failed, tearing down..."
    cmd_down
    exit 1
  }

  echo ""
  echo "=== E2E Environment Ready ==="
  echo "Root App:  http://localhost:${E2E_PORT_ROOT}"
  echo "Frontend:  http://localhost:${E2E_PORT_FRONT}"
  echo "API:       http://localhost:${E2E_PORT_API}"
  echo "Worker:    http://localhost:${E2E_PORT_WORKER}"
  echo "============================="
}

cmd_down() {
  local worktree_name
  worktree_name="$(detect_worktree)"
  local all=false

  if [[ "${1:-}" == "--all" ]]; then
    all=true
  fi

  echo "Stopping worktree services (${worktree_name})..."
  compose_worktree "$worktree_name" down --remove-orphans 2>/dev/null || true

  if $all; then
    echo "Stopping shared infra..."
    compose_infra down --remove-orphans 2>/dev/null || true
  fi

  # .env.e2e 삭제
  rm -f "${PROJECT_DIR}/.env.e2e"
  echo "Done"
}

cmd_status() {
  echo "=== Shared Infra ==="
  compose_infra ps 2>/dev/null || echo "Not running"
  echo ""
  echo "=== Worktree Services ($(detect_worktree)) ==="
  local worktree_name
  worktree_name="$(detect_worktree)"
  compose_worktree "$worktree_name" ps 2>/dev/null || echo "Not running"

  if [[ -f "${PROJECT_DIR}/.env.e2e" ]]; then
    echo ""
    echo "=== Ports ==="
    cat "${PROJECT_DIR}/.env.e2e"
  fi
}

# --- main ---
case "${1:-}" in
  up)     cmd_up ;;
  down)   cmd_down "${2:-}" ;;
  status) cmd_status ;;
  *)
    echo "Usage: e2e-server.sh {up|down [--all]|status}"
    exit 1
    ;;
esac
```

- [ ] **Step 2: 실행 권한 부여**

```bash
chmod +x .claude/skills/e2e/scripts/e2e-server.sh
```

- [ ] **Step 3: 스크립트 단독 테스트 — `status` 명령**

```bash
.claude/skills/e2e/scripts/e2e-server.sh status
```

Expected: "Not running" 출력 (아직 서비스를 띄우지 않았으므로)

- [ ] **Step 4: 커밋**

```bash
git add .claude/skills/e2e/scripts/e2e-server.sh
git commit -m "feat(e2e): add e2e-server.sh lifecycle management script"
```

---

## Task 4: 통합 테스트 — Docker 서비스 기동/종료

**Files:**
- 신규 생성 없음 (Task 1-3에서 만든 파일 검증)

- [ ] **Step 1: `e2e-server.sh up` 실행**

```bash
.claude/skills/e2e/scripts/e2e-server.sh up
```

Expected:
- `superbear-infra` 네트워크에 postgres, redis 기동
- `superbear-e2e-main` 네트워크에 api, worker, root-app, frontend 기동
- Health check 통과
- `.env.e2e` 파일 생성됨

- [ ] **Step 2: 서비스 상태 확인**

```bash
.claude/skills/e2e/scripts/e2e-server.sh status
```

Expected: 모든 서비스 running, 포트 정보 출력

- [ ] **Step 3: 각 서비스 접근 테스트**

```bash
# .env.e2e에서 포트 읽기
source .env.e2e

# API health
curl -sf http://localhost:${E2E_PORT_API}/health

# Root App
curl -sf http://localhost:${E2E_PORT_ROOT} | head -5

# Frontend
curl -sf http://localhost:${E2E_PORT_FRONT} | head -5
```

Expected: 모든 서비스 응답 성공

- [ ] **Step 4: `e2e-server.sh down` 실행**

```bash
.claude/skills/e2e/scripts/e2e-server.sh down
```

Expected: worktree 서비스만 종료, 공용 인프라는 유지

- [ ] **Step 5: 공용 인프라 확인 후 전체 종료**

```bash
# 인프라가 아직 떠있는지 확인
docker compose -p superbear-infra ps

# 전체 종료
.claude/skills/e2e/scripts/e2e-server.sh down --all
```

Expected: 모든 서비스 종료

- [ ] **Step 6: 문제 발견 시 수정 후 커밋**

발견된 이슈를 수정하고 커밋. 이슈 없으면 이 스텝은 skip.

---

## Task 5: `playwright.config.ts` 수정

**Files:**
- Modify: `playwright.config.ts`

- [ ] **Step 1: `webServer` 제거 + 포트 환경변수화**

`playwright.config.ts`를 다음과 같이 수정:

```typescript
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 30000,
  expect: {
    timeout: 5000,
  },
  retries: 0,
  use: {
    headless: true,
    screenshot: "only-on-failure",
  },
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

- [ ] **Step 2: 커밋**

```bash
git add playwright.config.ts
git commit -m "refactor(e2e): use env vars for ports in playwright config"
```

---

## Task 6: E2E 테스트 파일 하드코딩 URL 수정

**Files:**
- Modify: `e2e/chart-api.spec.ts`
- Modify: `e2e/chart-watchlist.spec.ts`
- Modify: `e2e/search-sse.spec.ts`
- Modify: `e2e/monitoring-api.spec.ts`

- [ ] **Step 1: `e2e/chart-api.spec.ts` 수정**

```typescript
// 변경 전:
const BACKEND_URL = "http://localhost:8080";

// 변경 후:
const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;
```

- [ ] **Step 2: `e2e/chart-watchlist.spec.ts` 수정**

```typescript
// 변경 전:
const BACKEND_URL = "http://localhost:8080";

// 변경 후:
const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;
```

- [ ] **Step 3: `e2e/search-sse.spec.ts` 수정**

```typescript
// 변경 전:
test.use({ baseURL: "http://localhost:3000" });

// 변경 후:
test.use({ baseURL: `http://localhost:${process.env.E2E_PORT_ROOT ?? 3100}` });
```

- [ ] **Step 4: `e2e/monitoring-api.spec.ts` 수정**

```typescript
// 변경 전:
const res = await request.get("http://localhost:8081/api/health/workers");

// 변경 후:
const res = await request.get(`http://localhost:${process.env.E2E_PORT_WORKER ?? 3400}/api/health/workers`);
```

- [ ] **Step 5: 커밋**

```bash
git add e2e/chart-api.spec.ts e2e/chart-watchlist.spec.ts e2e/search-sse.spec.ts e2e/monitoring-api.spec.ts
git commit -m "refactor(e2e): replace hardcoded URLs with env vars"
```

---

## Task 7: `.gitignore` 업데이트

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: `.env.e2e` 추가**

`.gitignore`의 `# Environment` 섹션에 이미 `.env*` 패턴이 있는지 확인. 현재 `.env*`로 전체 차단되어 있으므로 `.env.e2e`는 이미 무시됨 — 추가 작업 불필요.

확인:
```bash
echo ".env.e2e" | git check-ignore --stdin
```

Expected: `.env.e2e` 출력 (이미 무시되고 있음)

---

## Task 8: SKILL.md 작성

**Files:**
- Create: `.claude/skills/e2e/SKILL.md`

- [ ] **Step 1: 스킬 파일 작성**

```markdown
---
name: e2e
description: Worktree별 격리된 Docker 환경에서 E2E 테스트를 실행하는 자동화 스킬. 백엔드/프론트엔드 서버 기동, 테스트 실행, 정리를 한 번에 처리한다.
---

# E2E Test Runner

Worktree별 격리된 Docker 환경에서 E2E 테스트를 자동 실행한다.

## Current Context

- Git dir: `!git rev-parse --git-dir`
- Current branch: `!git branch --show-current`
- Working tree status: `!git status --short`

## Instructions

### Phase 1: 서버 기동

1. 스킬 디렉토리의 `scripts/e2e-server.sh up`을 실행한다.
   경로: `.claude/skills/e2e/scripts/e2e-server.sh`

2. `.env.e2e` 파일에서 할당된 포트를 확인하고 사용자에게 출력한다.

3. 기동에 실패하면 에러 메시지를 보여주고 중단한다.

### Phase 2: E2E 테스트

1. `.env.e2e`의 환경변수를 export한다:
   ```bash
   export $(cat .env.e2e | xargs)
   ```

2. Skill("playwright-best-practices")을 호출하여 E2E 테스트를 실행한다.
   테스트 범위와 실행 방법은 해당 스킬에 위임한다.

### Phase 3: 정리

테스트 성공/실패와 무관하게 반드시 실행:

1. `scripts/e2e-server.sh down`을 실행하여 worktree별 서비스를 종료한다.
   공용 인프라(postgres, redis)는 유지된다.

2. 테스트 결과 요약을 출력한다.

## Notes

- 공용 인프라(postgres, redis)를 포함하여 전체 종료하려면: `scripts/e2e-server.sh down --all`
- 현재 상태 확인: `scripts/e2e-server.sh status`
- 스크립트는 단독으로도 사용 가능 (스킬 밖에서 직접 호출)
```

- [ ] **Step 2: 커밋**

```bash
git add .claude/skills/e2e/SKILL.md
git commit -m "feat(e2e): add /e2e skill for automated E2E test workflow"
```

---

## Task 9: E2E 통합 검증

**Files:**
- 신규 생성 없음 (전체 파이프라인 검증)

- [ ] **Step 1: 전체 환경 기동**

```bash
.claude/skills/e2e/scripts/e2e-server.sh up
```

Expected: 모든 서비스 기동 + health check 통과

- [ ] **Step 2: 환경변수 export 후 Playwright 실행**

```bash
export $(cat .env.e2e | xargs)
npx playwright test --project=monitoring-api
```

Expected: monitoring-api 테스트가 `E2E_PORT_API` 포트로 연결하여 실행됨

- [ ] **Step 3: UI 테스트 실행**

```bash
npx playwright test --project=root-app
```

Expected: root-app 테스트가 `E2E_PORT_ROOT` 포트로 연결하여 실행됨

- [ ] **Step 4: 정리**

```bash
.claude/skills/e2e/scripts/e2e-server.sh down --all
```

Expected: 모든 서비스 + 인프라 종료

- [ ] **Step 5: 발견된 이슈 수정 후 최종 커밋**

이슈 수정 후 커밋. 이슈 없으면 skip.
