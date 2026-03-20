package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/llm"
	"github.com/dev-superbear/nexus-backend/internal/service"
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

	// SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	events, err := h.nlSvc.Stream(c.Request.Context(), req.Query)
	if err != nil {
		writeSSE(c, "error", gin.H{"message": err.Error()})
		return
	}

	var finalDSL string
	for event := range events {
		writeSSE(c, string(event.Type), event)
		if event.Type == llm.EventDSLReady {
			finalDSL = event.DSL
		}
	}

	if finalDSL != "" {
		results, err := h.searchSvc.Execute(c.Request.Context(), finalDSL)
		if err != nil {
			writeSSE(c, "error", gin.H{"message": "DSL execution failed: " + err.Error()})
			return
		}
		writeSSE(c, "done", gin.H{"results": results, "count": len(results)})
	}
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

	explanation, err := h.nlSvc.Explain(c.Request.Context(), req.DSL)
	if err != nil {
		slog.Error("explain failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"explanation": explanation})
}

func writeSSE(c *gin.Context, eventType string, data any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventType, b)
	c.Writer.Flush()
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
