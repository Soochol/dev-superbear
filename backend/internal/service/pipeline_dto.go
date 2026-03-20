package service

// CreatePipelineRequest is the JSON body for POST /pipelines.
type CreatePipelineRequest struct {
	Name          string              `json:"name" binding:"required,min=1,max=200"`
	Description   string              `json:"description"`
	Stages        []StageRequest      `json:"stages"`
	Monitors      []MonitorRequest    `json:"monitors"`
	SuccessScript *string             `json:"successScript"`
	FailureScript *string             `json:"failureScript"`
	PriceAlerts   []PriceAlertRequest `json:"priceAlerts"`
	IsPublic      bool                `json:"isPublic"`
}

// UpdatePipelineRequest is the JSON body for PUT /pipelines/:id.
type UpdatePipelineRequest struct {
	Name          string              `json:"name" binding:"required,min=1,max=200"`
	Description   string              `json:"description"`
	Stages        []StageRequest      `json:"stages"`
	Monitors      []MonitorRequest    `json:"monitors"`
	SuccessScript *string             `json:"successScript"`
	FailureScript *string             `json:"failureScript"`
	PriceAlerts   []PriceAlertRequest `json:"priceAlerts"`
	IsPublic      bool                `json:"isPublic"`
}

// StageRequest represents a stage with its blocks in a create/update pipeline request.
type StageRequest struct {
	Section string              `json:"section" binding:"required"`
	Order   int                 `json:"order"`
	Blocks  []AgentBlockRequest `json:"blocks"`
}

// AgentBlockRequest represents an agent block within a stage or monitor.
type AgentBlockRequest struct {
	Name         string   `json:"name" binding:"required,min=1"`
	Objective    string   `json:"objective"`
	InputDesc    string   `json:"inputDesc"`
	Tools        []string `json:"tools"`
	OutputFormat string   `json:"outputFormat"`
	Constraints  *string  `json:"constraints"`
	Examples     *string  `json:"examples"`
	Instruction  string   `json:"instruction"`
}

// MonitorRequest represents a monitor with its block in a create/update pipeline request.
type MonitorRequest struct {
	Block   AgentBlockRequest `json:"block"`
	Cron    string            `json:"cron" binding:"required"`
	Enabled bool              `json:"enabled"`
}

// PriceAlertRequest represents a price alert in a create/update pipeline request.
type PriceAlertRequest struct {
	Condition string `json:"condition" binding:"required"`
	Label     string `json:"label" binding:"required"`
}

// ExecutePipelineRequest is the JSON body for POST /pipelines/:id/execute.
type ExecutePipelineRequest struct {
	Symbol string `json:"symbol" binding:"required,min=1"`
}
