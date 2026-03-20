package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/infra/pgutil"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
	"github.com/dev-superbear/nexus-backend/internal/worker"
)

// MonitoringService manages pause/resume of individual blocks and entire cases.
type MonitoringService struct {
	queries          *sqlc.Queries
	schedulerManager *worker.SchedulerManager
}

func NewMonitoringService(pool *pgxpool.Pool, sm *worker.SchedulerManager) *MonitoringService {
	return &MonitoringService{queries: sqlc.New(pool), schedulerManager: sm}
}

// ToggleMonitorBlock enables or disables a single monitor block
// and re-syncs the asynq scheduler.
func (s *MonitoringService) ToggleMonitorBlock(monitorBlockID string, enabled bool) error {
	ctx := context.Background()
	id, err := parseUUID(monitorBlockID)
	if err != nil {
		return err
	}
	err = s.queries.UpdateMonitorBlockEnabled(ctx, sqlc.UpdateMonitorBlockEnabledParams{
		ID:      id,
		Enabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("update monitor block: %w", err)
	}

	if err := s.schedulerManager.SyncMonitorSchedules(); err != nil {
		return fmt.Errorf("sync schedules after toggle block: %w", err)
	}
	return nil
}

// ToggleCaseMonitoring enables or disables all monitor blocks for a case.
// If keepDSLPolling is true and enabled is false, DSL polling remains active.
func (s *MonitoringService) ToggleCaseMonitoring(caseID string, enabled bool, keepDSLPolling bool) error {
	ctx := context.Background()
	id, err := parseUUID(caseID)
	if err != nil {
		return err
	}

	err = s.queries.UpdateMonitorBlocksEnabledByCase(ctx, sqlc.UpdateMonitorBlocksEnabledByCaseParams{
		CaseID:  id,
		Enabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("update monitor blocks: %w", err)
	}

	if !keepDSLPolling && !enabled {
		err = s.queries.UpdateCaseDSLPollingEnabled(ctx, sqlc.UpdateCaseDSLPollingEnabledParams{
			ID:                id,
			DslPollingEnabled: false,
		})
		if err != nil {
			return fmt.Errorf("disable dsl polling: %w", err)
		}
	} else if enabled {
		err = s.queries.UpdateCaseDSLPollingEnabled(ctx, sqlc.UpdateCaseDSLPollingEnabledParams{
			ID:                id,
			DslPollingEnabled: true,
		})
		if err != nil {
			return fmt.Errorf("enable dsl polling: %w", err)
		}
	}

	if err := s.schedulerManager.SyncMonitorSchedules(); err != nil {
		return fmt.Errorf("sync schedules after toggle case: %w", err)
	}
	return nil
}

// ListMonitorBlocks returns all monitor blocks for a case.
func (s *MonitoringService) ListMonitorBlocks(caseID string) ([]MonitorBlockInfo, error) {
	ctx := context.Background()
	id, err := parseUUID(caseID)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries.ListMonitorBlocksByCase(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("query monitor blocks: %w", err)
	}
	blocks := make([]MonitorBlockInfo, len(rows))
	for i, r := range rows {
		var lastExec *string
		if r.LastExecutedAt.Valid {
			s := r.LastExecutedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			lastExec = &s
		}
		blocks[i] = MonitorBlockInfo{
			ID:             uuidToString(r.ID),
			Enabled:        r.Enabled,
			Cron:           r.Cron,
			LastExecutedAt: lastExec,
			Instruction:    r.Instruction,
		}
	}
	return blocks, nil
}

// MonitorBlockInfo is a read DTO for monitor block listing.
type MonitorBlockInfo struct {
	ID             string  `json:"id"`
	Enabled        bool    `json:"enabled"`
	Cron           string  `json:"cron"`
	LastExecutedAt *string `json:"last_executed_at"`
	Instruction    string  `json:"instruction"`
}

func parseUUID(s string) (pgtype.UUID, error) {
	return pgutil.ParseUUID(s)
}

func uuidToString(u pgtype.UUID) string {
	return pgutil.UUIDToString(u)
}
