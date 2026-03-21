package tools_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/llm/tools"
)

func TestDSLToolDefinitions(t *testing.T) {
	defs := tools.DSLToolDefinitions()
	assert.Len(t, defs, 4)

	names := make([]string, len(defs))
	for i, d := range defs {
		names[i] = d.Name
	}
	assert.Contains(t, names, "get_dsl_grammar")
	assert.Contains(t, names, "list_available_fields")
	assert.Contains(t, names, "validate_dsl")
	assert.Contains(t, names, "submit_dsl")
}

func TestDSLToolDefinitions_SubmitDSLHasRequiredFields(t *testing.T) {
	defs := tools.DSLToolDefinitions()
	var submitDSL tools.ToolDef
	for _, d := range defs {
		if d.Name == "submit_dsl" {
			submitDSL = d
			break
		}
	}
	assert.NotEmpty(t, submitDSL.Name)

	required, ok := submitDSL.Parameters["required"].([]string)
	assert.True(t, ok)
	assert.Contains(t, required, "dsl")
	assert.Contains(t, required, "explanation")
}
