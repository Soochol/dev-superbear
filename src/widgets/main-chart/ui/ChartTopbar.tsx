"use client";

import { useChartStore } from "@/features/chart";
import { useSearchModalStore } from "@/widgets/stock-search-modal";
import type { Timeframe } from "@/features/chart";
import { IndicatorSelector } from "./IndicatorSelector";

const TIMEFRAME_GROUPS: { label: string; items: { tf: Timeframe; display: string }[] }[] = [
  {
    label: "min",
    items: [
      { tf: "1m", display: "1" },
      { tf: "5m", display: "5" },
      { tf: "15m", display: "15" },
      { tf: "30m", display: "30" },
    ],
  },
  {
    label: "hour",
    items: [
      { tf: "1H", display: "1H" },
      { tf: "4H", display: "4H" },
    ],
  },
  {
    label: "day",
    items: [
      { tf: "1D", display: "D" },
      { tf: "1W", display: "W" },
      { tf: "1M", display: "M" },
    ],
  },
];

export function ChartTopbar() {
  const { currentStock, timeframe, setTimeframe } = useChartStore();
  const openModal = useSearchModalStore((s) => s.openModal);

  return (
    <div className="flex items-center justify-between px-4 py-2 bg-nexus-surface border-b border-nexus-border">
      <button
        onClick={openModal}
        data-testid="stock-search-trigger"
        className="flex items-center gap-4 hover:bg-nexus-border/30 rounded-lg px-3 py-1 transition-colors"
      >
        {currentStock ? (
          <>
            <span className="font-mono text-nexus-text-secondary text-sm">{currentStock.symbol}</span>
            <span className="font-semibold">{currentStock.name}</span>
            <span className="font-mono text-lg">{currentStock.price.toLocaleString()}</span>
            <span
              className={`font-mono text-sm ${
                currentStock.changePct >= 0 ? "text-nexus-success" : "text-nexus-failure"
              }`}
            >
              {currentStock.changePct >= 0 ? "+" : ""}
              {currentStock.changePct.toFixed(2)}%
            </span>
          </>
        ) : (
          <span className="text-nexus-text-muted flex items-center gap-2">
            <span>🔍</span> 종목을 검색하세요
          </span>
        )}
      </button>
      <div className="flex items-center">
        {TIMEFRAME_GROUPS.map((group, gi) => (
          <div key={group.label} className="flex items-center">
            {gi > 0 && <div className="w-px h-4 bg-nexus-border mx-1.5" />}
            <div className="flex gap-0.5">
              {group.items.map(({ tf, display }) => (
                <button
                  key={tf}
                  data-testid={`tf-${tf}`}
                  onClick={() => setTimeframe(tf)}
                  className={`px-2 py-1 text-xs font-medium rounded transition-colors ${
                    timeframe === tf
                      ? "bg-nexus-accent text-white"
                      : "text-nexus-text-secondary hover:text-nexus-text-primary"
                  }`}
                >
                  {display}
                </button>
              ))}
            </div>
          </div>
        ))}
        <div className="w-px h-4 bg-nexus-border mx-2" />
        <IndicatorSelector />
      </div>
    </div>
  );
}
