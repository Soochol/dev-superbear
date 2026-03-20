package service

import (
	"context"
	"fmt"
	"log/slog"
)

type PipelineGenerator struct{}

func NewPipelineGenerator() *PipelineGenerator {
	return &PipelineGenerator{}
}

type GeneratedPipeline struct {
	Name          string              `json:"name"`
	Description   string              `json:"description"`
	Stages        []StageRequest      `json:"stages"`
	Monitors      []MonitorRequest    `json:"monitors"`
	SuccessScript *string             `json:"successScript"`
	FailureScript *string             `json:"failureScript"`
	PriceAlerts   []PriceAlertRequest `json:"priceAlerts"`
}

type GenerateRequest struct {
	Description string `json:"description" binding:"required,min=1"`
}

func (g *PipelineGenerator) Generate(ctx context.Context, description string) (*GeneratedPipeline, error) {
	slog.Info("PipelineGenerator: generating from description", "description", description)

	systemPrompt := buildGeneratorSystemPrompt()
	_ = systemPrompt
	// TODO: Integrate with Google ADK Go SDK for LLM call

	return nil, fmt.Errorf("AI pipeline generation is not yet available")
}

func buildGeneratorSystemPrompt() string {
	return `You are a pipeline structure generator. Given a natural language description of an
investment analysis workflow, produce a structured pipeline definition with stages, agent blocks,
monitors, and price alerts. Each agent block should have a clear objective, input description,
output format, and list of tools it can use.`
}
