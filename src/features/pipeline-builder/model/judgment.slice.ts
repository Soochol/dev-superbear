import type { StateCreator } from "zustand";
import type { AnalysisSlice } from "./analysis.slice";
import type { MonitorSlice } from "./monitor.slice";
import type { MetaSlice } from "./pipeline.store";

export interface PriceAlertState {
  id: string;
  condition: string;
  label: string;
}

export interface JudgmentSlice {
  successScript: string;
  failureScript: string;
  priceAlerts: PriceAlertState[];
  setSuccessScript: (script: string) => void;
  setFailureScript: (script: string) => void;
  addPriceAlert: (condition: string, label: string) => void;
  removePriceAlert: (id: string) => void;
  setJudgment: (
    success: string,
    failure: string,
    alerts: PriceAlertState[],
  ) => void;
}

type PipelineStore = AnalysisSlice & MonitorSlice & JudgmentSlice & MetaSlice;

export const createJudgmentSlice: StateCreator<
  PipelineStore,
  [],
  [],
  JudgmentSlice
> = (set) => ({
  successScript: "",
  failureScript: "",
  priceAlerts: [],

  setSuccessScript: (script) => set({ successScript: script }),

  setFailureScript: (script) => set({ failureScript: script }),

  addPriceAlert: (condition, label) =>
    set((state) => ({
      priceAlerts: [
        ...state.priceAlerts,
        { id: crypto.randomUUID(), condition, label },
      ],
    })),

  removePriceAlert: (id) =>
    set((state) => ({
      priceAlerts: state.priceAlerts.filter((a) => a.id !== id),
    })),

  setJudgment: (success, failure, alerts) =>
    set({
      successScript: success,
      failureScript: failure,
      priceAlerts: alerts,
    }),
});
