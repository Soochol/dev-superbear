"use client";

import { useChartStore } from "@/features/chart";
import { FinancialsPanel } from "./FinancialsPanel";
import { AIFusionPanel } from "./AIFusionPanel";
import { SectorComparePanel } from "./SectorComparePanel";

export function BottomInfoPanel() {
  const { currentStock } = useChartStore();

  if (!currentStock) {
    return (
      <div data-testid="bottom-panel-empty" className="h-48 border-t border-nexus-border bg-nexus-surface flex items-center justify-center">
        <span className="text-nexus-text-muted">Select a stock to view details</span>
      </div>
    );
  }

  return (
    <div className="border-t border-nexus-border bg-nexus-surface">
      <div data-testid="bottom-panel-grid" className="grid grid-cols-3 divide-x divide-nexus-border min-h-[200px]">
        <FinancialsPanel symbol={currentStock.symbol} />
        <AIFusionPanel symbol={currentStock.symbol} />
        <SectorComparePanel symbol={currentStock.symbol} />
      </div>
    </div>
  );
}
