'use client';

import { useState } from 'react';
import type { Case } from '@/entities/case';
import { ConditionProgress } from './ConditionProgress';
import { ReturnTrackingTable } from './ReturnTrackingTable';
import { TradeHistory } from '@/features/manage-trades';
import { PriceAlertsList } from '@/features/manage-alerts';

interface CaseDetailPanelProps {
  caseData: Case;
}

interface SectionProps {
  title: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

function Section({ title, defaultOpen = true, children }: SectionProps) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="border-b border-nexus-border pb-3 mb-3">
      <button onClick={() => setOpen(!open)} className="flex items-center gap-2 w-full text-left mb-2">
        <span className="text-xs text-nexus-text-muted">{open ? '▾' : '▸'}</span>
        <span className="text-sm font-medium text-nexus-text-primary">{title}</span>
      </button>
      {open && children}
    </div>
  );
}

export function CaseDetailPanel({ caseData }: CaseDetailPanelProps) {
  const isClosed = caseData.status === 'CLOSED_SUCCESS' || caseData.status === 'CLOSED_FAILURE';

  return (
    <div className="space-y-1">
      <Section title="Conditions">
        <ConditionProgress caseData={caseData} />
      </Section>
      <Section title="Return Tracking">
        <ReturnTrackingTable caseId={caseData.id} />
      </Section>
      <Section title="Trade History">
        <TradeHistory caseId={caseData.id} isClosed={isClosed} />
      </Section>
      <Section title="Price Alerts">
        <PriceAlertsList caseId={caseData.id} isClosed={isClosed} />
      </Section>
    </div>
  );
}
