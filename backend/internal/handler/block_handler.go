package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// BlockHandler is a thin controller for agent-block-related endpoints.
type BlockHandler struct {
	queries *sqlc.Queries
}

// NewBlockHandler creates a BlockHandler backed by the given queries.
func NewBlockHandler(queries *sqlc.Queries) *BlockHandler {
	return &BlockHandler{queries: queries}
}

// List returns a paginated list of agent blocks visible to the authenticated user.
func (h *BlockHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	p := GetPagination(c)

	blocks, err := h.queries.ListAgentBlocksByUser(c.Request.Context(), sqlc.ListAgentBlocksByUserParams{
		UserID: parseUUID(userID),
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	count, err := h.queries.CountAgentBlocksByUser(c.Request.Context(), parseUUID(userID))
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	Paginated(c, blocks, count, p.Page, p.PageSize)
}

// Get returns a single agent block by ID.
func (h *BlockHandler) Get(c *gin.Context) {
	id := c.Param("id")

	block, err := h.queries.GetAgentBlock(c.Request.Context(), parseUUID(id))
	if err != nil {
		Error(c, http.StatusNotFound, "block not found")
		return
	}

	Success(c, block)
}

// Create validates the request body and creates a new agent block (placeholder).
func (h *BlockHandler) Create(c *gin.Context) {
	var req CreateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	// TODO: wire up full CreateAgentBlock with JSON marshalling for allowed_tools/output_schema
	Created(c, gin.H{"message": "block created (placeholder)"})
}

// Delete removes an agent block by ID, scoped to the authenticated user.
func (h *BlockHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")

	err := h.queries.DeleteAgentBlock(c.Request.Context(), sqlc.DeleteAgentBlockParams{
		ID:     parseUUID(id),
		UserID: parseUUID(userID),
	})
	if err != nil {
		Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
