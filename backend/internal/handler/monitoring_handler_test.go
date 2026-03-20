package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dev-superbear/nexus-backend/internal/middleware"
)

// ─── No Auth → 401 Tests ───

func TestMonitoringHandler_ListMonitors_NoAuth(t *testing.T) {
	r := setupRouter()
	h := NewMonitoringHandler(nil)
	r.GET("/cases/:id/monitors", h.ListMonitors)

	req := httptest.NewRequest(http.MethodGet, "/cases/some-id/monitors", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMonitoringHandler_ToggleBlock_NoAuth(t *testing.T) {
	r := setupRouter()
	h := NewMonitoringHandler(nil)
	r.PATCH("/cases/:id/monitors/:monitorId", h.ToggleBlock)

	req := httptest.NewRequest(http.MethodPatch, "/cases/id1/monitors/mid1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMonitoringHandler_ToggleCaseMonitoring_NoAuth(t *testing.T) {
	r := setupRouter()
	h := NewMonitoringHandler(nil)
	r.PATCH("/cases/:id/monitoring-status", h.ToggleCaseMonitoring)

	req := httptest.NewRequest(http.MethodPatch, "/cases/id1/monitoring-status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Invalid Body → 400 Tests (with auth) ───

func TestMonitoringHandler_ToggleBlock_InvalidBody(t *testing.T) {
	secret := "test-secret"
	token, _ := middleware.GenerateJWT("user-1", "test@test.com", secret)

	r := setupRouter()
	h := NewMonitoringHandler(nil)
	auth := r.Group("")
	auth.Use(middleware.AuthRequired(secret))
	auth.PATCH("/cases/:id/monitors/:monitorId", h.ToggleBlock)

	req := httptest.NewRequest(http.MethodPatch, "/cases/id1/monitors/mid1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMonitoringHandler_ToggleCaseMonitoring_InvalidBody(t *testing.T) {
	secret := "test-secret"
	token, _ := middleware.GenerateJWT("user-1", "test@test.com", secret)

	r := setupRouter()
	h := NewMonitoringHandler(nil)
	auth := r.Group("")
	auth.Use(middleware.AuthRequired(secret))
	auth.PATCH("/cases/:id/monitoring-status", h.ToggleCaseMonitoring)

	req := httptest.NewRequest(http.MethodPatch, "/cases/id1/monitoring-status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
