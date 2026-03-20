package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/dsl"
)

// SearchHandler is a thin controller for search/scan endpoints.
type SearchHandler struct{}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler() *SearchHandler {
	return &SearchHandler{}
}

// Scan validates the DSL query and returns matching results (placeholder).
func (h *SearchHandler) Scan(c *gin.Context) {
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	if err := dsl.ValidateDSL(req.Query); err != nil {
		Error(c, http.StatusBadRequest, "DSL syntax error: "+err.Error())
		return
	}

	Success(c, gin.H{
		"query":   req.Query,
		"results": []interface{}{},
		"message": "scan placeholder -- DSL validated successfully",
	})
}
