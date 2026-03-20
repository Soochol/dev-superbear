package handler

import (
	"net/http"

	"github.com/dev-superbear/nexus-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type FinancialsHandler struct {
	financialsSvc *service.FinancialsService
}

func NewFinancialsHandler(financialsSvc *service.FinancialsService) *FinancialsHandler {
	return &FinancialsHandler{financialsSvc: financialsSvc}
}

func (h *FinancialsHandler) GetFinancials(c *gin.Context) {
	symbol := c.Param("symbol")

	data, err := h.financialsSvc.GetFinancials(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *FinancialsHandler) GetSectorCompare(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": []any{}})
}
