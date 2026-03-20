import { create } from "zustand";
import type { Timeframe, BottomPanelTab } from "./types";
import type { StockInfo } from "@/entities/stock";
import type { CandleData } from "@/entities/candle";

interface ChartState {
  currentStock: StockInfo | null;
  setCurrentStock: (stock: StockInfo | null) => void;
  timeframe: Timeframe;
  setTimeframe: (tf: Timeframe) => void;
  candles: CandleData[];
  setCandles: (candles: CandleData[]) => void;
  isLoading: boolean;
  setIsLoading: (v: boolean) => void;
  activeIndicators: string[];
  toggleIndicator: (id: string) => void;
  activeSubIndicators: string[];
  toggleSubIndicator: (id: string) => void;
  bottomPanelTab: BottomPanelTab;
  setBottomPanelTab: (tab: BottomPanelTab) => void;
}

export const useChartStore = create<ChartState>()((set) => ({
  currentStock: null,
  setCurrentStock: (stock) => set({ currentStock: stock }),
  timeframe: "1D",
  setTimeframe: (tf) => set({ timeframe: tf }),
  candles: [],
  setCandles: (candles) => set({ candles }),
  isLoading: false,
  setIsLoading: (v) => set({ isLoading: v }),
  activeIndicators: ["ma20", "ma60"],
  toggleIndicator: (id) =>
    set((state) => ({
      activeIndicators: state.activeIndicators.includes(id)
        ? state.activeIndicators.filter((i) => i !== id)
        : [...state.activeIndicators, id],
    })),
  activeSubIndicators: [],
  toggleSubIndicator: (id) =>
    set((state) => ({
      activeSubIndicators: state.activeSubIndicators.includes(id)
        ? state.activeSubIndicators.filter((i) => i !== id)
        : [...state.activeSubIndicators, id],
    })),
  bottomPanelTab: "financials",
  setBottomPanelTab: (tab) => set({ bottomPanelTab: tab }),
}));
