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

	pipelines, err := h.queries.ListPipelinesByUser(c.Request.Context(), sqlc.ListPipelinesByUserParams{
		UserID: userUUID,
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		slog.Error("failed to list pipelines", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	count, err := h.queries.CountPipelinesByUser(c.Request.Context(), userUUID)
	if err != nil {
		slog.Error("failed to count pipelines", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Paginated(c, pipelines, count, p.Page, p.PageSize)
}

// Get returns a single pipeline by ID, scoped to the authenticated user.
func (h *PipelineHandler) Get(c *gin.Context) {
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

	pipeline, err := h.queries.GetPipelineByID(c.Request.Context(), sqlc.GetPipelineByIDParams{
		ID:     idUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get pipeline", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
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

	err = h.queries.DeletePipeline(c.Request.Context(), sqlc.DeletePipelineParams{
		ID:     idUUID,
		UserID: userUUID,
	})
	if err != nil {
		slog.Error("failed to delete pipeline", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}
