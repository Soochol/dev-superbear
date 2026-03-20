package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/infra"
	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// DSLPollerHandler handles monitor:dsl-poll tasks.
type DSLPollerHandler struct {
	queries  *sqlc.Queries
	enqueuer *asynq.Client
}

func NewDSLPollerHandler(pool *pgxpool.Pool, enqueuer *asynq.Client) *DSLPollerHandler {
	return &DSLPollerHandler{queries: sqlc.New(pool), enqueuer: enqueuer}
}

func (h *DSLPollerHandler) HandleDSLPoller(ctx context.Context, _ *asynq.Task) error {
	if !infra.IsMarketHours(time.Now()) {
		slog.Debug("outside market hours, skipping DSL poll")
		return nil
	}

	slog.Info("running DSL polling cycle")
	if err := h.runDSLPollingCycle(ctx); err != nil {
		return fmt.Errorf("DSL polling cycle: %w", err)
	}
	slog.Info("DSL polling cycle complete")
	return nil
}

func (h *DSLPollerHandler) runDSLPollingCycle(ctx context.Context) error {
	liveCases, err := h.queries.ListLiveCases(ctx)
	if err != nil {
		return fmt.Errorf("query live cases: %w", err)
	}
	if len(liveCases) == 0 {
		return nil
	}

	// 고유 심볼 추출 및 배치 가격 조회
	symbolSet := make(map[string]struct{})
	for _, c := range liveCases {
		symbolSet[c.Symbol] = struct{}{}
	}
	symbols := make([]string, 0, len(symbolSet))
	for s := range symbolSet {
		symbols = append(symbols, s)
	}
	prices, err := kis.FetchPricesBatch(symbols)
	if err != nil {
		return fmt.Errorf("fetch prices batch: %w", err)
	}

	for _, c := range liveCases {
		price, ok := prices[c.Symbol]
		if !ok {
			continue
		}
		if err := h.evaluateCaseConditions(ctx, c, price); err != nil {
			slog.Error("evaluate case conditions failed", "case_id", uuidToString(c.ID), "error", err)
		}
	}
	return nil
}

// BuildDSLContext creates a typed DSL context from a case event snapshot and a live price.
func BuildDSLContext(eventSnapshot []byte, price *kis.PriceSnapshot) DSLContext {
	var snapshot map[string]interface{}
	_ = json.Unmarshal(eventSnapshot, &snapshot)

	floatVal := func(key string) float64 {
		if v, ok := snapshot[key]; ok {
			if f, ok := v.(float64); ok {
				return f
			}
		}
		return 0
	}
	maVal := func(period int) float64 {
		if preMa, ok := snapshot["preMa"]; ok {
			if m, ok := preMa.(map[string]interface{}); ok {
				key := fmt.Sprintf("%d", period)
				if v, ok := m[key]; ok {
					if f, ok := v.(float64); ok {
						return f
					}
				}
			}
		}
		return 0
	}

	return DSLContext{
		Close:         price.Close,
		High:          price.High,
		Low:           price.Low,
		Volume:        price.Volume,
		EventHigh:     floatVal("high"),
		EventLow:      floatVal("low"),
		EventClose:    floatVal("close"),
		EventVolume:   floatVal("volume"),
		PreEventMA5:   maVal(5),
		PreEventMA20:  maVal(20),
		PreEventMA60:  maVal(60),
		PreEventMA120: maVal(120),
		PreEventMA200: maVal(200),
		PreEventClose: floatVal("preClose"),
	}
}

func (h *DSLPollerHandler) evaluateCaseConditions(
	ctx context.Context,
	caseRow sqlc.ListLiveCasesRow,
	price *kis.PriceSnapshot,
) error {
	dslCtx := BuildDSLContext(caseRow.EventSnapshot, price)
	caseID := uuidToString(caseRow.ID)

	// 성공 조건 체크
	if caseRow.SuccessScript != "" {
		if evaluateDSL(caseRow.SuccessScript, dslCtx) {
			payload := LifecyclePayload{
				CaseID: caseID,
				Action: "CLOSE_SUCCESS",
				Reason: fmt.Sprintf("성공 조건 도달: %s (close=%.2f)", caseRow.SuccessScript, price.Close),
			}
			task, err := NewLifecycleTask(payload)
			if err != nil {
				return err
			}
			if _, err := h.enqueuer.EnqueueContext(ctx, task); err != nil {
				return fmt.Errorf("enqueue lifecycle CLOSE_SUCCESS: %w", err)
			}
			return nil
		}
	}

	// 실패 조건 체크
	if caseRow.FailureScript != "" {
		if evaluateDSL(caseRow.FailureScript, dslCtx) {
			payload := LifecyclePayload{
				CaseID: caseID,
				Action: "CLOSE_FAILURE",
				Reason: fmt.Sprintf("실패 조건 도달: %s (close=%.2f)", caseRow.FailureScript, price.Close),
			}
			task, err := NewLifecycleTask(payload)
			if err != nil {
				return err
			}
			if _, err := h.enqueuer.EnqueueContext(ctx, task); err != nil {
				return fmt.Errorf("enqueue lifecycle CLOSE_FAILURE: %w", err)
			}
			return nil
		}
	}

	// 가격 알림 체크
	alerts, err := h.queries.ListUntriggeredAlertsByCase(ctx, caseRow.ID)
	if err != nil {
		return fmt.Errorf("query untriggered alerts: %w", err)
	}
	for _, alert := range alerts {
		if evaluateDSL(alert.Condition, dslCtx) {
			payload := LifecyclePayload{
				CaseID:  caseID,
				Action:  "TRIGGER_ALERT",
				Reason:  fmt.Sprintf("가격 알림 도달: %s", alert.Label),
				AlertID: uuidToString(alert.ID),
			}
			task, err := NewLifecycleTask(payload)
			if err != nil {
				slog.Error("create lifecycle task for alert", "alert_id", uuidToString(alert.ID), "error", err)
				continue
			}
			if _, err := h.enqueuer.EnqueueContext(ctx, task); err != nil {
				slog.Error("enqueue lifecycle TRIGGER_ALERT", "alert_id", uuidToString(alert.ID), "error", err)
			}
		}
	}

	return nil
}

// evaluateDSL is a stub for the DSL engine (Plan 1).
func evaluateDSL(_ string, _ DSLContext) bool {
	// TODO: Plan 1 DSL 엔진 연동
	return false
}
