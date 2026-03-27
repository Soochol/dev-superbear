import { calculateMA } from "../lib/ma";
import { calculateRSI } from "../lib/rsi";
import { calculateMACD, type MACDResult } from "../lib/macd";
import { calculateBollingerBands } from "../lib/bollinger";

export type IndicatorType = "overlay" | "panel";
export type IndicatorCategory = "moving-average" | "oscillator" | "band";

export interface IndicatorConfig {
  id: string;
  name: string;
  category: IndicatorCategory;
  type: IndicatorType;
  color?: string;
  colors?: Record<string, string>;
}

export const INDICATOR_REGISTRY: IndicatorConfig[] = [
  // Moving Averages (overlay)
  { id: "ma5", name: "MA(5)", category: "moving-average", type: "overlay", color: "#f59e0b" },
  { id: "ma20", name: "MA(20)", category: "moving-average", type: "overlay", color: "#6366f1" },
  { id: "ma60", name: "MA(60)", category: "moving-average", type: "overlay", color: "#22c55e" },
  { id: "ma120", name: "MA(120)", category: "moving-average", type: "overlay", color: "#ef4444" },
  { id: "ma200", name: "MA(200)", category: "moving-average", type: "overlay", color: "#8b5cf6" },
  // Bollinger Bands (overlay)
  {
    id: "bb",
    name: "BB(20,2)",
    category: "band",
    type: "overlay",
    colors: { upper: "#7c3aed", middle: "#6366f1", lower: "#7c3aed" },
  },
  // RSI (panel)
  { id: "rsi", name: "RSI(14)", category: "oscillator", type: "panel", color: "#f59e0b" },
  // MACD (panel)
  {
    id: "macd",
    name: "MACD(12,26,9)",
    category: "oscillator",
    type: "panel",
    colors: { macd: "#6366f1", signal: "#f59e0b", histUp: "#22c55e", histDown: "#ef4444" },
  },
];

export function getIndicator(id: string): IndicatorConfig | undefined {
  return INDICATOR_REGISTRY.find((ind) => ind.id === id);
}

export function getOverlayIndicators(activeIds: string[]): IndicatorConfig[] {
  return activeIds
    .map(getIndicator)
    .filter((ind): ind is IndicatorConfig => ind != null && ind.type === "overlay");
}

export function getPanelIndicators(activeIds: string[]): IndicatorConfig[] {
  return activeIds
    .map(getIndicator)
    .filter((ind): ind is IndicatorConfig => ind != null && ind.type === "panel");
}

export { calculateMA, calculateRSI, calculateMACD, calculateBollingerBands };
export type { MACDResult };
