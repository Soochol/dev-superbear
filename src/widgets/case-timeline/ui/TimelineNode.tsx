'use client';

import type { TimelineItem } from '../lib/timeline-formatter';

interface TimelineNodeProps {
  item: TimelineItem;
}

export function TimelineNode({ item }: TimelineNodeProps) {
  return (
    <div className="flex gap-3 py-3">
      <div className="flex flex-col items-center">
        <div className={`w-4 h-4 rounded-full border-2 ${item.color} border-current`} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-xs font-mono text-nexus-text-muted">{item.label}</span>
          <span className={`text-xs px-1.5 py-0.5 rounded ${item.bgColor} ${item.color}`}>
            {item.type.replace('_', ' ')}
          </span>
        </div>
        <p className="text-sm font-medium text-nexus-text-primary mt-1">{item.title}</p>
        {item.content && (
          <p className="text-xs text-nexus-text-secondary mt-0.5">{item.content}</p>
        )}
        <span className="text-xs text-nexus-text-muted">{item.date}</span>
      </div>
    </div>
  );
}
