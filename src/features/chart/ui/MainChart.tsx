"use client";

import { useEffect, useRef } from "react";
import { createChart, CandlestickSeries, LineSeries, type IChartApi, type ISeriesApi } from "lightweight-charts";
import { useChartStore } from "../model/chart.store";
import { useChartData } from "../lib/use-chart-data";
import { calculateMA } from "@/entities/indicator";
import { CHART_THEME, toLineData } from "../lib/chart-theme";

export function MainChart() {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<"Candlestick"> | null>(null);
  const overlaySeriesRef = useRef<Map<string, ISeriesApi<"Line">>>(new Map());

  const { candles, activeIndicators, isLoading } = useChartStore();
  useChartData();

  useEffect(() => {
    if (!chartContainerRef.current) return;

    const chart = createChart(chartContainerRef.current, {
      ...CHART_THEME,
      crosshair: { mode: 0 },
    });

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: "#22c55e",
      downColor: "#ef4444",
      borderDownColor: "#ef4444",
      borderUpColor: "#22c55e",
      wickDownColor: "#ef4444",
      wickUpColor: "#22c55e",
    });

    chartRef.current = chart;
    candleSeriesRef.current = candleSeries;

    const resizeObserver = new ResizeObserver((entries) => {
      const { width, height } = entries[0].contentRect;
      chart.applyOptions({ width, height });
    });
    resizeObserver.observe(chartContainerRef.current);

    return () => {
      resizeObserver.disconnect();
      chart.remove();
    };
  }, []);

  useEffect(() => {
    if (!candleSeriesRef.current || candles.length === 0) return;

    const candleData = candles.map((c) => ({
      time: c.time,
      open: c.open,
      high: c.high,
      low: c.low,
      close: c.close,
    }));

    candleSeriesRef.current.setData(candleData);
    updateOverlays();
  }, [candles, activeIndicators]);

  const updateOverlays = () => {
    if (!chartRef.current) return;

    const closes = candles.map((c) => c.close);
    const overlayConfigs: Record<string, { period: number; color: string }> = {
      ma5: { period: 5, color: "#f59e0b" },
      ma20: { period: 20, color: "#6366f1" },
      ma60: { period: 60, color: "#22c55e" },
      ma120: { period: 120, color: "#ef4444" },
      ma200: { period: 200, color: "#8b5cf6" },
    };

    overlaySeriesRef.current.forEach((series) => {
      chartRef.current!.removeSeries(series);
    });
    overlaySeriesRef.current.clear();

    for (const id of activeIndicators) {
      const config = overlayConfigs[id];
      if (!config) continue;

      const maValues = calculateMA(closes, config.period);
      const lineData = toLineData(candles, maValues);

      const series = chartRef.current!.addSeries(LineSeries, {
        color: config.color,
        lineWidth: 1,
        priceLineVisible: false,
      });
      series.setData(lineData);
      overlaySeriesRef.current.set(id, series);
    }
  };

  return (
    <div className="relative w-full h-full">
      {isLoading && (
        <div className="absolute inset-0 flex items-center justify-center bg-nexus-bg/50 z-10">
          <span className="text-nexus-text-muted">Loading chart data...</span>
        </div>
      )}
      <div ref={chartContainerRef} className="w-full h-full" />
    </div>
  );
}
