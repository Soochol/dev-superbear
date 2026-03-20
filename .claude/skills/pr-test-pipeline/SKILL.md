---
name: pr-test-pipeline
description: Use when the user wants to run tests on PR changes, start/stop Docker test environments, or clean up test containers. Triggers on "PR 테스트", "테스트 돌려줘", "이 브랜치 테스트", "e2e 테스트", "스크린샷 테스트", "lint 돌려줘", "unit test", "테스트 정리", "서버 내려줘", "docker 컨테이너 정리", "run tests", "test this PR", "run e2e", "run playwright", "go test", "백엔드 테스트", "API 테스트", "통합 테스트", "integration test". PR 변경사항을 검증하려는 의도가 보이면 이 스킬이 적합함.
---

# PR Test Pipeline

프롬프트 또는 PR 변경사항에 대한 테스트 전략을 수립하고, worktree별 완전 격리 환경에서 실행한다.

## 모드

**프롬프트 모드** — 인자가 있으면 (`pr-test-pipeline <prompt>`):
해당 프롬프트의 내용을 분석해서 테스트 전략을 수립한다.

**PR 모드** — 인자가 없으면 (`pr-test-pipeline`):
현재 브랜치의 변경사항을 base branch와 비교해서 테스트 전략을 수립한다.

## 테스트 전략 수립

### 1. 분석

**프롬프트 모드:** 프롬프트가 요구하는 변경의 성격을 파악한다 — 어떤 레이어(DB, 백엔드, 프론트엔드, 인프라)에 영향을 주는지, 어떤 기능이 변경되는지.

**PR 모드:** base branch를 감지하고, 변경 파일 목록 + 커밋 히스토리를 분석한다. 변경 파일의 성격(프론트엔드, 백엔드, DB/스키마, API, 설정/인프라, 문서)을 판단한다.

### 2. 프로젝트 감지

프로젝트 루트의 설정 파일을 탐색해서 사용 언어, 패키지 매니저, lint/unit/e2e 도구, Docker 구성을 파악한다. 가정하지 말고, 실제 파일에서 확인한다.

- **폴리글랏:** package.json + go.mod 공존 시 각 언어를 별도 테스트 도메인으로 취급
- **멀티앱:** 변경된 파일이 속한 앱만 테스트 대상

### 3. 전략 출력

다음을 포함하는 테스트 전략을 사용자에게 제시한다:

- **변경 분류**: 프론트엔드 / 백엔드 / DB / 인프라 / 문서
- **테스트 범위**: 실행할 테스트 종류와 대상 파일/모듈
- **격리 요구사항**: 서버 기동 필요 여부, Docker 서비스, 포트 할당
- **실행 순서**: 빠른 테스트부터 (lint → unit → integration → e2e)
- **예상 명령어**: 프로젝트의 실제 설정에서 확인한 테스트 명령어

| 변경 유형 | lint | unit | API | integration | e2e | visual |
|-----------|:----:|:----:|:---:|:-----------:|:---:|:------:|
| 프론트엔드만 | O | O | - | - | O | O |
| 백엔드만 | O | O | O | O | - | - |
| 프론트+백엔드 | O | O | O | O | O | O |
| DB/스키마 | O | O | O | O | O | - |
| 설정/인프라 | O | O | O | O | O | - |
| 문서만 | - | - | - | - | - | - |

사용자 확인 후 실행에 들어간다.

## Worktree 격리 실행

이 스킬의 핵심은 **N개 worktree에서 동시에 테스트할 때 서버를 완전히 독립시키는 것**이다.

### 격리 원리

각 worktree는 고유한 Docker Compose project name (`pr-<worktree-name>`)을 받는다. 컨테이너, 네트워크, 볼륨이 모두 분리되어 서로 간섭하지 않는다. 포트 충돌은 자동 offset으로 회피한다.

### 격리 도구

이 스킬 디렉토리의 `scripts/pr-docker.sh`가 격리를 담당한다:

```
pr-docker.sh up [--port-offset N]   # 격리된 서비스 기동
pr-docker.sh down                   # 이 worktree만 정리
pr-docker.sh down-all               # 전체 PR 환경 정리
pr-docker.sh status                 # 현재 상태 확인
```

- Docker compose 파일을 자동 감지한다 (docker-compose.test.yml 우선)
- worktree 이름에서 project name을 자동 생성한다
- port offset을 자동 계산하거나 수동 지정할 수 있다

### 서버 기동 규칙

- e2e/integration 테스트가 필요하면 서버를 기동한다. lint/unit만이면 건너뛴다.
- Playwright config에 webServer 설정이 있으면 Playwright가 프론트엔드를 기동한다 — 수동 기동 불필요. 단 백엔드/DB는 별도 준비.
- 서버 기동 후 반드시 healthcheck (curl 또는 포트 확인) 통과를 확인한 뒤 테스트에 진입한다. 30초 타임아웃.

### 정리

테스트 완료 후 `pr-docker.sh down`으로 기동한 서비스를 정리한다. 스크린샷, trace 등 결과 파일은 보존.

## 흔한 실수

| 실수 | 해결 |
|------|------|
| 서버 기동 전에 e2e 실행 | 기동 → healthcheck → 테스트 순서 |
| 변경 분석 없이 전체 테스트 | 매핑 테이블 참조 |
| 정리 안 하고 세션 종료 | 항상 정리 실행 |
| 포트 충돌 시 반복 재시도 | `--port-offset N`으로 변경 |
| healthcheck 없이 테스트 진입 | 서버 기동 후 반드시 응답 확인 |
