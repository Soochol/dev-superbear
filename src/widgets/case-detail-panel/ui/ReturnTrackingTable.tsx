'use client';

import { useEffect, useState } from 'react';
import { apiGet } from '@/shared/api/client';

interface ReturnPeriod {
  label: string;
  return_pct: number;
  vs_kospi: number;
  vs_sector: number;
  day_offset: number;
}

interface ReturnTrackingTableProps {
  caseId: string;
}

export function ReturnTrackingTable({ caseId }: ReturnTrackingTableProps) {
  const [periods, setPeriods] = useState<ReturnPeriod[]>([]);

  useEffect(() => {
    apiGet<{ data: { periods: ReturnPeriod[] } }>(`/api/v1/cases/${caseId}/return-tracking`)
      .then((res) => setPeriods(res.data.periods))
      .catch(() => setPeriods([]));
  }, [caseId]);

  if (periods.length === 0) {
    return <p className="text-xs text-nexus-text-muted">No return data available yet</p>;
  }

  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="text-nexus-text-muted border-b border-nexus-border">
          <th className="text-left py-1.5 font-medium">Period</th>
          <th className="text-right py-1.5 font-medium">Return</th>
          <th className="text-right py-1.5 font-medium">vs KOSPI</th>
          <th className="text-right py-1.5 font-medium">vs Sector</th>
        </tr>
      </thead>
      <tbody>
        {periods.map((p) => (
          <tr key={p.label} className="border-b border-nexus-border/50">
            <td className="py-1.5 text-nexus-text-secondary">{p.label}</td>
            <td className={`py-1.5 text-right font-mono ${p.return_pct >= 0 ? 'text-nexus-success' : 'text-nexus-failure'}`}>
              {p.return_pct >= 0 ? '+' : ''}{p.return_pct.toFixed(1)}%
            </td>
            <td className={`py-1.5 text-right font-mono ${p.vs_kospi >= 0 ? 'text-nexus-success' : 'text-nexus-failure'}`}>
              {p.vs_kospi >= 0 ? '+' : ''}{p.vs_kospi.toFixed(1)}%
            </td>
            <td className={`py-1.5 text-right font-mono ${p.vs_sector >= 0 ? 'text-nexus-success' : 'text-nexus-failure'}`}>
              {p.vs_sector >= 0 ? '+' : ''}{p.vs_sector.toFixed(1)}%
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
