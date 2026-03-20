"use client";

import { useState, useCallback } from "react";
import type { AgentBlock } from "@/entities/agent-block";
import type { StageState, MonitorBlockState, PriceAlertState } from "@/features/pipeline-builder";
import { usePipelineStore, PipelineTopbar, NodePalette, PipelineCanvas, useRegisterAndRun } from "@/features/pipeline-builder";
import { AgentBlockEditor } from "@/features/agent-block-editor";
import { AIGenerateModal } from "@/features/pipeline-generator";

export default function PipelinePage() {
  const pipelineName = usePipelineStore((s) => s.pipelineName);
  const setPipelineName = usePipelineStore((s) => s.setPipelineName);
  const selectedSymbol = usePipelineStore((s) => s.selectedSymbol);
  const setSelectedSymbol = usePipelineStore((s) => s.setSelectedSymbol);
  const isRunning = usePipelineStore((s) => s.isRunning);

  const setAnalysisStages = usePipelineStore((s) => s.setAnalysisStages);
  const setMonitorBlocks = usePipelineStore((s) => s.setMonitorBlocks);
  const setJudgment = usePipelineStore((s) => s.setJudgment);
  const setPipelineDescription = usePipelineStore(
    (s) => s.setPipelineDescription,
  );

  // Agent block editor state
  const [editingBlock, setEditingBlock] = useState<AgentBlock | null>(null);

  // AI generate modal state
  const [aiModalOpen, setAiModalOpen] = useState(false);

  // Error state
  const [runError, setRunError] = useState<string | null>(null);

  const { registerAndRun } = useRegisterAndRun();

  const handleEditBlock = useCallback((block: AgentBlock) => {
    setEditingBlock(block);
  }, []);

  const handleSaveBlock = useCallback(
    (updated: AgentBlock) => {
      // Update block in analysis stages
      const state = usePipelineStore.getState();
      const updatedStages = state.analysisStages.map((stage) => ({
        ...stage,
        blocks: stage.blocks.map((b) => (b.id === updated.id ? updated : b)),
      }));
      setAnalysisStages(updatedStages);

      // Update block in monitor blocks
      const updatedMonitors = state.monitorBlocks.map((m) =>
        m.block.id === updated.id ? { ...m, block: updated } : m,
      );
      setMonitorBlocks(updatedMonitors);
    },
    [setAnalysisStages, setMonitorBlocks],
  );

  const handleApplyGenerated = useCallback(
    (data: {
      name: string;
      description: string;
      stages: StageState[];
      monitors: MonitorBlockState[];
      successScript: string;
      failureScript: string;
      priceAlerts: PriceAlertState[];
    }) => {
      setPipelineName(data.name);
      setPipelineDescription(data.description);
      setAnalysisStages(data.stages);
      setMonitorBlocks(data.monitors);
      setJudgment(data.successScript, data.failureScript, data.priceAlerts);
    },
    [
      setPipelineName,
      setPipelineDescription,
      setAnalysisStages,
      setMonitorBlocks,
      setJudgment,
    ],
  );

  const handleRegisterAndRun = useCallback(async () => {
    setRunError(null);
    try {
      await registerAndRun();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Register & Run failed";
      setRunError(msg);
    }
  }, [registerAndRun]);

  return (
    <>
      <PipelineTopbar
        pipelineName={pipelineName}
        onPipelineNameChange={setPipelineName}
        selectedSymbol={selectedSymbol}
        onSymbolChange={setSelectedSymbol}
        onOpenAIGenerate={() => setAiModalOpen(true)}
        onRegisterAndRun={handleRegisterAndRun}
        isRunning={isRunning}
      />

      {runError && (
        <div className="mx-4 mt-2 px-4 py-2 text-sm text-nexus-failure bg-nexus-failure/10 border border-nexus-failure/20 rounded-md flex items-center justify-between">
          <span>{runError}</span>
          <button type="button" onClick={() => setRunError(null)} className="text-nexus-failure hover:text-nexus-failure/80 ml-4">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
          </button>
        </div>
      )}

      <div className="flex flex-1 overflow-hidden">
        <NodePalette />
        <PipelineCanvas onEditBlock={handleEditBlock} />
      </div>

      {/* Modals */}
      <AgentBlockEditor
        block={editingBlock}
        open={editingBlock !== null}
        onClose={() => setEditingBlock(null)}
        onSave={handleSaveBlock}
      />

      <AIGenerateModal
        open={aiModalOpen}
        onClose={() => setAiModalOpen(false)}
        onApply={handleApplyGenerated}
      />
    </>
  );
}
