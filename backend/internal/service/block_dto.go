package service

// CreateBlockRequest is the JSON body for POST /blocks.
type CreateBlockRequest struct {
	AgentBlockRequest
	IsTemplate bool `json:"isTemplate"`
}

// UpdateBlockRequest is the JSON body for PUT /blocks/:id.
type UpdateBlockRequest struct {
	Name         string   `json:"name" binding:"required,min=1"`
	Objective    string   `json:"objective"`
	InputDesc    string   `json:"inputDesc"`
	Tools        []string `json:"tools"`
	OutputFormat string   `json:"outputFormat"`
	Constraints  *string  `json:"constraints"`
	Examples     *string  `json:"examples"`
	Instruction  string   `json:"instruction"`
	SystemPrompt *string  `json:"systemPrompt"`
	AllowedTools []string `json:"allowedTools"`
	IsPublic     bool     `json:"isPublic"`
}

// CopyFromTemplateRequest is the JSON body for POST /blocks/copy-template.
type CopyFromTemplateRequest struct {
	TemplateID string `json:"templateId" binding:"required,uuid"`
	StageID    string `json:"stageId" binding:"required,uuid"`
}
