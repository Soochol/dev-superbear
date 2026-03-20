"use client";

import type { DragEvent } from "react";
import { BlockCard, type AgentBlock } from "@/entities/agent-block";
import { usePipelineStore } from "../../model/pipeline.store";
import { usePipelineDragDrop } from "../../lib/usePipelineDragDrop";

interface AnalysisSectionProps {
  onEditBlock?: (block: AgentBlock) => void;
}

export default function AnalysisSection({ onEditBlock }: AnalysisSectionProps) {
  const analysisStages = usePipelineStore((s) => s.analysisStages);
  const removeBlock = usePipelineStore((s) => s.removeBlock);
  const addStage = usePipelineStore((s) => s.addStage);
  const removeStage = usePipelineStore((s) => s.removeStage);
  const { handleDragOver, handleDropOnStage } = usePipelineDragDrop();

  return (
    <div className="border border-nexus-border rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold text-nexus-text-primary uppercase tracking-wider">
          Analysis Stages
        </h3>
        <button
          type="button"
          onClick={addStage}
          className="text-xs text-nexus-accent hover:text-nexus-accent/80 transition-colors"
        >
          + Add Stage
        </button>
      </div>

      <div className="space-y-3">
        {analysisStages.map((stage, stageIdx) => (
          <div key={stage.order}>
            {stageIdx > 0 && (
              <div className="flex justify-center py-1">
                <svg
                  width="16"
                  height="24"
                  viewBox="0 0 16 24"
                  fill="none"
                  className="text-nexus-text-muted"
                >
                  <path
                    d="M8 0v20M3 16l5 5 5-5"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                  />
                </svg>
              </div>
            )}

            <div
              onDragOver={handleDragOver}
              onDrop={(e: DragEvent<HTMLDivElement>) =>
                handleDropOnStage(e, stage.order)
              }
              className="relative bg-nexus-bg/50 border border-dashed border-nexus-border rounded-md p-3 min-h-[72px] transition-colors hover:border-nexus-accent/30"
            >
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] font-mono text-nexus-text-muted uppercase">
                  Stage {stage.order}
                  {stage.blocks.length > 1 && " (parallel)"}
                </span>
                {analysisStages.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeStage(stage.order)}
                    className="text-[10px] text-nexus-text-muted hover:text-nexus-failure transition-colors"
                  >
                    Remove
                  </button>
                )}
              </div>

              {stage.blocks.length === 0 ? (
                <p className="text-xs text-nexus-text-muted text-center py-3">
                  Drop agent blocks here
                </p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {stage.blocks.map((block) => (
                    <BlockCard
                      key={block.id}
                      block={block}
                      onEdit={onEditBlock}
                      onDelete={removeBlock}
                    />
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
