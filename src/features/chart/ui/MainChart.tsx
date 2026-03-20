"use client";

import { useEffect, useRef, useCallback } from "react";
import { createChart, CandlestickSeries, LineSeries, type IChartApi, type ISeriesApi } from "lightweight-charts";
import { useChartStore } from "../model/chart.store";
import { useChartData } from "../lib/use-chart-data";
import { calculateMA, calculateBollingerBands, getIndicator } from "@/entities/indicator";
import { CHART_THEME, toLineData } from "../lib/chart-theme";

export function MainChart() {
  const chartContainerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<"Candlestick"> | null>(null);
  const overlaySeriesRef = useRef<Map<string, ISeriesApi<"Line">[]>>(new Map());

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

  const updateOverlays = useCallback(() => {
    if (!chartRef.current || candles.length === 0) return;

    const closes = candles.map((c) => c.close);

    // Remove all existing overlay series
    overlaySeriesRef.current.forEach((seriesList) => {
      seriesList.forEach((s) => chartRef.current!.removeSeries(s));
    });
    overlaySeriesRef.current.clear();

    for (const id of activeIndicators) {
      const config = getIndicator(id);
      if (!config || config.type !== "overlay") continue;

      if (id.startsWith("ma")) {
        const period = parseInt(id.replace("ma", ""), 10);
        if (isNaN(period)) continue;
        const maValues = calculateMA(closes, period);
        const lineData = toLineData(candles, maValues);
        const series = chartRef.current!.addSeries(LineSeries, {
          color: config.color,
          lineWidth: 1,
          priceLineVisible: false,
        });
        series.setData(lineData);
        overlaySeriesRef.current.set(id, [series]);
      } else if (id === "bb") {
        const { upper, middle, lower } = calculateBollingerBands(closes, 20, 2);
        const colors = config.colors ?? { upper: "#7c3aed", middle: "#6366f1", lower: "#7c3aed" };
        const seriesList: ISeriesApi<"Line">[] = [];
        for (const [key, values] of Object.entries({ upper, middle, lower })) {
          const series = chartRef.current!.addSeries(LineSeries, {
            color: colors[key] ?? "#6366f1",
            lineWidth: 1,
            lineStyle: key === "middle" ? 0 : 2, // 0=solid, 2=dashed
            priceLineVisible: false,
          });
          series.setData(toLineData(candles, values));
          seriesList.push(series);
        }
        overlaySeriesRef.current.set(id, seriesList);
      }
    }
  }, [candles, activeIndicators]);

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
  }, [candles, activeIndicators, updateOverlays]);

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
