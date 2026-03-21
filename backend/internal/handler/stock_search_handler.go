package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

type StockSearchHandler struct {
	repo *repository.StockRepository
}

func NewStockSearchHandler(repo *repository.StockRepository) *StockSearchHandler {
	return &StockSearchHandler{repo: repo}
}

func (h *StockSearchHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusOK, gin.H{"data": []any{}})
		return
	}

	results, err := h.repo.Search(c.Request.Context(), query, 20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
