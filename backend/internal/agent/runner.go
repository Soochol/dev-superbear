package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dev-superbear/nexus-backend/internal/domain"
)

// Runner defines the interface for agent execution (mockable for tests).
type Runner interface {
	Execute(ctx context.Context, block *domain.AgentBlock, input *domain.AgentInput) (*domain.AgentOutput, error)
}

// ADKRunner implements Runner using Google ADK Go SDK (stub for now).
type ADKRunner struct{}

func NewADKRunner() *ADKRunner {
	return &ADKRunner{}
}

func (r *ADKRunner) Execute(ctx context.Context, block *domain.AgentBlock, input *domain.AgentInput) (*domain.AgentOutput, error) {
	slog.Info("AgentRunner: executing block", "blockId", block.ID, "blockName", block.Name)

	systemPrompt := buildSystemPrompt(block)
	userMessage := buildUserMessage(input)
	tools := block.Tools
	if len(tools) == 0 {
		tools = block.AllowedTools
	}

	_ = systemPrompt
	_ = userMessage
	_ = tools
	// TODO: Integrate with Google ADK Go SDK

	return &domain.AgentOutput{
		Summary: fmt.Sprintf("Result from %s", block.Name),
	}, nil
}

func buildSystemPrompt(block *domain.AgentBlock) string {
	prompt := fmt.Sprintf("You are an agent named '%s'.\n", block.Name)
	if block.Objective != "" {
		prompt += fmt.Sprintf("Objective: %s\n", block.Objective)
	}
	if block.OutputFormat != "" {
		prompt += fmt.Sprintf("Output Format: %s\n", block.OutputFormat)
	}
	if block.Constraints != nil {
		prompt += fmt.Sprintf("Constraints: %s\n", *block.Constraints)
	}
	if block.Examples != nil {
		prompt += fmt.Sprintf("Examples: %s\n", *block.Examples)
	}
	return prompt
}

func buildUserMessage(input *domain.AgentInput) string {
	msg := input.Instruction + "\n"
	msg += fmt.Sprintf("Symbol: %s (%s)\n", input.Context.Symbol, input.Context.SymbolName)
	msg += fmt.Sprintf("Event Date: %s\n", input.Context.EventDate)
	if len(input.Context.PreviousResults) > 0 {
		msg += "Previous Results:\n"
		for _, pr := range input.Context.PreviousResults {
			msg += fmt.Sprintf("- %s: %s\n", pr.BlockName, pr.Summary)
		}
	}
	return msg
}
