"use client";

import { MainChart, IndicatorPanel, useChartStore } from "@/features/chart";
import { getPanelIndicators } from "@/entities/indicator";
import { ChartTopbar } from "./ChartTopbar";
import { StockSearchModal } from "@/widgets/stock-search-modal";
import { BottomInfoPanel } from "@/widgets/bottom-info-panel";

export function ChartPageLayout() {
  const { activeIndicators, toggleIndicator } = useChartStore();
  const panelIndicators = getPanelIndicators(activeIndicators);

  return (
    <div className="flex flex-col h-full">
      <ChartTopbar />
      <div className="flex-1 min-h-0 flex flex-col">
        <div className="flex-1 min-h-0">
          <MainChart />
        </div>
        {panelIndicators.map((ind) => (
          <IndicatorPanel
            key={ind.id}
            indicatorId={ind.id}
            onRemove={toggleIndicator}
          />
        ))}
      </div>
      <BottomInfoPanel />
      <StockSearchModal />
    </div>
  );
}
