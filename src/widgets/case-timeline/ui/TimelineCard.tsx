'use client';

import { useState } from 'react';
import type { TimelineItem } from '../lib/timeline-formatter';

interface TimelineCardProps {
  item: TimelineItem;
}

export function TimelineCard({ item }: TimelineCardProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="flex gap-3 py-2">
      <div className="flex flex-col items-center">
        <div className={`w-3 h-3 rounded-full border-2 ${item.color} border-current`} />
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-xs font-mono text-nexus-text-muted">{item.label}</span>
          <span className={`text-xs px-1.5 py-0.5 rounded ${item.bgColor} ${item.color}`}>
            {item.type.replace('_', ' ')}
          </span>
        </div>
        <div
          className={`rounded border border-nexus-border bg-nexus-surface p-3 cursor-pointer hover:border-nexus-accent/30 transition-colors`}
          onClick={() => setExpanded(!expanded)}
        >
          <p className="text-sm font-medium text-nexus-text-primary">{item.title}</p>
          {item.content && (
            <p className={`text-xs text-nexus-text-secondary mt-1 ${expanded ? '' : 'line-clamp-2'}`}>
              {item.content}
            </p>
          )}
          {item.aiAnalysis && expanded && (
            <div className="mt-2 pt-2 border-t border-nexus-border">
              <p className="text-xs text-nexus-accent font-medium mb-1">AI Analysis</p>
              <p className="text-xs text-nexus-text-secondary">{item.aiAnalysis}</p>
            </div>
          )}
          {(item.content || item.aiAnalysis) && (
            <button className="text-xs text-nexus-text-muted mt-1 hover:text-nexus-accent">
              {expanded ? '접기' : '자세히 보기'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
