'use client';

import type { TimelineItem } from '../lib/timeline-formatter';

interface TimelineDotProps {
  item: TimelineItem;
}

export function TimelineDot({ item }: TimelineDotProps) {
  return (
    <div className="flex gap-3 py-1.5">
      <div className="flex flex-col items-center">
        <div className={`w-2 h-2 rounded-full ${item.color} bg-current mt-1.5`} />
      </div>
      <div className="flex items-center gap-2 flex-1 min-w-0">
        <span className="text-xs font-mono text-nexus-text-muted">{item.label}</span>
        <span className="text-xs text-nexus-text-secondary truncate">{item.title}</span>
      </div>
    </div>
  );
}
