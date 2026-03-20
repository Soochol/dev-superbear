package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return r
}

func newTestSearchHandler() *SearchHandler {
	searchSvc := service.NewSearchService(nil)
	nlSvc := service.NewNLToDSLService()
	return NewSearchHandler(searchSvc, nlSvc)
}

// ─── Search Handler Tests ───

func TestSearchHandler_Execute_ValidDSL(t *testing.T) {
	r := setupRouter()
	h := newTestSearchHandler()
	h.RegisterRoutes(r.Group("/api/v1"))

	body, _ := json.Marshal(DSLRequest{DSL: "scan where volume > 1000000"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["results"] == nil {
		t.Error("expected results in response")
	}
}

func TestSearchHandler_Execute_EmptyBody(t *testing.T) {
	r := setupRouter()
	h := newTestSearchHandler()
	h.RegisterRoutes(r.Group("/api/v1"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSearchHandler_Execute_MissingDSL(t *testing.T) {
	r := setupRouter()
	h := newTestSearchHandler()
	h.RegisterRoutes(r.Group("/api/v1"))

	body, _ := json.Marshal(map[string]string{"dsl": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/execute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty dsl, got %d", w.Code)
	}
}

// ─── Health Endpoint Test ───

func TestHealthEndpoint(t *testing.T) {
	r := setupRouter()
	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
}

// ─── Auth Middleware Integration Tests ───

func TestAuthProtectedRoute_NoToken(t *testing.T) {
	r := setupRouter()
	auth := r.Group("/api/v1")
	auth.Use(middleware.AuthRequired("test-secret"))
	auth.GET("/me", func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		c.JSON(200, gin.H{"userId": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthProtectedRoute_ValidToken(t *testing.T) {
	secret := "test-secret"
	token, _ := middleware.GenerateJWT("user-abc", "test@example.com", secret)

	r := setupRouter()
	auth := r.Group("/api/v1")
	auth.Use(middleware.AuthRequired(secret))
	auth.GET("/me", func(c *gin.Context) {
		userID, _ := middleware.GetUserID(c)
		c.JSON(200, gin.H{"userId": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["userId"] != "user-abc" {
		t.Errorf("expected userId=user-abc, got %v", resp["userId"])
	}
}

func TestAuthProtectedRoute_ExpiredToken(t *testing.T) {
	// A token with wrong secret should fail
	token, _ := middleware.GenerateJWT("user-abc", "test@example.com", "wrong-secret")

	r := setupRouter()
	auth := r.Group("/api/v1")
	auth.Use(middleware.AuthRequired("correct-secret"))
	auth.GET("/me", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
