package worker

import (
	"log/slog"

	"github.com/robfig/cron/v3"
)

// StartScheduleSync launches a background cron job that re-syncs
// the asynq scheduler with the DB every 5 minutes as a safety net.
func StartScheduleSync(sm *SchedulerManager) *cron.Cron {
	c := cron.New()
	if _, err := c.AddFunc("@every 5m", func() {
		if err := sm.SyncMonitorSchedules(); err != nil {
			slog.Error("periodic schedule sync failed", "error", err)
			return
		}
		slog.Info("periodic schedule sync completed")
	}); err != nil {
		slog.Error("failed to register periodic schedule sync", "error", err)
	}
	c.Start()
	return c
}
