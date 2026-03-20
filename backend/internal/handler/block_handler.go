package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

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
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	p := GetPagination(c)

	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	blocks, err := h.queries.ListAgentBlocksByUser(c.Request.Context(), sqlc.ListAgentBlocksByUserParams{
		UserID: userUUID,
		Limit:  int32(p.PageSize),
		Offset: int32(p.Offset),
	})
	if err != nil {
		slog.Error("failed to list blocks", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	count, err := h.queries.CountAgentBlocksByUser(c.Request.Context(), userUUID)
	if err != nil {
		slog.Error("failed to count blocks", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	Paginated(c, blocks, count, p.Page, p.PageSize)
}

// Get returns a single agent block by ID.
func (h *BlockHandler) Get(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	idUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	block, err := h.queries.GetAgentBlock(c.Request.Context(), idUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			Error(c, http.StatusNotFound, "not found")
		} else {
			slog.Error("failed to get block", "error", err, "userId", userID)
			Error(c, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// Ownership check: block is accessible if public or owned by the user
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if !block.IsPublic && block.UserID != userUUID {
		Error(c, http.StatusNotFound, "not found")
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
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := c.Param("id")

	idUUID, err := parseUUID(id)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	err = h.queries.DeleteAgentBlock(c.Request.Context(), sqlc.DeleteAgentBlockParams{
		ID:     idUUID,
		UserID: userUUID,
	})
	if err != nil {
		slog.Error("failed to delete block", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	c.Status(http.StatusNoContent)
}
