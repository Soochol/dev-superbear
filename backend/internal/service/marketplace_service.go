package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	mkt "backend/internal/domain/marketplace"
	"backend/internal/repository"
)

// MarketplaceService implements marketplace listing, detail, publish, like,
// usage-tracking, and verification (backtest-based verified badge).
type MarketplaceService struct {
	repo   *repository.MarketplaceRepo
	logger *slog.Logger
}

func NewMarketplaceService(repo *repository.MarketplaceRepo, logger *slog.Logger) *MarketplaceService {
	return &MarketplaceService{repo: repo, logger: logger}
}

// ---------------------------------------------------------------------------
// List & Search
// ---------------------------------------------------------------------------

// ListItems returns a paginated, filtered, sorted list of marketplace items.
func (s *MarketplaceService) ListItems(ctx context.Context, query mkt.ListQuery, currentUserID *uuid.UUID) (*mkt.ListResponse, error) {
	query.Defaults()

	rows, total, err := s.repo.ListItems(ctx, query)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list marketplace items", "error", err)
		return nil, fmt.Errorf("list marketplace items: %w", err)
	}

	items := make([]mkt.ItemDetail, 0, len(rows))
	for _, row := range rows {
		detail := rowToDetail(row)

		// Check if current user liked this item
		if currentUserID != nil {
			liked, err := s.repo.IsLiked(ctx, *currentUserID, row.ID)
			if err != nil {
				s.logger.WarnContext(ctx, "failed to check like status", "itemId", row.ID, "error", err)
			}
			detail.IsLikedByMe = liked
		}

		items = append(items, detail)
	}

	return &mkt.ListResponse{
		Items: items,
		Total: total,
		Page:  query.Page,
	}, nil
}

// GetDetail returns a single marketplace item with view-count tracking.
func (s *MarketplaceService) GetDetail(ctx context.Context, itemID uuid.UUID, currentUserID *uuid.UUID) (*mkt.ItemDetail, error) {
	// Deduplicated view count
	if currentUserID != nil {
		s.trackDeduplicatedView(ctx, *currentUserID, itemID)
	}

	row, err := s.repo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get marketplace item: %w", err)
	}

	detail := rowToDetail(*row)

	if currentUserID != nil {
		liked, err := s.repo.IsLiked(ctx, *currentUserID, itemID)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to check like status", "itemId", itemID, "error", err)
		}
		detail.IsLikedByMe = liked
	}

	return &detail, nil
}

// trackDeduplicatedView increments view_count only if the same user has not
// viewed this item in the last 60 minutes.
func (s *MarketplaceService) trackDeduplicatedView(ctx context.Context, userID, itemID uuid.UUID) {
	since := time.Now().Add(-60 * time.Minute)

	hasRecent, err := s.repo.HasRecentView(ctx, userID, itemID, since)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to check recent view", "error", err)
		return
	}
	if hasRecent {
		return
	}

	if err := s.repo.IncrementViewCount(ctx, itemID); err != nil {
		s.logger.WarnContext(ctx, "failed to increment view count", "error", err)
	}
	if err := s.repo.CreateUsageLog(ctx, userID, itemID, mkt.ActionView); err != nil {
		s.logger.WarnContext(ctx, "failed to create view log", "error", err)
	}
}

// ---------------------------------------------------------------------------
// Publish
// ---------------------------------------------------------------------------

// Publish registers an existing resource (pipeline, agent block, etc.) as a marketplace item.
func (s *MarketplaceService) Publish(ctx context.Context, userID uuid.UUID, req mkt.PublishRequest) (uuid.UUID, error) {
	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid resource ID: %w", err)
	}

	// TODO: verify ownership of the underlying resource via a separate resource repo
	// For now, ownership is enforced at the handler layer by checking session user.

	item := &mkt.MarketplaceItem{
		UserID:      userID,
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Tags:        req.Tags,
	}

	// Set the correct foreign key based on type
	switch req.Type {
	case mkt.ItemTypePipeline:
		item.PipelineID = &resourceID
	case mkt.ItemTypeAgentBlock:
		item.AgentBlockID = &resourceID
	case mkt.ItemTypeSearchPreset:
		item.SearchPresetID = &resourceID
	case mkt.ItemTypeJudgmentScript:
		item.JudgmentScriptID = &resourceID
	default:
		return uuid.Nil, fmt.Errorf("unknown item type: %s", req.Type)
	}

	created, err := s.repo.CreateItem(ctx, item)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create marketplace item", "error", err)
		return uuid.Nil, fmt.Errorf("publish item: %w", err)
	}

	s.logger.InfoContext(ctx, "marketplace item published",
		"itemId", created.ID, "type", req.Type, "userId", userID)

	return created.ID, nil
}

// ---------------------------------------------------------------------------
// Like Toggle
// ---------------------------------------------------------------------------

// ToggleLike adds or removes a like and returns the new state.
func (s *MarketplaceService) ToggleLike(ctx context.Context, userID, itemID uuid.UUID) (*mkt.LikeToggleResult, error) {
	liked, err := s.repo.IsLiked(ctx, userID, itemID)
	if err != nil {
		return nil, fmt.Errorf("check like: %w", err)
	}

	if liked {
		// Unlike
		if err := s.repo.DeleteLike(ctx, userID, itemID); err != nil {
			return nil, fmt.Errorf("delete like: %w", err)
		}
		if err := s.repo.DecrementLikeCount(ctx, itemID); err != nil {
			return nil, fmt.Errorf("decrement like count: %w", err)
		}
	} else {
		// Like
		if _, err := s.repo.CreateLike(ctx, userID, itemID); err != nil {
			return nil, fmt.Errorf("create like: %w", err)
		}
		if err := s.repo.IncrementLikeCount(ctx, itemID); err != nil {
			return nil, fmt.Errorf("increment like count: %w", err)
		}
		// Log usage
		if err := s.repo.CreateUsageLog(ctx, userID, itemID, mkt.ActionLike); err != nil {
			s.logger.WarnContext(ctx, "failed to log like action", "error", err)
		}
	}

	// Fetch fresh count
	count, err := s.repo.GetItemLikeCount(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get like count: %w", err)
	}

	return &mkt.LikeToggleResult{
		Liked:     !liked,
		LikeCount: count,
	}, nil
}

// ---------------------------------------------------------------------------
// Usage Tracking
// ---------------------------------------------------------------------------

// TrackUsage records an EXECUTE action against a marketplace item looked up by its resource.
func (s *MarketplaceService) TrackUsage(ctx context.Context, resourceID uuid.UUID, resourceType mkt.ItemType, userID uuid.UUID) error {
	itemID, err := s.repo.FindItemIDByResourceID(ctx, resourceID, resourceType)
	if err != nil {
		return fmt.Errorf("find item by resource: %w", err)
	}
	if itemID == nil {
		// Resource is not published on the marketplace; nothing to track.
		return nil
	}

	if err := s.repo.IncrementUsageCount(ctx, *itemID); err != nil {
		return fmt.Errorf("increment usage count: %w", err)
	}
	if err := s.repo.CreateUsageLog(ctx, userID, *itemID, mkt.ActionExecute); err != nil {
		return fmt.Errorf("create usage log: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Verification (Backtest → Verified Badge)
// ---------------------------------------------------------------------------

// BacktestJobInfo is a minimal representation needed for verification.
// In production this would come from a backtest repository / service.
type BacktestJobInfo struct {
	ID         uuid.UUID
	Status     string
	PipelineID *uuid.UUID
	Stats      *BacktestStatsInfo
}

type BacktestStatsInfo struct {
	WinRate     float64
	AvgReturn   float64
	TotalEvents int
}

// VerifyItem attaches a completed backtest to the marketplace item and grants the verified badge
// if the performance meets the criteria.
func (s *MarketplaceService) VerifyItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID, job BacktestJobInfo) (*mkt.VerificationResult, error) {
	// 1. Fetch item
	row, err := s.repo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get item for verification: %w", err)
	}

	// 2. Ownership check
	if row.UserID != userID {
		return &mkt.VerificationResult{
			Verified: false,
			Reason:   "본인의 아이템만 검증할 수 있습니다.",
		}, nil
	}

	// 3. Backtest status
	if job.Status != "COMPLETED" {
		return &mkt.VerificationResult{
			Verified: false,
			Reason:   "완료된 백테스트만 사용할 수 있습니다.",
		}, nil
	}

	// 4. Pipeline match
	if row.Type == mkt.ItemTypePipeline && row.PipelineID != nil && job.PipelineID != nil {
		if *row.PipelineID != *job.PipelineID {
			return &mkt.VerificationResult{
				Verified: false,
				Reason:   "백테스트의 파이프라인이 아이템과 일치하지 않습니다.",
			}, nil
		}
	}

	// 5. Stats present?
	if job.Stats == nil {
		return &mkt.VerificationResult{
			Verified: false,
			Reason:   "백테스트 통계가 없습니다.",
		}, nil
	}

	criteria := mkt.DefaultVerificationCriteria()

	// 6. Check criteria
	if job.Stats.TotalEvents < criteria.MinTotalEvents {
		return &mkt.VerificationResult{
			Verified: false,
			Reason:   fmt.Sprintf("최소 %d건 이상의 이벤트가 필요합니다. (현재: %d건)", criteria.MinTotalEvents, job.Stats.TotalEvents),
		}, nil
	}

	if job.Stats.WinRate < criteria.MinWinRate {
		return &mkt.VerificationResult{
			Verified: false,
			Reason:   fmt.Sprintf("승률이 %.0f%% 이상이어야 합니다. (현재: %.1f%%)", criteria.MinWinRate, job.Stats.WinRate),
		}, nil
	}

	if job.Stats.AvgReturn < criteria.MinAvgReturn {
		return &mkt.VerificationResult{
			Verified: false,
			Reason:   fmt.Sprintf("평균 수익률이 %.0f%% 이상이어야 합니다. (현재: %.1f%%)", criteria.MinAvgReturn, job.Stats.AvgReturn),
		}, nil
	}

	// 7. Grant badge
	if err := s.repo.SetVerification(ctx, itemID, job.ID, job.Stats.WinRate, job.Stats.AvgReturn, job.Stats.TotalEvents); err != nil {
		return nil, fmt.Errorf("set verification: %w", err)
	}

	s.logger.InfoContext(ctx, "marketplace item verified",
		"itemId", itemID, "winRate", job.Stats.WinRate, "avgReturn", job.Stats.AvgReturn)

	return &mkt.VerificationResult{
		Verified: true,
		Reason:   "검증 완료",
		Stats: &mkt.BacktestStats{
			WinRate:     job.Stats.WinRate,
			AvgReturn:   job.Stats.AvgReturn,
			TotalEvents: job.Stats.TotalEvents,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Update / Delete
// ---------------------------------------------------------------------------

func (s *MarketplaceService) UpdateItem(ctx context.Context, itemID, userID uuid.UUID, title, description string, tags []string) error {
	return s.repo.UpdateItem(ctx, itemID, userID, title, description, tags)
}

func (s *MarketplaceService) DeleteItem(ctx context.Context, itemID, userID uuid.UUID) error {
	return s.repo.SoftDeleteItem(ctx, itemID, userID)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func rowToDetail(row repository.ItemRow) mkt.ItemDetail {
	detail := mkt.ItemDetail{
		ID:          row.ID,
		Type:        row.Type,
		Title:       row.Title,
		Description: row.Description,
		Tags:        row.Tags,
		Author: mkt.Author{
			ID:     row.AuthorID,
			Name:   row.AuthorName,
			Avatar: row.AuthorImage,
		},
		Stats: mkt.ItemStats{
			UsageCount: row.UsageCount,
			ForkCount:  row.ForkCount,
			ViewCount:  row.ViewCount,
			LikeCount:  row.LikeCount,
		},
		Verified:    row.Verified,
		PublishedAt: row.PublishedAt,
	}

	if detail.Tags == nil {
		detail.Tags = []string{}
	}

	// Backtest stats
	if row.BacktestWinRate != nil {
		avgReturn := 0.0
		if row.BacktestAvgReturn != nil {
			avgReturn = *row.BacktestAvgReturn
		}
		totalEvents := 0
		if row.BacktestTotalEvents != nil {
			totalEvents = *row.BacktestTotalEvents
		}
		detail.BacktestStats = &mkt.BacktestStats{
			WinRate:     *row.BacktestWinRate,
			AvgReturn:   avgReturn,
			TotalEvents: totalEvents,
		}
	}

	// Fork origin
	if row.ForkOriginID != nil {
		originTitle := ""
		if row.ForkOriginTitle != nil {
			originTitle = *row.ForkOriginTitle
		}
		originAuthor := ""
		if row.ForkOriginAuthorName != nil {
			originAuthor = *row.ForkOriginAuthorName
		}
		detail.ForkedFrom = &mkt.ForkOrigin{
			ID:         *row.ForkOriginID,
			Title:      originTitle,
			AuthorName: originAuthor,
		}
	}

	return detail
}
