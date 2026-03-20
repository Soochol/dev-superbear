'use client';

import { useEffect } from 'react';
import { useCaseStore } from '@/entities/case/model/case.store';
import { CaseTabBar } from '@/widgets/case-tab-bar/ui/CaseTabBar';
import { CaseSummaryHeader } from '@/widgets/case-tab-bar/ui/CaseSummaryHeader';
import { CaseDetailLayout } from '@/widgets/case-detail-panel/ui/CaseDetailLayout';

export default function CasesPage() {
  const { cases, selectedCase, fetchCases, selectCase } = useCaseStore();

  useEffect(() => {
    fetchCases();
  }, [fetchCases]);

  return (
    <div className="flex flex-col h-screen bg-nexus-bg">
      <CaseTabBar
        cases={cases}
        selectedCaseId={selectedCase?.id ?? null}
        onSelectCase={selectCase}
      />
      {selectedCase && <CaseSummaryHeader caseData={selectedCase} />}
      <CaseDetailLayout selectedCaseId={selectedCase?.id ?? null} />
    </div>
  );
}
