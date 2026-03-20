import type { StateCreator } from "zustand";
import type { AgentBlock } from "@/entities/agent-block";
import type { MonitorSlice } from "./monitor.slice";
import type { JudgmentSlice } from "./judgment.slice";
import type { MetaSlice } from "./pipeline.store";

export interface StageState {
  order: number;
  blocks: AgentBlock[];
}

export interface AnalysisSlice {
  analysisStages: StageState[];
  addBlockToStage: (order: number, block: AgentBlock) => void;
  removeBlock: (blockId: string) => void;
  addStage: () => void;
  removeStage: (order: number) => void;
  setAnalysisStages: (stages: StageState[]) => void;
}

type PipelineStore = AnalysisSlice & MonitorSlice & JudgmentSlice & MetaSlice;

export const createAnalysisSlice: StateCreator<
  PipelineStore,
  [],
  [],
  AnalysisSlice
> = (set) => ({
  analysisStages: [{ order: 0, blocks: [] }],

  addBlockToStage: (order, block) =>
    set((state) => {
      const stages = state.analysisStages.map((s) =>
        s.order === order ? { ...s, blocks: [...s.blocks, block] } : s,
      );
      // If no stage with this order exists, create one
      if (!stages.find((s) => s.order === order)) {
        stages.push({ order, blocks: [block] });
        stages.sort((a, b) => a.order - b.order);
      }
      return { analysisStages: stages };
    }),

  removeBlock: (blockId) =>
    set((state) => ({
      analysisStages: state.analysisStages.map((s) => ({
        ...s,
        blocks: s.blocks.filter((b) => b.id !== blockId),
      })),
    })),

  addStage: () =>
    set((state) => {
      const maxOrder = state.analysisStages.reduce(
        (max, s) => Math.max(max, s.order),
        -1,
      );
      return {
        analysisStages: [
          ...state.analysisStages,
          { order: maxOrder + 1, blocks: [] },
        ],
      };
    }),

  removeStage: (order) =>
    set((state) => ({
      analysisStages: state.analysisStages
        .filter((s) => s.order !== order)
        .map((s, idx) => ({ ...s, order: idx })),
    })),

  setAnalysisStages: (stages) => set({ analysisStages: stages }),
});
