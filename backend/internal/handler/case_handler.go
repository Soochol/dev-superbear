package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dev-superbear/nexus-backend/internal/domain/casedomain"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// CaseHandler is a thin controller for case-related endpoints.
type CaseHandler struct {
	queries *sqlc.Queries
}

// NewCaseHandler creates a CaseHandler backed by the given queries.
func NewCaseHandler(queries *sqlc.Queries) *CaseHandler {
	return &CaseHandler{queries: queries}
}

// List returns a paginated list of cases for the authenticated user.
func (h *CaseHandler) List(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	p := GetPagination(c)

	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	cases, err := h.queries.ListCasesByUser(c.Request.Context(), sqlc.ListCasesByUserParams{
		UserID: userUUID,
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		slog.Error("failed to list cases", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	count, err := h.queries.CountCasesByUser(c.Request.Context(), userUUID)
	if err != nil {
		slog.Error("failed to count cases", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Paginated(c, cases, count, p.Page, p.PageSize)
}

// Get returns a single case by ID, scoped to the authenticated user.
func (h *CaseHandler) Get(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	idUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	cs, err := h.queries.GetCase(c.Request.Context(), sqlc.GetCaseParams{
		ID:     idUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get case", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, cs)
}

// Create validates the request body and creates a new case (placeholder).
func (h *CaseHandler) Create(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req CreateCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: wire up full CreateCase with JSON marshalling for event_snapshot
	_ = userID
	Created(c, gin.H{"message": "case created (placeholder)"})
}

// Delete removes a case by ID, scoped to the authenticated user.
func (h *CaseHandler) Delete(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	idUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.queries.DeleteCase(c.Request.Context(), sqlc.DeleteCaseParams{
		ID:     idUUID,
		UserID: userUUID,
	})
	if err != nil {
		slog.Error("failed to delete case", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}

// closeCaseRequest is the JSON body for POST /cases/:id/close.
type closeCaseRequest struct {
	Status string `json:"status" binding:"required,oneof=CLOSED_SUCCESS CLOSED_FAILURE"`
	Reason string `json:"reason"`
}

// Close transitions a case to a closed status.
func (h *CaseHandler) Close(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	idUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var req closeCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	result, err := h.queries.UpdateCaseStatus(c.Request.Context(), sqlc.UpdateCaseStatusParams{
		ID:           idUUID,
		Status:       sqlc.CaseStatus(req.Status),
		ClosedAt:     pgtype.Date{Time: time.Now(), Valid: true},
		ClosedReason: pgtype.Text{String: req.Reason, Valid: true},
		UserID:       userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to close case", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, result)
}

// GetTimeline returns timeline events for a case, with optional type filter and paging.
func (h *CaseHandler) GetTimeline(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	caseUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx := c.Request.Context()

	_, err = h.queries.GetCase(ctx, sqlc.GetCaseParams{
		ID:     caseUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get case for timeline", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	eventType := c.Query("type")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	if eventType != "" {
		events, err := h.queries.ListTimelineEventsByType(ctx, sqlc.ListTimelineEventsByTypeParams{
			CaseID: caseUUID,
			Type:   sqlc.TimelineEventType(eventType),
		})
		if err != nil {
			slog.Error("failed to list timeline events by type", "error", err, "caseId", id)
			Error(c, http.StatusInternalServerError, "internal server error")
			return
		}
		Success(c, events)
		return
	}

	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			Error(c, http.StatusBadRequest, "invalid limit parameter")
			return
		}
		if limit < 1 {
			limit = 20
		}
		offset := 0
		if offsetStr != "" {
			offset, err = strconv.Atoi(offsetStr)
			if err != nil {
				Error(c, http.StatusBadRequest, "invalid offset parameter")
				return
			}
			if offset < 0 {
				offset = 0
			}
		}
		events, err := h.queries.ListTimelineEventsWithPaging(ctx, sqlc.ListTimelineEventsWithPagingParams{
			CaseID: caseUUID,
			Limit:  int32(limit),
			Offset: int32(offset),
		})
		if err != nil {
			slog.Error("failed to list timeline events with paging", "error", err, "caseId", id)
			Error(c, http.StatusInternalServerError, "internal server error")
			return
		}
		Success(c, events)
		return
	}

	events, err := h.queries.ListTimelineEventsByCase(ctx, caseUUID)
	if err != nil {
		slog.Error("failed to list timeline events", "error", err, "caseId", id)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	Success(c, events)
}

// GetReturnTracking returns return tracking data for a case.
// Price history is not yet available; returns a stub with empty periods.
func (h *CaseHandler) GetReturnTracking(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	idUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	_, err = h.queries.GetCase(c.Request.Context(), sqlc.GetCaseParams{
		ID:     idUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get case for return tracking", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	result := casedomain.ReturnTrackingData{Periods: []casedomain.ReturnPeriod{}}
	Success(c, result)
}
