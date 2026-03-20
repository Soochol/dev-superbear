"use client";

import { useEffect, useRef } from "react";
import { createChart, LineSeries, type IChartApi } from "lightweight-charts";
import { useChartStore } from "@/features/chart";
import { calculateRSI } from "@/entities/indicator";

export function RSIChart() {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const { candles } = useChartStore();

  useEffect(() => {
    if (!containerRef.current) return;

    const chart = createChart(containerRef.current, {
      layout: { background: { color: "#0a0a0f" }, textColor: "#94a3b8" },
      grid: { vertLines: { color: "#1e1e2e" }, horzLines: { color: "#1e1e2e" } },
      rightPriceScale: { borderColor: "#1e1e2e" },
      timeScale: { visible: false },
      height: 96,
    });

    chartRef.current = chart;

    const resizeObserver = new ResizeObserver((entries) => {
      chart.applyOptions({ width: entries[0].contentRect.width });
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      chart.remove();
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current || candles.length === 0) return;

    const closes = candles.map((c) => c.close);
    const rsiValues = calculateRSI(closes);

    const rsiData = candles
      .map((c, i) => {
        const val = rsiValues[i];
        if (val === null) return null;
        return { time: c.time, value: val };
      })
      .filter((d): d is { time: string; value: number } => d !== null);

    const series = chartRef.current.addSeries(LineSeries, {
      color: "#8b5cf6",
      lineWidth: 1,
      priceLineVisible: false,
    });
    series.setData(rsiData);
  }, [candles]);

  return <div ref={containerRef} className="w-full h-full" />;
}
