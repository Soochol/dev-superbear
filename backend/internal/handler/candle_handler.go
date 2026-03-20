package handler

import (
	"net/http"

	"github.com/dev-superbear/nexus-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type CandleHandler struct {
	candleSvc *service.CandleService
}

func NewCandleHandler(candleSvc *service.CandleService) *CandleHandler {
	return &CandleHandler{candleSvc: candleSvc}
}

func (h *CandleHandler) GetCandles(c *gin.Context) {
	symbol := c.Param("symbol")
	period := c.DefaultQuery("period", "D")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	candles, err := h.candleSvc.GetCandles(c.Request.Context(), symbol, startDate, endDate, period)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"candles": candles,
			"symbol":  symbol,
		},
	})
}
