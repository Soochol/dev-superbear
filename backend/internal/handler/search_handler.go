package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"backend/internal/service"
)

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

type ExecuteSearchRequest struct {
	DSL string `json:"dsl" binding:"required"`
}

func (h *SearchHandler) Execute(c *gin.Context) {
	var req ExecuteSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.DSL) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL input exceeds maximum length of 10000 characters"})
		return
	}

	validation := h.searchSvc.Validate(c.Request.Context(), req.DSL)
	if !validation.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL: " + validation.Error})
		return
	}

	results, err := h.searchSvc.Execute(c.Request.Context(), req.DSL)
	if err != nil {
		slog.Error("failed to execute search", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

type ValidateRequest struct {
	DSL string `json:"dsl" binding:"required"`
}

func (h *SearchHandler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.DSL) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL input exceeds maximum length of 10000 characters"})
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

	if len(req.Query) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query exceeds maximum length of 10000 characters"})
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

type ExplainRequest struct {
	DSL string `json:"dsl" binding:"required"`
}

func (h *SearchHandler) Explain(c *gin.Context) {
	var req ExplainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.DSL) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL input exceeds maximum length of 10000 characters"})
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
