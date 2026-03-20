package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

// MonitoringHandler exposes HTTP endpoints for monitoring control.
type MonitoringHandler struct {
	svc *service.MonitoringService
}

func NewMonitoringHandler(svc *service.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{svc: svc}
}

// ToggleBlock handles PATCH /api/v1/cases/:id/monitors/:monitorId
func (h *MonitoringHandler) ToggleBlock(c *gin.Context) {
	if _, err := middleware.GetUserID(c); err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	monitorID := c.Param("monitorId")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ToggleMonitorBlock(monitorID, body.Enabled); err != nil {
		slog.Error("failed to toggle monitor block", "monitorId", monitorID, "error", err)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ToggleCaseMonitoring handles PATCH /api/v1/cases/:id/monitoring-status
func (h *MonitoringHandler) ToggleCaseMonitoring(c *gin.Context) {
	if _, err := middleware.GetUserID(c); err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	caseID := c.Param("id")
	var body struct {
		Enabled        bool `json:"enabled"`
		KeepDSLPolling bool `json:"keep_dsl_polling"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ToggleCaseMonitoring(caseID, body.Enabled, body.KeepDSLPolling); err != nil {
		slog.Error("failed to toggle case monitoring", "caseId", caseID, "error", err)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ListMonitors handles GET /api/v1/cases/:id/monitors
func (h *MonitoringHandler) ListMonitors(c *gin.Context) {
	if _, err := middleware.GetUserID(c); err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	caseID := c.Param("id")
	blocks, err := h.svc.ListMonitorBlocks(caseID)
	if err != nil {
		slog.Error("failed to list monitors", "caseId", caseID, "error", err)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	Success(c, blocks)
}
