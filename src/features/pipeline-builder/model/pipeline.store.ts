import { create, type StateCreator } from "zustand";
import type { Pipeline } from "@/entities/pipeline/model/types";
import {
  createAnalysisSlice,
  type AnalysisSlice,
  type StageState,
} from "./analysis.slice";
import { createMonitorSlice, type MonitorSlice } from "./monitor.slice";
import { createJudgmentSlice, type JudgmentSlice } from "./judgment.slice";

export interface MetaSlice {
  pipelineId: string | null;
  pipelineName: string;
  pipelineDescription: string;
  selectedSymbol: string;
  isRunning: boolean;
  setPipelineId: (id: string | null) => void;
  setPipelineName: (name: string) => void;
  setPipelineDescription: (desc: string) => void;
  setSelectedSymbol: (symbol: string) => void;
  setIsRunning: (running: boolean) => void;
  resetStore: () => void;
  loadPipeline: (pipeline: Pipeline) => void;
}

export type PipelineStore = AnalysisSlice &
  MonitorSlice &
  JudgmentSlice &
  MetaSlice;

const createMetaSlice: StateCreator<PipelineStore, [], [], MetaSlice> = (
  set,
) => ({
  pipelineId: null,
  pipelineName: "",
  pipelineDescription: "",
  selectedSymbol: "",
  isRunning: false,

  setPipelineId: (id) => set({ pipelineId: id }),
  setPipelineName: (name) => set({ pipelineName: name }),
  setPipelineDescription: (desc) => set({ pipelineDescription: desc }),
  setSelectedSymbol: (symbol) => set({ selectedSymbol: symbol }),
  setIsRunning: (running) => set({ isRunning: running }),

  resetStore: () =>
    set({
      pipelineId: null,
      pipelineName: "",
      pipelineDescription: "",
      selectedSymbol: "",
      isRunning: false,
      analysisStages: [{ order: 0, blocks: [] }],
      monitorBlocks: [],
      successScript: "",
      failureScript: "",
      priceAlerts: [],
    }),

  loadPipeline: (pipeline) => {
    const analysisStages: StageState[] = (pipeline.stages ?? [])
      .filter((s) => s.section === "analysis")
      .sort((a, b) => a.order - b.order)
      .map((s) => ({
        order: s.order,
        blocks: s.blocks ?? [],
      }));

    if (analysisStages.length === 0) {
      analysisStages.push({ order: 0, blocks: [] });
    }

    const monitorBlocks = (pipeline.monitors ?? []).map((m) => ({
      id: m.id,
      block: m.block ?? ({
        id: m.blockId,
        userId: "",
        name: "Monitor Block",
        objective: "",
        inputDesc: "",
        tools: [],
        outputFormat: "",
        constraints: null,
        examples: null,
        instruction: "",
        systemPrompt: null,
        allowedTools: [],
        isPublic: false,
        isTemplate: false,
        createdAt: "",
        updatedAt: "",
      }),
      cron: m.cron,
      enabled: m.enabled,
    }));

    const priceAlerts = (pipeline.priceAlerts ?? []).map((a) => ({
      id: a.id,
      condition: a.condition,
      label: a.label,
    }));

    set({
      pipelineId: pipeline.id,
      pipelineName: pipeline.name,
      pipelineDescription: pipeline.description,
      analysisStages,
      monitorBlocks,
      successScript: pipeline.successScript ?? "",
      failureScript: pipeline.failureScript ?? "",
      priceAlerts,
    });
  },
});

export const usePipelineStore = create<PipelineStore>()((...args) => ({
  ...createAnalysisSlice(...args),
  ...createMonitorSlice(...args),
  ...createJudgmentSlice(...args),
  ...createMetaSlice(...args),
}));
