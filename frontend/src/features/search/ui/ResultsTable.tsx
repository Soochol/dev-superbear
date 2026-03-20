"use client";

import { useRouter } from "next/navigation";
import { useSearchStore } from "../model/search.store";
import { useStockListStore } from "@/entities/stock";
import type { SearchResult } from "@/entities/search-result";
import { btnMini } from "./styles";

export function ResultsTable() {
  const router = useRouter();
  const results = useSearchStore((s) => s.results);
  const { setSearchResults, setSelectedSymbol, addToRecent } = useStockListStore();

  const handleChartClick = (symbol: string) => {
    setSearchResults(results);
    setSelectedSymbol(symbol);
    const found = results.find((r) => r.symbol === symbol);
    if (found) addToRecent(found);
    router.push(`/chart?symbol=${symbol}`);
  };

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-nexus-border text-nexus-text-secondary">
          <th className="text-left py-2 px-3">Code</th>
          <th className="text-left py-2 px-3">Name</th>
          <th className="text-right py-2 px-3">Matched Value</th>
          <th className="text-right py-2 px-3">Close</th>
          <th className="text-right py-2 px-3">Change %</th>
          <th className="text-center py-2 px-3"></th>
        </tr>
      </thead>
      <tbody>
        {results.map((row: SearchResult) => (
          <tr key={row.symbol} className="border-b border-nexus-border/50 hover:bg-nexus-border/20">
            <td className="py-2 px-3 font-mono text-nexus-text-secondary">{row.symbol}</td>
            <td className="py-2 px-3">{row.name}</td>
            <td className="py-2 px-3 text-right font-mono">{String(row.matchedValue)}</td>
            <td className="py-2 px-3 text-right font-mono">{row.close ?? "-"}</td>
            <td className="py-2 px-3 text-right font-mono">
              {row.changePct != null
                ? `${row.changePct > 0 ? "+" : ""}${row.changePct}%`
                : "-"}
            </td>
            <td className="py-2 px-3 text-center">
              <button
                onClick={() => handleChartClick(row.symbol)}
                aria-label="Chart"
                className={btnMini}
              >
                Chart
              </button>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
