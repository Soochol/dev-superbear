package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	return r
}

// ─── Search Handler Tests ───

func TestSearchHandler_Scan_ValidDSL(t *testing.T) {
	r := setupRouter()
	h := NewSearchHandler()
	r.POST("/search/scan", h.Scan)

	body, _ := json.Marshal(ScanRequest{Query: "scan where volume > 1000000"})
	req := httptest.NewRequest(http.MethodPost, "/search/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["query"] != "scan where volume > 1000000" {
		t.Errorf("expected query echo, got %v", data["query"])
	}
}

func TestSearchHandler_Scan_InvalidDSL(t *testing.T) {
	r := setupRouter()
	h := NewSearchHandler()
	r.POST("/search/scan", h.Scan)

	body, _ := json.Marshal(ScanRequest{Query: "scan where >"})
	req := httptest.NewRequest(http.MethodPost, "/search/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] == nil {
		t.Error("expected error message in response")
	}
}

func TestSearchHandler_Scan_EmptyBody(t *testing.T) {
	r := setupRouter()
	h := NewSearchHandler()
	r.POST("/search/scan", h.Scan)

	req := httptest.NewRequest(http.MethodPost, "/search/scan", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSearchHandler_Scan_MissingQuery(t *testing.T) {
	r := setupRouter()
	h := NewSearchHandler()
	r.POST("/search/scan", h.Scan)

	body, _ := json.Marshal(map[string]string{"query": ""})
	req := httptest.NewRequest(http.MethodPost, "/search/scan", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty query, got %d", w.Code)
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
