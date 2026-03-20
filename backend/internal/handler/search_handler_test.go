package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/handler"
	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }

func (m *mockProvider) Explain(_ context.Context, dsl string) (string, error) {
	return "Mock explanation for: " + dsl, nil
}

func (m *mockProvider) NLToDSL(_ context.Context, _ string) (<-chan llm.Event, error) {
	ch := make(chan llm.Event, 3)
	ch <- llm.Event{Type: llm.EventThinking, Message: "분석 중..."}
	ch <- llm.Event{Type: llm.EventDSLReady, DSL: "scan where volume > 1000000", Explanation: "거래량 100만 이상", Message: "생성 완료"}
	close(ch)
	return ch, nil
}

func setupSearchRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	searchSvc := service.NewSearchService(nil)
	nlSvc := service.NewNLToDSLService(&mockProvider{})
	h := handler.NewSearchHandler(searchSvc, nlSvc)

	api := r.Group("/api/v1")
	h.RegisterRoutes(api)

	return r
}

func TestSearchHandler_Execute(t *testing.T) {
	r := setupSearchRouter()

	t.Run("returns results for valid DSL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "results")
	})

	t.Run("rejects missing dsl", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_Validate(t *testing.T) {
	r := setupSearchRouter()

	t.Run("returns valid for correct DSL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/validate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, true, resp["valid"])
	})
}

func TestSearchHandler_NLToDSL_SSE(t *testing.T) {
	r := setupSearchRouter()

	body, _ := json.Marshal(map[string]string{"query": "거래량 많은 종목"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/nl-to-dsl", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	responseBody := w.Body.String()
	assert.Contains(t, responseBody, "event: thinking")
	assert.Contains(t, responseBody, "event: dsl_ready")
	assert.Contains(t, responseBody, "event: done")
}

func TestSearchHandler_Explain_WithProvider(t *testing.T) {
	r := setupSearchRouter()

	body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/explain", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Contains(t, resp["explanation"], "Mock explanation")
}
