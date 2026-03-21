package gemini_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/gemini"
)

func TestProvider_ImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*gemini.Provider)(nil)
}

func TestProvider_Name(t *testing.T) {
	p := gemini.New(config.LLMConfig{}, nil)
	assert.Equal(t, "gemini", p.Name())
}
