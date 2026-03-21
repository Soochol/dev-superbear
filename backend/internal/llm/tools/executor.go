package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

// Executor provides DSL tool functions shared by API providers and MCP server.
type Executor struct {
	dslExec *dsl.Executor
}

func NewExecutor(dslExec *dsl.Executor) *Executor {
	return &Executor{dslExec: dslExec}
}

func (e *Executor) GetGrammar() string {
	return e.dslExec.GrammarText()
}

func (e *Executor) ListFields() string {
	fields := e.dslExec.AvailableFields()
	ops := e.dslExec.AllowedOps()

	lines := make([]string, 0, len(fields)+3)
	lines = append(lines, "Available fields:")

	maxLen := 0
	for _, f := range fields {
		if len(f.Name) > maxLen {
			maxLen = len(f.Name)
		}
	}

	for _, f := range fields {
		padding := strings.Repeat(" ", maxLen-len(f.Name))
		lines = append(lines, fmt.Sprintf("  %s%s — %s (%s)", f.Name, padding, f.Description, f.Unit))
	}

	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("All fields support operators: %s", strings.Join(ops, ", ")))
	lines = append(lines, "All fields can be used in sort by clause.")
	return strings.Join(lines, "\n")
}

func (e *Executor) ValidateDSL(dslCode string) (string, error) {
	result := e.dslExec.Validate(dslCode)
	if !result.Valid {
		return "", fmt.Errorf("%s", result.Error)
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

// DispatchTool routes a tool call to the appropriate function.
// NOTE: submit_dsl is NOT handled here — it is a terminal action
// handled directly by the provider's function-calling loop.
func (e *Executor) DispatchTool(name string, args json.RawMessage) (string, error) {
	switch name {
	case "get_dsl_grammar":
		return e.GetGrammar(), nil
	case "list_available_fields":
		return e.ListFields(), nil
	case "validate_dsl":
		var a struct {
			DSL string `json:"dsl"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("invalid validate_dsl args: %w", err)
		}
		return e.ValidateDSL(a.DSL)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
