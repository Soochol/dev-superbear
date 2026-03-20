package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/lib/pq"

	mkt "github.com/dev-superbear/nexus-backend/internal/domain/marketplace"
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

// ForkService implements the deep-copy fork engine for marketplace items.
// It copies the underlying resource (pipeline + stages + blocks, agent block,
// search preset, or judgment script) and records the fork relationship.
type ForkService struct {
	db     *sql.DB
	repo   *repository.MarketplaceRepo
	logger *slog.Logger
}

func NewForkService(db *sql.DB, repo *repository.MarketplaceRepo, logger *slog.Logger) *ForkService {
	return &ForkService{db: db, repo: repo, logger: logger}
}

// ForkItem deep-copies the resource behind a marketplace item and creates a new
// marketplace item referencing the copy.
func (s *ForkService) ForkItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*mkt.ForkResult, error) {
	// 1. Fetch original
	row, err := s.repo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("fork: get original item: %w", err)
	}

	if row.Status != mkt.StatusActive {
		return nil, fmt.Errorf("fork: item is not active")
	}

	// 2. Deep-copy in a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fork: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	var newResourceID uuid.UUID
	switch row.Type {
	case mkt.ItemTypePipeline:
		if row.PipelineID == nil {
			return nil, fmt.Errorf("fork: pipeline_id is nil")
		}
		newResourceID, err = s.forkPipeline(ctx, tx, *row.PipelineID, userID)
	case mkt.ItemTypeAgentBlock:
		if row.AgentBlockID == nil {
			return nil, fmt.Errorf("fork: agent_block_id is nil")
		}
		newResourceID, err = s.forkAgentBlock(ctx, tx, *row.AgentBlockID, userID)
	case mkt.ItemTypeSearchPreset:
		if row.SearchPresetID == nil {
			return nil, fmt.Errorf("fork: search_preset_id is nil")
		}
		newResourceID, err = s.forkSearchPreset(ctx, tx, *row.SearchPresetID, userID)
	case mkt.ItemTypeJudgmentScript:
		if row.JudgmentScriptID == nil {
			return nil, fmt.Errorf("fork: judgment_script_id is nil")
		}
		newResourceID, err = s.forkJudgmentScript(ctx, tx, *row.JudgmentScriptID, userID)
	default:
		return nil, fmt.Errorf("fork: unknown item type: %s", row.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("fork: deep copy %s: %w", row.Type, err)
	}

	// 3. Create new marketplace item referencing the fork
	newItem := &mkt.MarketplaceItem{
		UserID:       userID,
		Type:         row.Type,
		Title:        row.Title + " (Fork)",
		Description:  row.Description,
		Tags:         row.Tags,
		ForkedFromID: &itemID,
	}
	switch row.Type {
	case mkt.ItemTypePipeline:
		newItem.PipelineID = &newResourceID
	case mkt.ItemTypeAgentBlock:
		newItem.AgentBlockID = &newResourceID
	case mkt.ItemTypeSearchPreset:
		newItem.SearchPresetID = &newResourceID
	case mkt.ItemTypeJudgmentScript:
		newItem.JudgmentScriptID = &newResourceID
	}

	// Insert new marketplace item inside the same transaction
	var newItemID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO marketplace_items (
			user_id, type, title, description, tags,
			pipeline_id, agent_block_id, search_preset_id, judgment_script_id,
			forked_from_id, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'ACTIVE')
		RETURNING id`,
		newItem.UserID, newItem.Type, newItem.Title, newItem.Description, pq.Array(newItem.Tags),
		newItem.PipelineID, newItem.AgentBlockID, newItem.SearchPresetID, newItem.JudgmentScriptID,
		newItem.ForkedFromID,
	).Scan(&newItemID)
	if err != nil {
		return nil, fmt.Errorf("fork: create new marketplace item: %w", err)
	}

	// 4. Increment original fork_count
	if _, err := tx.ExecContext(ctx,
		`UPDATE marketplace_items SET fork_count = fork_count + 1 WHERE id = $1`, itemID,
	); err != nil {
		return nil, fmt.Errorf("fork: increment fork_count: %w", err)
	}

	// 5. Record usage log
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO marketplace_usage_logs (user_id, item_id, action) VALUES ($1, $2, 'FORK')`,
		userID, itemID,
	); err != nil {
		return nil, fmt.Errorf("fork: record usage log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("fork: commit: %w", err)
	}

	s.logger.InfoContext(ctx, "marketplace item forked",
		"originalItemId", itemID, "newItemId", newItemID, "newResourceId", newResourceID, "userId", userID)

	return &mkt.ForkResult{
		NewItemID:     newItemID,
		NewResourceID: newResourceID,
	}, nil
}

// ---------------------------------------------------------------------------
// Type-specific deep copy helpers
// ---------------------------------------------------------------------------

// forkPipeline deep-copies a pipeline including all analysis_stages, their blocks,
// and monitors with their blocks.
func (s *ForkService) forkPipeline(ctx context.Context, tx *sql.Tx, originalID uuid.UUID, userID uuid.UUID) (uuid.UUID, error) {
	// Copy pipeline row
	var newPipelineID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		INSERT INTO pipelines (user_id, name, description, success_script, failure_script, is_public)
		SELECT $1, name || ' (Fork)', description, success_script, failure_script, false
		FROM pipelines WHERE id = $2
		RETURNING id`, userID, originalID,
	).Scan(&newPipelineID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy pipeline: %w", err)
	}

	// Copy analysis stages
	stageRows, err := tx.QueryContext(ctx, `
		SELECT id, "order" FROM analysis_stages WHERE pipeline_id = $1 ORDER BY "order"`, originalID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("list stages: %w", err)
	}
	defer stageRows.Close()

	type stageMapping struct {
		oldID uuid.UUID
		order int
	}
	var stages []stageMapping
	for stageRows.Next() {
		var sm stageMapping
		if err := stageRows.Scan(&sm.oldID, &sm.order); err != nil {
			return uuid.Nil, err
		}
		stages = append(stages, sm)
	}
	if err := stageRows.Err(); err != nil {
		return uuid.Nil, err
	}

	for _, stage := range stages {
		var newStageID uuid.UUID
		err := tx.QueryRowContext(ctx, `
			INSERT INTO analysis_stages (pipeline_id, "order")
			VALUES ($1, $2)
			RETURNING id`, newPipelineID, stage.order,
		).Scan(&newStageID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("copy stage: %w", err)
		}

		// Copy blocks within this stage
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO agent_blocks (user_id, name, instruction, system_prompt, allowed_tools, output_schema, is_public, stage_id)
			SELECT $1, name || ' (Fork)', instruction, system_prompt, allowed_tools, output_schema, false, $2
			FROM agent_blocks WHERE stage_id = $3`,
			userID, newStageID, stage.oldID,
		); err != nil {
			return uuid.Nil, fmt.Errorf("copy stage blocks: %w", err)
		}
	}

	// Copy monitors and their blocks
	monitorRows, err := tx.QueryContext(ctx, `
		SELECT id, cron FROM monitors WHERE pipeline_id = $1`, originalID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("list monitors: %w", err)
	}
	defer monitorRows.Close()

	type monitorMapping struct {
		oldID uuid.UUID
		cron  string
	}
	var monitors []monitorMapping
	for monitorRows.Next() {
		var mm monitorMapping
		if err := monitorRows.Scan(&mm.oldID, &mm.cron); err != nil {
			return uuid.Nil, err
		}
		monitors = append(monitors, mm)
	}
	if err := monitorRows.Err(); err != nil {
		return uuid.Nil, err
	}

	for _, mon := range monitors {
		// Copy the monitor's block first
		var newBlockID uuid.UUID
		err := tx.QueryRowContext(ctx, `
			INSERT INTO agent_blocks (user_id, name, instruction, system_prompt, allowed_tools, output_schema, is_public)
			SELECT $1, name || ' (Fork)', instruction, system_prompt, allowed_tools, output_schema, false
			FROM agent_blocks WHERE id = (SELECT block_id FROM monitors WHERE id = $2)
			RETURNING id`, userID, mon.oldID,
		).Scan(&newBlockID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("copy monitor block: %w", err)
		}

		// Copy the monitor itself
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO monitors (pipeline_id, block_id, cron, enabled)
			VALUES ($1, $2, $3, true)`,
			newPipelineID, newBlockID, mon.cron,
		); err != nil {
			return uuid.Nil, fmt.Errorf("copy monitor: %w", err)
		}
	}

	return newPipelineID, nil
}

// forkAgentBlock deep-copies a standalone agent block.
func (s *ForkService) forkAgentBlock(ctx context.Context, tx *sql.Tx, originalID uuid.UUID, userID uuid.UUID) (uuid.UUID, error) {
	var newID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		INSERT INTO agent_blocks (user_id, name, instruction, system_prompt, allowed_tools, output_schema, is_public)
		SELECT $1, name || ' (Fork)', instruction, system_prompt, allowed_tools, output_schema, false
		FROM agent_blocks WHERE id = $2
		RETURNING id`, userID, originalID,
	).Scan(&newID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy agent block: %w", err)
	}
	return newID, nil
}

// forkSearchPreset deep-copies a search preset.
func (s *ForkService) forkSearchPreset(ctx context.Context, tx *sql.Tx, originalID uuid.UUID, userID uuid.UUID) (uuid.UUID, error) {
	var newID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		INSERT INTO search_presets (user_id, name, description, mode, query, is_public)
		SELECT $1, name || ' (Fork)', description, mode, query, false
		FROM search_presets WHERE id = $2
		RETURNING id`, userID, originalID,
	).Scan(&newID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy search preset: %w", err)
	}
	return newID, nil
}

// forkJudgmentScript deep-copies a judgment script.
func (s *ForkService) forkJudgmentScript(ctx context.Context, tx *sql.Tx, originalID uuid.UUID, userID uuid.UUID) (uuid.UUID, error) {
	var newID uuid.UUID
	err := tx.QueryRowContext(ctx, `
		INSERT INTO judgment_scripts (user_id, name, description, success_script, failure_script, price_alerts, is_public)
		SELECT $1, name || ' (Fork)', description, success_script, failure_script, price_alerts, false
		FROM judgment_scripts WHERE id = $2
		RETURNING id`, userID, originalID,
	).Scan(&newID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("copy judgment script: %w", err)
	}
	return newID, nil
}
