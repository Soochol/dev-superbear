// Package handler provides Gin HTTP handlers for search preset CRUD.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"backend/internal/repository"
)

// PresetHandler groups all search-preset-related HTTP endpoints.
type PresetHandler struct {
	repo *repository.PresetRepository
}

// NewPresetHandler creates a new handler backed by the given repository.
func NewPresetHandler(repo *repository.PresetRepository) *PresetHandler {
	return &PresetHandler{repo: repo}
}

// RegisterRoutes mounts preset routes onto a Gin router group.
//
//	GET    /search/presets     → ListPresets
//	POST   /search/presets     → CreatePreset
//	DELETE /search/presets/:id → DeletePreset
func (h *PresetHandler) RegisterRoutes(rg *gin.RouterGroup) {
	presets := rg.Group("/search/presets")
	{
		presets.GET("", h.ListPresets)
		presets.POST("", h.CreatePreset)
		presets.DELETE("/:id", h.DeletePreset)
	}
}

// ListPresets returns a paginated list of presets visible to the current user.
func (h *PresetHandler) ListPresets(c *gin.Context) {
	userID := c.GetString("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	result, err := h.repo.FindMany(c.Request.Context(), userID, int32(pageSize), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (int(result.Total) + pageSize - 1) / pageSize
	c.JSON(http.StatusOK, gin.H{
		"data": result.Presets,
		"pagination": gin.H{
			"total":      result.Total,
			"page":       page,
			"pageSize":   pageSize,
			"totalPages": totalPages,
		},
	})
}

// CreatePresetRequest is the JSON body for creating a preset.
type CreatePresetRequest struct {
	Name     string  `json:"name" binding:"required"`
	DSL      string  `json:"dsl" binding:"required"`
	NLQuery  *string `json:"nlQuery,omitempty"`
	IsPublic bool    `json:"isPublic"`
}

// CreatePreset creates a new search preset for the current user.
func (h *PresetHandler) CreatePreset(c *gin.Context) {
	userID := c.GetString("userID")

	var req CreatePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preset, err := h.repo.Create(c.Request.Context(), repository.CreatePresetParams{
		UserID:   userID,
		Name:     req.Name,
		DSL:      req.DSL,
		NLQuery:  req.NLQuery,
		IsPublic: req.IsPublic,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": preset})
}

// DeletePreset removes a preset owned by the current user.
func (h *PresetHandler) DeletePreset(c *gin.Context) {
	userID := c.GetString("userID")
	presetID := c.Param("id")

	if err := h.repo.Delete(c.Request.Context(), presetID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
