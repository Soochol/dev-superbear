package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	mkt "backend/internal/domain/marketplace"
	"backend/internal/service"
)

// MarketplaceHandler exposes REST endpoints for the marketplace feature.
//
// Routes (registered via RegisterRoutes):
//
//	GET    /api/marketplace             — list / search
//	GET    /api/marketplace/:id         — detail
//	POST   /api/marketplace/publish     — publish a resource
//	PUT    /api/marketplace/:id         — update title/desc/tags
//	DELETE /api/marketplace/:id         — soft-delete
//	POST   /api/marketplace/:id/fork    — deep-copy fork
//	POST   /api/marketplace/:id/like    — toggle like
//	POST   /api/marketplace/:id/verify  — request verified badge
type MarketplaceHandler struct {
	marketplaceSvc *service.MarketplaceService
	forkSvc        *service.ForkService
	logger         *slog.Logger
}

func NewMarketplaceHandler(
	marketplaceSvc *service.MarketplaceService,
	forkSvc *service.ForkService,
	logger *slog.Logger,
) *MarketplaceHandler {
	return &MarketplaceHandler{
		marketplaceSvc: marketplaceSvc,
		forkSvc:        forkSvc,
		logger:         logger,
	}
}

// RegisterRoutes mounts all marketplace routes under the given router group.
func (h *MarketplaceHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/marketplace")
	{
		g.GET("", h.List)
		g.GET("/:id", h.GetDetail)
		g.POST("/publish", h.Publish)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/fork", h.Fork)
		g.POST("/:id/like", h.ToggleLike)
		g.POST("/:id/verify", h.Verify)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// currentUserID extracts the authenticated user ID from the Gin context.
// The auth middleware is expected to set "userId" as a string.
func currentUserID(c *gin.Context) (*uuid.UUID, error) {
	raw, exists := c.Get("userId")
	if !exists {
		return nil, nil
	}
	s, ok := raw.(string)
	if !ok {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// requireUserID is like currentUserID but returns 401 if missing.
func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	uid, err := currentUserID(c)
	if err != nil || uid == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "인증이 필요합니다."})
		return uuid.Nil, false
	}
	return *uid, true
}

// parseUUIDParam extracts and parses a UUID path parameter.
func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "유효하지 않은 ID입니다."})
		return uuid.Nil, false
	}
	return id, true
}

// ---------------------------------------------------------------------------
// GET /api/marketplace — List / Search
// ---------------------------------------------------------------------------

// List handles paginated listing with type/tag/verified filters and full-text search.
//
//	Query params: type, sort, search, tags (comma-sep), verifiedOnly, page, limit
func (h *MarketplaceHandler) List(c *gin.Context) {
	var query mkt.ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uid, _ := currentUserID(c)

	resp, err := h.marketplaceSvc.ListItems(c.Request.Context(), query, uid)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "list marketplace items failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "목록 조회에 실패했습니다."})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// GET /api/marketplace/:id — Detail
// ---------------------------------------------------------------------------

func (h *MarketplaceHandler) GetDetail(c *gin.Context) {
	itemID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	uid, _ := currentUserID(c)

	detail, err := h.marketplaceSvc.GetDetail(c.Request.Context(), itemID, uid)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "get marketplace detail failed",
			"itemId", itemID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "아이템을 찾을 수 없습니다."})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// ---------------------------------------------------------------------------
// POST /api/marketplace/publish — Publish
// ---------------------------------------------------------------------------

func (h *MarketplaceHandler) Publish(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req mkt.PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	itemID, err := h.marketplaceSvc.Publish(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "publish failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "게시에 실패했습니다."})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"itemId": itemID})
}

// ---------------------------------------------------------------------------
// PUT /api/marketplace/:id — Update
// ---------------------------------------------------------------------------

type updateRequest struct {
	Title       string   `json:"title" binding:"required,min=1,max=200"`
	Description string   `json:"description" binding:"required,min=1"`
	Tags        []string `json:"tags"`
}

func (h *MarketplaceHandler) Update(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.marketplaceSvc.UpdateItem(c.Request.Context(), itemID, userID, req.Title, req.Description, req.Tags); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "update failed", "itemId", itemID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "수정 권한이 없거나 아이템을 찾을 수 없습니다."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "수정 완료"})
}

// ---------------------------------------------------------------------------
// DELETE /api/marketplace/:id — Soft Delete
// ---------------------------------------------------------------------------

func (h *MarketplaceHandler) Delete(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.marketplaceSvc.DeleteItem(c.Request.Context(), itemID, userID); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "delete failed", "itemId", itemID, "error", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "삭제 권한이 없거나 아이템을 찾을 수 없습니다."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "삭제 완료"})
}

// ---------------------------------------------------------------------------
// POST /api/marketplace/:id/fork — Fork (Deep Copy)
// ---------------------------------------------------------------------------

func (h *MarketplaceHandler) Fork(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.forkSvc.ForkItem(c.Request.Context(), userID, itemID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "fork failed", "itemId", itemID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Fork에 실패했습니다."})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ---------------------------------------------------------------------------
// POST /api/marketplace/:id/like — Toggle Like
// ---------------------------------------------------------------------------

func (h *MarketplaceHandler) ToggleLike(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.marketplaceSvc.ToggleLike(c.Request.Context(), userID, itemID)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "toggle like failed", "itemId", itemID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "좋아요 처리에 실패했습니다."})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// POST /api/marketplace/:id/verify — Request Verified Badge
// ---------------------------------------------------------------------------

func (h *MarketplaceHandler) Verify(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req mkt.VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backtestJobID, err := uuid.Parse(req.BacktestJobID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "유효하지 않은 백테스트 작업 ID입니다."})
		return
	}

	// In production, fetch the BacktestJobInfo from the backtest service/repo.
	// This is a placeholder showing the expected integration point.
	job := service.BacktestJobInfo{
		ID:     backtestJobID,
		Status: "COMPLETED", // would come from actual DB lookup
		Stats:  nil,          // would come from actual DB lookup
	}

	// TODO: Integrate with actual backtest service:
	//   job, err := h.backtestSvc.GetJobInfo(ctx, backtestJobID)
	_ = job

	result, err := h.marketplaceSvc.VerifyItem(c.Request.Context(), itemID, userID, job)
	if err != nil {
		h.logger.ErrorContext(c.Request.Context(), "verify failed", "itemId", itemID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "검증 처리에 실패했습니다."})
		return
	}

	if result.Verified {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusUnprocessableEntity, result)
	}
}
