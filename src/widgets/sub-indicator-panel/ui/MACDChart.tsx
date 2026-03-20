"use client";

import { useEffect, useRef } from "react";
import { createChart, LineSeries, HistogramSeries, type IChartApi } from "lightweight-charts";
import { useChartStore } from "@/features/chart";
import { calculateMACD } from "@/entities/indicator";

export function MACDChart() {
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
    const { macd, signal, histogram } = calculateMACD(closes);

    const macdData = candles
      .map((c, i) => {
        const val = macd[i];
        if (val === null) return null;
        return { time: c.time, value: val };
      })
      .filter((d): d is { time: string; value: number } => d !== null);

    const signalData = candles
      .map((c, i) => {
        const val = signal[i];
        if (val === null) return null;
        return { time: c.time, value: val };
      })
      .filter((d): d is { time: string; value: number } => d !== null);

    const histogramData = candles
      .map((c, i) => {
        const val = histogram[i];
        if (val === null) return null;
        return { time: c.time, value: val, color: val >= 0 ? "#22c55e80" : "#ef444480" };
      })
      .filter((d): d is { time: string; value: number; color: string } => d !== null);

    const histSeries = chartRef.current.addSeries(HistogramSeries, { priceLineVisible: false });
    histSeries.setData(histogramData);

    const macdSeries = chartRef.current.addSeries(LineSeries, {
      color: "#6366f1",
      lineWidth: 1,
      priceLineVisible: false,
    });
    macdSeries.setData(macdData);

    const signalSeries = chartRef.current.addSeries(LineSeries, {
      color: "#f59e0b",
      lineWidth: 1,
      priceLineVisible: false,
    });
    signalSeries.setData(signalData);
  }, [candles]);

  return <div ref={containerRef} className="w-full h-full" />;
}
