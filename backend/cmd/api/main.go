package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/agent"
	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/handler"
	"github.com/dev-superbear/nexus-backend/internal/infra/kis"
	"github.com/dev-superbear/nexus-backend/internal/llm/claudecli"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	queries := sqlc.New(pool)

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	api := r.Group("/api/v1")

	// Public routes (no auth required)
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Auth routes (public)
	authH := handler.NewAuthHandler(queries, cfg.JWTSecret, cfg.Env)
	api.POST("/auth/google/callback", authH.GoogleCallback)
	api.POST("/auth/logout", authH.Logout)

	// Protected routes
	auth := api.Group("")
	auth.Use(middleware.AuthRequired(cfg.JWTSecret, cfg.Env))
	auth.GET("/auth/me", authH.Me)

	// Register resource routes
	registerRoutes(auth, queries, pool, cfg)

	slog.Info("starting server", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func registerRoutes(rg *gin.RouterGroup, queries *sqlc.Queries, pool *pgxpool.Pool, cfg *config.Config) {
	caseH := handler.NewCaseHandler(queries)
	rg.GET("/cases", caseH.List)
	rg.POST("/cases", caseH.Create)
	rg.GET("/cases/:id", caseH.Get)
	rg.DELETE("/cases/:id", caseH.Delete)

	// Pipeline & block repos + services
	pipelineRepo := repository.NewPipelineRepository(pool)
	blockRepo := repository.NewBlockRepository(pool)

	runner := agent.NewADKRunner()
	orchestrator := service.NewPipelineOrchestrator(runner)

	pipelineSvc := service.NewPipelineService(pipelineRepo, blockRepo, orchestrator)
	blockSvc := service.NewBlockService(blockRepo)
	generator := service.NewPipelineGenerator()

	pipeH := handler.NewPipelineHandler(pipelineSvc, generator)
	pipeH.RegisterRoutes(rg)

	blockH := handler.NewBlockHandler(blockSvc)
	blockH.RegisterRoutes(rg)

	llmProvider := claudecli.New(cfg.LLM)
	searchSvc := service.NewSearchService(dsl.NewExecutor(pool))
	nlSvc := service.NewNLToDSLService(llmProvider)
	searchH := handler.NewSearchHandler(searchSvc, nlSvc)
	searchH.RegisterRoutes(rg)

	presetRepo := repository.NewPresetRepository(pool)
	presetH := handler.NewPresetHandler(presetRepo)
	presetH.RegisterRoutes(rg)

	// Case extensions
	rg.POST("/cases/:id/close", caseH.Close)
	rg.GET("/cases/:id/timeline", caseH.GetTimeline)
	rg.GET("/cases/:id/return-tracking", caseH.GetReturnTracking)

	// Trade routes
	tradeH := handler.NewTradeHandler(queries)
	rg.POST("/cases/:id/trades", tradeH.CreateTrade)
	rg.GET("/cases/:id/trades", tradeH.ListTrades)

	// Alert routes
	alertH := handler.NewAlertHandler(queries)
	rg.GET("/cases/:id/alerts", alertH.ListAlerts)
	rg.POST("/cases/:id/alerts", alertH.CreateAlert)
	rg.DELETE("/cases/:id/alerts/:alertId", alertH.DeleteAlert)

	// Monitoring routes
	monitoringSvc := service.NewMonitoringService(pool, nil) // nil scheduler — API server doesn't manage schedules
	monitorH := handler.NewMonitoringHandler(monitoringSvc)
	rg.GET("/cases/:id/monitors", monitorH.ListMonitors)
	rg.PATCH("/cases/:id/monitors/:monitorId", monitorH.ToggleBlock)
	rg.PATCH("/cases/:id/monitoring-status", monitorH.ToggleCaseMonitoring)

	// Candle routes
	kisClient := kis.NewClient(cfg.KISAppKey, cfg.KISAppSecret, cfg.KISBaseURL)
	candleSvc := service.NewCandleService(kisClient)
	candleH := handler.NewCandleHandler(candleSvc)
	rg.GET("/candles/:symbol", candleH.GetCandles)
}
