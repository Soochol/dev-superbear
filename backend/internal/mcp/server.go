package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// JSON-RPC types

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP content types

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

func textResult(text string) toolResult {
	return toolResult{Content: []textContent{{Type: "text", Text: text}}}
}

func errorResult(text string) toolResult {
	return toolResult{Content: []textContent{{Type: "text", Text: text}}, IsError: true}
}

// Tool schema types

type toolInputSchema struct {
	Type       string                     `json:"type"`
	Properties map[string]toolInputProp   `json:"properties,omitempty"`
	Required   []string                   `json:"required,omitempty"`
}

type toolInputProp struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type toolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema toolInputSchema `json:"inputSchema"`
}

// toolsList returns the MCP tool definitions.
func toolsList() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "get_dsl_grammar",
			Description: "Returns the DSL grammar and syntax reference for writing scan queries.",
			InputSchema: toolInputSchema{Type: "object"},
		},
		{
			Name:        "list_available_fields",
			Description: "Lists all fields available in scan queries with their types and descriptions.",
			InputSchema: toolInputSchema{Type: "object"},
		},
		{
			Name:        "validate_dsl",
			Description: "Validates a DSL query string and returns a JSON result indicating validity.",
			InputSchema: toolInputSchema{
				Type: "object",
				Properties: map[string]toolInputProp{
					"dsl": {
						Type:        "string",
						Description: "The DSL query string to validate.",
					},
				},
				Required: []string{"dsl"},
			},
		},
	}
}

// HandleRequest processes a single JSON-RPC request and returns the response bytes.
func (s *Server) HandleRequest(data []byte) ([]byte, error) {
	var req rpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return s.errorResponse(nil, -32700, "parse error"), nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return s.errorResponse(req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method)), nil
	}
}

func (s *Server) handleInitialize(req rpcRequest) ([]byte, error) {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "nexus-dsl",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
	}
	return s.successResponse(req.ID, result), nil
}

func (s *Server) handleToolsList(req rpcRequest) ([]byte, error) {
	result := map[string]any{
		"tools": toolsList(),
	}
	return s.successResponse(req.ID, result), nil
}

func (s *Server) handleToolsCall(req rpcRequest) ([]byte, error) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "invalid params"), nil
	}

	switch params.Name {
	case "get_dsl_grammar":
		return s.successResponse(req.ID, textResult(s.HandleGetDSLGrammar())), nil

	case "list_available_fields":
		return s.successResponse(req.ID, textResult(s.HandleListAvailableFields())), nil

	case "validate_dsl":
		var args struct {
			DSL string `json:"dsl"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return s.errorResponse(req.ID, -32602, "invalid arguments"), nil
		}
		text, err := s.HandleValidateDSL(args.DSL)
		if err != nil {
			return s.successResponse(req.ID, errorResult(err.Error())), nil
		}
		return s.successResponse(req.ID, textResult(text)), nil

	default:
		return s.errorResponse(req.ID, -32602, fmt.Sprintf("unknown tool: %s", params.Name)), nil
	}
}

func (s *Server) successResponse(id any, result any) []byte {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return s.errorResponse(id, -32603, "internal: failed to marshal response")
	}
	return b
}

func (s *Server) errorResponse(id any, code int, message string) []byte {
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   rpcError{Code: code, Message: message},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		// Last-resort fallback — hardcoded valid JSON-RPC error
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal marshal error"}}`)
	}
	return b
}

// Run reads JSON-RPC requests from stdin line-by-line, dispatches each to
// HandleRequest, and writes the response to stdout.
func (s *Server) Run(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("stdin read error: %w", err)
			}
			return nil // EOF
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp, err := s.HandleRequest(line)
		if err != nil {
			return fmt.Errorf("handle request: %w", err)
		}

		if _, err := fmt.Fprintf(os.Stdout, "%s\n", resp); err != nil {
			return fmt.Errorf("stdout write error: %w", err)
		}
	}
}
