package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/dev-superbear/nexus-backend/internal/repository"
)

// Task type constants for marketplace async workers.
const (
	// TypeMarketplaceStatsRefresh triggers a full stats recalculation for all active items.
	TypeMarketplaceStatsRefresh = "marketplace:stats:refresh"

	// TypeMarketplaceStatsRefreshItem triggers stats recalculation for a single item.
	TypeMarketplaceStatsRefreshItem = "marketplace:stats:refresh_item"
)

// MarketplaceStatsWorker handles asynchronous statistics recalculation.
// Counters (fork_count, like_count, usage_count, view_count) are normally
// incremented inline, but this worker re-derives them from the source tables
// to correct any drift caused by race conditions or failed transactions.
type MarketplaceStatsWorker struct {
	repo   *repository.MarketplaceRepo
	logger *slog.Logger
}

func NewMarketplaceStatsWorker(repo *repository.MarketplaceRepo, logger *slog.Logger) *MarketplaceStatsWorker {
	return &MarketplaceStatsWorker{repo: repo, logger: logger}
}

// ---------------------------------------------------------------------------
// Payload types
// ---------------------------------------------------------------------------

// StatsRefreshItemPayload is the payload for TypeMarketplaceStatsRefreshItem.
type StatsRefreshItemPayload struct {
	ItemID string `json:"itemId"`
}

// ---------------------------------------------------------------------------
// Task constructors (called by the publisher side)
// ---------------------------------------------------------------------------

// NewMarketplaceStatsRefreshTask creates a task to refresh all marketplace stats.
func NewMarketplaceStatsRefreshTask() (*asynq.Task, error) {
	return asynq.NewTask(TypeMarketplaceStatsRefresh, nil), nil
}

// NewMarketplaceStatsRefreshItemTask creates a task to refresh a single item's stats.
func NewMarketplaceStatsRefreshItemTask(itemID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(StatsRefreshItemPayload{ItemID: itemID.String()})
	if err != nil {
		return nil, fmt.Errorf("marshal stats refresh payload: %w", err)
	}
	return asynq.NewTask(TypeMarketplaceStatsRefreshItem, payload), nil
}

// ---------------------------------------------------------------------------
// Handlers (registered with the asynq server)
// ---------------------------------------------------------------------------

// HandleStatsRefresh processes TypeMarketplaceStatsRefresh — refreshes all active items.
func (w *MarketplaceStatsWorker) HandleStatsRefresh(ctx context.Context, t *asynq.Task) error {
	w.logger.InfoContext(ctx, "marketplace stats refresh started")

	ids, err := w.repo.ListActiveItemIDs(ctx)
	if err != nil {
		return fmt.Errorf("list active item IDs: %w", err)
	}

	var errs []error
	for _, id := range ids {
		if err := w.refreshSingleItem(ctx, id); err != nil {
			w.logger.ErrorContext(ctx, "failed to refresh item stats",
				"itemId", id, "error", err)
			errs = append(errs, err)
		}
	}

	w.logger.InfoContext(ctx, "marketplace stats refresh completed",
		"total", len(ids), "errors", len(errs))

	if len(errs) > 0 {
		return fmt.Errorf("refresh had %d errors (first: %w)", len(errs), errs[0])
	}
	return nil
}

// HandleStatsRefreshItem processes TypeMarketplaceStatsRefreshItem — refreshes a single item.
func (w *MarketplaceStatsWorker) HandleStatsRefreshItem(ctx context.Context, t *asynq.Task) error {
	var payload StatsRefreshItemPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	itemID, err := uuid.Parse(payload.ItemID)
	if err != nil {
		return fmt.Errorf("parse item ID: %w", err)
	}

	return w.refreshSingleItem(ctx, itemID)
}

// refreshSingleItem recalculates all counters for one item from the source tables.
func (w *MarketplaceStatsWorker) refreshSingleItem(ctx context.Context, id uuid.UUID) error {
	// Each refresh is independent and idempotent; errors on one counter
	// should not block the others.
	var firstErr error
	setErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	setErr(w.repo.RefreshForkCount(ctx, id))
	setErr(w.repo.RefreshLikeCount(ctx, id))
	setErr(w.repo.RefreshUsageCount(ctx, id))
	setErr(w.repo.RefreshViewCount(ctx, id))

	if firstErr != nil {
		return fmt.Errorf("refresh item %s: %w", id, firstErr)
	}

	w.logger.DebugContext(ctx, "item stats refreshed", "itemId", id)
	return nil
}

// ---------------------------------------------------------------------------
// Registration helper
// ---------------------------------------------------------------------------

// RegisterHandlers registers all marketplace worker handlers with an asynq mux.
func (w *MarketplaceStatsWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeMarketplaceStatsRefresh, w.HandleStatsRefresh)
	mux.HandleFunc(TypeMarketplaceStatsRefreshItem, w.HandleStatsRefreshItem)
}
