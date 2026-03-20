package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dev-superbear/nexus-backend/internal/domain"
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

// BlockService orchestrates agent block CRUD via the repository.
type BlockService struct {
	repo *repository.BlockRepository
}

// NewBlockService creates a BlockService.
func NewBlockService(repo *repository.BlockRepository) *BlockService {
	return &BlockService{repo: repo}
}

// ListBlocks returns standalone blocks for a user.
func (s *BlockService) ListBlocks(ctx context.Context, userID string) ([]domain.AgentBlock, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return s.repo.FindMany(ctx, uid)
}

// ListTemplates returns template blocks visible to a user.
func (s *BlockService) ListTemplates(ctx context.Context, userID string) ([]domain.AgentBlock, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return s.repo.FindTemplates(ctx, uid)
}

// GetBlock returns a single block by ID, enforcing ownership for non-public blocks.
func (s *BlockService) GetBlock(ctx context.Context, userID, id string) (*domain.AgentBlock, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	bid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid block ID: %w", err)
	}
	block, err := s.repo.FindByID(ctx, bid)
	if err != nil {
		return nil, err
	}
	if !block.IsPublic && block.UserID != uid {
		return nil, fmt.Errorf("block: %w", domain.ErrNotFound)
	}
	return block, nil
}

// CreateBlock creates a new standalone agent block.
func (s *BlockService) CreateBlock(ctx context.Context, userID string, req *CreateBlockRequest) (*domain.AgentBlock, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.repo.Create(ctx, uid, nil, &repository.CreateBlockInput{
		Name:         req.Name,
		Objective:    req.Objective,
		InputDesc:    req.InputDesc,
		Tools:        req.Tools,
		OutputFormat: req.OutputFormat,
		Constraints:  req.Constraints,
		Examples:     req.Examples,
		Instruction:  req.Instruction,
		IsTemplate:   req.IsTemplate,
	})
}

// CopyFromTemplate copies a template block into a user's stage.
func (s *BlockService) CopyFromTemplate(ctx context.Context, userID, templateID, stageID string) (*domain.AgentBlock, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	tid, err := uuid.Parse(templateID)
	if err != nil {
		return nil, fmt.Errorf("invalid template ID: %w", err)
	}
	sid, err := uuid.Parse(stageID)
	if err != nil {
		return nil, fmt.Errorf("invalid stage ID: %w", err)
	}
	return s.repo.CreateFromTemplate(ctx, tid, uid, sid)
}

// UpdateBlock updates an existing agent block.
func (s *BlockService) UpdateBlock(ctx context.Context, userID, id string, req *UpdateBlockRequest) (*domain.AgentBlock, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	bid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid block ID: %w", err)
	}

	// Verify ownership
	existing, err := s.repo.FindByID(ctx, bid)
	if err != nil {
		return nil, fmt.Errorf("block not found: %w", err)
	}
	if existing.UserID != uid {
		return nil, fmt.Errorf("block: %w", domain.ErrNotFound)
	}

	return s.repo.Update(ctx, bid, &repository.UpdateBlockInput{
		Name:         req.Name,
		Objective:    req.Objective,
		InputDesc:    req.InputDesc,
		Tools:        req.Tools,
		OutputFormat: req.OutputFormat,
		Constraints:  req.Constraints,
		Examples:     req.Examples,
		Instruction:  req.Instruction,
		SystemPrompt: req.SystemPrompt,
		AllowedTools: req.AllowedTools,
		IsPublic:     req.IsPublic,
	})
}

// DeleteBlock removes an agent block.
func (s *BlockService) DeleteBlock(ctx context.Context, userID, id string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	bid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid block ID: %w", err)
	}
	return s.repo.Delete(ctx, bid, uid)
}
