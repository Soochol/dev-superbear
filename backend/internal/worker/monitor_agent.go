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

// AgentHandler handles monitor:agent tasks.
type AgentHandler struct {
	queries *sqlc.Queries
}

func NewAgentHandler(pool *pgxpool.Pool) *AgentHandler {
	return &AgentHandler{queries: sqlc.New(pool)}
}

func (h *AgentHandler) HandleMonitorAgent(ctx context.Context, t *asynq.Task) error {
	var payload MonitorAgentPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal MonitorAgentPayload: %w", err)
	}

	slog.Info("executing monitor agent",
		"case_id", payload.CaseID,
		"monitor_block_id", payload.MonitorBlockID,
	)

	caseUUID, err := stringToUUID(payload.CaseID)
	if err != nil {
		return fmt.Errorf("parse case UUID: %w", err)
	}

	caseRecord, err := h.queries.GetCaseEventSnapshot(ctx, caseUUID)
	if err != nil {
		return fmt.Errorf("query case %s: %w", payload.CaseID, err)
	}

	agentResult, err := executeAgentBlock(ctx, payload, caseRecord.EventSnapshot)
	if err != nil {
		return fmt.Errorf("execute agent block: %w", err)
	}

	title := fmt.Sprintf("[모니터링] %.80s", agentResult.Summary)
	now := time.Now()
	_, err = h.queries.CreateTimelineEvent(ctx, sqlc.CreateTimelineEventParams{
		CaseID: caseUUID,
		Date:   pgtype.Date{Time: now, Valid: true},
		Type:   sqlc.TimelineEventTypePIPELINERESULT,
		Title:  title,
		Content: agentResult.Summary,
		AiAnalysis: pgtype.Text{String: agentResult.Summary, Valid: true},
		Data:   agentResult.Data,
	})
	if err != nil {
		return fmt.Errorf("insert timeline event: %w", err)
	}

	// 마지막 실행 시간 업데이트
	monitorUUID, err := stringToUUID(payload.MonitorBlockID)
	if err != nil {
		slog.Error("invalid monitor block ID, skipping last-executed update",
			"monitor_block_id", payload.MonitorBlockID, "error", err)
	} else if err := h.queries.UpdateMonitorBlockLastExecuted(ctx, sqlc.UpdateMonitorBlockLastExecutedParams{
		ID:             monitorUUID,
		LastExecutedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}); err != nil {
		slog.Error("failed to update last-executed timestamp",
			"monitor_block_id", payload.MonitorBlockID, "error", err)
	}

	slog.Info("monitor agent completed", "case_id", payload.CaseID)
	return nil
}

// ── agent runtime stub (Plan 4 제공) ────────────────────────────

type agentBlockResult struct {
	Summary string
	Data    json.RawMessage
}

func executeAgentBlock(
	_ context.Context,
	_ MonitorAgentPayload,
	_ []byte,
) (*agentBlockResult, error) {
	// TODO: Plan 4 에이전트 런타임 연동
	return &agentBlockResult{
		Summary: "stub agent result",
		Data:    json.RawMessage(`{}`),
	}, nil
}
