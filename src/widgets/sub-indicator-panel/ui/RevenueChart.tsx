"use client";

import { useEffect, useRef } from "react";
import { createChart, HistogramSeries, type IChartApi, type ISeriesApi } from "lightweight-charts";
import { useChartStore, CHART_THEME } from "@/features/chart";

export function RevenueChart() {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const volumeSeriesRef = useRef<ISeriesApi<"Histogram"> | null>(null);
  const { candles } = useChartStore();

  useEffect(() => {
    if (!containerRef.current) return;

    const chart = createChart(containerRef.current, {
      ...CHART_THEME,
      timeScale: { visible: false },
      height: 128,
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

    if (volumeSeriesRef.current) {
      chartRef.current.removeSeries(volumeSeriesRef.current);
      volumeSeriesRef.current = null;
    }

    const volumeData = candles.map((c) => ({
      time: c.time,
      value: c.volume,
      color: c.close >= c.open ? "#22c55e80" : "#ef444480",
    }));

    const series = chartRef.current.addSeries(HistogramSeries, {
      priceLineVisible: false,
    });
    series.setData(volumeData);
    volumeSeriesRef.current = series;
  }, [candles]);

  return <div ref={containerRef} className="w-full h-full" />;
}
