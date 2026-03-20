// Package handler provides Gin HTTP handlers for the portfolio API.
// All handlers are thin controllers that delegate to the portfolio service.
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	domain "backend/internal/domain/portfolio"
	"backend/internal/service"
)

// ---------------------------------------------------------------------------
// PortfolioHandler
// ---------------------------------------------------------------------------

// PortfolioHandler groups all portfolio-related HTTP endpoints.
type PortfolioHandler struct {
	svc *service.PortfolioService
}

// NewPortfolioHandler creates a new handler backed by the given service.
func NewPortfolioHandler(svc *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{svc: svc}
}

// RegisterRoutes mounts portfolio routes onto a Gin router group.
//
//	GET  /portfolio          → GetPortfolio
//	GET  /portfolio/summary  → GetPortfolioSummary (alias)
//	GET  /portfolio/sectors  → GetSectorWeights
//	GET  /portfolio/history  → GetPnLHistory
//	GET  /portfolio/tax      → SimulateTax
//	POST /portfolio/rebalance→ SimulateRebalance
func (h *PortfolioHandler) RegisterRoutes(rg *gin.RouterGroup) {
	pg := rg.Group("/portfolio")
	{
		pg.GET("", h.GetPortfolio)
		pg.GET("/summary", h.GetPortfolio) // alias
		pg.GET("/sectors", h.GetSectorWeights)
		pg.GET("/history", h.GetPnLHistory)
		pg.GET("/tax", h.SimulateTax)
		pg.POST("/rebalance", h.SimulateRebalance)
	}
}

// ---------------------------------------------------------------------------
// GET /portfolio — 포트폴리오 현황
// ---------------------------------------------------------------------------

// GetPortfolio returns the full portfolio summary with live prices.
func (h *PortfolioHandler) GetPortfolio(c *gin.Context) {
	userID := c.GetString("userID") // set by auth middleware
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.svc.GetPortfolioSummary(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// ---------------------------------------------------------------------------
// GET /portfolio/sectors — 섹터 비중
// ---------------------------------------------------------------------------

// GetSectorWeights returns sector-grouped portfolio data for donut chart.
func (h *PortfolioHandler) GetSectorWeights(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	sectors, totalValue, err := h.svc.GetSectorWeights(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sectors":    sectors,
		"totalValue": totalValue,
	})
}

// ---------------------------------------------------------------------------
// GET /portfolio/history — 손익 히스토리
// ---------------------------------------------------------------------------

// GetPnLHistory returns realized PnL history within a date range.
//
// Query params:
//
//	from=2025-01-01  (default: 1 year ago)
//	to=2025-12-31    (default: today)
func (h *PortfolioHandler) GetPnLHistory(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	now := time.Now()
	from := now.AddDate(-1, 0, 0)
	to := now

	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			to = t
		}
	}

	entries, err := h.svc.GetPnLHistory(c.Request.Context(), userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": entries,
	})
}

// ---------------------------------------------------------------------------
// GET /portfolio/tax — 세금 시뮬레이션
// ---------------------------------------------------------------------------

// SimulateTax returns the annual tax breakdown (KR + US).
func (h *PortfolioHandler) SimulateTax(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	tax, err := h.svc.SimulateTax(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tax)
}

// ---------------------------------------------------------------------------
// POST /portfolio/rebalance — 리밸런싱 시뮬레이션
// ---------------------------------------------------------------------------

type rebalanceRequest struct {
	Targets []domain.TargetAllocation `json:"targets" binding:"required"`
}

// SimulateRebalance computes rebalancing actions for the given targets.
func (h *PortfolioHandler) SimulateRebalance(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req rebalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	result, err := h.svc.SimulateRebalance(c.Request.Context(), userID, req.Targets)
	if err != nil {
		if err == domain.ErrTargetWeightExceeds100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
