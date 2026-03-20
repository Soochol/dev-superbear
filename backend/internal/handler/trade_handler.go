package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dev-superbear/nexus-backend/internal/domain/trade"
	"github.com/dev-superbear/nexus-backend/internal/middleware"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// TradeHandler is a thin controller for trade-related endpoints.
type TradeHandler struct {
	queries *sqlc.Queries
}

// NewTradeHandler creates a TradeHandler backed by the given queries.
func NewTradeHandler(queries *sqlc.Queries) *TradeHandler {
	return &TradeHandler{queries: queries}
}

// CreateTrade records a new trade for a case and creates a corresponding timeline event.
func (h *TradeHandler) CreateTrade(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	caseID := c.Param("id")
	caseUUID, err := parseUUID(caseID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	var req CreateTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "Validation error: "+err.Error())
		return
	}

	t, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		Error(c, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
		return
	}
	tradeDate := pgtype.Date{Time: t, Valid: true}

	noteText := pgtype.Text{}
	if req.Note != nil {
		noteText = pgtype.Text{String: *req.Note, Valid: true}
	}

	ctx := c.Request.Context()

	created, err := h.queries.CreateTrade(ctx, sqlc.CreateTradeParams{
		CaseID:   caseUUID,
		UserID:   userUUID,
		Type:     sqlc.TradeType(req.Type),
		Price:    req.Price,
		Quantity: int32(req.Quantity),
		Fee:      req.Fee,
		Date:     tradeDate,
		Note:     noteText,
	})
	if err != nil {
		slog.Error("failed to create trade", "error", err, "userId", userID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	noteStr := ""
	if req.Note != nil {
		noteStr = *req.Note
	}

	_, timelineErr := h.queries.CreateTimelineEvent(ctx, sqlc.CreateTimelineEventParams{
		CaseID:     caseUUID,
		Date:       tradeDate,
		DayOffset:  0,
		Type:       sqlc.TimelineEventTypeTRADE,
		Title:      fmt.Sprintf("%s %d shares @ %.0f", req.Type, req.Quantity, req.Price),
		Content:    noteStr,
		AiAnalysis: pgtype.Text{},
		Data:       nil,
	})
	if timelineErr != nil {
		slog.Error("failed to create timeline event for trade", "error", timelineErr, "tradeId", created.ID)
	}

	Created(c, created)
}

// ListTrades returns all trades for a case with an optional PnL summary.
func (h *TradeHandler) ListTrades(c *gin.Context) {
	caseID := c.Param("id")
	caseUUID, err := parseUUID(caseID)
	if err != nil {
		Error(c, http.StatusBadRequest, err.Error())
		return
	}

	currentPrice := 0.0
	if cp := c.Query("currentPrice"); cp != "" {
		if v, err := strconv.ParseFloat(cp, 64); err == nil {
			currentPrice = v
		}
	}

	trades, err := h.queries.ListTradesByCase(c.Request.Context(), caseUUID)
	if err != nil {
		slog.Error("failed to list trades", "error", err, "caseId", caseID)
		Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	inputs := make([]trade.TradeInput, 0, len(trades))
	for _, tr := range trades {
		tradedAt := time.Time{}
		if tr.Date.Valid {
			tradedAt = tr.Date.Time
		}
		inputs = append(inputs, trade.TradeInput{
			Type:     string(tr.Type),
			Price:    tr.Price,
			Quantity: int(tr.Quantity),
			Fee:      tr.Fee,
			TradedAt: tradedAt,
		})
	}

	summary := trade.CalculatePnL(inputs, currentPrice)

	Success(c, gin.H{
		"trades":  trades,
		"summary": summary,
	})
}
