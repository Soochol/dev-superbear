package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	AllowedOrigins     []string
	Env                string
	KISAppKey          string
	KISAppSecret       string
	KISBaseURL         string
	DARTApiKey         string
	RedisAddr          string
	RedisPassword      string
	LLM                LLMConfig
}

type LLMConfig struct {
	Provider       string
	ClaudeCLIPath  string
	MCPConfigPath  string
	AnthropicKey   string
	MaxConcurrent  int
	TimeoutSeconds int
}

func Load() *Config {
	env := getEnv("APP_ENV", "development")
	cfg := &Config{
		Env:            env,
		Port:           getEnv("PORT", "8080"),
		AllowedOrigins: parseOrigins(getEnv("ALLOWED_ORIGINS",
			getEnv("ALLOWED_ORIGIN", "http://localhost:3000"))),
		KISAppKey:      getEnv("KIS_APP_KEY", ""),
		KISAppSecret:   getEnv("KIS_APP_SECRET", ""),
		KISBaseURL:     getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
		DARTApiKey:     getEnv("DART_API_KEY", ""),
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
	cfg.RedisAddr = getEnv("REDIS_ADDR", "localhost:6379")
	cfg.RedisPassword = getEnv("REDIS_PASSWORD", "")
	maxConcurrent, _ := strconv.Atoi(getEnv("LLM_MAX_CONCURRENT", "5"))
	timeoutSec, _ := strconv.Atoi(getEnv("LLM_TIMEOUT_SECONDS", "60"))
	cfg.LLM = LLMConfig{
		Provider:       getEnv("LLM_PROVIDER", "claude-cli"),
		ClaudeCLIPath:  getEnv("CLAUDE_CLI_PATH", "claude"),
		MCPConfigPath:  getEnv("MCP_CONFIG_PATH", ""),
		AnthropicKey:   getEnv("ANTHROPIC_API_KEY", ""),
		MaxConcurrent:  maxConcurrent,
		TimeoutSeconds: timeoutSec,
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

func parseOrigins(s string) []string {
	parts := strings.Split(s, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			origins = append(origins, t)
		}
	}
	return origins
}
