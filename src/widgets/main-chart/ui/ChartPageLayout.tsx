"use client";

export function ChartPageLayout() {
  return (
    <div className="flex flex-col h-full">
      {/* Topbar: 종목 정보 + 타임프레임 선택 */}
      <div className="px-4 py-2 bg-nexus-surface border-b border-nexus-border">
        <span className="text-nexus-text-muted">Chart Topbar (placeholder)</span>
      </div>

      {/* Main Area: 차트(좌) + 사이드바(우) */}
      <div className="flex flex-1 min-h-0">
        {/* 좌측: 메인 차트 + 보조 지표 */}
        <div className="flex-1 flex flex-col min-w-0">
          <div className="flex-1 min-h-0 flex items-center justify-center text-nexus-text-muted">
            Main Chart (placeholder)
          </div>
          <div className="border-t border-nexus-border p-2 text-nexus-text-muted text-sm">
            Sub-Indicator Panel (placeholder)
          </div>
        </div>

        {/* 우측: 종목 리스트 사이드바 */}
        <div className="w-72 border-l border-nexus-border flex-shrink-0 flex items-center justify-center text-nexus-text-muted">
          Stock List Sidebar (placeholder)
        </div>
      </div>

      {/* 하단: Financials | AI Fusion | Sector Compare (full-width 3칼럼) */}
      <div className="h-48 border-t border-nexus-border bg-nexus-surface flex items-center justify-center text-nexus-text-muted">
        Bottom Info Panel (placeholder)
      </div>
    </div>
  );
}
