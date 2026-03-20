package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/handler"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
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
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	auth := api.Group("")
	auth.Use(middleware.AuthRequired(cfg.JWTSecret))

	registerRoutes(auth, queries, cfg)

	slog.Info("starting server", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func registerRoutes(rg *gin.RouterGroup, queries *sqlc.Queries, cfg *config.Config) {
	authH := handler.NewAuthHandler(queries, cfg.JWTSecret)
	_ = authH

	caseH := handler.NewCaseHandler(queries)
	rg.GET("/cases", caseH.List)
	rg.POST("/cases", caseH.Create)
	rg.GET("/cases/:id", caseH.Get)
	rg.DELETE("/cases/:id", caseH.Delete)

	pipeH := handler.NewPipelineHandler(queries)
	rg.GET("/pipelines", pipeH.List)
	rg.POST("/pipelines", pipeH.Create)
	rg.GET("/pipelines/:id", pipeH.Get)
	rg.PUT("/pipelines/:id", pipeH.Update)
	rg.DELETE("/pipelines/:id", pipeH.Delete)

	blockH := handler.NewBlockHandler(queries)
	rg.GET("/blocks", blockH.List)
	rg.POST("/blocks", blockH.Create)
	rg.GET("/blocks/:id", blockH.Get)
	rg.DELETE("/blocks/:id", blockH.Delete)

	searchH := handler.NewSearchHandler()
	rg.POST("/search/scan", searchH.Scan)
}
