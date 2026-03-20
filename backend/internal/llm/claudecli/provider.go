// Package claudecli implements the llm.Provider interface using claude -p as a subprocess.
package claudecli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/dev-superbear/nexus-backend/internal/config"
	"github.com/dev-superbear/nexus-backend/internal/llm"
)

// Provider executes claude -p as a subprocess and streams events.
type Provider struct {
	cfg       config.LLMConfig
	sem       *semaphore.Weighted
	nlPrompt  string
	expPrompt string
}

// New creates a ClaudeCLI provider with the given configuration.
// It uses embedded system prompts from the llm package.
func New(cfg config.LLMConfig) *Provider {
	if cfg.ClaudeCLIPath == "" {
		cfg.ClaudeCLIPath = "claude"
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 60
	}
	return &Provider{
		cfg:       cfg,
		sem:       semaphore.NewWeighted(int64(cfg.MaxConcurrent)),
		nlPrompt:  llm.NLToDSLPrompt,
		expPrompt: llm.ExplainPrompt,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string { return "claude-cli" }

// newCommand creates a configured exec.Cmd with common options:
// process-group isolation, signal-based cancellation, and optional MCP config.
func (p *Provider) newCommand(ctx context.Context, outputFormat string) *exec.Cmd {
	args := []string{"-p", "--output-format", outputFormat}
	if p.cfg.MCPConfigPath != "" {
		args = append(args, "--mcp-config", p.cfg.MCPConfigPath)
	}

	cmd := exec.CommandContext(ctx, p.cfg.ClaudeCLIPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 2 * time.Second
	return cmd
}

// NLToDSL streams events converting natural language to DSL via claude -p subprocess.
func (p *Provider) NLToDSL(ctx context.Context, query string) (<-chan llm.Event, error) {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquiring semaphore: %w", err)
	}

	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)

	// cleanup releases resources if setup fails before the goroutine takes ownership.
	started := false
	cleanup := func() {
		if !started {
			cancel()
			p.sem.Release(1)
		}
	}
	defer cleanup()

	ch := make(chan llm.Event, 16)
	cmd := p.newCommand(ctx, "stream-json")
	cmd.Stdin = strings.NewReader(p.nlPrompt + "\n\n---\nUser query: " + query)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting claude process: %w", err)
	}

	started = true

	// Emit initial thinking event.
	ch <- llm.Event{Type: llm.EventThinking, Message: "Analyzing query..."}

	go func() {
		defer close(ch)
		defer cancel()
		defer p.sem.Release(1)

		// Read stderr in the background.
		var stderrBuf bytes.Buffer
		stderrDone := make(chan struct{})
		go func() {
			defer close(stderrDone)
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				stderrBuf.Write(scanner.Bytes())
				stderrBuf.WriteByte('\n')
			}
		}()

		scanner := bufio.NewScanner(stdout)
		// Allow up to 1MB lines (claude output can be large).
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		var lastText string
		var dslEmitted bool

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			event, err := ParseStreamLine(line)
			if err != nil {
				slog.Debug("skipping unparseable stream line", "error", err, "line", string(line))
				continue
			}
			if event == nil {
				continue
			}

			// Track DSL emission and text for post-processing.
			if event.Type == llm.EventDSLReady {
				dslEmitted = true
			}
			if event.Type == llm.EventDone && event.Message != "" {
				lastText = event.Message
			}

			select {
			case ch <- *event:
			case <-ctx.Done():
				ch <- llm.Event{Type: llm.EventError, Message: "context canceled"}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			slog.Warn("scanner error reading claude stdout", "error", err)
		}

		// Wait for process to finish.
		waitErr := cmd.Wait()
		<-stderrDone

		// Check context cancellation first.
		if ctx.Err() != nil {
			ch <- llm.Event{Type: llm.EventError, Message: "request timed out or canceled"}
			return
		}

		if waitErr != nil {
			stderrStr := strings.TrimSpace(stderrBuf.String())
			msg := fmt.Sprintf("claude process exited with error: %v", waitErr)
			if stderrStr != "" {
				msg += "; stderr: " + stderrStr
			}
			ch <- llm.Event{Type: llm.EventError, Message: msg}
			return
		}

		// If DSL was already emitted via streaming, we're done.
		if dslEmitted {
			return
		}

		// If we captured text from a "result" event, try to extract DSL.
		if lastText != "" {
			dsl, explanation := extractDSL(lastText)
			if dsl != "" {
				ch <- llm.Event{
					Type:        llm.EventDSLReady,
					Message:     "DSL generated",
					DSL:         dsl,
					Explanation: explanation,
				}
				return
			}
		}

		// If no DSL was emitted by stream parsing, emit error.
		ch <- llm.Event{Type: llm.EventError, Message: "no DSL found in claude output"}
	}()

	return ch, nil
}

// Explain runs claude -p synchronously and returns a plain-text explanation.
func (p *Provider) Explain(ctx context.Context, dsl string) (string, error) {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return "", fmt.Errorf("acquiring semaphore: %w", err)
	}
	defer p.sem.Release(1)

	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := p.newCommand(ctx, "text")
	cmd.Stdin = strings.NewReader(p.expPrompt + "\n\n" + dsl)

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("explain timed out or canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("claude explain failed: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// --- Stream JSON parsing ---

// streamMessage represents a claude stream-json line.
type streamMessage struct {
	Type    string       `json:"type"`
	Message *messageBody `json:"message,omitempty"`
	Result  string       `json:"result,omitempty"`
}

type messageBody struct {
	Content []contentItem `json:"content,omitempty"`
}

type contentItem struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`
	ID    string `json:"id,omitempty"`
	Input any    `json:"input,omitempty"`
}

// ParseStreamLine parses a single line from claude -p stream-json output
// and converts it into an llm.Event. Returns (nil, nil) for lines that
// should be skipped (e.g., unknown or irrelevant event types).
func ParseStreamLine(line []byte) (*llm.Event, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}

	var msg streamMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	switch msg.Type {
	case "assistant":
		return parseAssistantMessage(&msg)
	case "result":
		return parseResultMessage(&msg)
	default:
		// Unknown type — skip silently.
		return nil, nil
	}
}

func parseAssistantMessage(msg *streamMessage) (*llm.Event, error) {
	if msg.Message == nil {
		return nil, nil
	}
	for _, item := range msg.Message.Content {
		switch item.Type {
		case "tool_use":
			return &llm.Event{
				Type:    llm.EventToolCall,
				Message: fmt.Sprintf("Calling tool: %s", item.Name),
				Data: map[string]any{
					"tool_name": item.Name,
					"tool_id":   item.ID,
					"input":     item.Input,
				},
			}, nil
		case "tool_result":
			return &llm.Event{
				Type:    llm.EventToolResult,
				Message: item.Text,
			}, nil
		case "text":
			// Check if text contains DSL output.
			dsl, explanation := extractDSL(item.Text)
			if dsl != "" {
				return &llm.Event{
					Type:        llm.EventDSLReady,
					Message:     "DSL generated",
					DSL:         dsl,
					Explanation: explanation,
				}, nil
			}
			// Otherwise it's intermediate text — emit as thinking.
			return &llm.Event{
				Type:    llm.EventThinking,
				Message: item.Text,
			}, nil
		}
	}
	return nil, nil
}

func parseResultMessage(msg *streamMessage) (*llm.Event, error) {
	if msg.Result == "" {
		return nil, nil
	}

	// Try to extract DSL from the result.
	dsl, explanation := extractDSL(msg.Result)
	if dsl != "" {
		return &llm.Event{
			Type:        llm.EventDSLReady,
			Message:     "DSL generated",
			DSL:         dsl,
			Explanation: explanation,
		}, nil
	}

	// Return as done event with the raw text for post-processing.
	return &llm.Event{
		Type:    llm.EventDone,
		Message: msg.Result,
	}, nil
}

// extractDSL looks for "DSL:" and "EXPLANATION:" lines in the text.
func extractDSL(text string) (dsl, explanation string) {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "DSL:") {
			dsl = strings.TrimSpace(strings.TrimPrefix(trimmed, "DSL:"))
		} else if strings.HasPrefix(trimmed, "EXPLANATION:") {
			explanation = strings.TrimSpace(strings.TrimPrefix(trimmed, "EXPLANATION:"))
		}
	}
	return dsl, explanation
}
