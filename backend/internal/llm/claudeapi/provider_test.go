package claudeapi_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/claudeapi"
)

func TestProvider_ImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*claudeapi.Provider)(nil)
}

func TestProvider_Name(t *testing.T) {
	p := claudeapi.New(config.LLMConfig{}, nil)
	assert.Equal(t, "claude-api", p.Name())
}

func TestProvider_DefaultConfig(t *testing.T) {
	p := claudeapi.New(config.LLMConfig{}, nil)
	assert.Equal(t, "claude-api", p.Name())
}
