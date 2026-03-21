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

func TestLoad_LLMNewFields(t *testing.T) {
	os.Setenv("APP_ENV", "development")
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	os.Setenv("LLM_MODEL", "gemini-2.0-flash")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("LLM_MODEL")
	}()

	cfg := config.Load()
	assert.Equal(t, "test-gemini-key", cfg.LLM.GeminiKey)
	assert.Equal(t, "gemini-2.0-flash", cfg.LLM.Model)
}
