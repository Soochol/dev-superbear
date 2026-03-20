"use client";

import { useEffect, useState } from "react";
import { logger } from "@/shared/lib/logger";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface SectorStock {
  symbol: string;
  name: string;
  per: number | null;
  roe: number | null;
  rsi: number | null;
  changePct: number;
}

export function SectorComparePanel({ symbol }: { symbol: string }) {
  const [stocks, setStocks] = useState<SectorStock[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    fetch(`${API_BASE}/api/v1/financials/${symbol}/sector`)
      .then((res) => res.json())
      .then((json) => setStocks((json as { data?: SectorStock[] }).data ?? []))
      .catch((err) => {
        logger.error("Failed to fetch sector data", { symbol, message: String(err) });
        setStocks([]);
      })
      .finally(() => setLoading(false));
  }, [symbol]);

  return (
    <div className="p-4">
      <h3 className="text-xs font-semibold text-nexus-text-secondary uppercase mb-3">
        Sector Compare
      </h3>
      {loading ? (
        <div className="text-nexus-text-muted text-sm">Loading...</div>
      ) : stocks.length === 0 ? (
        <div className="text-nexus-text-muted text-sm">No sector data available</div>
      ) : (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-nexus-text-muted">
              <th className="text-left py-1">Name</th>
              <th className="text-right py-1">PER</th>
              <th className="text-right py-1">ROE</th>
              <th className="text-right py-1">RSI</th>
              <th className="text-right py-1">Chg%</th>
            </tr>
          </thead>
          <tbody>
            {stocks.map((s) => (
              <tr
                key={s.symbol}
                className={s.symbol === symbol ? "text-nexus-accent font-medium" : ""}
              >
                <td className="py-1 truncate max-w-[100px]">{s.name}</td>
                <td className="text-right font-mono">{s.per?.toFixed(1) ?? "-"}</td>
                <td className="text-right font-mono">{s.roe?.toFixed(1) ?? "-"}%</td>
                <td className="text-right font-mono">{s.rsi?.toFixed(0) ?? "-"}</td>
                <td className={`text-right font-mono ${s.changePct >= 0 ? "text-nexus-success" : "text-nexus-failure"}`}>
                  {s.changePct >= 0 ? "+" : ""}{s.changePct.toFixed(2)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
