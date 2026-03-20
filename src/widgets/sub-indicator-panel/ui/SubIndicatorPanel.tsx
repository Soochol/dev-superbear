"use client";

import { useChartStore } from "@/features/chart";
import { RSIChart } from "./RSIChart";
import { MACDChart } from "./MACDChart";
import { RevenueChart } from "./RevenueChart";

const SUB_INDICATORS = [
  { id: "rsi", label: "RSI" },
  { id: "macd", label: "MACD" },
  { id: "revenue", label: "Revenue" },
];

export function SubIndicatorPanel() {
  const { activeSubIndicators, toggleSubIndicator } = useChartStore();

  return (
    <div className="border-t border-nexus-border">
      <div className="flex items-center gap-1 px-3 py-1 bg-nexus-surface border-b border-nexus-border">
        {SUB_INDICATORS.map((ind) => (
          <button
            key={ind.id}
            onClick={() => toggleSubIndicator(ind.id)}
            className={`px-2 py-0.5 text-xs rounded transition-colors ${
              activeSubIndicators.includes(ind.id)
                ? "bg-nexus-accent/20 text-nexus-accent"
                : "text-nexus-text-muted hover:text-nexus-text-secondary"
            }`}
          >
            [{ind.label}]
          </button>
        ))}
      </div>
      <div className="flex flex-col">
        {activeSubIndicators.includes("rsi") && (
          <div className="h-24 border-b border-nexus-border">
            <RSIChart />
          </div>
        )}
        {activeSubIndicators.includes("macd") && (
          <div className="h-24 border-b border-nexus-border">
            <MACDChart />
          </div>
        )}
        {activeSubIndicators.includes("revenue") && (
          <div className="h-32 border-b border-nexus-border">
            <RevenueChart />
          </div>
        )}
      </div>
    </div>
  );
}
