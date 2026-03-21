package llm

import "context"

type EventType string

const (
	EventThinking   EventType = "thinking"
	EventToolCall   EventType = "tool_call"
	EventToolResult EventType = "tool_result"
	EventDSLReady   EventType = "dsl_ready"
	EventDone       EventType = "done"
	EventError      EventType = "error"
)

type Event struct {
	Type        EventType `json:"type"`
	Message     string    `json:"message"`
	DSL         string    `json:"dsl,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
	Data        any       `json:"data,omitempty"`
}

// Provider abstracts LLM backends for NL-to-DSL conversion.
// Implementations: ClaudeCLIProvider (claude -p subprocess), future: ClaudeAPIProvider, GoogleADKProvider.
type Provider interface {
	// NLToDSL streams events converting natural language to DSL.
	// The final meaningful event MUST be EventDSLReady (with DSL + Explanation) or EventError.
	// The provider does NOT execute the DSL — the handler does that after receiving dsl_ready.
	NLToDSL(ctx context.Context, query string) (<-chan Event, error)

	// Explain returns a natural language explanation of a DSL query (synchronous).
	Explain(ctx context.Context, dsl string) (string, error)

	// Name returns the provider identifier (e.g. "claude-cli").
	Name() string
}
