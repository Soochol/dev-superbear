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
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

func setupPresetRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := repository.NewPresetRepository(nil)
	h := handler.NewPresetHandler(repo)

	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		c.Set("userID", "test-user-123")
		c.Next()
	})
	h.RegisterRoutes(api)

	return r
}

func TestPresetHandler_Routes_Registered(t *testing.T) {
	r := setupPresetRouter()

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/search/presets"},
		{http.MethodPost, "/api/v1/search/presets"},
		{http.MethodDelete, "/api/v1/search/presets/some-id"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route should be registered")
		})
	}
}

func TestPresetHandler_CreatePreset_Validation(t *testing.T) {
	r := setupPresetRouter()

	t.Run("rejects empty body", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/presets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects missing name", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"dsl": "scan where volume > 100"})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/search/presets", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPresetHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	repo := repository.NewPresetRepository(nil)
	h := handler.NewPresetHandler(repo)

	api := r.Group("/api/v1")
	h.RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/presets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "authentication required", resp["error"])
}
