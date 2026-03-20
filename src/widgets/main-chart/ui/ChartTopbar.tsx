"use client";

import { useChartStore } from "@/features/chart";
import { useSearchModalStore } from "@/widgets/stock-search-modal";
import type { Timeframe } from "@/features/chart";

const TIMEFRAMES: Timeframe[] = ["1m", "5m", "15m", "1H", "1D", "1W", "1M"];

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
      <div className="flex gap-1">
        {TIMEFRAMES.map((tf) => (
          <button
            key={tf}
            onClick={() => setTimeframe(tf)}
            className={`px-2 py-1 text-xs font-medium rounded transition-colors ${
              timeframe === tf
                ? "bg-nexus-accent text-white"
                : "text-nexus-text-secondary hover:text-nexus-text-primary"
            }`}
          >
            {tf}
          </button>
        ))}
      </div>
    </div>
  );
}
