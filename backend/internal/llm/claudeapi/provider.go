// Package claudeapi implements the llm.Provider interface using the Anthropic Messages API
// with function calling for NL-to-DSL conversion.
package claudeapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"golang.org/x/sync/semaphore"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/tools"
)

const (
	defaultModel          = anthropic.ModelClaudeSonnet4_6
	defaultMaxConcurrent  = 5
	defaultTimeoutSeconds = 120
	defaultMaxTokens      = 4096
	maxToolRounds         = 10
)

// Provider implements llm.Provider using the Anthropic Messages API.
type Provider struct {
	client   *anthropic.Client
	cfg      config.LLMConfig
	sem      *semaphore.Weighted
	toolExec *tools.Executor
}

// New creates a Claude API provider with the given configuration and optional tool executor.
func New(cfg config.LLMConfig, toolExec *tools.Executor) *Provider {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = defaultMaxConcurrent
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = defaultTimeoutSeconds
	}
	if cfg.Model == "" {
		cfg.Model = defaultModel
	}

	var client *anthropic.Client
	if cfg.AnthropicKey != "" {
		c := anthropic.NewClient(option.WithAPIKey(cfg.AnthropicKey))
		client = &c
	} else {
		// Use environment variable ANTHROPIC_API_KEY if set.
		c := anthropic.NewClient()
		client = &c
	}

	return &Provider{
		client:   client,
		cfg:      cfg,
		sem:      semaphore.NewWeighted(int64(cfg.MaxConcurrent)),
		toolExec: toolExec,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "claude-api" }

// NLToDSL converts a natural language query to DSL using the Anthropic Messages API
// with a function-calling loop. Events are streamed via the returned channel.
func (p *Provider) NLToDSL(ctx context.Context, query string) (<-chan llm.Event, error) {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquiring semaphore: %w", err)
	}

	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)

	started := false
	cleanup := func() {
		if !started {
			cancel()
			p.sem.Release(1)
		}
	}
	defer cleanup()

	ch := make(chan llm.Event, 16)
	started = true

	go func() {
		defer close(ch)
		defer cancel()
		defer p.sem.Release(1)

		p.runNLToDSL(ctx, query, ch)
	}()

	return ch, nil
}

// runNLToDSL executes the function-calling loop and sends events to ch.
func (p *Provider) runNLToDSL(ctx context.Context, query string, ch chan<- llm.Event) {
	send := func(e llm.Event) bool {
		select {
		case ch <- e:
			return true
		case <-ctx.Done():
			ch <- llm.Event{Type: llm.EventError, Message: "context canceled"}
			return false
		}
	}

	if !send(llm.Event{Type: llm.EventThinking, Message: "Analyzing query..."}) {
		return
	}

	// Build tool list from DSLToolDefinitions.
	apiTools := buildTools()

	// Initial message history.
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(query)),
	}

	for round := 0; round < maxToolRounds; round++ {
		if ctx.Err() != nil {
			ch <- llm.Event{Type: llm.EventError, Message: "context canceled"}
			return
		}

		resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     p.cfg.Model,
			MaxTokens: defaultMaxTokens,
			System: []anthropic.TextBlockParam{
				{Text: llm.NLToDSLAPIPrompt},
			},
			Messages: messages,
			Tools:    apiTools,
		})
		if err != nil {
			if ctx.Err() != nil {
				ch <- llm.Event{Type: llm.EventError, Message: "request timed out or canceled"}
			} else {
				ch <- llm.Event{Type: llm.EventError, Message: fmt.Sprintf("API call failed: %v", err)}
			}
			return
		}

		// Check if any tool_use blocks are present.
		hasToolUse := false
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				hasToolUse = true
				break
			}
		}

		if !hasToolUse {
			// No tool calls — try to extract DSL from text response as fallback.
			for _, block := range resp.Content {
				if block.Type == "text" && block.Text != "" {
					dslStr, explanation := extractDSL(block.Text)
					if dslStr != "" {
						send(llm.Event{
							Type:        llm.EventDSLReady,
							Message:     "DSL generated",
							DSL:         dslStr,
							Explanation: explanation,
						})
						return
					}
				}
			}
			send(llm.Event{Type: llm.EventError, Message: "model did not call submit_dsl and no DSL found in text response"})
			return
		}

		// Process tool calls. Collect assistant content blocks to add to message history.
		assistantContentBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(resp.Content))
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					assistantContentBlocks = append(assistantContentBlocks, anthropic.NewTextBlock(block.Text))
					if !send(llm.Event{Type: llm.EventThinking, Message: block.Text}) {
						return
					}
				}
			case "tool_use":
				assistantContentBlocks = append(assistantContentBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))

				if block.Name == "submit_dsl" {
					// Terminal tool: extract dsl and explanation from input.
					var input struct {
						DSL         string `json:"dsl"`
						Explanation string `json:"explanation"`
					}
					if err := json.Unmarshal(block.Input, &input); err != nil {
						slog.Warn("failed to parse submit_dsl input", "error", err)
						send(llm.Event{Type: llm.EventError, Message: fmt.Sprintf("failed to parse submit_dsl input: %v", err)})
						return
					}
					send(llm.Event{
						Type:        llm.EventDSLReady,
						Message:     "DSL generated",
						DSL:         input.DSL,
						Explanation: input.Explanation,
					})
					return
				}

				// Non-terminal tool call.
				if !send(llm.Event{
					Type:    llm.EventToolCall,
					Message: fmt.Sprintf("Calling tool: %s", block.Name),
					Data: map[string]any{
						"tool_name": block.Name,
						"tool_id":   block.ID,
					},
				}) {
					return
				}
			}
		}

		// Add assistant turn to message history.
		messages = append(messages, anthropic.NewAssistantMessage(assistantContentBlocks...))

		// Execute non-submit tool calls and collect results.
		toolResultBlocks := make([]anthropic.ContentBlockParamUnion, 0)
		for _, block := range resp.Content {
			if block.Type != "tool_use" || block.Name == "submit_dsl" {
				continue
			}

			result, dispatchErr := p.toolExec.DispatchTool(block.Name, block.Input)
			var isError bool
			var resultContent string
			if dispatchErr != nil {
				slog.Warn("tool dispatch error", "tool", block.Name, "error", dispatchErr)
				resultContent = fmt.Sprintf("error: %v", dispatchErr)
				isError = true
			} else {
				resultContent = result
			}

			if !send(llm.Event{
				Type:    llm.EventToolResult,
				Message: fmt.Sprintf("Tool %s result", block.Name),
				Data:    resultContent,
			}) {
				return
			}

			toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(block.ID, resultContent, isError))
		}

		if len(toolResultBlocks) > 0 {
			messages = append(messages, anthropic.NewUserMessage(toolResultBlocks...))
		}
	}

	// Exceeded max rounds.
	send(llm.Event{Type: llm.EventError, Message: "max tool-calling rounds exceeded"})
}

// Explain returns a natural language explanation of a DSL query.
func (p *Provider) Explain(ctx context.Context, dsl string) (string, error) {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return "", fmt.Errorf("acquiring semaphore: %w", err)
	}
	defer p.sem.Release(1)

	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.cfg.Model,
		MaxTokens: defaultMaxTokens,
		System: []anthropic.TextBlockParam{
			{Text: llm.ExplainPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(dsl)),
		},
	})
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("explain timed out or canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("explain API call failed: %w", err)
	}

	for _, block := range resp.Content {
		if block.Type == "text" {
			return strings.TrimSpace(block.Text), nil
		}
	}
	return "", fmt.Errorf("no text content in explain response")
}

// buildTools converts DSLToolDefinitions to Anthropic SDK tool format.
func buildTools() []anthropic.ToolUnionParam {
	defs := tools.DSLToolDefinitions()
	result := make([]anthropic.ToolUnionParam, 0, len(defs))
	for _, def := range defs {
		toolParam := anthropic.ToolParam{
			Name:        def.Name,
			Description: anthropic.String(def.Description),
			InputSchema: buildInputSchema(def.Parameters),
		}
		result = append(result, anthropic.ToolUnionParam{OfTool: &toolParam})
	}
	return result
}

// buildInputSchema converts a generic map to ToolInputSchemaParam.
func buildInputSchema(params map[string]any) anthropic.ToolInputSchemaParam {
	schema := anthropic.ToolInputSchemaParam{}

	if props, ok := params["properties"]; ok {
		schema.Properties = props
	}
	if req, ok := params["required"]; ok {
		if reqSlice, ok := req.([]string); ok {
			schema.Required = reqSlice
		}
	}
	return schema
}

// extractDSL looks for "DSL:" and "EXPLANATION:" labels in text output.
// Used as a fallback when the model returns text instead of calling submit_dsl.
func extractDSL(text string) (dsl, explanation string) {
	lines := strings.Split(text, "\n")
	dslNextLine := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.ReplaceAll(trimmed, "**", "")

		if dslNextLine {
			if strings.HasPrefix(trimmed, "```") {
				continue
			}
			if trimmed != "" {
				dsl = strings.Trim(trimmed, "`")
				dslNextLine = false
			}
			continue
		}

		if strings.HasPrefix(trimmed, "DSL:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "DSL:"))
			val = strings.Trim(val, "`")
			if val != "" {
				dsl = val
			} else {
				dslNextLine = true
			}
		} else if strings.HasPrefix(trimmed, "EXPLANATION:") {
			explanation = strings.TrimSpace(strings.TrimPrefix(trimmed, "EXPLANATION:"))
		}
	}
	return dsl, explanation
}
