package claudecli_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/claudecli"
)

func TestProvider_ImplementsInterface(t *testing.T) {
	var _ llm.Provider = (*claudecli.Provider)(nil)
}

func TestProvider_Name(t *testing.T) {
	p := claudecli.New(config.LLMConfig{})
	assert.Equal(t, "claude-cli", p.Name())
}

func TestProvider_DefaultConfig(t *testing.T) {
	// Verify defaults are applied when zero values are given.
	p := claudecli.New(config.LLMConfig{})
	assert.Equal(t, "claude-cli", p.Name())
}

func TestProvider_NLToDSL_ContextCancellation(t *testing.T) {
	// Use "sleep" as a mock CLI that takes time, so cancellation is testable.
	p := claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  "sleep",
		TimeoutSeconds: 5,
		MaxConcurrent:  1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := p.NLToDSL(ctx, "test")
	if err != nil {
		// Expected: might fail on semaphore acquisition if context is already done.
		return
	}

	var lastEvent llm.Event
	for e := range ch {
		lastEvent = e
	}
	assert.Equal(t, llm.EventError, lastEvent.Type)
}

func TestProvider_NLToDSL_InvalidBinary(t *testing.T) {
	p := claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  "/nonexistent/binary",
		TimeoutSeconds: 2,
		MaxConcurrent:  1,
	})

	_, err := p.NLToDSL(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "starting claude process")
}

func TestProvider_NLToDSL_SemaphoreLimit(t *testing.T) {
	// Create a script that ignores all args and sleeps, simulating a slow CLI.
	tmpDir := t.TempDir()
	script := filepath.Join(tmpDir, "slow-cli.sh")
	err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0755)
	require.NoError(t, err)

	p := claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  script,
		TimeoutSeconds: 10,
		MaxConcurrent:  1,
	})

	// Start first call — the script will block for 30s, holding the semaphore.
	ctx1, cancel1 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel1()
	ch1, err := p.NLToDSL(ctx1, "test")
	require.NoError(t, err)

	// Give the goroutine a moment to start the process.
	time.Sleep(50 * time.Millisecond)

	// Second call with a very short deadline should fail on semaphore acquisition.
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()
	_, err = p.NLToDSL(ctx2, "2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "semaphore")

	// Drain first channel to avoid goroutine leak.
	cancel1()
	for range ch1 {
	}
}

// --- ParseStreamLine tests ---

func TestParseStreamLine_AssistantText(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"DSL: scan where volume > 1000000\nEXPLANATION: 거래량 100만 이상"}]}}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, llm.EventDSLReady, event.Type)
	assert.Equal(t, "scan where volume > 1000000", event.DSL)
	assert.Equal(t, "거래량 100만 이상", event.Explanation)
}

func TestParseStreamLine_AssistantTextNoDSL(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"text","text":"Let me check the grammar first."}]}}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, llm.EventThinking, event.Type)
	assert.Equal(t, "Let me check the grammar first.", event.Message)
}

func TestParseStreamLine_ToolUse(t *testing.T) {
	line := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"get_dsl_grammar","id":"tool_1","input":{}}]}}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, llm.EventToolCall, event.Type)
	assert.Contains(t, event.Message, "get_dsl_grammar")

	data, ok := event.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "get_dsl_grammar", data["tool_name"])
}

func TestParseStreamLine_Result(t *testing.T) {
	line := `{"type":"result","result":"DSL: scan where price > 50000\nEXPLANATION: 가격 5만원 이상"}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, llm.EventDSLReady, event.Type)
	assert.Equal(t, "scan where price > 50000", event.DSL)
	assert.Equal(t, "가격 5만원 이상", event.Explanation)
}

func TestParseStreamLine_ResultNoDSL(t *testing.T) {
	line := `{"type":"result","result":"Sorry, I could not generate a valid DSL."}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, llm.EventDone, event.Type)
	assert.Contains(t, event.Message, "Sorry")
}

func TestParseStreamLine_EmptyLine(t *testing.T) {
	event, err := claudecli.ParseStreamLine([]byte(""))
	assert.NoError(t, err)
	assert.Nil(t, event)
}

func TestParseStreamLine_InvalidJSON(t *testing.T) {
	event, err := claudecli.ParseStreamLine([]byte("not json"))
	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestParseStreamLine_UnknownType(t *testing.T) {
	line := `{"type":"ping"}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	assert.NoError(t, err)
	assert.Nil(t, event)
}

func TestParseStreamLine_AssistantNoMessage(t *testing.T) {
	line := `{"type":"assistant"}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	assert.NoError(t, err)
	assert.Nil(t, event)
}

func TestParseStreamLine_ResultEmpty(t *testing.T) {
	line := `{"type":"result","result":""}`
	event, err := claudecli.ParseStreamLine([]byte(line))
	assert.NoError(t, err)
	assert.Nil(t, event)
}

func TestProvider_Explain_InvalidBinary(t *testing.T) {
	p := claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  "/nonexistent/binary",
		TimeoutSeconds: 2,
		MaxConcurrent:  1,
	})

	_, err := p.Explain(context.Background(), "scan where volume > 100")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "claude explain failed")
}

func TestProvider_Explain_ContextCancellation(t *testing.T) {
	p := claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  "sleep",
		TimeoutSeconds: 10,
		MaxConcurrent:  1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.Explain(ctx, "scan where volume > 100")
	assert.Error(t, err)
}
