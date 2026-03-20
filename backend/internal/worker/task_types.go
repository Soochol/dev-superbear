package worker

import (
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

// ── Task type constants ─────────────────────────────────────────
const (
	TypeMonitorAgent     = "monitor:agent"
	TypeDSLPoller        = "monitor:dsl-poll"
	TypeMonitorLifecycle = "monitor:lifecycle"
)

// ── Payloads ────────────────────────────────────────────────────

type MonitorAgentPayload struct {
	CaseID           string   `json:"case_id"`
	MonitorBlockID   string   `json:"monitor_block_id"`
	PipelineID       string   `json:"pipeline_id"`
	Symbol           string   `json:"symbol"`
	BlockInstruction string   `json:"block_instruction"`
	AllowedTools     []string `json:"allowed_tools"`
}

type DSLPollingPayload struct {
	CaseID        string `json:"case_id"`
	Symbol        string `json:"symbol"`
	SuccessScript string `json:"success_script"`
	FailureScript string `json:"failure_script"`
	PriceAlerts   []struct {
		ID        string `json:"id"`
		Condition string `json:"condition"`
		Label     string `json:"label"`
	} `json:"price_alerts"`
	EventSnapshot map[string]interface{} `json:"event_snapshot"`
}

type LifecyclePayload struct {
	CaseID  string `json:"case_id"`
	Action  string `json:"action"` // "CLOSE_SUCCESS" | "CLOSE_FAILURE" | "TRIGGER_ALERT"
	Reason  string `json:"reason"`
	AlertID string `json:"alert_id,omitempty"`
}

// DSLContext built from a Case + live price snapshot
type DSLContext struct {
	Close         float64 `json:"close"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Volume        float64 `json:"volume"`
	EventHigh     float64 `json:"event_high"`
	EventLow      float64 `json:"event_low"`
	EventClose    float64 `json:"event_close"`
	EventVolume   float64 `json:"event_volume"`
	PreEventMA5   float64 `json:"pre_event_ma_5"`
	PreEventMA20  float64 `json:"pre_event_ma_20"`
	PreEventMA60  float64 `json:"pre_event_ma_60"`
	PreEventMA120 float64 `json:"pre_event_ma_120"`
	PreEventMA200 float64 `json:"pre_event_ma_200"`
	PreEventClose float64 `json:"pre_event_close"`
}

// MonitoringMetrics collected from queue inspection and DB counts.
type MonitoringMetrics struct {
	DSLPollingDurationMs     int64 `json:"dsl_polling_duration_ms"`
	AgentExecutionDurationMs int64 `json:"agent_execution_duration_ms"`
	ActiveCases              int   `json:"active_cases"`
	ActiveMonitorBlocks      int   `json:"active_monitor_blocks"`
	QueueDepth               struct {
		Agent     int `json:"agent"`
		DSL       int `json:"dsl"`
		Lifecycle int `json:"lifecycle"`
	} `json:"queue_depth"`
}

// ── Task constructors ───────────────────────────────────────────

func NewMonitorAgentTask(p MonitorAgentPayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal MonitorAgentPayload: %w", err)
	}
	return asynq.NewTask(TypeMonitorAgent, data, asynq.MaxRetry(3)), nil
}

func NewDSLPollerTask() *asynq.Task {
	return asynq.NewTask(TypeDSLPoller, nil, asynq.MaxRetry(2))
}

func NewLifecycleTask(p LifecyclePayload) (*asynq.Task, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal LifecyclePayload: %w", err)
	}
	return asynq.NewTask(TypeMonitorLifecycle, data), nil
}
