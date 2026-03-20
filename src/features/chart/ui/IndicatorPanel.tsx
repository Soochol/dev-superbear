"use client";

import { useEffect, useRef } from "react";
import {
  createChart,
  LineSeries,
  HistogramSeries,
  type IChartApi,
} from "lightweight-charts";
import { useChartStore } from "../model/chart.store";
import {
  calculateRSI,
  calculateMACD,
  getIndicator,
} from "@/entities/indicator";
import { CHART_THEME, toLineData } from "../lib/chart-theme";

interface Props {
  indicatorId: string;
  onRemove: (id: string) => void;
}

export function IndicatorPanel({ indicatorId, onRemove }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candles = useChartStore((s) => s.candles);
  const config = getIndicator(indicatorId);

  useEffect(() => {
    if (!containerRef.current) return;

    const chart = createChart(containerRef.current, {
      ...CHART_THEME,
      crosshair: { mode: 0 },
      height: 120,
    });

    chartRef.current = chart;

    const resizeObserver = new ResizeObserver((entries) => {
      const { width } = entries[0].contentRect;
      chart.applyOptions({ width });
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      chart.remove();
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current || candles.length === 0) return;
    const chart = chartRef.current;
    const closes = candles.map((c) => c.close);

    // lightweight-charts v5: recreate chart to clear all series
    chart.remove();

    const newChart = createChart(containerRef.current!, {
      ...CHART_THEME,
      crosshair: { mode: 0 },
      height: containerRef.current!.clientHeight || 120,
    });
    chartRef.current = newChart;

    if (indicatorId === "rsi") {
      const rsiValues = calculateRSI(closes, 14);
      const lineData = toLineData(candles, rsiValues);
      const rsiSeries = newChart.addSeries(LineSeries, {
        color: config?.color ?? "#f59e0b",
        lineWidth: 1,
        priceLineVisible: false,
      });
      rsiSeries.setData(lineData);

      // Add overbought/oversold reference lines
      newChart.applyOptions({
        rightPriceScale: { scaleMargins: { top: 0.05, bottom: 0.05 } },
      });
    } else if (indicatorId === "macd") {
      const { macd, signal, histogram } = calculateMACD(closes);
      const colors = config?.colors ?? { macd: "#6366f1", signal: "#f59e0b", histUp: "#22c55e", histDown: "#ef4444" };

      // Histogram
      const histData = candles
        .map((c, i) => {
          if (histogram[i] === null) return null;
          return {
            time: c.time,
            value: histogram[i]!,
            color: histogram[i]! >= 0 ? colors.histUp : colors.histDown,
          };
        })
        .filter((d): d is NonNullable<typeof d> => d !== null);

      const histSeries = newChart.addSeries(HistogramSeries, {
        priceLineVisible: false,
      });
      histSeries.setData(histData);

      // MACD line
      const macdSeries = newChart.addSeries(LineSeries, {
        color: colors.macd,
        lineWidth: 1,
        priceLineVisible: false,
      });
      macdSeries.setData(toLineData(candles, macd));

      // Signal line
      const signalSeries = newChart.addSeries(LineSeries, {
        color: colors.signal,
        lineWidth: 1,
        priceLineVisible: false,
      });
      signalSeries.setData(toLineData(candles, signal));
    }

    newChart.timeScale().fitContent();
  }, [candles, indicatorId, config]);

  if (!config) return null;

  return (
    <div className="flex flex-col border-t border-nexus-border" data-testid={`indicator-panel-${indicatorId}`}>
      <div className="flex items-center justify-between px-3 py-1 bg-nexus-surface border-b border-nexus-border">
        <span className="text-[10px] font-medium text-nexus-text-secondary">{config.name}</span>
        <button
          onClick={() => onRemove(indicatorId)}
          className="text-nexus-text-muted hover:text-nexus-text-primary text-xs"
          aria-label={`Remove ${config.name}`}
        >
          ✕
        </button>
      </div>
      <div ref={containerRef} className="h-[120px]" />
    </div>
  );
}
