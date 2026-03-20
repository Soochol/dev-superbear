'use client';

interface CaseDetailLayoutProps {
  selectedCaseId: string | null;
  leftSlot?: React.ReactNode;
  rightSlot?: React.ReactNode;
}

export function CaseDetailLayout({ selectedCaseId, leftSlot, rightSlot }: CaseDetailLayoutProps) {
  if (!selectedCaseId) {
    return (
      <div className="flex-1 flex items-center justify-center text-nexus-text-muted">
        Select a case to view details
      </div>
    );
  }

  return (
    <div className="flex-1 flex overflow-hidden">
      <div className="w-1/2 border-r border-nexus-border overflow-y-auto p-4">
        {leftSlot ?? (
          <div className="text-nexus-text-muted text-sm">Timeline will appear here</div>
        )}
      </div>
      <div className="w-1/2 overflow-y-auto p-4">
        {rightSlot ?? (
          <div className="text-nexus-text-muted text-sm">Details will appear here</div>
        )}
      </div>
    </div>
  );
}
