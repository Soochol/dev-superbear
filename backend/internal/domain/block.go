package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AgentBlock struct {
	ID           uuid.UUID        `json:"id"`
	UserID       uuid.UUID        `json:"userId"`
	StageID      *uuid.UUID       `json:"stageId,omitempty"`
	Name         string           `json:"name"`
	Objective    string           `json:"objective"`
	InputDesc    string           `json:"inputDesc"`
	Tools        []string         `json:"tools"`
	OutputFormat string           `json:"outputFormat"`
	Constraints  *string          `json:"constraints,omitempty"`
	Examples     *string          `json:"examples,omitempty"`
	Instruction  string           `json:"instruction"`
	SystemPrompt *string          `json:"systemPrompt,omitempty"`
	AllowedTools []string         `json:"allowedTools"`
	OutputSchema *json.RawMessage `json:"outputSchema,omitempty"`
	IsPublic     bool             `json:"isPublic"`
	IsTemplate   bool             `json:"isTemplate"`
	TemplateID   *uuid.UUID       `json:"templateId,omitempty"`
	CreatedAt    time.Time        `json:"createdAt"`
	UpdatedAt    time.Time        `json:"updatedAt"`

	// Relations
	MonitorBlock *MonitorBlock `json:"monitorBlock,omitempty"`
}

type MonitorBlock struct {
	ID         uuid.UUID `json:"id"`
	PipelineID uuid.UUID `json:"pipelineId"`
	BlockID    uuid.UUID `json:"blockId"`
	Cron       string    `json:"cron"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	// Relations
	Block *AgentBlock `json:"block,omitempty"`
}

type PriceAlert struct {
	ID          uuid.UUID  `json:"id"`
	PipelineID  *uuid.UUID `json:"pipelineId,omitempty"`
	CaseID      *uuid.UUID `json:"caseId,omitempty"`
	Condition   string     `json:"condition"`
	Label       string     `json:"label"`
	Triggered   bool       `json:"triggered"`
	TriggeredAt *time.Time `json:"triggeredAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// AgentBlockTemplate is a read-only view for palette display.
type AgentBlockTemplate struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Objective    string    `json:"objective"`
	InputDesc    string    `json:"inputDesc"`
	Tools        []string  `json:"tools"`
	OutputFormat string    `json:"outputFormat"`
	Constraints  *string   `json:"constraints,omitempty"`
	Examples     *string   `json:"examples,omitempty"`
	IsPublic     bool      `json:"isPublic"`
}
