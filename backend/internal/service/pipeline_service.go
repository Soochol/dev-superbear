package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dev-superbear/nexus-backend/internal/domain"
	"github.com/dev-superbear/nexus-backend/internal/repository"
)

// PipelineService orchestrates pipeline CRUD via repositories.
type PipelineService struct {
	pipelineRepo *repository.PipelineRepository
	blockRepo    *repository.BlockRepository
}

// NewPipelineService creates a PipelineService with its dependencies.
func NewPipelineService(pr *repository.PipelineRepository, br *repository.BlockRepository) *PipelineService {
	return &PipelineService{pipelineRepo: pr, blockRepo: br}
}

// List returns a paginated list of pipelines for a user.
func (s *PipelineService) List(ctx context.Context, userID string, limit, offset int32) ([]domain.Pipeline, int64, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid user ID: %w", err)
	}
	return s.pipelineRepo.FindMany(ctx, uid, limit, offset)
}

// GetByID returns a pipeline with all relations loaded.
func (s *PipelineService) GetByID(ctx context.Context, userID, id string) (*domain.Pipeline, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline ID: %w", err)
	}
	return s.pipelineRepo.FindByID(ctx, pid, uid)
}

// Create builds a full pipeline: pipeline row, stages, blocks per stage,
// monitors (with their blocks), and price alerts.
func (s *PipelineService) Create(ctx context.Context, userID string, req *CreatePipelineRequest) (*domain.Pipeline, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 1. Create pipeline
	p := &domain.Pipeline{
		UserID:        uid,
		Name:          req.Name,
		Description:   req.Description,
		SuccessScript: req.SuccessScript,
		FailureScript: req.FailureScript,
		IsPublic:      req.IsPublic,
	}
	created, err := s.pipelineRepo.Create(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	// 2. Create stages and their blocks
	for _, sr := range req.Stages {
		stage, err := s.pipelineRepo.CreateStage(ctx, created.ID, sr.Section, sr.Order)
		if err != nil {
			return nil, fmt.Errorf("create stage: %w", err)
		}

		for _, br := range sr.Blocks {
			_, err := s.blockRepo.Create(ctx, uid, &stage.ID, &repository.CreateBlockInput{
				Name:         br.Name,
				Objective:    br.Objective,
				InputDesc:    br.InputDesc,
				Tools:        br.Tools,
				OutputFormat: br.OutputFormat,
				Constraints:  br.Constraints,
				Examples:     br.Examples,
				Instruction:  br.Instruction,
			})
			if err != nil {
				return nil, fmt.Errorf("create block in stage: %w", err)
			}
		}
	}

	// 3. Create monitors: first create the block, then the monitor_block
	for _, mr := range req.Monitors {
		block, err := s.blockRepo.Create(ctx, uid, nil, &repository.CreateBlockInput{
			Name:         mr.Block.Name,
			Objective:    mr.Block.Objective,
			InputDesc:    mr.Block.InputDesc,
			Tools:        mr.Block.Tools,
			OutputFormat: mr.Block.OutputFormat,
			Constraints:  mr.Block.Constraints,
			Examples:     mr.Block.Examples,
			Instruction:  mr.Block.Instruction,
		})
		if err != nil {
			return nil, fmt.Errorf("create monitor block: %w", err)
		}

		_, err = s.pipelineRepo.CreateMonitor(ctx, created.ID, block.ID, mr.Cron, mr.Enabled)
		if err != nil {
			return nil, fmt.Errorf("create monitor: %w", err)
		}
	}

	// 4. Create price alerts
	for _, ar := range req.PriceAlerts {
		_, err := s.pipelineRepo.CreatePriceAlert(ctx, created.ID, ar.Condition, ar.Label)
		if err != nil {
			return nil, fmt.Errorf("create price alert: %w", err)
		}
	}

	// 5. Reload with all relations
	return s.pipelineRepo.FindByID(ctx, created.ID, uid)
}

// Update replaces a pipeline's stages, blocks, monitors, and price alerts.
func (s *PipelineService) Update(ctx context.Context, userID, id string, req *UpdatePipelineRequest) (*domain.Pipeline, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline ID: %w", err)
	}

	// 1. Update pipeline core fields
	p := &domain.Pipeline{
		Name:          req.Name,
		Description:   req.Description,
		SuccessScript: req.SuccessScript,
		FailureScript: req.FailureScript,
		IsPublic:      req.IsPublic,
	}
	_, err = s.pipelineRepo.Update(ctx, pid, uid, p)
	if err != nil {
		return nil, fmt.Errorf("update pipeline: %w", err)
	}

	// 2. Delete existing stages (CASCADE deletes blocks within stages)
	if err := s.pipelineRepo.DeleteStages(ctx, pid); err != nil {
		return nil, fmt.Errorf("delete stages: %w", err)
	}

	// 3. Delete existing monitors and their blocks
	existingMonitors, err := s.pipelineRepo.ListMonitors(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("list existing monitors: %w", err)
	}
	for _, mon := range existingMonitors {
		if err := s.pipelineRepo.DeleteMonitor(ctx, mon.ID); err != nil {
			return nil, fmt.Errorf("delete monitor: %w", err)
		}
		// Delete the monitor's block
		if err := s.blockRepo.Delete(ctx, mon.BlockID, uid); err != nil {
			// Block may already be deleted by CASCADE; ignore
		}
	}

	// 4. Delete existing price alerts
	existingAlerts, err := s.pipelineRepo.ListPriceAlerts(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("list existing alerts: %w", err)
	}
	for _, alert := range existingAlerts {
		if err := s.pipelineRepo.DeletePriceAlert(ctx, alert.ID); err != nil {
			return nil, fmt.Errorf("delete price alert: %w", err)
		}
	}

	// 5. Recreate stages + blocks
	for _, sr := range req.Stages {
		stage, err := s.pipelineRepo.CreateStage(ctx, pid, sr.Section, sr.Order)
		if err != nil {
			return nil, fmt.Errorf("create stage: %w", err)
		}

		for _, br := range sr.Blocks {
			_, err := s.blockRepo.Create(ctx, uid, &stage.ID, &repository.CreateBlockInput{
				Name:         br.Name,
				Objective:    br.Objective,
				InputDesc:    br.InputDesc,
				Tools:        br.Tools,
				OutputFormat: br.OutputFormat,
				Constraints:  br.Constraints,
				Examples:     br.Examples,
				Instruction:  br.Instruction,
			})
			if err != nil {
				return nil, fmt.Errorf("create block in stage: %w", err)
			}
		}
	}

	// 6. Recreate monitors
	for _, mr := range req.Monitors {
		block, err := s.blockRepo.Create(ctx, uid, nil, &repository.CreateBlockInput{
			Name:         mr.Block.Name,
			Objective:    mr.Block.Objective,
			InputDesc:    mr.Block.InputDesc,
			Tools:        mr.Block.Tools,
			OutputFormat: mr.Block.OutputFormat,
			Constraints:  mr.Block.Constraints,
			Examples:     mr.Block.Examples,
			Instruction:  mr.Block.Instruction,
		})
		if err != nil {
			return nil, fmt.Errorf("create monitor block: %w", err)
		}

		_, err = s.pipelineRepo.CreateMonitor(ctx, pid, block.ID, mr.Cron, mr.Enabled)
		if err != nil {
			return nil, fmt.Errorf("create monitor: %w", err)
		}
	}

	// 7. Recreate price alerts
	for _, ar := range req.PriceAlerts {
		_, err := s.pipelineRepo.CreatePriceAlert(ctx, pid, ar.Condition, ar.Label)
		if err != nil {
			return nil, fmt.Errorf("create price alert: %w", err)
		}
	}

	// 8. Reload with all relations
	return s.pipelineRepo.FindByID(ctx, pid, uid)
}

// Delete removes a pipeline.
func (s *PipelineService) Delete(ctx context.Context, userID, id string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid pipeline ID: %w", err)
	}
	return s.pipelineRepo.Delete(ctx, pid, uid)
}

// Execute creates a pipeline job for execution.
func (s *PipelineService) Execute(ctx context.Context, userID, id, symbol string) (*domain.PipelineJob, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline ID: %w", err)
	}

	// Verify pipeline exists and belongs to user
	_, err = s.pipelineRepo.FindByID(ctx, pid, uid)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	return s.pipelineRepo.CreateJob(ctx, pid, symbol)
}

// GetJob returns a pipeline job by ID.
func (s *PipelineService) GetJob(ctx context.Context, id string) (*domain.PipelineJob, error) {
	jobID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}
	return s.pipelineRepo.GetJob(ctx, jobID)
}
