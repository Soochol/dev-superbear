package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// PipelineHandler is a thin controller for pipeline-related endpoints.
type PipelineHandler struct {
	queries *sqlc.Queries
}

// NewPipelineHandler creates a PipelineHandler backed by the given queries.
func NewPipelineHandler(queries *sqlc.Queries) *PipelineHandler {
	return &PipelineHandler{queries: queries}
}

// List returns a paginated list of pipelines for the authenticated user.
func (h *PipelineHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	p := GetPagination(c)

	pipelines, err := h.queries.ListPipelinesByUser(c.Request.Context(), sqlc.ListPipelinesByUserParams{
		UserID: parseUUID(userID),
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	count, err := h.queries.CountPipelinesByUser(c.Request.Context(), parseUUID(userID))
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Paginated(c, pipelines, count, p.Page, p.PageSize)
}

// Get returns a single pipeline by ID, scoped to the authenticated user.
func (h *PipelineHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	pipeline, err := h.queries.GetPipeline(c.Request.Context(), sqlc.GetPipelineParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusNotFound, "pipeline not found")
		return
	}

	Success(c, pipeline)
}

// Create validates the request body and creates a new pipeline (placeholder).
func (h *PipelineHandler) Create(c *gin.Context) {
	var req CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: wire up full CreatePipeline with JSON marshalling for analysis_stages/monitors
	Created(c, gin.H{"message": "pipeline created (placeholder)"})
}

// Update validates the request body and updates an existing pipeline (placeholder).
func (h *PipelineHandler) Update(c *gin.Context) {
	var req UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: wire up full UpdatePipeline with JSON marshalling
	Success(c, gin.H{"message": "pipeline updated (placeholder)"})
}

// Delete removes a pipeline by ID, scoped to the authenticated user.
func (h *PipelineHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	err := h.queries.DeletePipeline(c.Request.Context(), sqlc.DeletePipelineParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
