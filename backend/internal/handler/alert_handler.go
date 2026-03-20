package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// AlertHandler is a thin controller for price alert endpoints.
type AlertHandler struct {
	queries *sqlc.Queries
}

// NewAlertHandler creates an AlertHandler backed by the given queries.
func NewAlertHandler(queries *sqlc.Queries) *AlertHandler {
	return &AlertHandler{queries: queries}
}

// ListAlerts returns pending and triggered alerts for a case.
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	caseID := c.Param("id")
	caseUUID, err := parseUUID(caseID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()

	pending, err := h.queries.ListPendingAlertsByCase(ctx, caseUUID)
	if err != nil {
		slog.Error("failed to list pending alerts", "error", err, "caseId", caseID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	triggered, err := h.queries.ListTriggeredAlertsByCase(ctx, caseUUID)
	if err != nil {
		slog.Error("failed to list triggered alerts", "error", err, "caseId", caseID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Success(c, gin.H{
		"pending":   pending,
		"triggered": triggered,
	})
}

// CreateAlert creates a new price alert for a case.
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	caseID := c.Param("id")
	caseUUID, err := parseUUID(caseID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	pipelineUUID := pgtype.UUID{}
	if req.PipelineID != nil {
		pipelineUUID, err = parseUUID(*req.PipelineID)
		if err != nil {
			Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}

	alert, err := h.queries.CreatePriceAlert(c.Request.Context(), sqlc.CreatePriceAlertParams{
		CaseID:     caseUUID,
		PipelineID: pipelineUUID,
		Condition:  req.Condition,
		Label:      req.Label,
	})
	if err != nil {
		slog.Error("failed to create alert", "error", err, "caseId", caseID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Created(c, alert)
}

// DeleteAlert removes a price alert by ID, scoped to the case.
func (h *AlertHandler) DeleteAlert(c *gin.Context) {
	caseID := c.Param("id")
	alertID := c.Param("alertId")

	caseUUID, err := parseUUID(caseID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	alertUUID, err := parseUUID(alertID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.queries.DeletePriceAlert(c.Request.Context(), sqlc.DeletePriceAlertParams{
		ID:     alertUUID,
		CaseID: caseUUID,
	})
	if err != nil {
		slog.Error("failed to delete alert", "error", err, "caseId", caseID, "alertId", alertID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}
