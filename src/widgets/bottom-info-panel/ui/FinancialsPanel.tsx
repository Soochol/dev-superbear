"use client";

import { useEffect, useState } from "react";
import { logger } from "@/shared/lib/logger";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface FinancialData {
  revenue: number | null;
  operatingProfit: number | null;
  netMargin: number | null;
  per: number | null;
  pbr: number | null;
  roe: number | null;
}

export function FinancialsPanel({ symbol }: { symbol: string }) {
  const [data, setData] = useState<FinancialData | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch(`${API_BASE}/api/v1/financials/${symbol}`)
      .then((res) => res.json())
      .then((json) => setData((json as { data?: FinancialData }).data ?? null))
      .catch((err) => {
        logger.error("Failed to fetch financials", { symbol, message: String(err) });
        setData(null);
      })
      .finally(() => setLoading(false));
  }, [symbol]);

  const metrics = [
    { label: "Revenue", value: data?.revenue ?? null, format: "억원" },
    { label: "Op.Profit", value: data?.operatingProfit ?? null, format: "억원" },
    { label: "Net Margin", value: data?.netMargin ?? null, format: "%" },
    { label: "PER", value: data?.per ?? null, format: "x" },
    { label: "PBR", value: data?.pbr ?? null, format: "x" },
    { label: "ROE", value: data?.roe ?? null, format: "%" },
  ];

  return (
    <div className="p-4">
      <h3 className="text-xs font-semibold text-nexus-text-secondary uppercase mb-3">
        Financials
      </h3>
      <div className="space-y-2">
        {metrics.map((m) => (
          <div key={m.label} className="flex justify-between text-sm">
            <span className="text-nexus-text-secondary">{m.label}</span>
            <span className={`font-mono ${loading ? "text-nexus-text-muted" : ""}`}>
              {m.value != null ? `${m.value.toLocaleString()}${m.format}` : "-"}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}
