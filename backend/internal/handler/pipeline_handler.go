package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/dev-superbear/nexus-backend/internal/domain"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

// PipelineHandler is a thin controller for pipeline-related endpoints.
type PipelineHandler struct {
	svc       *service.PipelineService
	generator *service.PipelineGenerator
}

// NewPipelineHandler creates a PipelineHandler backed by the given service.
func NewPipelineHandler(svc *service.PipelineService, generator *service.PipelineGenerator) *PipelineHandler {
	return &PipelineHandler{svc: svc, generator: generator}
}

// RegisterRoutes mounts pipeline routes on the given router group.
func (h *PipelineHandler) RegisterRoutes(rg *gin.RouterGroup) {
	pipelines := rg.Group("/pipelines")
	pipelines.GET("", h.List)
	pipelines.POST("", h.Create)
	pipelines.POST("/generate", h.Generate)
	pipelines.GET("/jobs/:jobId", h.GetJob)
	pipelines.GET("/:id", h.Get)
	pipelines.PUT("/:id", h.Update)
	pipelines.DELETE("/:id", h.Delete)
	pipelines.POST("/:id/execute", h.Execute)
}

// List returns a paginated list of pipelines for the authenticated user.
func (h *PipelineHandler) List(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	p := GetPagination(c)

	pipelines, count, err := h.svc.List(c.Request.Context(), userID, int32(p.PageSize), int32(p.Offset))
	if err != nil {
		slog.Error("failed to list pipelines", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Paginated(c, pipelines, count, p.Page, p.PageSize)
}

// Get returns a single pipeline by ID with all relations.
func (h *PipelineHandler) Get(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	pipeline, err := h.svc.GetByID(c.Request.Context(), userID, id)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get pipeline", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, pipeline)
}

// Create validates the request body and creates a new pipeline with stages, blocks, monitors, and alerts.
func (h *PipelineHandler) Create(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.CreatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	pipeline, err := h.svc.Create(c.Request.Context(), userID, &req)
	if err != nil {
		slog.Error("failed to create pipeline", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Created(c, pipeline)
}

// Update validates the request body and updates an existing pipeline.
func (h *PipelineHandler) Update(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	var req service.UpdatePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	pipeline, err := h.svc.Update(c.Request.Context(), userID, id, &req)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to update pipeline", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, pipeline)
}

// Delete removes a pipeline by ID.
func (h *PipelineHandler) Delete(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		slog.Error("failed to delete pipeline", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}

// Execute creates a pipeline execution job.
func (h *PipelineHandler) Execute(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	var req service.ExecutePipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	job, err := h.svc.Execute(c.Request.Context(), userID, id, req.Symbol)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "pipeline not found")
		} else {
			slog.Error("failed to execute pipeline", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Created(c, job)
}

// GetJob returns a pipeline job by ID.
func (h *PipelineHandler) GetJob(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobID := c.Param("jobId")

	job, err := h.svc.GetJob(c.Request.Context(), userID, jobID)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "job not found")
		} else {
			slog.Error("failed to get job", "error", err, "jobId", jobID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, job)
}

// Generate creates a pipeline structure from a natural language description.
func (h *PipelineHandler) Generate(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	pipeline, err := h.generator.Generate(c.Request.Context(), req.Description)
	if err != nil {
		slog.Error("failed to generate pipeline", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	Success(c, pipeline)
}

// isNotFound checks if an error indicates a "not found" condition.
func isNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || errors.Is(err, domain.ErrNotFound)
}
