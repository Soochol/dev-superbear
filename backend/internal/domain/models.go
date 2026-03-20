package domain

import (
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
)

type CaseStatus string

const (
	CaseStatusLive          CaseStatus = "LIVE"
	CaseStatusClosedSuccess CaseStatus = "CLOSED_SUCCESS"
	CaseStatusClosedFailure CaseStatus = "CLOSED_FAILURE"
	CaseStatusBacktest      CaseStatus = "BACKTEST"
)

type TimelineEventType string

const (
	TimelineEventTypeNews           TimelineEventType = "NEWS"
	TimelineEventTypeDisclosure     TimelineEventType = "DISCLOSURE"
	TimelineEventTypeSector         TimelineEventType = "SECTOR"
	TimelineEventTypePriceAlert     TimelineEventType = "PRICE_ALERT"
	TimelineEventTypeTrade          TimelineEventType = "TRADE"
	TimelineEventTypePipelineResult TimelineEventType = "PIPELINE_RESULT"
)

type TradeType string

const (
	TradeTypeBuy  TradeType = "BUY"
	TradeTypeSell TradeType = "SELL"
)

type User struct {
	ID        pgtype.UUID        `json:"id"`
	Email     string             `json:"email"`
	Name      *string            `json:"name"`
	Image     *string            `json:"image"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

type Case struct {
	ID            pgtype.UUID        `json:"id"`
	UserID        pgtype.UUID        `json:"user_id"`
	PipelineID    pgtype.UUID        `json:"pipeline_id"`
	Symbol        string             `json:"symbol"`
	Status        CaseStatus         `json:"status"`
	EventDate     pgtype.Date        `json:"event_date"`
	EventSnapshot json.RawMessage    `json:"event_snapshot"`
	SuccessScript string             `json:"success_script"`
	FailureScript string             `json:"failure_script"`
	ClosedAt      pgtype.Date        `json:"closed_at,omitempty"`
	ClosedReason  *string            `json:"closed_reason,omitempty"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
	UpdatedAt     pgtype.Timestamptz `json:"updated_at"`
}

type TimelineEvent struct {
	ID         pgtype.UUID        `json:"id"`
	CaseID     pgtype.UUID        `json:"case_id"`
	Date       pgtype.Date        `json:"date"`
	Type       TimelineEventType  `json:"type"`
	Title      string             `json:"title"`
	Content    string             `json:"content"`
	AIAnalysis *string            `json:"ai_analysis,omitempty"`
	Data       json.RawMessage    `json:"data,omitempty"`
	CreatedAt  pgtype.Timestamptz `json:"created_at"`
}

type Trade struct {
	ID        pgtype.UUID        `json:"id"`
	CaseID    pgtype.UUID        `json:"case_id"`
	UserID    pgtype.UUID        `json:"user_id"`
	Type      TradeType          `json:"type"`
	Price     float64            `json:"price"`
	Quantity  int32              `json:"quantity"`
	Fee       float64            `json:"fee"`
	Date      pgtype.Date        `json:"date"`
	Note      *string            `json:"note,omitempty"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

type Pipeline struct {
	ID             pgtype.UUID        `json:"id"`
	UserID         pgtype.UUID        `json:"user_id"`
	Name           string             `json:"name"`
	Description    string             `json:"description"`
	AnalysisStages json.RawMessage    `json:"analysis_stages"`
	Monitors       json.RawMessage    `json:"monitors"`
	SuccessScript  string             `json:"success_script"`
	FailureScript  string             `json:"failure_script"`
	IsPublic       bool               `json:"is_public"`
	CreatedAt      pgtype.Timestamptz `json:"created_at"`
	UpdatedAt      pgtype.Timestamptz `json:"updated_at"`
}

type AgentBlock struct {
	ID           pgtype.UUID        `json:"id"`
	UserID       pgtype.UUID        `json:"user_id"`
	Name         string             `json:"name"`
	Instruction  string             `json:"instruction"`
	SystemPrompt *string            `json:"system_prompt,omitempty"`
	AllowedTools json.RawMessage    `json:"allowed_tools,omitempty"`
	OutputSchema json.RawMessage    `json:"output_schema,omitempty"`
	IsPublic     bool               `json:"is_public"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
}

type PriceAlert struct {
	ID          pgtype.UUID        `json:"id"`
	CaseID      pgtype.UUID        `json:"case_id"`
	PipelineID  pgtype.UUID        `json:"pipeline_id,omitempty"`
	Condition   string             `json:"condition"`
	Label       string             `json:"label"`
	Triggered   bool               `json:"triggered"`
	TriggeredAt pgtype.Date        `json:"triggered_at,omitempty"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
}

type EventSnapshot struct {
	High       float64         `json:"high"`
	Low        float64         `json:"low"`
	Close      float64         `json:"close"`
	Volume     float64         `json:"volume"`
	TradeValue float64         `json:"trade_value"`
	PreMA      map[int]float64 `json:"pre_ma"`
}

type AnalysisStage struct {
	Order    int      `json:"order"`
	BlockIDs []string `json:"blockIds"`
}

type MonitorConfig struct {
	BlockID string `json:"blockId"`
	Cron    string `json:"cron"`
	Enabled bool   `json:"enabled"`
}
