"use client";

import { useEffect } from "react";
import { useChartStore } from "../model/chart.store";
import { chartApi } from "../api/chart-api";

export function useChartData() {
  const currentStock = useChartStore((s) => s.currentStock);
  const timeframe = useChartStore((s) => s.timeframe);

  useEffect(() => {
    if (!currentStock?.symbol) return;

    const { setCandles, setIsLoading } = useChartStore.getState();

    const fetchCandles = async () => {
      setIsLoading(true);
      try {
        const candles = await chartApi.fetchCandles(currentStock.symbol, timeframe);
        setCandles(candles);
      } finally {
        setIsLoading(false);
      }
    };

    fetchCandles();
  }, [currentStock?.symbol, timeframe]);
}
