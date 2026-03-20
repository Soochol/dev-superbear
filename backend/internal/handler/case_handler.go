package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	userID := middleware.GetUserID(c)
	p := GetPagination(c)

	cases, err := h.queries.ListCasesByUser(c.Request.Context(), sqlc.ListCasesByUserParams{
		UserID: parseUUID(userID),
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	count, err := h.queries.CountCasesByUser(c.Request.Context(), parseUUID(userID))
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Paginated(c, cases, count, p.Page, p.PageSize)
}

// Get returns a single case by ID, scoped to the authenticated user.
func (h *CaseHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	cs, err := h.queries.GetCase(c.Request.Context(), sqlc.GetCaseParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusNotFound, "case not found")
		return
	}

	Success(c, cs)
}

// Create validates the request body and creates a new case (placeholder).
func (h *CaseHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
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
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	err := h.queries.DeleteCase(c.Request.Context(), sqlc.DeleteCaseParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
