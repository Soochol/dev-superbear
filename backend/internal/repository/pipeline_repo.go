package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dev-superbear/nexus-backend/internal/domain"
	"github.com/dev-superbear/nexus-backend/internal/repository/sqlc"
)

// PipelineRepository wraps sqlc queries and converts between pgtype and domain types.
type PipelineRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

// NewPipelineRepository creates a PipelineRepository backed by the given pool.
func NewPipelineRepository(pool *pgxpool.Pool) *PipelineRepository {
	return &PipelineRepository{
		q:    sqlc.New(pool),
		pool: pool,
	}
}

// ---------------------------------------------------------------------------
// Type conversion helpers
// ---------------------------------------------------------------------------

func toPgtypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toUUID(pt pgtype.UUID) uuid.UUID {
	if !pt.Valid {
		return uuid.Nil
	}
	return uuid.UUID(pt.Bytes)
}

func toPgtypeUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func toUUIDPtr(pt pgtype.UUID) *uuid.UUID {
	if !pt.Valid {
		return nil
	}
	id := uuid.UUID(pt.Bytes)
	return &id
}

func toPgtypeText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func toStringPtr(pt pgtype.Text) *string {
	if !pt.Valid {
		return nil
	}
	return &pt.String
}

func toTime(pt pgtype.Timestamptz) time.Time {
	if !pt.Valid {
		return time.Time{}
	}
	return pt.Time
}

func toTimePtr(pt pgtype.Timestamptz) *time.Time {
	if !pt.Valid {
		return nil
	}
	return &pt.Time
}

func toPgtypeTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: !t.IsZero()}
}

// ---------------------------------------------------------------------------
// Domain conversion helpers
// ---------------------------------------------------------------------------

func toDomainPipeline(row sqlc.Pipeline) domain.Pipeline {
	return domain.Pipeline{
		ID:            toUUID(row.ID),
		UserID:        toUUID(row.UserID),
		Name:          row.Name,
		Description:   row.Description,
		SuccessScript: toStringPtr(row.SuccessScript),
		FailureScript: toStringPtr(row.FailureScript),
		IsPublic:      row.IsPublic,
		CreatedAt:     toTime(row.CreatedAt),
		UpdatedAt:     toTime(row.UpdatedAt),
	}
}

func toDomainStage(row sqlc.Stage) domain.Stage {
	return domain.Stage{
		ID:         toUUID(row.ID),
		PipelineID: toUUID(row.PipelineID),
		Section:    row.Section,
		OrderIndex: int(row.OrderIndex),
		CreatedAt:  toTime(row.CreatedAt),
	}
}

func toDomainBlock(row sqlc.AgentBlock) domain.AgentBlock {
	var allowedTools []string
	if row.AllowedTools != nil {
		if err := json.Unmarshal(row.AllowedTools, &allowedTools); err != nil {
			slog.Warn("failed to unmarshal allowedTools", "error", err)
		}
	}

	var outputSchema *json.RawMessage
	if row.OutputSchema != nil {
		raw := json.RawMessage(row.OutputSchema)
		outputSchema = &raw
	}

	tools := row.Tools
	if tools == nil {
		tools = []string{}
	}

	return domain.AgentBlock{
		ID:           toUUID(row.ID),
		UserID:       toUUID(row.UserID),
		StageID:      toUUIDPtr(row.StageID),
		Name:         row.Name,
		Objective:    row.Objective,
		InputDesc:    row.InputDesc,
		Tools:        tools,
		OutputFormat: row.OutputFormat,
		Constraints:  toStringPtr(row.Constraints),
		Examples:     toStringPtr(row.Examples),
		Instruction:  row.Instruction,
		SystemPrompt: toStringPtr(row.SystemPrompt),
		AllowedTools: allowedTools,
		OutputSchema: outputSchema,
		IsPublic:     row.IsPublic,
		IsTemplate:   row.IsTemplate,
		TemplateID:   toUUIDPtr(row.TemplateID),
		CreatedAt:    toTime(row.CreatedAt),
		UpdatedAt:    toTime(row.UpdatedAt),
	}
}

func toDomainMonitor(row sqlc.MonitorBlock) domain.MonitorBlock {
	return domain.MonitorBlock{
		ID:         toUUID(row.ID),
		PipelineID: toUUID(row.PipelineID),
		BlockID:    toUUID(row.BlockID),
		Cron:       row.Cron,
		Enabled:    row.Enabled,
		CreatedAt:  toTime(row.CreatedAt),
		UpdatedAt:  toTime(row.UpdatedAt),
	}
}

func toDomainPriceAlert(row sqlc.PriceAlert) domain.PriceAlert {
	return domain.PriceAlert{
		ID:         toUUID(row.ID),
		PipelineID: toUUIDPtr(row.PipelineID),
		CaseID:     toUUIDPtr(row.CaseID),
		Condition:  row.Condition,
		Label:      row.Label,
		Triggered:  row.Triggered,
		CreatedAt:  toTime(row.CreatedAt),
	}
}

func toDomainJob(row sqlc.PipelineJob) domain.PipelineJob {
	var result *json.RawMessage
	if row.Result != nil {
		raw := json.RawMessage(row.Result)
		result = &raw
	}

	return domain.PipelineJob{
		ID:          toUUID(row.ID),
		PipelineID:  toUUID(row.PipelineID),
		Symbol:      row.Symbol,
		Status:      row.Status,
		Result:      result,
		Error:       toStringPtr(row.Error),
		StartedAt:   toTimePtr(row.StartedAt),
		CompletedAt: toTimePtr(row.CompletedAt),
		CreatedAt:   toTime(row.CreatedAt),
	}
}

// ---------------------------------------------------------------------------
// Pipeline CRUD
// ---------------------------------------------------------------------------

// FindMany returns a paginated list of pipelines for a user plus total count.
func (r *PipelineRepository) FindMany(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.Pipeline, int64, error) {
	uid := toPgtypeUUID(userID)

	rows, err := r.q.ListPipelinesByUser(ctx, sqlc.ListPipelinesByUserParams{
		UserID: uid,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list pipelines: %w", err)
	}

	count, err := r.q.CountPipelinesByUser(ctx, uid)
	if err != nil {
		return nil, 0, fmt.Errorf("count pipelines: %w", err)
	}

	pipelines := make([]domain.Pipeline, len(rows))
	for i, row := range rows {
		pipelines[i] = toDomainPipeline(row)
	}
	return pipelines, count, nil
}

// FindByID returns a pipeline with its stages, blocks, monitors, and price alerts loaded.
func (r *PipelineRepository) FindByID(ctx context.Context, id, userID uuid.UUID) (*domain.Pipeline, error) {
	row, err := r.q.GetPipelineByID(ctx, sqlc.GetPipelineByIDParams{
		ID:     toPgtypeUUID(id),
		UserID: toPgtypeUUID(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	p := toDomainPipeline(row)

	// Load stages with blocks
	stages, err := r.loadStagesWithBlocks(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Stages = stages

	// Load monitors with their blocks
	monitors, err := r.loadMonitorsWithBlocks(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Monitors = monitors

	// Load price alerts
	alertRows, err := r.q.ListPriceAlertsByPipeline(ctx, toPgtypeUUID(id))
	if err != nil {
		return nil, fmt.Errorf("list price alerts: %w", err)
	}
	alerts := make([]domain.PriceAlert, len(alertRows))
	for i, a := range alertRows {
		alerts[i] = toDomainPriceAlert(a)
	}
	p.PriceAlerts = alerts

	return &p, nil
}

// Create inserts a new pipeline row.
func (r *PipelineRepository) Create(ctx context.Context, p *domain.Pipeline) (*domain.Pipeline, error) {
	row, err := r.q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		UserID:        toPgtypeUUID(p.UserID),
		Name:          p.Name,
		Description:   p.Description,
		SuccessScript: toPgtypeText(p.SuccessScript),
		FailureScript: toPgtypeText(p.FailureScript),
		IsPublic:      p.IsPublic,
	})
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}
	result := toDomainPipeline(row)
	return &result, nil
}

// Update modifies an existing pipeline's core fields.
func (r *PipelineRepository) Update(ctx context.Context, id, userID uuid.UUID, p *domain.Pipeline) (*domain.Pipeline, error) {
	row, err := r.q.UpdatePipeline(ctx, sqlc.UpdatePipelineParams{
		ID:            toPgtypeUUID(id),
		UserID:        toPgtypeUUID(userID),
		Name:          p.Name,
		Description:   p.Description,
		SuccessScript: toPgtypeText(p.SuccessScript),
		FailureScript: toPgtypeText(p.FailureScript),
		IsPublic:      p.IsPublic,
	})
	if err != nil {
		return nil, fmt.Errorf("update pipeline: %w", err)
	}
	result := toDomainPipeline(row)
	return &result, nil
}

// Delete removes a pipeline scoped to a user.
func (r *PipelineRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeletePipeline(ctx, sqlc.DeletePipelineParams{
		ID:     toPgtypeUUID(id),
		UserID: toPgtypeUUID(userID),
	})
}

// ---------------------------------------------------------------------------
// Stage operations
// ---------------------------------------------------------------------------

// CreateStage inserts a new stage for a pipeline.
func (r *PipelineRepository) CreateStage(ctx context.Context, pipelineID uuid.UUID, section string, order int) (*domain.Stage, error) {
	row, err := r.q.CreateStage(ctx, sqlc.CreateStageParams{
		PipelineID: toPgtypeUUID(pipelineID),
		Section:    section,
		OrderIndex: int32(order),
	})
	if err != nil {
		return nil, fmt.Errorf("create stage: %w", err)
	}
	s := toDomainStage(row)
	return &s, nil
}

// DeleteStages removes all stages (and their blocks via CASCADE) for a pipeline.
func (r *PipelineRepository) DeleteStages(ctx context.Context, pipelineID uuid.UUID) error {
	return r.q.DeleteStagesByPipeline(ctx, toPgtypeUUID(pipelineID))
}

// ---------------------------------------------------------------------------
// Job operations
// ---------------------------------------------------------------------------

// CreateJob creates a new pipeline execution job.
func (r *PipelineRepository) CreateJob(ctx context.Context, pipelineID uuid.UUID, symbol string) (*domain.PipelineJob, error) {
	row, err := r.q.CreatePipelineJob(ctx, sqlc.CreatePipelineJobParams{
		PipelineID: toPgtypeUUID(pipelineID),
		Symbol:     symbol,
	})
	if err != nil {
		return nil, fmt.Errorf("create pipeline job: %w", err)
	}
	j := toDomainJob(row)
	return &j, nil
}

// GetJob fetches a pipeline job by ID.
func (r *PipelineRepository) GetJob(ctx context.Context, id uuid.UUID) (*domain.PipelineJob, error) {
	row, err := r.q.GetPipelineJob(ctx, toPgtypeUUID(id))
	if err != nil {
		return nil, fmt.Errorf("get pipeline job: %w", err)
	}
	j := toDomainJob(row)
	return &j, nil
}

// UpdateJobStatus updates a job's status, result, and error.
func (r *PipelineRepository) UpdateJobStatus(ctx context.Context, id uuid.UUID, status string, result interface{}, errMsg *string) error {
	var resultBytes []byte
	if result != nil {
		var err error
		resultBytes, err = json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshal job result: %w", err)
		}
	}

	now := time.Now()
	params := sqlc.UpdatePipelineJobStatusParams{
		ID:     toPgtypeUUID(id),
		Status: status,
		Result: resultBytes,
		Error:  toPgtypeText(errMsg),
	}

	switch status {
	case domain.JobStatusRunning:
		params.StartedAt = toPgtypeTimestamptz(now)
	case domain.JobStatusCompleted, domain.JobStatusFailed:
		params.CompletedAt = toPgtypeTimestamptz(now)
	}

	_, err := r.q.UpdatePipelineJobStatus(ctx, params)
	return err
}

// ---------------------------------------------------------------------------
// Monitor operations
// ---------------------------------------------------------------------------

// CreateMonitor creates a monitor_block row linking a block to a pipeline.
func (r *PipelineRepository) CreateMonitor(ctx context.Context, pipelineID, blockID uuid.UUID, cron string, enabled bool) (*domain.MonitorBlock, error) {
	row, err := r.q.CreateMonitorBlock(ctx, sqlc.CreateMonitorBlockParams{
		PipelineID: toPgtypeUUID(pipelineID),
		BlockID:    toPgtypeUUID(blockID),
		Cron:       cron,
		Enabled:    enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("create monitor: %w", err)
	}
	m := toDomainMonitor(row)
	return &m, nil
}

// ListMonitors lists all monitor_blocks for a pipeline.
func (r *PipelineRepository) ListMonitors(ctx context.Context, pipelineID uuid.UUID) ([]domain.MonitorBlock, error) {
	rows, err := r.q.ListMonitorsByPipeline(ctx, toPgtypeUUID(pipelineID))
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	monitors := make([]domain.MonitorBlock, len(rows))
	for i, row := range rows {
		monitors[i] = toDomainMonitor(row)
	}
	return monitors, nil
}

// DeleteMonitor deletes a single monitor_block.
func (r *PipelineRepository) DeleteMonitor(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteMonitorBlock(ctx, toPgtypeUUID(id))
}

// ---------------------------------------------------------------------------
// Price alert operations
// ---------------------------------------------------------------------------

// CreatePriceAlert inserts a price alert linked to a pipeline.
func (r *PipelineRepository) CreatePriceAlert(ctx context.Context, pipelineID uuid.UUID, condition, label string) (*domain.PriceAlert, error) {
	row, err := r.q.CreatePriceAlert(ctx, sqlc.CreatePriceAlertParams{
		PipelineID: toPgtypeUUID(pipelineID),
		Condition:  condition,
		Label:      label,
	})
	if err != nil {
		return nil, fmt.Errorf("create price alert: %w", err)
	}
	a := toDomainPriceAlert(row)
	return &a, nil
}

// ListPriceAlerts returns all price alerts for a pipeline.
func (r *PipelineRepository) ListPriceAlerts(ctx context.Context, pipelineID uuid.UUID) ([]domain.PriceAlert, error) {
	rows, err := r.q.ListPriceAlertsByPipeline(ctx, toPgtypeUUID(pipelineID))
	if err != nil {
		return nil, fmt.Errorf("list price alerts: %w", err)
	}
	alerts := make([]domain.PriceAlert, len(rows))
	for i, row := range rows {
		alerts[i] = toDomainPriceAlert(row)
	}
	return alerts, nil
}

// DeletePriceAlert deletes a single price alert.
func (r *PipelineRepository) DeletePriceAlert(ctx context.Context, id uuid.UUID) error {
	return r.q.DeletePriceAlert(ctx, toPgtypeUUID(id))
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// loadStagesWithBlocks loads all stages for a pipeline, then populates each
// stage's Blocks slice.
func (r *PipelineRepository) loadStagesWithBlocks(ctx context.Context, pipelineID uuid.UUID) ([]domain.Stage, error) {
	stageRows, err := r.q.ListStagesByPipeline(ctx, toPgtypeUUID(pipelineID))
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	stages := make([]domain.Stage, len(stageRows))
	for i, sr := range stageRows {
		stages[i] = toDomainStage(sr)

		blockRows, err := r.q.ListBlocksByStage(ctx, sr.ID)
		if err != nil {
			return nil, fmt.Errorf("list blocks for stage %s: %w", toUUID(sr.ID), err)
		}
		blocks := make([]domain.AgentBlock, len(blockRows))
		for j, br := range blockRows {
			blocks[j] = toDomainBlock(br)
		}
		stages[i].Blocks = blocks
	}
	return stages, nil
}

// loadMonitorsWithBlocks loads all monitors for a pipeline, then attaches each
// monitor's associated block.
func (r *PipelineRepository) loadMonitorsWithBlocks(ctx context.Context, pipelineID uuid.UUID) ([]domain.MonitorBlock, error) {
	monitorRows, err := r.q.ListMonitorsByPipeline(ctx, toPgtypeUUID(pipelineID))
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}

	monitors := make([]domain.MonitorBlock, len(monitorRows))
	for i, mr := range monitorRows {
		monitors[i] = toDomainMonitor(mr)

		block, err := r.q.GetBlockByID(ctx, mr.BlockID)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("get monitor block: %w", err)
		}
		if err == nil {
			b := toDomainBlock(block)
			monitors[i].Block = &b
		}
	}
	return monitors, nil
}

// Queries exposes the underlying sqlc.Queries for advanced use.
func (r *PipelineRepository) Queries() *sqlc.Queries {
	return r.q
}
