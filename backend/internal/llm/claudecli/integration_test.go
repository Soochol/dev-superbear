//go:build integration

package claudecli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/claudecli"
)

var (
	mcpBinaryPath string
	mcpConfigPath string
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "nl-to-dsl-integration-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Build the MCP server binary.
	mcpBinaryPath = filepath.Join(tmpDir, "mcp-server")
	buildCmd := exec.Command("go", "build", "-o", mcpBinaryPath, "./cmd/mcp-server")
	buildCmd.Dir = filepath.Join(projectRoot(), "backend")
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build mcp-server: %v\n", err)
		os.Exit(1)
	}

	// Write MCP config pointing to the binary.
	mcpConfigPath = filepath.Join(tmpDir, "mcp-config.json")
	mcpCfg := map[string]any{
		"mcpServers": map[string]any{
			"nexus-dsl": map[string]any{
				"command": mcpBinaryPath,
			},
		},
	}
	cfgBytes, _ := json.Marshal(mcpCfg)
	if err := os.WriteFile(mcpConfigPath, cfgBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write mcp config: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func projectRoot() string {
	// Walk up from the test file to find the project root (where go.mod is).
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Dir(dir) // go.mod is in backend/, project root is parent
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

func newProvider(t *testing.T) *claudecli.Provider {
	t.Helper()
	return claudecli.New(config.LLMConfig{
		ClaudeCLIPath:  "claude",
		MCPConfigPath:  mcpConfigPath,
		MaxConcurrent:  2,
		TimeoutSeconds: 120,
	})
}

func collectEvents(t *testing.T, ctx context.Context, ch <-chan llm.Event) []llm.Event {
	t.Helper()
	var events []llm.Event
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, e)
			t.Logf("event: type=%s message=%q dsl=%q", e.Type, e.Message, e.DSL)
		case <-ctx.Done():
			t.Fatal("context expired while collecting events")
			return events
		}
	}
}

func findDSLEvent(events []llm.Event) *llm.Event {
	for i := range events {
		if events[i].Type == llm.EventDSLReady {
			return &events[i]
		}
	}
	return nil
}

func findErrorEvent(events []llm.Event) *llm.Event {
	for i := range events {
		if events[i].Type == llm.EventError {
			return &events[i]
		}
	}
	return nil
}

type nlTestCase struct {
	name           string
	query          string
	expectFields   []string // DSL에 포함되어야 하는 필드명
	expectKeywords []string // DSL에 포함되어야 하는 키워드 (sort, limit 등)
}

var nlTestCases = []nlTestCase{
	// --- 기본 단일 필터 ---
	{
		name:         "거래량 필터링",
		query:        "거래량 100만 이상 종목",
		expectFields: []string{"volume"},
	},
	{
		name:         "종가 필터링",
		query:        "현재가 5만원 이상인 종목",
		expectFields: []string{"close"},
	},
	{
		name:         "저가 필터링",
		query:        "저가가 1만원 이하인 주식",
		expectFields: []string{"low"},
	},
	{
		name:         "시가 필터링",
		query:        "시가 2만원 넘는 종목",
		expectFields: []string{"open"},
	},
	{
		name:         "등락률 필터",
		query:        "오늘 3% 이상 오른 종목",
		expectFields: []string{"change_pct"},
	},

	// --- 정렬 포함 ---
	{
		name:           "거래대금 정렬",
		query:          "거래대금이 많은 순서로 종목 보여줘",
		expectFields:   []string{"trade_value"},
		expectKeywords: []string{"sort"},
	},
	{
		name:           "거래량 내림차순 정렬",
		query:          "거래량 많은 순으로 상위 종목",
		expectFields:   []string{"volume"},
		expectKeywords: []string{"sort"},
	},
	{
		name:           "종가 오름차순 정렬",
		query:          "주가 낮은 순서로 보여줘",
		expectKeywords: []string{"sort", "asc"},
	},

	// --- 갯수 제한 ---
	{
		name:           "갯수 제한",
		query:          "고가 10만원 이상 종목 20개만",
		expectFields:   []string{"high"},
		expectKeywords: []string{"limit"},
	},
	{
		name:           "상위 10개",
		query:          "거래대금 상위 10개 종목",
		expectFields:   []string{"trade_value"},
		expectKeywords: []string{"limit"},
	},

	// --- 복합 조건 ---
	{
		name:         "종가 + 거래량",
		query:        "종가 5만원 이상이고 거래량 50만 이상인 종목",
		expectFields: []string{"close", "volume"},
	},
	{
		name:         "거래량 + 등락률",
		query:        "거래량 100만 이상이면서 5% 이상 상승한 종목",
		expectFields: []string{"volume", "change_pct"},
	},
	{
		name:           "3중 조건 + 정렬",
		query:          "종가 1만원 이상, 거래량 50만 이상, 거래대금 100억 이상 종목 거래대금 순으로",
		expectFields:   []string{"close", "volume", "trade_value"},
		expectKeywords: []string{"sort"},
	},

	// --- 구어체/비형식적 표현 ---
	{
		name:         "구어체 표현",
		query:        "많이 거래된 주식 찾아줘",
		expectFields: []string{"volume"},
	},
	{
		name:         "비형식적 질문",
		query:        "싼 주식 뭐 있어?",
		expectFields: []string{"close"},
	},
	{
		name:         "대화체",
		query:        "거래가 활발한 종목 좀 알려줘",
		expectFields: []string{"volume"},
	},

	// --- 단위 변환 ---
	{
		name:         "억 단위 거래대금",
		query:        "거래대금 100억 이상",
		expectFields: []string{"trade_value"},
	},
	{
		name:         "만 단위 종가",
		query:        "5만원짜리 이상 종목",
		expectFields: []string{"close"},
	},
}

func TestNLToDSL_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	executor := dsl.NewExecutor()
	provider := newProvider(t)

	for _, tc := range nlTestCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			ch, err := provider.NLToDSL(ctx, tc.query)
			require.NoError(t, err, "NLToDSL should start without error")

			events := collectEvents(t, ctx, ch)

			// Should not have errors.
			errEvt := findErrorEvent(events)
			if errEvt != nil {
				t.Fatalf("unexpected error event: %s", errEvt.Message)
			}

			// Must produce a DSL.
			dslEvt := findDSLEvent(events)
			require.NotNil(t, dslEvt, "should produce a dsl_ready event")
			require.NotEmpty(t, dslEvt.DSL, "DSL should not be empty")

			t.Logf("generated DSL: %s", dslEvt.DSL)
			t.Logf("explanation: %s", dslEvt.Explanation)

			// Verify DSL is syntactically valid.
			result := executor.Validate(dslEvt.DSL)
			assert.True(t, result.Valid, "generated DSL should be valid, got error: %s (DSL: %s)", result.Error, dslEvt.DSL)

			// Verify expected fields appear in the DSL.
			dslLower := strings.ToLower(dslEvt.DSL)
			for _, field := range tc.expectFields {
				assert.Contains(t, dslLower, field,
					"DSL should contain field %q for query %q", field, tc.query)
			}

			// Verify expected keywords appear in the DSL.
			for _, kw := range tc.expectKeywords {
				assert.Contains(t, dslLower, kw,
					"DSL should contain keyword %q for query %q", kw, tc.query)
			}

			// Explanation is expected but not guaranteed — LLM may omit it.
			if dslEvt.Explanation == "" {
				t.Log("WARNING: explanation was empty (LLM may have omitted EXPLANATION: line)")
			}
		})
	}
}

// TestNLToDSL_ToolCallSequence verifies the LLM follows the expected workflow:
// 1. Call get_dsl_grammar
// 2. Call list_available_fields
// 3. Generate DSL
// 4. Call validate_dsl
func TestNLToDSL_ToolCallSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	provider := newProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	ch, err := provider.NLToDSL(ctx, "거래량 100만 이상 종목")
	require.NoError(t, err)

	events := collectEvents(t, ctx, ch)

	// Verify tool calls happened.
	// Claude CLI prefixes MCP tool names with "mcp__<server>__", e.g. "mcp__nexus-dsl__get_dsl_grammar".
	var toolCalls []string
	for _, e := range events {
		if e.Type == llm.EventToolCall {
			if data, ok := e.Data.(map[string]any); ok {
				if name, ok := data["tool_name"].(string); ok {
					toolCalls = append(toolCalls, name)
				}
			}
		}
	}

	t.Logf("tool call sequence: %v", toolCalls)

	toolCallsJoined := strings.Join(toolCalls, ",")

	// At minimum, grammar and fields should be consulted.
	assert.Contains(t, toolCallsJoined, "get_dsl_grammar", "should call get_dsl_grammar")
	assert.Contains(t, toolCallsJoined, "list_available_fields", "should call list_available_fields")

	// validate_dsl should be called (prompt instructs this).
	assert.Contains(t, toolCallsJoined, "validate_dsl", "should call validate_dsl")

	// DSL should be produced.
	dslEvt := findDSLEvent(events)
	require.NotNil(t, dslEvt, "should produce a dsl_ready event")
}
