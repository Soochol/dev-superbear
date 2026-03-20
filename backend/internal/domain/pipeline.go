package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Pipeline struct {
	ID            uuid.UUID      `json:"id"`
	UserID        uuid.UUID      `json:"userId"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	SuccessScript *string        `json:"successScript"`
	FailureScript *string        `json:"failureScript"`
	IsPublic      bool           `json:"isPublic"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`

	// Relations (populated by repository)
	Stages      []Stage        `json:"stages,omitempty"`
	Monitors    []MonitorBlock `json:"monitors,omitempty"`
	PriceAlerts []PriceAlert   `json:"priceAlerts,omitempty"`
}

type Stage struct {
	ID         uuid.UUID `json:"id"`
	PipelineID uuid.UUID `json:"pipelineId"`
	Section    string    `json:"section"`
	OrderIndex int       `json:"order"`
	CreatedAt  time.Time `json:"createdAt"`

	// Relations
	Blocks []AgentBlock `json:"blocks,omitempty"`
}

type PipelineJob struct {
	ID          uuid.UUID        `json:"id"`
	PipelineID  uuid.UUID        `json:"pipelineId"`
	Symbol      string           `json:"symbol"`
	Status      string           `json:"status"`
	Result      *json.RawMessage `json:"result,omitempty"`
	Error       *string          `json:"error,omitempty"`
	StartedAt   *time.Time       `json:"startedAt,omitempty"`
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	CreatedAt   time.Time        `json:"createdAt"`
}

// PipelineJob status constants
const (
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
)

// Stage section constants
const (
	SectionAnalysis   = "analysis"
	SectionMonitoring = "monitoring"
	SectionJudgment   = "judgment"
)
