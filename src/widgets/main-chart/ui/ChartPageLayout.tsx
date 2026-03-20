"use client";

import { MainChart } from "@/features/chart";
import { ChartTopbar } from "./ChartTopbar";
import { StockSearchModal } from "@/widgets/stock-search-modal";
import { BottomInfoPanel } from "@/widgets/bottom-info-panel";

export function ChartPageLayout() {
  return (
    <div className="flex flex-col h-full">
      <ChartTopbar />
      <div className="flex-1 min-h-0">
        <MainChart />
      </div>
      <BottomInfoPanel />
      <StockSearchModal />
    </div>
  );
}
