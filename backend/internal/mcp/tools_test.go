package mcp_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
	"github.com/dev-superbear/nexus-backend/internal/mcp"
)

func TestGetDSLGrammar(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())
	result := s.HandleGetDSLGrammar()
	assert.Contains(t, result, "scan")
	assert.Contains(t, result, "where")
	assert.Contains(t, result, "sort")
	assert.Contains(t, result, "limit")
	assert.Contains(t, result, "AND")
}

func TestListAvailableFields(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())
	result := s.HandleListAvailableFields()
	assert.Contains(t, result, "volume")
	assert.Contains(t, result, "close")
	assert.Contains(t, result, "trade_value")
	assert.Contains(t, result, "change_pct")
}

func TestValidateDSL(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())

	t.Run("valid query", func(t *testing.T) {
		result, err := s.HandleValidateDSL("scan where volume > 1000000")
		require.NoError(t, err)
		assert.Contains(t, result, `"valid":true`)
	})

	t.Run("invalid query", func(t *testing.T) {
		_, err := s.HandleValidateDSL("bad query")
		assert.Error(t, err)
	})

	t.Run("empty query", func(t *testing.T) {
		_, err := s.HandleValidateDSL("")
		assert.Error(t, err)
	})
}

func TestServer_HandleRequest(t *testing.T) {
	s := mcp.NewServer(dsl.NewExecutor())

	t.Run("initialize", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), `"name":"nexus-dsl"`)
	})

	t.Run("tools/list", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "get_dsl_grammar")
		assert.Contains(t, string(resp), "list_available_fields")
		assert.Contains(t, string(resp), "validate_dsl")
	})

	t.Run("tools/call get_dsl_grammar", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_dsl_grammar","arguments":{}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "scan")
	})

	t.Run("tools/call validate_dsl valid", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"validate_dsl","arguments":{"dsl":"scan where volume > 1000000"}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "valid")
		assert.NotContains(t, string(resp), "isError")
	})

	t.Run("tools/call validate_dsl invalid", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"validate_dsl","arguments":{"dsl":"bad"}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "isError")
	})

	t.Run("tools/call unknown tool", func(t *testing.T) {
		req := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"unknown","arguments":{}}}`
		resp, err := s.HandleRequest([]byte(req))
		require.NoError(t, err)
		assert.Contains(t, string(resp), "error")
	})
}
