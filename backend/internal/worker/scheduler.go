package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// SchedulerManager manages asynq scheduler entries for active monitor blocks.
type SchedulerManager struct {
	mu        sync.Mutex
	queries   *sqlc.Queries
	redisOpt  asynq.RedisClientOpt
	scheduler *asynq.Scheduler
}

func NewSchedulerManager(pool *pgxpool.Pool, redisOpt asynq.RedisClientOpt) *SchedulerManager {
	return &SchedulerManager{
		queries:  sqlc.New(pool),
		redisOpt: redisOpt,
	}
}

// SyncMonitorSchedules reads all active MonitorBlocks from the DB and registers
// them as periodic tasks with the asynq scheduler. It shuts down the old scheduler,
// builds a new one, and starts it in a goroutine.
func (sm *SchedulerManager) SyncMonitorSchedules() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	rows, err := sm.queries.ListActiveMonitorBlocks(context.Background())
	if err != nil {
		return fmt.Errorf("query active monitors: %w", err)
	}

	newScheduler := asynq.NewScheduler(sm.redisOpt, nil)

	// DSL 가격 폴링: 장중 1분마다 (월~금 09:00~15:59 KST)
	if _, err := newScheduler.Register("*/1 9-15 * * 1-5", NewDSLPollerTask()); err != nil {
		return fmt.Errorf("register DSL poller schedule: %w", err)
	}

	for _, row := range rows {
		var tools []string
		if row.AllowedTools != nil {
			if err := json.Unmarshal(row.AllowedTools, &tools); err != nil {
				slog.Error("failed to unmarshal allowed_tools, using empty list",
					"monitor_block_id", uuidToString(row.MonitorBlockID), "error", err)
			}
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
		if _, err := newScheduler.Register(row.Cron, task); err != nil {
			slog.Error("failed to register agent schedule", "monitor_block_id", uuidToString(row.MonitorBlockID), "cron", row.Cron, "error", err)
		}
	}

	// Shutdown old scheduler before replacing
	if sm.scheduler != nil {
		sm.scheduler.Shutdown()
	}

	sm.scheduler = newScheduler

	// Start new scheduler in a goroutine
	go func() {
		if err := newScheduler.Run(); err != nil {
			slog.Error("scheduler stopped", "error", err)
		}
	}()

	slog.Info("monitor schedules synced", "count", len(rows))
	return nil
}

// Shutdown stops the scheduler gracefully.
func (sm *SchedulerManager) Shutdown() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.scheduler != nil {
		sm.scheduler.Shutdown()
	}
}
