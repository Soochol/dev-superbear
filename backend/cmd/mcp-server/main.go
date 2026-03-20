package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/mcp"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	dbURL := os.Getenv("DATABASE_URL")
	var pool *pgxpool.Pool
	if dbURL != "" {
		var err error
		pool, err = pgxpool.New(context.Background(), dbURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer pool.Close()
		slog.Info("mcp-server: connected to database")
	} else {
		slog.Warn("mcp-server: no DATABASE_URL, validate-only mode")
	}

	executor := dsl.NewExecutor(pool)
	server := mcp.NewServer(executor)

	if err := server.Run(context.Background()); err != nil {
		slog.Error("mcp-server exited with error", "error", err)
		os.Exit(1)
	}
}
