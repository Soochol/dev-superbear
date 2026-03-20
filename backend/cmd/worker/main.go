package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/infra"
	"github.com/dev-superbear/nexus-backend/internal/service"
	"github.com/dev-superbear/nexus-backend/internal/worker"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	cfg := config.Load()
	redisOpt := infra.NewRedisClientOpt(cfg.RedisAddr, cfg.RedisPassword)

	// ── DB (pgxpool + sqlc) ─────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	slog.Info("connected to database")

	// ── asynq Client (for enqueuing tasks from within handlers) ─
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	// ── Handlers ────────────────────────────────────────────────
	agentHandler := worker.NewAgentHandler(pool)
	dslHandler := worker.NewDSLPollerHandler(pool, client)
	schedulerMgr := worker.NewSchedulerManager(pool, redisOpt)
	lifecycleHandler := worker.NewLifecycleHandler(pool, schedulerMgr)

	// ── asynq Server (task processing) ──────────────────────────
	srv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.Error("task failed", "type", task.Type(), "error", err)
			}),
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(worker.TypeMonitorAgent, agentHandler.HandleMonitorAgent)
	mux.HandleFunc(worker.TypeDSLPoller, dslHandler.HandleDSLPoller)
	mux.HandleFunc(worker.TypeMonitorLifecycle, lifecycleHandler.HandleLifecycle)

	// ── Initial schedule sync (also starts scheduler goroutine) ─
	if err := schedulerMgr.SyncMonitorSchedules(); err != nil {
		log.Fatalf("initial schedule sync failed: %v", err)
	}
	slog.Info("initial schedule sync completed")

	// ── Schedule sync cron (5-minute safety net) ────────────────
	syncCron := worker.StartScheduleSync(schedulerMgr)
	defer syncCron.Stop()

	// ── Health check HTTP server ────────────────────────────────
	metricsSvc := service.NewMetricsService(redisOpt)

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("GET /api/health/workers", func(w http.ResponseWriter, r *http.Request) {
		metrics, err := metricsSvc.CollectMetrics()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"metrics": metrics,
		}); err != nil {
			slog.Error("failed to encode health response", "error", err)
		}
	})

	go func() {
		addr := ":8081"
		slog.Info("health HTTP server starting", "addr", addr)
		if err := http.ListenAndServe(addr, httpMux); err != nil && err != http.ErrServerClosed {
			log.Fatalf("health HTTP server failed: %v", err)
		}
	}()

	// ── Start asynq worker server ───────────────────────────────
	slog.Info("starting asynq worker server")
	go func() {
		if err := srv.Run(mux); err != nil {
			slog.Error("asynq server stopped", "error", err)
		}
	}()

	// ── Graceful shutdown ───────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("received shutdown signal", "signal", sig)

	srv.Shutdown()
	schedulerMgr.Shutdown()
	slog.Info("all workers shut down gracefully")
}
