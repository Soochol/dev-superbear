// Package gemini implements the llm.Provider interface using the Google GenAI SDK
// with function calling for NL-to-DSL conversion.
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
	"google.golang.org/genai"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/llm/tools"
)

const (
	defaultModel          = "gemini-2.0-flash"
	defaultMaxConcurrent  = 5
	defaultTimeoutSeconds = 120
	defaultMaxOutputTokens = int32(4096)
	maxToolRounds         = 10
)

// Provider implements llm.Provider using the Google GenAI SDK.
type Provider struct {
	cfg      config.LLMConfig
	sem      *semaphore.Weighted
	toolExec *tools.Executor
}

// New creates a Gemini provider with the given configuration and optional tool executor.
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

	return &Provider{
		cfg:      cfg,
		sem:      semaphore.NewWeighted(int64(cfg.MaxConcurrent)),
		toolExec: toolExec,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "gemini" }

// NLToDSL converts a natural language query to DSL using the Gemini API
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

	// Create a new client for this request.
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  p.cfg.GeminiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		send(llm.Event{Type: llm.EventError, Message: fmt.Sprintf("failed to create Gemini client: %v", err)})
		return
	}

	// Build tool declarations from DSLToolDefinitions.
	funcDecls := buildFunctionDeclarations()

	// Build the conversation contents, starting with the user query.
	contents := []*genai.Content{
		{Role: genai.RoleUser, Parts: []*genai.Part{{Text: query}}},
	}

	genConfig := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: llm.NLToDSLAPIPrompt}},
		},
		MaxOutputTokens: defaultMaxOutputTokens,
		Tools: []*genai.Tool{
			{FunctionDeclarations: funcDecls},
		},
	}

	for round := 0; round < maxToolRounds; round++ {
		if ctx.Err() != nil {
			ch <- llm.Event{Type: llm.EventError, Message: "context canceled"}
			return
		}

		resp, err := client.Models.GenerateContent(ctx, p.cfg.Model, contents, genConfig)
		if err != nil {
			if ctx.Err() != nil {
				ch <- llm.Event{Type: llm.EventError, Message: "request timed out or canceled"}
			} else {
				ch <- llm.Event{Type: llm.EventError, Message: fmt.Sprintf("API call failed: %v", err)}
			}
			return
		}

		if len(resp.Candidates) == 0 {
			send(llm.Event{Type: llm.EventError, Message: "no candidates in Gemini response"})
			return
		}

		candidate := resp.Candidates[0]
		if candidate.Content == nil {
			send(llm.Event{Type: llm.EventError, Message: "empty content in Gemini response"})
			return
		}

		// Check if any function calls are present.
		hasFunctionCall := false
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				hasFunctionCall = true
				break
			}
		}

		if !hasFunctionCall {
			// No function calls — try to extract DSL from text response as fallback.
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					dslStr, explanation := extractDSL(part.Text)
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

		// Add model's response turn to conversation history.
		contents = append(contents, candidate.Content)

		// Process function calls and collect function response parts.
		funcResponseParts := make([]*genai.Part, 0)

		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				if !send(llm.Event{Type: llm.EventThinking, Message: part.Text}) {
					return
				}
			}

			if part.FunctionCall == nil {
				continue
			}

			fc := part.FunctionCall

			if fc.Name == "submit_dsl" {
				// Terminal tool: extract dsl and explanation from args map.
				dslVal, _ := fc.Args["dsl"].(string)
				explanationVal, _ := fc.Args["explanation"].(string)
				send(llm.Event{
					Type:        llm.EventDSLReady,
					Message:     "DSL generated",
					DSL:         dslVal,
					Explanation: explanationVal,
				})
				return
			}

			// Non-terminal tool call.
			if !send(llm.Event{
				Type:    llm.EventToolCall,
				Message: fmt.Sprintf("Calling tool: %s", fc.Name),
				Data: map[string]any{
					"tool_name": fc.Name,
					"tool_id":   fc.ID,
				},
			}) {
				return
			}

			// Marshal args to json.RawMessage for DispatchTool.
			argsJSON, marshalErr := json.Marshal(fc.Args)
			if marshalErr != nil {
				slog.Warn("failed to marshal function call args", "tool", fc.Name, "error", marshalErr)
				argsJSON = []byte("{}")
			}

			result, dispatchErr := p.toolExec.DispatchTool(fc.Name, argsJSON)
			var resultContent string
			if dispatchErr != nil {
				slog.Warn("tool dispatch error", "tool", fc.Name, "error", dispatchErr)
				resultContent = fmt.Sprintf("error: %v", dispatchErr)
			} else {
				resultContent = result
			}

			if !send(llm.Event{
				Type:    llm.EventToolResult,
				Message: fmt.Sprintf("Tool %s result", fc.Name),
				Data:    resultContent,
			}) {
				return
			}

			funcResponseParts = append(funcResponseParts, genai.NewPartFromFunctionResponse(
				fc.Name,
				map[string]any{"output": resultContent},
			))
		}

		if len(funcResponseParts) > 0 {
			contents = append(contents, &genai.Content{
				Role:  genai.RoleUser,
				Parts: funcResponseParts,
			})
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

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  p.cfg.GeminiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	resp, err := client.Models.GenerateContent(
		ctx,
		p.cfg.Model,
		genai.Text(dsl),
		&genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: llm.ExplainPrompt}},
			},
			MaxOutputTokens: defaultMaxOutputTokens,
		},
	)
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("explain timed out or canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("explain API call failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in explain response")
	}
	candidate := resp.Candidates[0]
	if candidate.Content == nil {
		return "", fmt.Errorf("empty content in explain response")
	}

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			return strings.TrimSpace(part.Text), nil
		}
	}
	return "", fmt.Errorf("no text content in explain response")
}

// buildFunctionDeclarations converts DSLToolDefinitions to GenAI FunctionDeclaration format.
// It uses ParametersJsonSchema to pass the raw JSON schema map directly.
func buildFunctionDeclarations() []*genai.FunctionDeclaration {
	defs := tools.DSLToolDefinitions()
	result := make([]*genai.FunctionDeclaration, 0, len(defs))
	for _, def := range defs {
		fd := &genai.FunctionDeclaration{
			Name:                def.Name,
			Description:         def.Description,
			ParametersJsonSchema: def.Parameters,
		}
		result = append(result, fd)
	}
	return result
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
