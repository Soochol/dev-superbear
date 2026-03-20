package worker

import (
	"log/slog"

	"github.com/robfig/cron/v3"
)

// StartScheduleSync launches a background cron job that re-syncs
// the asynq scheduler with the DB every 5 minutes as a safety net.
func StartScheduleSync(sm *SchedulerManager) *cron.Cron {
	c := cron.New()
	_, _ = c.AddFunc("@every 5m", func() {
		if err := sm.SyncMonitorSchedules(); err != nil {
			slog.Error("periodic schedule sync failed", "error", err)
			return
		}
		slog.Info("periodic schedule sync completed")
	})
	c.Start()
	return c
}
