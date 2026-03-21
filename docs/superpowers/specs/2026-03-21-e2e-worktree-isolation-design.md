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

헬퍼 스크립트 + 스킬 (접근법 B). 셸 스크립트가 Docker 라이프사이클을 관리하고, 스킬이 오케스트레이션한다.

---

## 아키텍처

### 서비스 분류

| 서비스 | 관리 방식 | 호스트 포트 노출 |
|--------|-----------|-----------------|
| PostgreSQL | **공용** (1개, main에서 기동) | 불필요 (Docker 내부만) |
| Redis | **공용** (1개, main에서 기동) | 불필요 (Docker 내부만) |
| API | **worktree별** 격리 | 불필요 (Docker 내부만) |
| Worker | **worktree별** 격리 | 불필요 (Docker 내부만) |
| Frontend | **worktree별** 격리 | **1개만 노출** (Playwright 진입점) |

### 네트워크 구조

```
[호스트]                        [Docker]
                                 ┌─────────────── 공용 네트워크 ──────────────┐
                                 │  postgres:5432                             │
                                 │  redis:6379                                │
                                 └────────────────────────────────────────────┘
                                        ↑               ↑
Playwright ──→ localhost:XXXX ──→ frontend:3000 → api:8080
                  (유일한                  └─────── worktree별 네트워크 ────────┘
                   호스트 포트)
```

컨테이너 간 통신은 Docker 내부 네트워크를 사용하므로, 호스트에 노출할 포트는 Playwright가 접근하는 **frontend 1개뿐**이다.

---

## 설계

### 파일 구조

```
.claude/skills/e2e/
  SKILL.md                # 스킬 정의 (워크플로우 오케스트레이션)
  scripts/
    e2e-server.sh         # Docker 라이프사이클 관리 (단독 사용 가능)
```

### Docker Compose 파일 분리

```
docker-compose.infra.yml      # 공용 인프라 (postgres, redis) — 새로 생성
docker-compose.test.yml       # worktree별 서비스 (api, worker, frontend) — 기존 파일 수정
```

#### `docker-compose.infra.yml` (신규)

공용 인프라 서비스. 한 번 띄우면 모든 worktree가 공유:

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

`networks.default.name`으로 네트워크 이름을 고정하여, worktree별 compose에서 참조 가능하게 한다.

#### `docker-compose.test.yml` (수정)

worktree별 서비스. 공용 인프라 네트워크에 연결:

```yaml
services:
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgresql://nexus:nexus@postgres:5432/nexus_test?sslmode=disable
      PORT: "8080"
      JWT_SECRET: test-secret
      ALLOWED_ORIGINS: http://frontend:3000
      APP_ENV: development
      REDIS_ADDR: redis:6379
    depends_on: []   # health check는 e2e-server.sh가 사전 확인

  worker:
    build:
      context: ./backend
      dockerfile: Dockerfile
    command: ["/worker"]
    environment:
      DATABASE_URL: postgresql://nexus:nexus@postgres:5432/nexus_test?sslmode=disable
      REDIS_ADDR: redis:6379
      APP_ENV: development

  frontend:
    build:
      context: .
      dockerfile: Dockerfile.frontend   # 신규 생성 필요
    ports:
      - "${E2E_PORT:-3100}:3000"         # 유일한 호스트 노출 포트
    environment:
      NEXT_PUBLIC_API_URL: http://api:8080

networks:
  default:
    name: superbear-e2e-${WORKTREE_NAME:-main}
  infra:
    name: superbear-infra
    external: true
```

API, Worker, Frontend가 `superbear-infra` 네트워크에도 연결되어 postgres/redis에 접근 가능.

### `e2e-server.sh` — Docker 라이프사이클 스크립트

#### 명령어

| 명령 | 동작 |
|------|------|
| `e2e-server.sh up` | 공용 인프라 확인/기동 → worktree명 감지 → E2E_PORT 할당 → worktree별 서비스 기동 → health check 대기 |
| `e2e-server.sh down` | worktree별 서비스만 종료 (공용 인프라는 유지) |
| `e2e-server.sh down --all` | worktree별 서비스 + 공용 인프라 모두 종료 |
| `e2e-server.sh status` | 공용 인프라 + 현재 worktree 서비스 상태 출력 |

#### 포트 할당 로직

관리할 포트가 `E2E_PORT` (frontend 호스트 노출) **1개뿐**:

- main repo: `E2E_PORT=3100` (기본값)
- worktree: 디렉토리명의 `cksum` 해시로 오프셋(1~9) 계산 → `3100 + offset`
- 충돌 시 `ss`로 검증 후 offset 증가
- `.env.e2e`에 기록:

```env
E2E_PORT=3101
WORKTREE_NAME=chart-search-modal
```

#### Docker Compose 실행

```bash
# 1. 공용 인프라 (없으면 띄우기)
docker compose -p superbear-infra -f docker-compose.infra.yml up -d

# 2. worktree별 서비스
docker compose -p "superbear-e2e-${WORKTREE_NAME}" \
  --env-file .env.e2e \
  -f docker-compose.test.yml up -d --build
```

#### Health Check

1. 공용 인프라: `docker compose -p superbear-infra ps`로 healthy 상태 확인
2. API: worktree 네트워크 내에서 `docker compose exec api curl localhost:8080/health`
3. Frontend: `curl http://localhost:${E2E_PORT}`

최대 60초 대기, 실패 시 자동으로 `down` 실행 후 에러 리턴.

### `Dockerfile.frontend` (신규)

```dockerfile
FROM node:22-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
EXPOSE 3000
CMD ["npx", "next", "dev", "--port", "3000"]
```

개발용이므로 `next dev`를 사용. 프로덕션 빌드가 아닌 HMR 지원 개발 서버.

### `playwright.config.ts` 수정

호스트 포트를 환경변수로 참조. `webServer` 설정은 제거 (Docker가 서버를 관리하므로):

```typescript
export default defineConfig({
  // webServer 제거 — Docker가 서버 라이프사이클 관리
  projects: [
    {
      name: "frontend-app",
      use: {
        browserName: "chromium",
        baseURL: `http://localhost:${process.env.E2E_PORT ?? 3100}`,
      },
    },
    // ... 다른 프로젝트도 E2E_PORT 사용
  ],
});
```

### `SKILL.md` — 워크플로우 오케스트레이션

```
/e2e 실행:
  Phase 1: 서버 기동
    - scripts/e2e-server.sh up 실행
    - .env.e2e에서 E2E_PORT 확인

  Phase 2: E2E 테스트
    - E2E_PORT 환경변수 export
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
- `.worktrees/*/docker-compose.test.yml` (main의 것을 공유)
- `playwright.config.ts`의 `webServer` 섹션

---

## 제약사항

- 공용 DB는 test DB이므로 마이그레이션 충돌 시 DB를 재생성하면 됨 (`e2e-server.sh down --all` → `up`)
- `Dockerfile.frontend` 신규 생성 필요
- `playwright-best-practices` 스킬이 `E2E_PORT` 환경변수를 인식해야 함
- worktree별 프론트엔드 Docker 빌드 시간이 추가됨 (캐시 활용으로 완화)
