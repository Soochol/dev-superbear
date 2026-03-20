-- name: ListActiveMonitorBlocks :many
-- 스케줄러 동기화: LIVE 케이스의 enabled MonitorBlock 조회
SELECT mb.id AS monitor_block_id,
       c.id  AS case_id,
       c.pipeline_id,
       c.symbol,
       mb.cron,
       ab.instruction AS block_instruction,
       ab.allowed_tools
FROM monitor_blocks mb
JOIN cases c ON c.id = mb.case_id
JOIN agent_blocks ab ON ab.id = mb.block_id
WHERE mb.enabled = true AND c.status = 'LIVE';

-- name: GetCaseEventSnapshot :one
-- 에이전트 핸들러: 케이스 이벤트 스냅샷 조회
SELECT id, event_snapshot
FROM cases
WHERE id = $1;

-- name: ListLiveCases :many
-- DSL 폴러: LIVE 상태 + DSL 폴링 활성 케이스 목록
SELECT id, symbol, success_script, failure_script, event_snapshot
FROM cases
WHERE status = 'LIVE' AND dsl_polling_enabled = true;

-- name: ListUntriggeredAlertsByCase :many
-- DSL 폴러: 미트리거 가격 알림 조회
SELECT id, condition, label
FROM price_alerts
WHERE case_id = $1 AND triggered = false;

-- name: DisableMonitorBlocksByCase :exec
-- 라이프사이클: 케이스의 모든 모니터 블록 비활성화
UPDATE monitor_blocks SET enabled = false WHERE case_id = $1;

-- name: UpdateMonitorBlockEnabled :exec
-- 서비스: 개별 모니터 블록 활성/비활성
UPDATE monitor_blocks SET enabled = $2 WHERE id = $1;

-- name: UpdateMonitorBlocksEnabledByCase :exec
-- 서비스: 케이스 전체 모니터 블록 활성/비활성
UPDATE monitor_blocks SET enabled = $2 WHERE case_id = $1;

-- name: UpdateCaseDSLPollingEnabled :exec
-- 서비스: DSL 폴링 활성/비활성 토글
UPDATE cases SET dsl_polling_enabled = $2 WHERE id = $1;

-- name: ListMonitorBlocksByCase :many
-- 서비스: 케이스별 모니터 블록 목록 조회
SELECT mb.id, mb.enabled, mb.cron, mb.last_executed_at, ab.instruction
FROM monitor_blocks mb
JOIN agent_blocks ab ON ab.id = mb.block_id
WHERE mb.case_id = $1;

-- name: UpdateMonitorBlockLastExecuted :exec
-- 에이전트 핸들러: 마지막 실행 시간 업데이트
UPDATE monitor_blocks SET last_executed_at = $2 WHERE id = $1;
