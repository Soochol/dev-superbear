"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";
import { ChartPageLayout } from "@/widgets/main-chart";

export default function ChartPage() {
  const searchParams = useSearchParams();
  const symbol = searchParams.get("symbol");
  const { selectedSymbol } = useStockListStore();
  const { setCurrentStock } = useChartStore();

  useEffect(() => {
    const targetSymbol = symbol ?? selectedSymbol;
    if (!targetSymbol) return;
    if (targetSymbol === useChartStore.getState().currentStock?.symbol) return;

    const searchResults = useStockListStore.getState().searchResults;
    const stockInfo = searchResults.find((r) => r.symbol === targetSymbol);

    setCurrentStock({
      symbol: targetSymbol,
      name: stockInfo?.name ?? targetSymbol,
      price: 0,
      change: 0,
      changePct: 0,
    });
  }, [symbol, selectedSymbol, setCurrentStock]);

  return <ChartPageLayout />;
}
