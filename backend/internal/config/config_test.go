package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/config"
)

func TestLoad_LLMDefaults(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	defer os.Unsetenv("APP_ENV")

	cfg := config.Load()

	assert.Equal(t, "claude-cli", cfg.LLM.Provider)
	assert.Equal(t, "claude", cfg.LLM.ClaudeCLIPath)
	assert.Equal(t, 5, cfg.LLM.MaxConcurrent)
	assert.Equal(t, 60, cfg.LLM.TimeoutSeconds)
}

func TestLoad_LLMFromEnv(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	os.Setenv("LLM_PROVIDER", "claude-api")
	os.Setenv("LLM_MAX_CONCURRENT", "10")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("LLM_PROVIDER")
		os.Unsetenv("LLM_MAX_CONCURRENT")
	}()

	cfg := config.Load()

	assert.Equal(t, "claude-api", cfg.LLM.Provider)
	assert.Equal(t, 10, cfg.LLM.MaxConcurrent)
}
