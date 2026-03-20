package service

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/dev-superbear/nexus-backend/internal/agent"
	"github.com/dev-superbear/nexus-backend/internal/domain"
)

type PipelineOrchestrator struct {
	agentRunner agent.Runner
}

func NewPipelineOrchestrator(runner agent.Runner) *PipelineOrchestrator {
	return &PipelineOrchestrator{agentRunner: runner}
}

func (o *PipelineOrchestrator) Execute(ctx context.Context, pipeline *domain.Pipeline, symbol string) (*domain.PipelineExecutionContext, error) {
	execCtx := &domain.PipelineExecutionContext{
		Symbol:          symbol,
		PreviousResults: make([]domain.PreviousResult, 0),
	}

	// Group stages by order_index
	stageGroups := groupStagesByOrder(pipeline.Stages)
	sortedOrders := sortedKeys(stageGroups)

	for _, order := range sortedOrders {
		stages := stageGroups[order]
		blocks := collectBlocks(stages)

		if len(blocks) == 0 {
			continue
		}

		// Same order blocks execute in parallel using goroutines
		type blockResult struct {
			index  int
			output *domain.AgentOutput
			err    error
		}

		results := make([]blockResult, len(blocks))
		var wg sync.WaitGroup

		for i, block := range blocks {
			wg.Add(1)
			go func(idx int, b domain.AgentBlock) {
				defer wg.Done()
				input := o.buildInput(&b, execCtx)
				output, err := o.agentRunner.Execute(ctx, &b, input)
				results[idx] = blockResult{index: idx, output: output, err: err}
			}(i, block)
		}

		wg.Wait()

		for i, r := range results {
			if r.err != nil {
				slog.Error("AgentBlock execution failed", "blockName", blocks[i].Name, "error", r.err)
				continue
			}
			if r.output != nil {
				execCtx.PreviousResults = append(execCtx.PreviousResults, domain.PreviousResult{
					BlockName: blocks[i].Name,
					Summary:   r.output.Summary,
					Data:      r.output.Data,
				})
			}
		}
	}

	return execCtx, nil
}

func (o *PipelineOrchestrator) buildInput(block *domain.AgentBlock, ctx *domain.PipelineExecutionContext) *domain.AgentInput {
	instruction := block.Instruction
	if instruction == "" {
		instruction = block.Objective
	}
	return &domain.AgentInput{
		Instruction: instruction,
		Context: domain.AgentInputContext{
			Symbol:          ctx.Symbol,
			PreviousResults: ctx.PreviousResults,
		},
	}
}

func groupStagesByOrder(stages []domain.Stage) map[int][]domain.Stage {
	groups := make(map[int][]domain.Stage)
	for _, s := range stages {
		groups[s.OrderIndex] = append(groups[s.OrderIndex], s)
	}
	return groups
}

func sortedKeys(m map[int][]domain.Stage) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func collectBlocks(stages []domain.Stage) []domain.AgentBlock {
	var blocks []domain.AgentBlock
	for _, s := range stages {
		blocks = append(blocks, s.Blocks...)
	}
	return blocks
}
