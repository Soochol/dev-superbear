package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/domain"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// BlockRepository wraps sqlc queries for agent_blocks.
type BlockRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// NewBlockRepository creates a BlockRepository backed by the given pool.
func NewBlockRepository(pool *pgxpool.Pool) *BlockRepository {
	return &BlockRepository{
		q:    sqlc.New(pool),
		pool: pool,
	}
}

// NewBlockRepositoryFromQueries creates a BlockRepository using pre-existing
// sqlc.Queries (e.g. transaction-scoped).
func NewBlockRepositoryFromQueries(q *sqlc.Queries) *BlockRepository {
	return &BlockRepository{q: q}
}

// ---------------------------------------------------------------------------
// Input structs (repo-level, avoids circular import with service)
// ---------------------------------------------------------------------------

// CreateBlockInput holds parameters for creating a new agent block.
type CreateBlockInput struct {
	Name         string
	Objective    string
	InputDesc    string
	Tools        []string
	OutputFormat string
	Constraints  *string
	Examples     *string
	Instruction  string
	SystemPrompt *string
	AllowedTools []string
	OutputSchema interface{}
	IsPublic     bool
	IsTemplate   bool
	TemplateID   *uuid.UUID
}

// UpdateBlockInput holds parameters for updating an agent block.
type UpdateBlockInput struct {
	Name         string
	Objective    string
	InputDesc    string
	Tools        []string
	OutputFormat string
	Constraints  *string
	Examples     *string
	Instruction  string
	SystemPrompt *string
	AllowedTools []string
	OutputSchema interface{}
	IsPublic     bool
}

// ---------------------------------------------------------------------------
// Block CRUD
// ---------------------------------------------------------------------------

// FindMany returns standalone blocks (not assigned to any stage) for a user.
func (r *BlockRepository) FindMany(ctx context.Context, userID uuid.UUID) ([]domain.AgentBlock, error) {
	rows, err := r.q.ListBlocksByUser(ctx, toPgtypeUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list blocks by user: %w", err)
	}
	blocks := make([]domain.AgentBlock, len(rows))
	for i, row := range rows {
		blocks[i] = toDomainBlock(row)
	}
	return blocks, nil
}

// FindTemplates returns template blocks visible to a user (own + public).
func (r *BlockRepository) FindTemplates(ctx context.Context, userID uuid.UUID) ([]domain.AgentBlock, error) {
	rows, err := r.q.ListTemplates(ctx, toPgtypeUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	blocks := make([]domain.AgentBlock, len(rows))
	for i, row := range rows {
		blocks[i] = toDomainBlock(row)
	}
	return blocks, nil
}

// FindByID returns a single block by ID.
func (r *BlockRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AgentBlock, error) {
	row, err := r.q.GetBlockByID(ctx, toPgtypeUUID(id))
	if err != nil {
		return nil, fmt.Errorf("get block: %w", err)
	}
	b := toDomainBlock(row)
	return &b, nil
}

// FindByStage returns all blocks belonging to a stage.
func (r *BlockRepository) FindByStage(ctx context.Context, stageID uuid.UUID) ([]domain.AgentBlock, error) {
	rows, err := r.q.ListBlocksByStage(ctx, toPgtypeUUID(stageID))
	if err != nil {
		return nil, fmt.Errorf("list blocks by stage: %w", err)
	}
	blocks := make([]domain.AgentBlock, len(rows))
	for i, row := range rows {
		blocks[i] = toDomainBlock(row)
	}
	return blocks, nil
}

// Create inserts a new agent block.
func (r *BlockRepository) Create(ctx context.Context, userID uuid.UUID, stageID *uuid.UUID, req *CreateBlockInput) (*domain.AgentBlock, error) {
	allowedToolsJSON, err := json.Marshal(req.AllowedTools)
	if err != nil {
		return nil, fmt.Errorf("marshal allowed tools: %w", err)
	}

	var outputSchemaJSON []byte
	if req.OutputSchema != nil {
		outputSchemaJSON, err = json.Marshal(req.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal output schema: %w", err)
		}
	}

	tools := req.Tools
	if tools == nil {
		tools = []string{}
	}

	row, err := r.q.CreateBlock(ctx, sqlc.CreateBlockParams{
		UserID:       toPgtypeUUID(userID),
		StageID:      toPgtypeUUIDPtr(stageID),
		Name:         req.Name,
		Objective:    req.Objective,
		InputDesc:    req.InputDesc,
		Tools:        tools,
		OutputFormat: req.OutputFormat,
		Constraints:  toPgtypeText(req.Constraints),
		Examples:     toPgtypeText(req.Examples),
		Instruction:  req.Instruction,
		SystemPrompt: toPgtypeText(req.SystemPrompt),
		AllowedTools: allowedToolsJSON,
		OutputSchema: outputSchemaJSON,
		IsPublic:     req.IsPublic,
		IsTemplate:   req.IsTemplate,
		TemplateID:   toPgtypeUUIDPtr(req.TemplateID),
	})
	if err != nil {
		return nil, fmt.Errorf("create block: %w", err)
	}
	b := toDomainBlock(row)
	return &b, nil
}

// CreateFromTemplate copies a template block into a user's stage.
func (r *BlockRepository) CreateFromTemplate(ctx context.Context, templateID, userID, stageID uuid.UUID) (*domain.AgentBlock, error) {
	tmpl, err := r.q.GetBlockByID(ctx, toPgtypeUUID(templateID))
	if err != nil {
		return nil, fmt.Errorf("get template block: %w", err)
	}

	row, err := r.q.CreateBlock(ctx, sqlc.CreateBlockParams{
		UserID:       toPgtypeUUID(userID),
		StageID:      toPgtypeUUID(stageID),
		Name:         tmpl.Name,
		Objective:    tmpl.Objective,
		InputDesc:    tmpl.InputDesc,
		Tools:        tmpl.Tools,
		OutputFormat: tmpl.OutputFormat,
		Constraints:  tmpl.Constraints,
		Examples:     tmpl.Examples,
		Instruction:  tmpl.Instruction,
		SystemPrompt: tmpl.SystemPrompt,
		AllowedTools: tmpl.AllowedTools,
		OutputSchema: tmpl.OutputSchema,
		IsPublic:     false,
		IsTemplate:   false,
		TemplateID:   toPgtypeUUID(templateID),
	})
	if err != nil {
		return nil, fmt.Errorf("create block from template: %w", err)
	}
	b := toDomainBlock(row)
	return &b, nil
}

// Update modifies an existing agent block.
func (r *BlockRepository) Update(ctx context.Context, id uuid.UUID, req *UpdateBlockInput) (*domain.AgentBlock, error) {
	allowedToolsJSON, err := json.Marshal(req.AllowedTools)
	if err != nil {
		return nil, fmt.Errorf("marshal allowed tools: %w", err)
	}

	var outputSchemaJSON []byte
	if req.OutputSchema != nil {
		outputSchemaJSON, err = json.Marshal(req.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal output schema: %w", err)
		}
	}

	tools := req.Tools
	if tools == nil {
		tools = []string{}
	}

	row, err := r.q.UpdateBlock(ctx, sqlc.UpdateBlockParams{
		ID:           toPgtypeUUID(id),
		Name:         req.Name,
		Objective:    req.Objective,
		InputDesc:    req.InputDesc,
		Tools:        tools,
		OutputFormat: req.OutputFormat,
		Constraints:  toPgtypeText(req.Constraints),
		Examples:     toPgtypeText(req.Examples),
		Instruction:  req.Instruction,
		SystemPrompt: toPgtypeText(req.SystemPrompt),
		AllowedTools: allowedToolsJSON,
		OutputSchema: outputSchemaJSON,
		IsPublic:     req.IsPublic,
	})
	if err != nil {
		return nil, fmt.Errorf("update block: %w", err)
	}
	b := toDomainBlock(row)
	return &b, nil
}

// Delete removes a block scoped to a user.
func (r *BlockRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteBlock(ctx, sqlc.DeleteBlockParams{
		ID:     toPgtypeUUID(id),
		UserID: toPgtypeUUID(userID),
	})
}

