package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CandleHandler struct {
	candleSvc CandleFetcher
}

func NewCandleHandler(candleSvc CandleFetcher) *CandleHandler {
	return &CandleHandler{candleSvc: candleSvc}
}

func (h *CandleHandler) GetCandles(c *gin.Context) {
	symbol := c.Param("symbol")
	period := c.DefaultQuery("period", "D")
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	candles, err := h.candleSvc.GetCandles(c.Request.Context(), symbol, startDate, endDate, period)
	if err != nil {
		slog.Error("failed to fetch candles", "error", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"candles": candles,
			"symbol":  symbol,
		},
	})
}
