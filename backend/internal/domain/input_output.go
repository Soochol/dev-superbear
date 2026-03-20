package domain

type AgentInput struct {
	Instruction string            `json:"instruction"`
	Context     AgentInputContext `json:"context"`
}

type AgentInputContext struct {
	Symbol          string           `json:"symbol"`
	SymbolName      string           `json:"symbolName"`
	EventDate       string           `json:"eventDate"`
	EventSnapshot   *EventSnapshot   `json:"eventSnapshot,omitempty"`
	PreviousResults []PreviousResult `json:"previousResults"`
}

type AgentOutput struct {
	Summary    string                 `json:"summary"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Confidence *float64               `json:"confidence,omitempty"`
}

type PreviousResult struct {
	BlockName string                 `json:"blockName"`
	Summary   string                 `json:"summary"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

type BlockError struct {
	BlockName string `json:"blockName"`
	Error     string `json:"error"`
}

type PipelineExecutionContext struct {
	Symbol          string           `json:"symbol"`
	PreviousResults []PreviousResult `json:"previousResults"`
	Errors          []BlockError     `json:"errors,omitempty"`
}
