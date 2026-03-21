# E2E 테스트 Worktree 격리 자동화 디자인 스펙

> 날짜: 2026-03-21
> 상태: Draft

## 개요

여러 git worktree에서 동시에 E2E 테스트를 실행할 수 있도록, 백엔드 서버(Docker)의 라이프사이클을 자동 관리하는 `/e2e` 스킬을 만든다.

## 문제

- worktree마다 `docker-compose.test.yml`을 복사하고 포트를 수동 변경해야 함
- 동시 실행 시 포트 충돌 발생
- `playwright.config.ts`의 포트도 하드코딩되어 있어 worktree별 대응 불가
- 서버 띄우기 → 테스트 → 정리 과정이 수동

## 접근법

헬퍼 스크립트 + 스킬 (접근법 B). 셸 스크립트가 Docker 라이프사이클을 관리하고, 스킬이 오케스트레이션한다.

---

## 설계

### 파일 구조

```
.claude/skills/e2e/
  SKILL.md                # 스킬 정의 (워크플로우 오케스트레이션)
  scripts/
    e2e-server.sh         # Docker 라이프사이클 관리 (단독 사용 가능)
```

### 1. `e2e-server.sh` — Docker 라이프사이클 스크립트

#### 명령어

| 명령 | 동작 |
|------|------|
| `e2e-server.sh up` | worktree 감지 → 포트 할당 → `.env.e2e` 생성 → `docker compose up -d` → health check 대기 |
| `e2e-server.sh down` | `docker compose down` → `.env.e2e` 삭제 |
| `e2e-server.sh status` | 현재 컨테이너 상태 + 할당된 포트 출력 |

#### 포트 할당 로직

- worktree 디렉토리명의 `cksum` 해시로 오프셋(1~9) 계산
- main repo(non-worktree)면 오프셋 0 → 기본 포트 사용
- 포트 매핑 (오프셋 N 기준):

| 서비스 | 기본 포트 (오프셋 0) | 오프셋 적용 |
|--------|---------------------|-------------|
| PostgreSQL | 5433 | 5433 + N |
| Redis | 6379 | 6379 + N |
| API | 8080 | 8080 + N×10 |
| Worker | 8081 | 8081 + N×10 |

- 오프셋 적용 후 포트 사용 가능 여부를 `ss` 또는 `lsof`로 검증
- 충돌 시 오프셋을 1씩 증가하며 빈 포트 탐색

#### `.env.e2e` 파일 생성

스크립트가 `up` 실행 시 프로젝트 루트에 `.env.e2e` 생성:

```env
PG_PORT=5434
REDIS_PORT=6380
API_PORT=8090
WORKER_PORT=8091
COMPOSE_PROJECT=superbear-chart-search-modal
```

#### Docker Compose 프로젝트 격리

`-p` (project name) 플래그로 worktree별 완전 격리:

```bash
docker compose -p "${COMPOSE_PROJECT}" \
  --env-file .env.e2e \
  -f docker-compose.test.yml up -d
```

동일한 YAML을 사용하더라도 네트워크, 볼륨, 컨테이너 이름이 모두 분리된다.

#### Health Check

`up` 후 서비스가 준비될 때까지 대기:

1. PostgreSQL: `pg_isready` (docker compose healthcheck 활용)
2. Redis: `redis-cli ping`
3. API: `curl http://localhost:${API_PORT}/health` 또는 TCP 체크

최대 60초 대기, 실패 시 자동으로 `down` 실행 후 에러 리턴.

### 2. `docker-compose.test.yml` 수정

기존 하드코딩 포트를 환경변수화. 기본값은 현재 포트와 동일하여 기존 사용 방식에 영향 없음:

```yaml
services:
  postgres:
    ports:
      - "${PG_PORT:-5433}:5432"    # 기존: "5433:5432"

  redis:
    ports:
      - "${REDIS_PORT:-6379}:6379"  # 기존: "6379:6379"

  api:
    ports:
      - "${API_PORT:-8080}:8080"    # 기존: "8080:8080"
    environment:
      ALLOWED_ORIGINS: http://localhost:${FRONTEND_PORT:-3000},http://localhost:${FRONTEND2_PORT:-3001}

  worker:
    ports:
      - "${WORKER_PORT:-8081}:8081"  # 기존: "8081:8081"
```

컨테이너 간 내부 통신(DATABASE_URL, REDIS_ADDR)은 Docker 네트워크를 사용하므로 변경 불필요.

### 3. `SKILL.md` — 워크플로우 오케스트레이션

#### 스킬 메타데이터

```yaml
---
name: e2e
description: Worktree별 격리된 백엔드 서버를 띄우고 E2E 테스트를 실행하는 자동화 스킬
---
```

#### 워크플로우

```
/e2e 실행:
  Phase 1: 서버 기동
    - scripts/e2e-server.sh up 실행
    - .env.e2e에서 할당된 포트 확인

  Phase 2: E2E 테스트
    - .env.e2e의 포트를 환경변수로 export
    - Skill("playwright-best-practices") 호출
    - 테스트 범위/실행 방법은 playwright-best-practices 스킬에 위임

  Phase 3: 정리 (테스트 성공/실패 무관)
    - scripts/e2e-server.sh down 실행
    - 결과 요약 출력
```

### 4. `.gitignore` 추가

```
.env.e2e
```

worktree별로 생성되는 임시 환경 파일이므로 버전 관리에서 제외.

---

## 기존 worktree별 중복 파일 정리

이 설계가 적용되면 다음 파일들은 불필요:

- `.worktrees/chart-search-modal/docker-compose.e2e.yml` (별도 e2e용 compose)
- `.worktrees/*/docker-compose.test.yml` (main의 것을 공유)

단, worktree에 독자적인 migration이나 서비스 변경이 있으면 해당 worktree의 `docker-compose.test.yml`은 유지.

---

## 제약사항

- `cksum` 해시 기반 오프셋은 0~9 범위로, worktree 10개 이상 동시 실행 시 충돌 가능 → 포트 검증 로직이 fallback 역할
- `playwright-best-practices` 스킬이 환경변수(`API_PORT` 등)를 인식해야 함 → `playwright.config.ts`에서 `process.env` 참조 필요
- `docker-compose.test.yml` 수정은 모든 worktree에 영향 → main에서 한 번 수정 후 worktree에 반영
