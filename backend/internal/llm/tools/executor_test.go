package tools_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/llm/tools"
)

func TestExecutor_GetGrammar(t *testing.T) {
	exec := tools.NewExecutor(dsl.NewExecutor(nil))
	grammar := exec.GetGrammar()
	assert.Contains(t, grammar, "scan where")
	assert.Contains(t, grammar, "volume")
}

func TestExecutor_ListFields(t *testing.T) {
	exec := tools.NewExecutor(dsl.NewExecutor(nil))
	fields := exec.ListFields()
	assert.Contains(t, fields, "Available fields:")
	assert.Contains(t, fields, "close")
	assert.Contains(t, fields, "volume")
}

func TestExecutor_ValidateDSL_Valid(t *testing.T) {
	exec := tools.NewExecutor(dsl.NewExecutor(nil))
	result, err := exec.ValidateDSL("scan where volume > 1000000")
	require.NoError(t, err)
	assert.Contains(t, result, `"valid":true`)
}

func TestExecutor_ValidateDSL_Invalid(t *testing.T) {
	exec := tools.NewExecutor(dsl.NewExecutor(nil))
	_, err := exec.ValidateDSL("")
	assert.Error(t, err)
}

func TestExecutor_DispatchTool(t *testing.T) {
	exec := tools.NewExecutor(dsl.NewExecutor(nil))

	t.Run("get_dsl_grammar", func(t *testing.T) {
		result, err := exec.DispatchTool("get_dsl_grammar", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "scan where")
	})

	t.Run("list_available_fields", func(t *testing.T) {
		result, err := exec.DispatchTool("list_available_fields", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "close")
	})

	t.Run("validate_dsl", func(t *testing.T) {
		result, err := exec.DispatchTool("validate_dsl", []byte(`{"dsl":"scan where volume > 100"}`))
		require.NoError(t, err)
		assert.Contains(t, result, "true")
	})

	t.Run("unknown_tool", func(t *testing.T) {
		_, err := exec.DispatchTool("unknown", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown tool")
	})
}
