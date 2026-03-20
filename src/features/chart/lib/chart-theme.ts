export const CHART_THEME = {
  layout: { background: { color: "#0a0a0f" }, textColor: "#94a3b8" },
  grid: { vertLines: { color: "#1e1e2e" }, horzLines: { color: "#1e1e2e" } },
  rightPriceScale: { borderColor: "#1e1e2e" },
  timeScale: { borderColor: "#1e1e2e" },
} as const;

export function toLineData(
  candles: { time: string }[],
  values: (number | null)[],
): { time: string; value: number }[] {
  const result: { time: string; value: number }[] = [];
  for (let i = 0; i < candles.length; i++) {
    const val = values[i];
    if (val !== null) result.push({ time: candles[i].time, value: val });
  }
  return result;
}
