'use client';

import type { TimelineEvent } from '@/entities/case';
import { formatTimeline, type TimelineItem } from '../lib/timeline-formatter';
import { TimelineNode } from './TimelineNode';
import { TimelineDot } from './TimelineDot';
import { TimelineCard } from './TimelineCard';
import { TimelineConnector } from './TimelineConnector';

interface TimelineProps {
  events: TimelineEvent[];
  eventDate: string;
}

function TimelineItemRenderer({ item }: { item: TimelineItem }) {
  switch (item.variant) {
    case 'node':
      return <TimelineNode item={item} />;
    case 'dot':
      return <TimelineDot item={item} />;
    case 'card':
      return <TimelineCard item={item} />;
  }
}

export function Timeline({ events, eventDate }: TimelineProps) {
  const items = formatTimeline(events, eventDate);

  if (items.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-nexus-text-muted text-sm">
        No timeline events yet
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <h3 className="text-sm font-medium text-nexus-text-primary mb-3">Timeline</h3>
      <div className="flex flex-col">
        {items.map((item, i) => (
          <div key={item.id}>
            <TimelineItemRenderer item={item} />
            {i < items.length - 1 && <TimelineConnector />}
          </div>
        ))}
      </div>
    </div>
  );
}
