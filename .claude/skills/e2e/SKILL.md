---
name: e2e
description: Worktree별 격리된 Docker 환경에서 E2E 테스트를 실행하는 자동화 스킬. 백엔드/프론트엔드 서버 기동, 테스트 실행, 정리를 한 번에 처리한다. /e2e 명령어로 실행하거나, E2E 테스트를 돌려야 할 때, worktree에서 서버를 띄워야 할 때 사용한다.
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
