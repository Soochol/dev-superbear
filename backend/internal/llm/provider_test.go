package llm_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/llm"
)

func TestEventType_Constants(t *testing.T) {
	assert.Equal(t, llm.EventType("thinking"), llm.EventThinking)
	assert.Equal(t, llm.EventType("tool_call"), llm.EventToolCall)
	assert.Equal(t, llm.EventType("tool_result"), llm.EventToolResult)
	assert.Equal(t, llm.EventType("dsl_ready"), llm.EventDSLReady)
	assert.Equal(t, llm.EventType("done"), llm.EventDone)
	assert.Equal(t, llm.EventType("error"), llm.EventError)
}

func TestEvent_JSONTags(t *testing.T) {
	e := llm.Event{
		Type:        llm.EventDSLReady,
		Message:     "test",
		DSL:         "scan where volume > 100",
		Explanation: "설명",
	}
	assert.Equal(t, llm.EventDSLReady, e.Type)
	assert.Equal(t, "scan where volume > 100", e.DSL)
	assert.Equal(t, "설명", e.Explanation)
}
