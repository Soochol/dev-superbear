package config

import (
	"log/slog"
	"os"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	AllowedOrigins     []string
	Env                string
}

func Load() *Config {
	env := getEnv("APP_ENV", "development")
	cfg := &Config{
		Env:            env,
		Port:           getEnv("PORT", "8080"),
		AllowedOrigins: []string{getEnv("ALLOWED_ORIGIN", "http://localhost:3000")},
	}
	if env == "production" {
		cfg.JWTSecret = requireEnv("JWT_SECRET")
		cfg.DatabaseURL = requireEnv("DATABASE_URL")
		cfg.GoogleClientID = requireEnv("GOOGLE_CLIENT_ID")
		cfg.GoogleClientSecret = requireEnv("GOOGLE_CLIENT_SECRET")
	} else {
		cfg.JWTSecret = getEnv("JWT_SECRET", "dev-secret-change-in-production")
		cfg.DatabaseURL = getEnv("DATABASE_URL", "postgresql://nexus:nexus@localhost:5432/nexus?sslmode=disable")
		cfg.GoogleClientID = getEnv("GOOGLE_CLIENT_ID", "")
		cfg.GoogleClientSecret = getEnv("GOOGLE_CLIENT_SECRET", "")
		slog.Warn("using development config defaults")
	}
	return cfg
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
