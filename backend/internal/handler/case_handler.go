package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

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
