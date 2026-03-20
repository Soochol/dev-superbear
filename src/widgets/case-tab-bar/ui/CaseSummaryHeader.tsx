'use client';

import type { Case } from '@/entities/case';

interface CaseSummaryHeaderProps {
  caseData: Case;
}

const STATUS_CONFIG: Record<string, { label: string; color: string }> = {
  LIVE: { label: 'LIVE', color: 'text-nexus-success' },
  CLOSED_SUCCESS: { label: 'SUCCESS', color: 'text-blue-400' },
  CLOSED_FAILURE: { label: 'FAILURE', color: 'text-nexus-failure' },
  BACKTEST: { label: 'BACKTEST', color: 'text-nexus-warning' },
};

export function CaseSummaryHeader({ caseData }: CaseSummaryHeaderProps) {
  const status = STATUS_CONFIG[caseData.status] ?? { label: caseData.status, color: 'text-gray-400' };

  // Calculate D+N from event_date
  const eventDate = new Date(caseData.event_date);
  const today = new Date();
  const dayOffset = Math.floor((today.getTime() - eventDate.getTime()) / (1000 * 60 * 60 * 24));

  return (
    <div className="flex items-center gap-4 px-4 py-2 bg-nexus-surface/50 border-b border-nexus-border text-sm">
      <span className="font-mono font-semibold text-nexus-text-primary">{caseData.symbol}</span>
      <span className="text-nexus-text-secondary">{caseData.symbol_name}</span>
      <span className={`font-medium ${status.color}`}>● {status.label}</span>
      <span className="text-nexus-text-muted">D+{dayOffset}</span>
      {caseData.sector && (
        <span className="text-nexus-text-muted text-xs px-2 py-0.5 rounded bg-nexus-border/50">
          {caseData.sector}
        </span>
      )}
    </div>
  );
}
