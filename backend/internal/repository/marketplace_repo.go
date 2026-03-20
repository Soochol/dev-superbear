package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	mkt "backend/internal/domain/marketplace"
)

// MarketplaceRepo wraps direct SQL access for marketplace tables.
// In production this will be replaced by sqlc-generated code; the interface
// is kept identical so the service layer needs no changes.
type MarketplaceRepo struct {
	db *sql.DB
}

func NewMarketplaceRepo(db *sql.DB) *MarketplaceRepo {
	return &MarketplaceRepo{db: db}
}

// ---------------------------------------------------------------------------
// Item Row — flat row returned by joined queries
// ---------------------------------------------------------------------------

type ItemRow struct {
	// marketplace_items columns
	ID                  uuid.UUID
	UserID              uuid.UUID
	Type                mkt.ItemType
	Title               string
	Description         string
	Tags                []string
	PipelineID          *uuid.UUID
	AgentBlockID        *uuid.UUID
	SearchPresetID      *uuid.UUID
	JudgmentScriptID    *uuid.UUID
	ForkedFromID        *uuid.UUID
	ForkCount           int
	UsageCount          int
	ViewCount           int
	LikeCount           int
	Verified            bool
	BacktestJobID       *uuid.UUID
	BacktestWinRate     *float64
	BacktestAvgReturn   *float64
	BacktestTotalEvents *int
	Status              mkt.Status
	PublishedAt         time.Time
	UpdatedAt           time.Time

	// joined author
	AuthorID    uuid.UUID
	AuthorName  string
	AuthorImage *string

	// joined fork origin (nullable)
	ForkOriginID         *uuid.UUID
	ForkOriginTitle      *string
	ForkOriginAuthorName *string

	// search rank (only for full-text queries)
	Rank *float64
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) CreateItem(ctx context.Context, item *mkt.MarketplaceItem) (*mkt.MarketplaceItem, error) {
	const q = `
		INSERT INTO marketplace_items (
			user_id, type, title, description, tags,
			pipeline_id, agent_block_id, search_preset_id, judgment_script_id,
			forked_from_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, status, published_at, updated_at, fork_count, usage_count, view_count, like_count, verified`

	err := r.db.QueryRowContext(ctx, q,
		item.UserID, item.Type, item.Title, item.Description, pq.Array(item.Tags),
		item.PipelineID, item.AgentBlockID, item.SearchPresetID, item.JudgmentScriptID,
		item.ForkedFromID,
	).Scan(
		&item.ID, &item.Status, &item.PublishedAt, &item.UpdatedAt,
		&item.ForkCount, &item.UsageCount, &item.ViewCount, &item.LikeCount, &item.Verified,
	)
	return item, err
}

// ---------------------------------------------------------------------------
// Read
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) GetItemByID(ctx context.Context, id uuid.UUID) (*ItemRow, error) {
	const q = `
		SELECT mi.id, mi.user_id, mi.type, mi.title, mi.description, mi.tags,
		       mi.pipeline_id, mi.agent_block_id, mi.search_preset_id, mi.judgment_script_id,
		       mi.forked_from_id, mi.fork_count, mi.usage_count, mi.view_count, mi.like_count,
		       mi.verified, mi.backtest_job_id, mi.backtest_win_rate, mi.backtest_avg_return, mi.backtest_total_events,
		       mi.status, mi.published_at, mi.updated_at,
		       u.id, u.name, u.image,
		       fo.id, fo.title, fu.name
		FROM marketplace_items mi
		JOIN users u ON mi.user_id = u.id
		LEFT JOIN marketplace_items fo ON mi.forked_from_id = fo.id
		LEFT JOIN users fu ON fo.user_id = fu.id
		WHERE mi.id = $1`

	row := &ItemRow{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&row.ID, &row.UserID, &row.Type, &row.Title, &row.Description, pq.Array(&row.Tags),
		&row.PipelineID, &row.AgentBlockID, &row.SearchPresetID, &row.JudgmentScriptID,
		&row.ForkedFromID, &row.ForkCount, &row.UsageCount, &row.ViewCount, &row.LikeCount,
		&row.Verified, &row.BacktestJobID, &row.BacktestWinRate, &row.BacktestAvgReturn, &row.BacktestTotalEvents,
		&row.Status, &row.PublishedAt, &row.UpdatedAt,
		&row.AuthorID, &row.AuthorName, &row.AuthorImage,
		&row.ForkOriginID, &row.ForkOriginTitle, &row.ForkOriginAuthorName,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace item not found: %w", err)
	}
	return row, nil
}

// ---------------------------------------------------------------------------
// List (filtered, sorted, paginated)
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) ListItems(ctx context.Context, q mkt.ListQuery) ([]ItemRow, int64, error) {
	q.Defaults()
	offset := (q.Page - 1) * q.Limit

	// If search text is provided, use full-text search path
	if q.Search != "" {
		return r.searchItems(ctx, q, offset)
	}

	orderClause := sortToOrderBy(q.Sort)

	baseWhere, args := r.buildWhereClause(q, 1)

	// Count
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM marketplace_items WHERE %s`, baseWhere)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch
	listQ := fmt.Sprintf(`
		SELECT mi.id, mi.user_id, mi.type, mi.title, mi.description, mi.tags,
		       mi.pipeline_id, mi.agent_block_id, mi.search_preset_id, mi.judgment_script_id,
		       mi.forked_from_id, mi.fork_count, mi.usage_count, mi.view_count, mi.like_count,
		       mi.verified, mi.backtest_job_id, mi.backtest_win_rate, mi.backtest_avg_return, mi.backtest_total_events,
		       mi.status, mi.published_at, mi.updated_at,
		       u.id, u.name, u.image
		FROM marketplace_items mi
		JOIN users u ON mi.user_id = u.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		baseWhere, orderClause, len(args)+1, len(args)+2)

	args = append(args, q.Limit, offset)

	rows, err := r.db.QueryContext(ctx, listQ, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ItemRow
	for rows.Next() {
		var row ItemRow
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Type, &row.Title, &row.Description, pq.Array(&row.Tags),
			&row.PipelineID, &row.AgentBlockID, &row.SearchPresetID, &row.JudgmentScriptID,
			&row.ForkedFromID, &row.ForkCount, &row.UsageCount, &row.ViewCount, &row.LikeCount,
			&row.Verified, &row.BacktestJobID, &row.BacktestWinRate, &row.BacktestAvgReturn, &row.BacktestTotalEvents,
			&row.Status, &row.PublishedAt, &row.UpdatedAt,
			&row.AuthorID, &row.AuthorName, &row.AuthorImage,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	return items, total, rows.Err()
}

// searchItems uses PostgreSQL tsvector full-text search.
func (r *MarketplaceRepo) searchItems(ctx context.Context, q mkt.ListQuery, offset int) ([]ItemRow, int64, error) {
	baseWhere, args := r.buildWhereClause(q, 2) // $1 reserved for search term
	searchTerm := q.Search

	ftsWhere := fmt.Sprintf(`mi.search_vector @@ plainto_tsquery('simple', $1) AND %s`, baseWhere)

	// Count
	countQ := fmt.Sprintf(`
		SELECT COUNT(*) FROM marketplace_items mi
		WHERE mi.search_vector @@ plainto_tsquery('simple', $1) AND %s`, baseWhere)
	countArgs := append([]interface{}{searchTerm}, args...)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch ranked
	n := len(args) + 1 // next param index after args
	listQ := fmt.Sprintf(`
		SELECT mi.id, mi.user_id, mi.type, mi.title, mi.description, mi.tags,
		       mi.pipeline_id, mi.agent_block_id, mi.search_preset_id, mi.judgment_script_id,
		       mi.forked_from_id, mi.fork_count, mi.usage_count, mi.view_count, mi.like_count,
		       mi.verified, mi.backtest_job_id, mi.backtest_win_rate, mi.backtest_avg_return, mi.backtest_total_events,
		       mi.status, mi.published_at, mi.updated_at,
		       u.id, u.name, u.image,
		       ts_rank(mi.search_vector, plainto_tsquery('simple', $1)) AS rank
		FROM marketplace_items mi
		JOIN users u ON mi.user_id = u.id
		WHERE %s
		ORDER BY rank DESC
		LIMIT $%d OFFSET $%d`,
		ftsWhere, n+1, n+2)

	fetchArgs := append([]interface{}{searchTerm}, args...)
	fetchArgs = append(fetchArgs, q.Limit, offset)

	rows, err := r.db.QueryContext(ctx, listQ, fetchArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ItemRow
	for rows.Next() {
		var row ItemRow
		if err := rows.Scan(
			&row.ID, &row.UserID, &row.Type, &row.Title, &row.Description, pq.Array(&row.Tags),
			&row.PipelineID, &row.AgentBlockID, &row.SearchPresetID, &row.JudgmentScriptID,
			&row.ForkedFromID, &row.ForkCount, &row.UsageCount, &row.ViewCount, &row.LikeCount,
			&row.Verified, &row.BacktestJobID, &row.BacktestWinRate, &row.BacktestAvgReturn, &row.BacktestTotalEvents,
			&row.Status, &row.PublishedAt, &row.UpdatedAt,
			&row.AuthorID, &row.AuthorName, &row.AuthorImage,
			&row.Rank,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, row)
	}
	return items, total, rows.Err()
}

// buildWhereClause creates the WHERE fragment (excluding full-text) with positional params.
// startParam is the first $N to use.
func (r *MarketplaceRepo) buildWhereClause(q mkt.ListQuery, startParam int) (string, []interface{}) {
	where := "mi.status = 'ACTIVE'"
	var args []interface{}
	n := startParam

	if q.Type != nil {
		where += fmt.Sprintf(" AND mi.type = $%d", n)
		args = append(args, *q.Type)
		n++
	}
	if q.VerifiedOnly != nil && *q.VerifiedOnly {
		where += fmt.Sprintf(" AND mi.verified = $%d", n)
		args = append(args, true)
		n++
	}
	if len(q.Tags) > 0 {
		where += fmt.Sprintf(" AND mi.tags && $%d", n)
		args = append(args, pq.Array(q.Tags))
		n++
	}

	return where, args
}

func sortToOrderBy(s mkt.SortOption) string {
	switch s {
	case mkt.SortPopular:
		return "mi.usage_count DESC"
	case mkt.SortPerformance:
		return "mi.backtest_win_rate DESC NULLS LAST"
	case mkt.SortMostForked:
		return "mi.fork_count DESC"
	default:
		return "mi.published_at DESC"
	}
}

// ---------------------------------------------------------------------------
// Update / Delete
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) UpdateItem(ctx context.Context, id, userID uuid.UUID, title, description string, tags []string) error {
	const q = `
		UPDATE marketplace_items
		SET title = $2, description = $3, tags = $4
		WHERE id = $1 AND user_id = $5`
	res, err := r.db.ExecContext(ctx, q, id, title, description, pq.Array(tags), userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("marketplace item not found or not owned by user")
	}
	return nil
}

func (r *MarketplaceRepo) SoftDeleteItem(ctx context.Context, id, userID uuid.UUID) error {
	const q = `UPDATE marketplace_items SET status = 'REMOVED' WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("marketplace item not found or not owned by user")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Counters
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) IncrementViewCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketplace_items SET view_count = view_count + 1 WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) IncrementForkCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketplace_items SET fork_count = fork_count + 1 WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) IncrementUsageCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketplace_items SET usage_count = usage_count + 1 WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) IncrementLikeCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketplace_items SET like_count = like_count + 1 WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) DecrementLikeCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE marketplace_items SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) SetVerification(ctx context.Context, id uuid.UUID, jobID uuid.UUID, winRate, avgReturn float64, totalEvents int) error {
	const q = `
		UPDATE marketplace_items
		SET verified = true, backtest_job_id = $2, backtest_win_rate = $3, backtest_avg_return = $4, backtest_total_events = $5
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id, jobID, winRate, avgReturn, totalEvents)
	return err
}

// ---------------------------------------------------------------------------
// Likes
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) CreateLike(ctx context.Context, userID, itemID uuid.UUID) (*mkt.MarketplaceLike, error) {
	const q = `
		INSERT INTO marketplace_likes (user_id, item_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, item_id) DO NOTHING
		RETURNING id, user_id, item_id, created_at`
	like := &mkt.MarketplaceLike{}
	err := r.db.QueryRowContext(ctx, q, userID, itemID).Scan(&like.ID, &like.UserID, &like.ItemID, &like.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // already liked — no-op
	}
	return like, err
}

func (r *MarketplaceRepo) DeleteLike(ctx context.Context, userID, itemID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM marketplace_likes WHERE user_id = $1 AND item_id = $2`, userID, itemID)
	return err
}

func (r *MarketplaceRepo) IsLiked(ctx context.Context, userID, itemID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM marketplace_likes WHERE user_id = $1 AND item_id = $2)`,
		userID, itemID,
	).Scan(&exists)
	return exists, err
}

// ---------------------------------------------------------------------------
// Usage Logs
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) CreateUsageLog(ctx context.Context, userID, itemID uuid.UUID, action mkt.UsageAction) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO marketplace_usage_logs (user_id, item_id, action) VALUES ($1, $2, $3)`,
		userID, itemID, action,
	)
	return err
}

func (r *MarketplaceRepo) HasRecentView(ctx context.Context, userID, itemID uuid.UUID, since time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM marketplace_usage_logs
			WHERE item_id = $1 AND user_id = $2 AND action = 'VIEW' AND created_at >= $3
		)`,
		itemID, userID, since,
	).Scan(&exists)
	return exists, err
}

// FindItemIDByResourceID resolves a resource to its marketplace item.
func (r *MarketplaceRepo) FindItemIDByResourceID(ctx context.Context, resourceID uuid.UUID, resourceType mkt.ItemType) (*uuid.UUID, error) {
	var col string
	switch resourceType {
	case mkt.ItemTypePipeline:
		col = "pipeline_id"
	case mkt.ItemTypeAgentBlock:
		col = "agent_block_id"
	case mkt.ItemTypeSearchPreset:
		col = "search_preset_id"
	case mkt.ItemTypeJudgmentScript:
		col = "judgment_script_id"
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	q := fmt.Sprintf(`SELECT id FROM marketplace_items WHERE %s = $1 AND status = 'ACTIVE' LIMIT 1`, col)
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, q, resourceID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// ---------------------------------------------------------------------------
// Batch refresh (for asynq worker)
// ---------------------------------------------------------------------------

func (r *MarketplaceRepo) ListActiveItemIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM marketplace_items WHERE status = 'ACTIVE'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *MarketplaceRepo) RefreshForkCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE marketplace_items SET fork_count = (SELECT COUNT(*) FROM marketplace_items WHERE forked_from_id = $1) WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) RefreshLikeCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE marketplace_items SET like_count = (SELECT COUNT(*) FROM marketplace_likes WHERE item_id = $1) WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) RefreshUsageCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE marketplace_items SET usage_count = (SELECT COUNT(*) FROM marketplace_usage_logs WHERE item_id = $1 AND action = 'EXECUTE') WHERE id = $1`, id)
	return err
}

func (r *MarketplaceRepo) RefreshViewCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE marketplace_items SET view_count = (SELECT COUNT(*) FROM marketplace_usage_logs WHERE item_id = $1 AND action = 'VIEW') WHERE id = $1`, id)
	return err
}

// GetItemLikeCount returns the current like_count for an item.
func (r *MarketplaceRepo) GetItemLikeCount(ctx context.Context, id uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT like_count FROM marketplace_items WHERE id = $1`, id).Scan(&count)
	return count, err
}
