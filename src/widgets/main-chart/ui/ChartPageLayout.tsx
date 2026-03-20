"use client";

import { MainChart } from "@/features/chart";
import { ChartTopbar } from "./ChartTopbar";
import { StockListSidebar } from "@/widgets/stock-list-sidebar";
import { SubIndicatorPanel } from "@/widgets/sub-indicator-panel";
import { BottomInfoPanel } from "@/widgets/bottom-info-panel";

export function ChartPageLayout() {
  return (
    <div className="flex flex-col h-full">
      <ChartTopbar />
      <div className="flex flex-1 min-h-0">
        <div className="flex-1 flex flex-col min-w-0">
          <div className="flex-1 min-h-0">
            <MainChart />
          </div>
          <SubIndicatorPanel />
        </div>
        <div className="w-72 border-l border-nexus-border flex-shrink-0">
          <StockListSidebar />
        </div>
      </div>
      <BottomInfoPanel />
    </div>
  );
}
