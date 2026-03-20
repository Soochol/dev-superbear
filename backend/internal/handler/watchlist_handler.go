package handler

import (
	"log/slog"
	"net/http"

	"github.com/dev-superbear/nexus-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

type WatchlistHandler struct {
	repo *repository.WatchlistRepo
}

func NewWatchlistHandler(repo *repository.WatchlistRepo) *WatchlistHandler {
	return &WatchlistHandler{repo: repo}
}

func (h *WatchlistHandler) GetWatchlist(c *gin.Context) {
	userID := int64(1) // TODO: extract from auth middleware

	items, err := h.repo.GetByUser(c.Request.Context(), userID)
	if err != nil {
		slog.Error("failed to fetch watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

type addWatchlistRequest struct {
	Symbol string `json:"symbol" binding:"required"`
	Name   string `json:"name" binding:"required"`
}

func (h *WatchlistHandler) AddToWatchlist(c *gin.Context) {
	userID := int64(1) // TODO: extract from auth middleware

	var req addWatchlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	item, err := h.repo.Add(c.Request.Context(), userID, req.Symbol, req.Name)
	if err != nil {
		slog.Error("failed to add to watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add to watchlist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *WatchlistHandler) RemoveFromWatchlist(c *gin.Context) {
	userID := int64(1) // TODO: extract from auth middleware
	symbol := c.Param("symbol")

	if err := h.repo.Remove(c.Request.Context(), userID, symbol); err != nil {
		slog.Error("failed to remove from watchlist", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove from watchlist"})
		return
	}

	c.Status(http.StatusNoContent)
}
