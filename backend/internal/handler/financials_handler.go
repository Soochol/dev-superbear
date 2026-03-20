package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FinancialsHandler struct {
	financialsSvc FinancialsFetcher
}

func NewFinancialsHandler(financialsSvc FinancialsFetcher) *FinancialsHandler {
	return &FinancialsHandler{financialsSvc: financialsSvc}
}

func (h *FinancialsHandler) GetFinancials(c *gin.Context) {
	symbol := c.Param("symbol")

	data, err := h.financialsSvc.GetFinancials(c.Request.Context(), symbol)
	if err != nil {
		slog.Error("failed to fetch financials", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *FinancialsHandler) GetSectorCompare(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}})
}
