package tools

// ToolDef describes a tool for LLM function calling.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// DSLToolDefinitions returns tool definitions for API-based providers.
func DSLToolDefinitions() []ToolDef {
	return []ToolDef{
		{
			Name:        "get_dsl_grammar",
			Description: "Returns the DSL grammar and syntax reference for writing scan queries.",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "list_available_fields",
			Description: "Lists all fields available in scan queries with their types and descriptions.",
			Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "validate_dsl",
			Description: "Validates a DSL query string and returns a JSON result indicating validity.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"dsl": map[string]any{
						"type":        "string",
						"description": "The DSL query string to validate.",
					},
				},
				"required": []string{"dsl"},
			},
		},
		{
			Name:        "submit_dsl",
			Description: "Submit the final validated DSL query and its Korean explanation. Call this ONLY after validate_dsl confirms the query is valid.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"dsl": map[string]any{
						"type":        "string",
						"description": "The validated DSL query string.",
					},
					"explanation": map[string]any{
						"type":        "string",
						"description": "Korean explanation of what the query does.",
					},
				},
				"required": []string{"dsl", "explanation"},
			},
		},
	}
}
