package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dev-superbear/nexus-backend/internal/handler"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

func setupSearchRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	searchSvc := service.NewSearchService(nil)
	nlSvc := service.NewNLToDSLService()
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

func TestSearchHandler_NLToDSL(t *testing.T) {
	r := setupSearchRouter()

	t.Run("converts NL query to DSL", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"query": "2년 최대거래량 종목"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/nl-to-dsl", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "dsl")
		assert.Contains(t, resp, "explanation")
		assert.Contains(t, resp, "results")
	})

	t.Run("rejects empty query", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"query": ""})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/nl-to-dsl", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestSearchHandler_Explain(t *testing.T) {
	r := setupSearchRouter()

	t.Run("explains DSL in natural language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 1000000"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/explain", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "explanation")
	})
}
