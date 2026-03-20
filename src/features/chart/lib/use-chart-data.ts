"use client";

import { useEffect } from "react";
import { useChartStore } from "../model/chart.store";
import { chartApi } from "../api/chart-api";

export function useChartData() {
  const { currentStock, timeframe, setCandles, setIsLoading } = useChartStore();

  useEffect(() => {
    if (!currentStock?.symbol) return;

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
  }, [currentStock?.symbol, timeframe, setCandles, setIsLoading]);

  return { isLoading: useChartStore((s) => s.isLoading) };
}
