import type { StateCreator } from "zustand";
import type { AgentBlock } from "@/entities/agent-block";
import type { AnalysisSlice } from "./analysis.slice";
import type { JudgmentSlice } from "./judgment.slice";
import type { MetaSlice } from "./pipeline.store";

export interface MonitorBlockState {
  id: string;
  block: AgentBlock;
  cron: string;
  enabled: boolean;
}

export interface MonitorSlice {
  monitorBlocks: MonitorBlockState[];
  addMonitorBlock: (block: AgentBlock, cron: string) => void;
  removeMonitorBlock: (id: string) => void;
  updateMonitorCron: (id: string, cron: string) => void;
  toggleMonitorEnabled: (id: string) => void;
  setMonitorBlocks: (blocks: MonitorBlockState[]) => void;
}

type PipelineStore = AnalysisSlice & MonitorSlice & JudgmentSlice & MetaSlice;

export const createMonitorSlice: StateCreator<
  PipelineStore,
  [],
  [],
  MonitorSlice
> = (set) => ({
  monitorBlocks: [],

  addMonitorBlock: (block, cron) =>
    set((state) => ({
      monitorBlocks: [
        ...state.monitorBlocks,
        { id: crypto.randomUUID(), block, cron, enabled: true },
      ],
    })),

  removeMonitorBlock: (id) =>
    set((state) => ({
      monitorBlocks: state.monitorBlocks.filter((m) => m.id !== id),
    })),

  updateMonitorCron: (id, cron) =>
    set((state) => ({
      monitorBlocks: state.monitorBlocks.map((m) =>
        m.id === id ? { ...m, cron } : m,
      ),
    })),

  toggleMonitorEnabled: (id) =>
    set((state) => ({
      monitorBlocks: state.monitorBlocks.map((m) =>
        m.id === id ? { ...m, enabled: !m.enabled } : m,
      ),
    })),

  setMonitorBlocks: (blocks) => set({ monitorBlocks: blocks }),
});
