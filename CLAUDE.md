@AGENTS.md

## 검증 체계

이 프로젝트는 kimoring 기반 검증 시스템을 사용합니다. verify-* 스킬이 코드 규칙을 자동 검증하고, `/manage-skills`로 새 스킬을 점진적으로 추가합니다.

## Skills

| 스킬 | 설명 |
|------|------|
| `/verify-implementation` | 등록된 verify-* 스킬 순차 실행, 통합 검증 보고서 생성 |
| `/manage-skills` | 세션 변경사항 분석 → verify 스킬 생성/업데이트 제안 |
| `/merge-worktree` | worktree 브랜치 squash-merge + 커밋 메시지 자동 생성 |
| `/verify-fsd` | FSD 아키텍처 규칙 검증 (barrel 파일, import 경계, store 위치, 디자인 토큰) |
