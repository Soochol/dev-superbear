package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/service"
)

// BlockHandler is a thin controller for agent-block-related endpoints.
type BlockHandler struct {
	svc *service.BlockService
}

// NewBlockHandler creates a BlockHandler backed by the given service.
func NewBlockHandler(svc *service.BlockService) *BlockHandler {
	return &BlockHandler{svc: svc}
}

// RegisterRoutes mounts block routes on the given router group.
func (h *BlockHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/blocks", h.List)
	rg.GET("/blocks/templates", h.ListTemplates)
	rg.POST("/blocks", h.Create)
	rg.POST("/blocks/copy-template", h.CopyFromTemplate)
	rg.GET("/blocks/:id", h.Get)
	rg.PUT("/blocks/:id", h.Update)
	rg.DELETE("/blocks/:id", h.Delete)
}

// List returns standalone agent blocks for the authenticated user.
func (h *BlockHandler) List(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	blocks, err := h.svc.ListBlocks(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to list blocks", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Success(c, blocks)
}

// ListTemplates returns template blocks visible to the authenticated user.
func (h *BlockHandler) ListTemplates(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	templates, err := h.svc.ListTemplates(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to list templates", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Success(c, templates)
}

// Get returns a single agent block by ID.
func (h *BlockHandler) Get(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	block, err := h.svc.GetBlock(c.Request.Context(), userID, id)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get block", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, block)
}

// Create validates the request body and creates a new agent block.
func (h *BlockHandler) Create(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.CreateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	block, err := h.svc.CreateBlock(c.Request.Context(), userID, &req)
	if err != nil {
		slog.Error("failed to create block", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Created(c, block)
}

// CopyFromTemplate copies a template block into a user's stage.
func (h *BlockHandler) CopyFromTemplate(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.CopyFromTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	block, err := h.svc.CopyFromTemplate(c.Request.Context(), userID, req.TemplateID, req.StageID)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "template not found")
		} else {
			slog.Error("failed to copy from template", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Created(c, block)
}

// Update validates the request body and updates an existing agent block.
func (h *BlockHandler) Update(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	var req service.UpdateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	block, err := h.svc.UpdateBlock(c.Request.Context(), userID, id, &req)
	if err != nil {
		if isNotFound(err) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to update block", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	Success(c, block)
}

// Delete removes an agent block by ID, scoped to the authenticated user.
func (h *BlockHandler) Delete(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	if err := h.svc.DeleteBlock(c.Request.Context(), userID, id); err != nil {
		slog.Error("failed to delete block", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}
