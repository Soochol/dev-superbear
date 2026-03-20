package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/dev-superbear/nexus-backend/internal/domain"
	"github.com/dev-superbear/nexus-backend/internal/repository"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// PipelineService orchestrates pipeline CRUD via repositories.
type PipelineService struct {
	pipelineRepo *repository.PipelineRepository
	blockRepo    *repository.BlockRepository
	orchestrator *PipelineOrchestrator
}

// NewPipelineService creates a PipelineService with its dependencies.
func NewPipelineService(pr *repository.PipelineRepository, br *repository.BlockRepository, orch *PipelineOrchestrator) *PipelineService {
	return &PipelineService{pipelineRepo: pr, blockRepo: br, orchestrator: orch}
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
// monitors (with their blocks), and price alerts. All writes are wrapped
// in a single database transaction.
func (s *PipelineService) Create(ctx context.Context, userID string, req *CreatePipelineRequest) (*domain.Pipeline, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var createdID uuid.UUID

	if err := s.pipelineRepo.WithTx(ctx, func(q *sqlc.Queries) error {
		txPipeRepo := repository.NewPipelineRepositoryFromQueries(q)
		txBlockRepo := repository.NewBlockRepositoryFromQueries(q)

		// 1. Create pipeline
		p := &domain.Pipeline{
			UserID:        uid,
			Name:          req.Name,
			Description:   req.Description,
			SuccessScript: req.SuccessScript,
			FailureScript: req.FailureScript,
			IsPublic:      req.IsPublic,
		}
		created, err := txPipeRepo.Create(ctx, p)
		if err != nil {
			return fmt.Errorf("create pipeline: %w", err)
		}
		createdID = created.ID

		// 2. Create stages, blocks, monitors, and price alerts
		if err := createChildren(ctx, txPipeRepo, txBlockRepo, created.ID, uid, req.Stages, req.Monitors, req.PriceAlerts); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// 5. Reload with all relations (outside transaction)
	return s.pipelineRepo.FindByID(ctx, createdID, uid)
}

// Update replaces a pipeline's stages, blocks, monitors, and price alerts.
// All writes are wrapped in a single database transaction.
func (s *PipelineService) Update(ctx context.Context, userID, id string, req *UpdatePipelineRequest) (*domain.Pipeline, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline ID: %w", err)
	}

	if err := s.pipelineRepo.WithTx(ctx, func(q *sqlc.Queries) error {
		txPipeRepo := repository.NewPipelineRepositoryFromQueries(q)
		txBlockRepo := repository.NewBlockRepositoryFromQueries(q)

		// 1. Update pipeline core fields
		p := &domain.Pipeline{
			Name:          req.Name,
			Description:   req.Description,
			SuccessScript: req.SuccessScript,
			FailureScript: req.FailureScript,
			IsPublic:      req.IsPublic,
		}
		_, err := txPipeRepo.Update(ctx, pid, uid, p)
		if err != nil {
			return fmt.Errorf("update pipeline: %w", err)
		}

		// 2. Delete existing stages (CASCADE deletes blocks within stages)
		if err := txPipeRepo.DeleteStages(ctx, pid); err != nil {
			return fmt.Errorf("delete stages: %w", err)
		}

		// 3. Delete existing monitors and their blocks
		existingMonitors, err := txPipeRepo.ListMonitors(ctx, pid)
		if err != nil {
			return fmt.Errorf("list existing monitors: %w", err)
		}
		for _, mon := range existingMonitors {
			if err := txPipeRepo.DeleteMonitor(ctx, mon.ID); err != nil {
				return fmt.Errorf("delete monitor: %w", err)
			}
			// Delete the monitor's block
			if err := txBlockRepo.Delete(ctx, mon.BlockID, uid); err != nil {
				// Block may already be deleted by CASCADE; ignore
			}
		}

		// 4. Delete existing price alerts
		existingAlerts, err := txPipeRepo.ListPriceAlerts(ctx, pid)
		if err != nil {
			return fmt.Errorf("list existing alerts: %w", err)
		}
		for _, alert := range existingAlerts {
			if err := txPipeRepo.DeletePriceAlert(ctx, alert.ID); err != nil {
				return fmt.Errorf("delete price alert: %w", err)
			}
		}

		// 5. Recreate stages, blocks, monitors, and price alerts
		if err := createChildren(ctx, txPipeRepo, txBlockRepo, pid, uid, req.Stages, req.Monitors, req.PriceAlerts); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	// 8. Reload with all relations (outside transaction)
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

// Execute creates a pipeline job and launches the orchestrator in a background goroutine.
func (s *PipelineService) Execute(ctx context.Context, userID, id, symbol string) (*domain.PipelineJob, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid pipeline ID: %w", err)
	}

	// Load pipeline with all relations
	pipeline, err := s.pipelineRepo.FindByID(ctx, pid, uid)
	if err != nil {
		return nil, fmt.Errorf("pipeline not found: %w", err)
	}

	// Create the job record
	job, err := s.pipelineRepo.CreateJob(ctx, pid, symbol)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	// Launch orchestrator in background goroutine
	go func() {
		bgCtx := context.Background()

		// Mark job as running
		if err := s.pipelineRepo.UpdateJobStatus(bgCtx, job.ID, domain.JobStatusRunning, nil, nil); err != nil {
			slog.Error("failed to update job status to RUNNING", "jobId", job.ID, "error", err)
			return
		}

		execCtx, err := s.orchestrator.Execute(bgCtx, pipeline, symbol)
		if err != nil {
			errMsg := err.Error()
			slog.Error("pipeline execution failed", "jobId", job.ID, "error", err)
			_ = s.pipelineRepo.UpdateJobStatus(bgCtx, job.ID, domain.JobStatusFailed, execCtx, &errMsg)
			return
		}

		if len(execCtx.Errors) > 0 {
			slog.Warn("pipeline completed with partial block failures", "jobId", job.ID, "errorCount", len(execCtx.Errors))
		}

		if err := s.pipelineRepo.UpdateJobStatus(bgCtx, job.ID, domain.JobStatusCompleted, execCtx, nil); err != nil {
			slog.Error("failed to update job status to COMPLETED", "jobId", job.ID, "error", err)
		}
	}()

	return job, nil
}

// createChildren creates stages with their blocks, monitors (with their blocks),
// and price alerts for a pipeline. It is shared by Create and Update.
func createChildren(
	ctx context.Context,
	pipeRepo *repository.PipelineRepository,
	blockRepo *repository.BlockRepository,
	pipelineID, userID uuid.UUID,
	stages []StageRequest,
	monitors []MonitorRequest,
	alerts []PriceAlertRequest,
) error {
	// Stages + blocks
	for _, sr := range stages {
		stage, err := pipeRepo.CreateStage(ctx, pipelineID, sr.Section, sr.Order)
		if err != nil {
			return fmt.Errorf("create stage: %w", err)
		}
		for _, br := range sr.Blocks {
			_, err := blockRepo.Create(ctx, userID, &stage.ID, &repository.CreateBlockInput{
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
				return fmt.Errorf("create block in stage: %w", err)
			}
		}
	}

	// Monitors: create the block first, then the monitor_block
	for _, mr := range monitors {
		block, err := blockRepo.Create(ctx, userID, nil, &repository.CreateBlockInput{
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
			return fmt.Errorf("create monitor block: %w", err)
		}
		_, err = pipeRepo.CreateMonitor(ctx, pipelineID, block.ID, mr.Cron, mr.Enabled)
		if err != nil {
			return fmt.Errorf("create monitor: %w", err)
		}
	}

	// Price alerts
	for _, ar := range alerts {
		_, err := pipeRepo.CreatePriceAlert(ctx, pipelineID, ar.Condition, ar.Label)
		if err != nil {
			return fmt.Errorf("create price alert: %w", err)
		}
	}

	return nil
}

// GetJob returns a pipeline job by ID, verifying the job's pipeline belongs to the user.
func (s *PipelineService) GetJob(ctx context.Context, userID, id string) (*domain.PipelineJob, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	jobID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}
	job, err := s.pipelineRepo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	// Verify ownership by loading the pipeline
	_, err = s.pipelineRepo.FindByID(ctx, job.PipelineID, uid)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}
	return job, nil
}
