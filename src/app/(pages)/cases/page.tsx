'use client';

import { useEffect } from 'react';
import { useCaseStore } from '@/entities/case/model/case.store';
import { useTradeStore } from '@/features/manage-trades/model/trade.store';
import { useAlertStore } from '@/features/manage-alerts/model/alert.store';
import { CaseTabBar } from '@/widgets/case-tab-bar/ui/CaseTabBar';
import { CaseSummaryHeader } from '@/widgets/case-tab-bar/ui/CaseSummaryHeader';
import { CaseDetailLayout } from '@/widgets/case-detail-panel/ui/CaseDetailLayout';
import { Timeline } from '@/widgets/case-timeline/ui/Timeline';
import { CaseDetailPanel } from '@/widgets/case-detail-panel/ui/CaseDetailPanel';

export default function CasesPage() {
  const { cases, selectedCase, timelineEvents, fetchCases, selectCase, fetchTimeline } = useCaseStore();
  const { fetchTrades } = useTradeStore();
  const { fetchAlerts } = useAlertStore();

  useEffect(() => {
    fetchCases();
  }, [fetchCases]);

  const handleSelectCase = async (id: string) => {
    await selectCase(id);
    await Promise.all([fetchTimeline(id), fetchTrades(id), fetchAlerts(id)]);
  };

  return (
    <div className="flex flex-col h-screen bg-nexus-bg">
      <CaseTabBar
        cases={cases}
        selectedCaseId={selectedCase?.id ?? null}
        onSelectCase={handleSelectCase}
      />
      {selectedCase && <CaseSummaryHeader caseData={selectedCase} />}
      <CaseDetailLayout
        selectedCaseId={selectedCase?.id ?? null}
        leftSlot={
          selectedCase && (
            <Timeline events={timelineEvents} eventDate={selectedCase.event_date} />
          )
        }
        rightSlot={
          selectedCase && (
            <CaseDetailPanel caseData={selectedCase} />
          )
        }
      />
    </div>
  );
}
