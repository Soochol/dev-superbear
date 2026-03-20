"use client";

import { useEffect, useRef } from "react";
import { createChart, LineSeries, HistogramSeries, type IChartApi, type ISeriesApi } from "lightweight-charts";
import { useChartStore } from "@/features/chart";
import { calculateMACD } from "@/entities/indicator";
import { CHART_THEME, toLineData } from "@/features/chart";

export function MACDChart() {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const histSeriesRef = useRef<ISeriesApi<"Histogram"> | null>(null);
  const macdSeriesRef = useRef<ISeriesApi<"Line"> | null>(null);
  const signalSeriesRef = useRef<ISeriesApi<"Line"> | null>(null);
  const { candles } = useChartStore();

  useEffect(() => {
    if (!containerRef.current) return;

    const chart = createChart(containerRef.current, {
      ...CHART_THEME,
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

    const macdData = toLineData(candles, macd);
    const signalData = toLineData(candles, signal);

    const histogramData = candles
      .map((c, i) => {
        const val = histogram[i];
        if (val === null) return null;
        return { time: c.time, value: val, color: val >= 0 ? "#22c55e80" : "#ef444480" };
      })
      .filter((d): d is { time: string; value: number; color: string } => d !== null);

    if (histSeriesRef.current) {
      chartRef.current.removeSeries(histSeriesRef.current);
      histSeriesRef.current = null;
    }
    if (macdSeriesRef.current) {
      chartRef.current.removeSeries(macdSeriesRef.current);
      macdSeriesRef.current = null;
    }
    if (signalSeriesRef.current) {
      chartRef.current.removeSeries(signalSeriesRef.current);
      signalSeriesRef.current = null;
    }

    const histSeries = chartRef.current.addSeries(HistogramSeries, { priceLineVisible: false });
    histSeries.setData(histogramData);
    histSeriesRef.current = histSeries;

    const macdSeries = chartRef.current.addSeries(LineSeries, {
      color: "#6366f1",
      lineWidth: 1,
      priceLineVisible: false,
    });
    macdSeries.setData(macdData);
    macdSeriesRef.current = macdSeries;

    const signalSeries = chartRef.current.addSeries(LineSeries, {
      color: "#f59e0b",
      lineWidth: 1,
      priceLineVisible: false,
    });
    signalSeries.setData(signalData);
    signalSeriesRef.current = signalSeries;
  }, [candles]);

  return <div ref={containerRef} className="w-full h-full" />;
}
