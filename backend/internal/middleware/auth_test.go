package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthRequired_NoToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired("test-secret"))
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	token, err := GenerateJWT("user-123", "test@example.com", secret)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}
	r := gin.New()
	r.Use(AuthRequired(secret))
	r.GET("/test", func(c *gin.Context) {
		userID, err := GetUserID(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(200, gin.H{"userId": userID})
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired("test-secret"))
	r.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_CookieToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"
	token, err := GenerateJWT("user-456", "cookie@example.com", secret)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}
	r := gin.New()
	r.Use(AuthRequired(secret))
	r.GET("/test", func(c *gin.Context) {
		userID, err := GetUserID(c)
		if err != nil {
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(200, gin.H{"userId": userID})
	})
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "nexus_token", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGenerateJWT(t *testing.T) {
	token, err := GenerateJWT("user-123", "test@example.com", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	claims, err := validateJWT(token, "secret")
	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("expected 'user-123', got '%s'", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("expected 'test@example.com', got '%s'", claims.Email)
	}
	if claims.Issuer != "nexus" {
		t.Errorf("expected issuer 'nexus', got '%s'", claims.Issuer)
	}
	if claims.Subject != "user-123" {
		t.Errorf("expected subject 'user-123', got '%s'", claims.Subject)
	}
}

func TestGetUserID_NoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GetUserID(c)
	if err == nil {
		t.Error("expected error when userId not in context")
	}
}

func TestGetUserID_EmptyString(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set(ContextKeyUserID, "")

	_, err := GetUserID(c)
	if err == nil {
		t.Error("expected error when userId is empty string")
	}
}
