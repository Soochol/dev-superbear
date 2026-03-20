'use client';

import type { Case } from '@/entities/case';

interface ConditionProgressProps {
  caseData: Case;
}

export function ConditionProgress({ caseData }: ConditionProgressProps) {
  return (
    <div className="space-y-3">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs text-nexus-success">✓</span>
          <span className="text-xs text-nexus-text-secondary">Success</span>
        </div>
        <p className="text-xs text-nexus-text-muted font-mono mb-1">{caseData.success_script}</p>
        <div className="w-full bg-nexus-border rounded-full h-1.5">
          <div className="bg-nexus-success h-1.5 rounded-full" style={{ width: '0%' }} />
        </div>
      </div>
      <div>
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs text-nexus-failure">✗</span>
          <span className="text-xs text-nexus-text-secondary">Failure</span>
        </div>
        <p className="text-xs text-nexus-text-muted font-mono mb-1">{caseData.failure_script}</p>
        <div className="w-full bg-nexus-border rounded-full h-1.5">
          <div className="bg-nexus-failure h-1.5 rounded-full" style={{ width: '0%' }} />
        </div>
      </div>
    </div>
  );
}
