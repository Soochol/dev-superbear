"use client";

import { useEffect, useRef } from "react";
import { createChart, LineSeries, type IChartApi, type ISeriesApi } from "lightweight-charts";
import { useChartStore } from "@/features/chart";
import { calculateRSI } from "@/entities/indicator";
import { CHART_THEME, toLineData } from "@/features/chart";

export function RSIChart() {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const rsiSeriesRef = useRef<ISeriesApi<"Line"> | null>(null);
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
    const rsiValues = calculateRSI(closes);

    const rsiData = toLineData(candles, rsiValues);

    if (rsiSeriesRef.current) {
      chartRef.current.removeSeries(rsiSeriesRef.current);
      rsiSeriesRef.current = null;
    }

    const series = chartRef.current.addSeries(LineSeries, {
      color: "#8b5cf6",
      lineWidth: 1,
      priceLineVisible: false,
    });
    series.setData(rsiData);
    rsiSeriesRef.current = series;
  }, [candles]);

  return <div ref={containerRef} className="w-full h-full" />;
}
