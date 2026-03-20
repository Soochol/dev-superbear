package handler

// CreateCaseRequest is the JSON body for POST /cases.
type CreateCaseRequest struct {
	PipelineID    string                 `json:"pipeline_id" binding:"required,uuid"`
	Symbol        string                 `json:"symbol" binding:"required,min=1,max=10"`
	EventDate     string                 `json:"event_date" binding:"required"`
	EventSnapshot map[string]interface{} `json:"event_snapshot" binding:"required"`
	SuccessScript string                 `json:"success_script" binding:"required,min=1"`
	FailureScript string                 `json:"failure_script" binding:"required,min=1"`
}

// CreatePipelineRequest is the JSON body for POST /pipelines.
type CreatePipelineRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=200"`
	Description   string  `json:"description"`
	SuccessScript *string `json:"success_script"`
	FailureScript *string `json:"failure_script"`
	IsPublic      bool    `json:"is_public"`
}

// UpdatePipelineRequest is the JSON body for PUT /pipelines/:id.
type UpdatePipelineRequest struct {
	Name          string  `json:"name" binding:"required,min=1,max=200"`
	Description   string  `json:"description"`
	SuccessScript *string `json:"success_script"`
	FailureScript *string `json:"failure_script"`
	IsPublic      bool    `json:"is_public"`
}

// CreateBlockRequest is the JSON body for POST /blocks.
type CreateBlockRequest struct {
	Name         string      `json:"name" binding:"required,min=1,max=200"`
	Instruction  string      `json:"instruction" binding:"required,min=1"`
	SystemPrompt *string     `json:"system_prompt"`
	AllowedTools []string    `json:"allowed_tools"`
	OutputSchema interface{} `json:"output_schema"`
	IsPublic     bool        `json:"is_public"`
}

// CreateTradeRequest is the JSON body for POST /cases/:id/trades.
type CreateTradeRequest struct {
	Type     string  `json:"type" binding:"required,oneof=BUY SELL"`
	Price    float64 `json:"price" binding:"required,gt=0"`
	Quantity int     `json:"quantity" binding:"required,gt=0"`
	Fee      float64 `json:"fee" binding:"min=0"`
	Date     string  `json:"date" binding:"required"`
	Note     *string `json:"note"`
}

// CreateAlertRequest is the JSON body for POST /alerts.
type CreateAlertRequest struct {
	Condition  string  `json:"condition" binding:"required,min=1"`
	Label      string  `json:"label" binding:"required,min=1"`
	PipelineID *string `json:"pipeline_id" binding:"omitempty,uuid"`
}

// ScanRequest is the JSON body for POST /search/scan.
type ScanRequest struct {
	Query string `json:"query" binding:"required,min=1"`
}
