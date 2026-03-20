package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/service"
)

const maxDSLLength = 10000

func validateInputLength(c *gin.Context, input, field string, max int) bool {
	if len(input) > max {
		c.JSON(http.StatusBadRequest, gin.H{"error": field + " exceeds maximum length"})
		return false
	}
	return true
}

type SearchHandler struct {
	searchSvc *service.SearchService
	nlSvc     *service.NLToDSLService
}

func NewSearchHandler(searchSvc *service.SearchService, nlSvc *service.NLToDSLService) *SearchHandler {
	return &SearchHandler{
		searchSvc: searchSvc,
		nlSvc:     nlSvc,
	}
}

// DSLRequest is the common request body for endpoints that accept a single DSL string.
type DSLRequest struct {
	DSL string `json:"dsl" binding:"required"`
}

func (h *SearchHandler) Execute(c *gin.Context) {
	var req DSLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateInputLength(c, req.DSL, "dsl", maxDSLLength) {
		return
	}

	results, err := h.searchSvc.Execute(c.Request.Context(), req.DSL)
	if err != nil {
		slog.Error("search execute failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *SearchHandler) Validate(c *gin.Context) {
	var req DSLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateInputLength(c, req.DSL, "dsl", maxDSLLength) {
		return
	}

	result := h.searchSvc.Validate(c.Request.Context(), req.DSL)
	c.JSON(http.StatusOK, result)
}

type NLToDSLRequest struct {
	Query string `json:"query" binding:"required"`
}

func (h *SearchHandler) NLToDSL(c *gin.Context) {
	var req NLToDSLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateInputLength(c, req.Query, "query", maxDSLLength) {
		return
	}

	dslResult, err := h.nlSvc.Convert(c.Request.Context(), req.Query)
	if err != nil {
		slog.Error("failed to convert NL to DSL", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	results, err := h.searchSvc.Execute(c.Request.Context(), dslResult.DSL)
	if err != nil {
		slog.Error("failed to execute search after NL conversion", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dsl":         dslResult.DSL,
		"explanation": dslResult.Explanation,
		"results":     results,
	})
}

func (h *SearchHandler) Explain(c *gin.Context) {
	var req DSLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateInputLength(c, req.DSL, "dsl", maxDSLLength) {
		return
	}

	explanation := "이 쿼리는 다음 조건으로 종목을 검색합니다: " + req.DSL
	c.JSON(http.StatusOK, gin.H{"explanation": explanation})
}

func (h *SearchHandler) RegisterRoutes(rg *gin.RouterGroup) {
	search := rg.Group("/search")
	{
		search.POST("/execute", h.Execute)
		search.POST("/validate", h.Validate)
		search.POST("/nl-to-dsl", h.NLToDSL)
		search.POST("/explain", h.Explain)
	}
}
