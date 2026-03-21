---
name: e2e
description: Worktree별 격리된 Docker 환경에서 E2E 테스트를 실행하는 자동화 스킬. 백엔드/프론트엔드 서버 기동, 테스트 실행, 정리를 한 번에 처리한다. /e2e 명령어로 실행하거나, E2E 테스트를 돌려야 할 때, worktree에서 서버를 띄워야 할 때 사용한다.
hooks:
  Stop:
    - matcher: ""
      hooks:
        - type: command
          command: "bash .claude/skills/e2e/scripts/e2e-server.sh down"
          timeout: 30000
---

# E2E Test Runner

Worktree별 완전 격리된 Docker 환경(postgres, redis, api, worker, frontend)에서 E2E 테스트를 자동 실행한다. 각 worktree는 독립된 DB와 서비스를 가지며, 다른 worktree와 간섭하지 않는다.

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
   (실패 시 trap handler가 자동으로 컨테이너를 정리한다)

### Phase 2: E2E 테스트

1. `.env.e2e`의 환경변수를 export한다:
   ```bash
   export $(cat .env.e2e | xargs)
   ```

2. Skill("playwright-best-practices")을 호출하여 E2E 테스트를 실행한다.
   테스트 범위와 실행 방법은 해당 스킬에 위임한다.

### Phase 3: 결과 보고

테스트 결과 요약을 출력한다.

> **정리(down)는 Stop hook이 세션 종료 시 자동 실행한다.** 수동 정리가 필요하면 `scripts/e2e-server.sh down`을 직접 호출한다.

## Notes

- 각 worktree는 독립된 postgres/redis/api/worker/frontend를 가진다. 다른 worktree와 DB나 네트워크를 공유하지 않는다.
- 현재 상태 확인: `scripts/e2e-server.sh status`
- 스크립트는 단독으로도 사용 가능 (스킬 밖에서 직접 호출)
- 프로세스 크래시 시 trap handler가 자동 정리하며, 다음 실행 시 orphan 컨테이너도 label 기반으로 감지/제거된다.
