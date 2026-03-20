package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// LifecycleHandler handles monitor:lifecycle tasks.
type LifecycleHandler struct {
	queries          *sqlc.Queries
	schedulerManager *SchedulerManager
}

func NewLifecycleHandler(pool *pgxpool.Pool, sm *SchedulerManager) *LifecycleHandler {
	return &LifecycleHandler{queries: sqlc.New(pool), schedulerManager: sm}
}

func (h *LifecycleHandler) HandleLifecycle(ctx context.Context, t *asynq.Task) error {
	var payload LifecyclePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal LifecyclePayload: %w", err)
	}

	slog.Info("processing lifecycle event",
		"case_id", payload.CaseID,
		"action", payload.Action,
	)

	switch payload.Action {
	case "CLOSE_SUCCESS":
		return h.closeCase(ctx, payload.CaseID, sqlc.CaseStatusCLOSEDSUCCESS, payload.Reason)
	case "CLOSE_FAILURE":
		return h.closeCase(ctx, payload.CaseID, sqlc.CaseStatusCLOSEDFAILURE, payload.Reason)
	case "TRIGGER_ALERT":
		return h.triggerAlert(ctx, payload.CaseID, payload.AlertID, payload.Reason)
	default:
		return fmt.Errorf("unknown lifecycle action: %s", payload.Action)
	}
}

func (h *LifecycleHandler) closeCase(ctx context.Context, caseID string, status sqlc.CaseStatus, reason string) error {
	now := time.Now()
	id, err := stringToUUID(caseID)
	if err != nil {
		return fmt.Errorf("parse case UUID: %w", err)
	}

	// 1. 케이스 상태 업데이트
	_, err = h.queries.UpdateCaseStatus(ctx, sqlc.UpdateCaseStatusParams{
		ID:           id,
		Status:       status,
		ClosedAt:     pgtype.Date{Time: now, Valid: true},
		ClosedReason: pgtype.Text{String: reason, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update case status: %w", err)
	}

	// 2. 타임라인 이벤트 생성
	title := "성공 조건 도달"
	if status == sqlc.CaseStatusCLOSEDFAILURE {
		title = "실패 조건 도달"
	}
	_, err = h.queries.CreateTimelineEvent(ctx, sqlc.CreateTimelineEventParams{
		CaseID:  id,
		Date:    pgtype.Date{Time: now, Valid: true},
		Type:    sqlc.TimelineEventTypePRICEALERT,
		Title:   title,
		Content: reason,
	})
	if err != nil {
		return fmt.Errorf("insert timeline event: %w", err)
	}

	// 3. 해당 케이스의 모니터링 스케줄 모두 해제
	err = h.queries.DisableMonitorBlocksByCase(ctx, id)
	if err != nil {
		return fmt.Errorf("disable monitor blocks: %w", err)
	}

	// 스케줄 재동기화
	if err := h.schedulerManager.SyncMonitorSchedules(); err != nil {
		return fmt.Errorf("sync schedules after case close: %w", err)
	}

	// 4. 알림 전파 (Plan 7)
	// TODO: emitNotification(NotificationEvent{Type: "CASE_CLOSED", CaseID: caseID, Status: status, Reason: reason})

	slog.Info("case closed", "case_id", caseID, "status", status)
	return nil
}

func (h *LifecycleHandler) triggerAlert(ctx context.Context, caseID, alertID, reason string) error {
	now := time.Now()
	caseUUID, err := stringToUUID(caseID)
	if err != nil {
		return fmt.Errorf("parse case UUID: %w", err)
	}
	alertUUID, err := stringToUUID(alertID)
	if err != nil {
		return fmt.Errorf("parse alert UUID: %w", err)
	}

	// 1. PriceAlert 트리거 상태 업데이트
	_, err = h.queries.TriggerPriceAlert(ctx, sqlc.TriggerPriceAlertParams{
		ID:          alertUUID,
		TriggeredAt: pgtype.Date{Time: now, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update price alert: %w", err)
	}

	// 2. 타임라인 이벤트 생성
	_, err = h.queries.CreateTimelineEvent(ctx, sqlc.CreateTimelineEventParams{
		CaseID:  caseUUID,
		Date:    pgtype.Date{Time: now, Valid: true},
		Type:    sqlc.TimelineEventTypePRICEALERT,
		Title:   "가격 알림 도달",
		Content: reason,
	})
	if err != nil {
		return fmt.Errorf("insert timeline event: %w", err)
	}

	// 3. 알림 전파 (Plan 7)
	// TODO: emitNotification(NotificationEvent{Type: "PRICE_ALERT", CaseID: caseID, AlertID: alertID, Reason: reason})

	slog.Info("price alert triggered", "case_id", caseID, "alert_id", alertID)
	return nil
}
