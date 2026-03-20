package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// SchedulerManager manages asynq scheduler entries for active monitor blocks.
type SchedulerManager struct {
	queries   *sqlc.Queries
	pool      *pgxpool.Pool
	redisOpt  asynq.RedisClientOpt
	scheduler *asynq.Scheduler
}

func NewSchedulerManager(pool *pgxpool.Pool, redisOpt asynq.RedisClientOpt) *SchedulerManager {
	scheduler := asynq.NewScheduler(redisOpt, nil)
	return &SchedulerManager{
		queries:   sqlc.New(pool),
		pool:      pool,
		redisOpt:  redisOpt,
		scheduler: scheduler,
	}
}

// SyncMonitorSchedules reads all active MonitorBlocks from the DB and registers
// them as periodic tasks with the asynq scheduler. It rebuilds the scheduler
// from scratch each time to handle additions, removals, and cron changes.
func (sm *SchedulerManager) SyncMonitorSchedules() error {
	rows, err := sm.queries.ListActiveMonitorBlocks(context.Background())
	if err != nil {
		return fmt.Errorf("query active monitors: %w", err)
	}

	newScheduler := asynq.NewScheduler(sm.redisOpt, nil)

	// DSL 가격 폴링: 장중 1분마다 (월~금 09:00~15:59 KST)
	_, err = newScheduler.Register("*/1 9-15 * * 1-5", NewDSLPollerTask())
	if err != nil {
		slog.Error("failed to register DSL poller schedule", "error", err)
	}

	for _, row := range rows {
		var tools []string
		if row.AllowedTools != nil {
			_ = json.Unmarshal(row.AllowedTools, &tools)
		}
		payload := MonitorAgentPayload{
			CaseID:           uuidToString(row.CaseID),
			MonitorBlockID:   uuidToString(row.MonitorBlockID),
			PipelineID:       uuidToString(row.PipelineID),
			Symbol:           row.Symbol,
			BlockInstruction: row.BlockInstruction,
			AllowedTools:     tools,
		}
		task, err := NewMonitorAgentTask(payload)
		if err != nil {
			slog.Error("failed to create agent task", "monitor_block_id", uuidToString(row.MonitorBlockID), "error", err)
			continue
		}
		_, err = newScheduler.Register(row.Cron, task)
		if err != nil {
			slog.Error("failed to register agent schedule", "monitor_block_id", uuidToString(row.MonitorBlockID), "cron", row.Cron, "error", err)
		}
	}

	sm.scheduler = newScheduler
	slog.Info("monitor schedules synced", "count", len(rows))
	return nil
}

// Run starts the underlying asynq scheduler (blocking).
func (sm *SchedulerManager) Run() error {
	return sm.scheduler.Run()
}

// Shutdown stops the scheduler gracefully.
func (sm *SchedulerManager) Shutdown() {
	sm.scheduler.Shutdown()
}
