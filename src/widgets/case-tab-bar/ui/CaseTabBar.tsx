'use client';

import { useRef } from 'react';
import type { Case } from '@/entities/case';

interface CaseTabBarProps {
  cases: Case[];
  selectedCaseId: string | null;
  onSelectCase: (id: string) => void;
}

const STATUS_COLORS: Record<string, string> = {
  LIVE: 'bg-nexus-success',
  CLOSED_SUCCESS: 'bg-blue-500',
  CLOSED_FAILURE: 'bg-nexus-failure',
  BACKTEST: 'bg-nexus-warning',
};

export function CaseTabBar({ cases, selectedCaseId, onSelectCase }: CaseTabBarProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  const scroll = (dir: number) => {
    scrollRef.current?.scrollBy({ left: dir * 200, behavior: 'smooth' });
  };

  if (cases.length === 0) {
    return (
      <div className="flex items-center justify-center h-12 bg-nexus-surface border-b border-nexus-border text-nexus-text-muted text-sm">
        No cases yet
      </div>
    );
  }

  return (
    <div className="flex items-center bg-nexus-surface border-b border-nexus-border">
      <button
        onClick={() => scroll(-1)}
        className="px-2 py-3 text-nexus-text-muted hover:text-nexus-text-primary shrink-0"
      >
        ◀
      </button>
      <div ref={scrollRef} className="flex gap-1 overflow-x-auto scrollbar-hide py-1 flex-1">
        {cases.map((c) => (
          <button
            key={c.id}
            onClick={() => onSelectCase(c.id)}
            className={`flex items-center gap-2 px-3 py-2 rounded text-sm whitespace-nowrap shrink-0 transition-colors ${
              c.id === selectedCaseId
                ? 'bg-nexus-accent/20 text-nexus-accent border border-nexus-accent/30'
                : 'text-nexus-text-secondary hover:bg-nexus-border/50'
            }`}
          >
            <span className="font-mono text-xs">{c.symbol}</span>
            <span className="text-xs text-nexus-text-muted">{c.symbol_name}</span>
            <span className={`w-2 h-2 rounded-full ${STATUS_COLORS[c.status] ?? 'bg-gray-500'}`} />
          </button>
        ))}
      </div>
      <button
        onClick={() => scroll(1)}
        className="px-2 py-3 text-nexus-text-muted hover:text-nexus-text-primary shrink-0"
      >
        ▶
      </button>
    </div>
  );
}
