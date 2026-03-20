import type { TimelineEvent, TimelineEventType } from '@/entities/case';

export type TimelineVariant = 'node' | 'dot' | 'card';

export interface TimelineItem {
  id: string;
  label: string;          // "D-Day", "D+7", etc.
  date: string;
  type: TimelineEventType;
  variant: TimelineVariant;
  title: string;
  content: string;
  aiAnalysis: string | null;
  color: string;           // tailwind color class
  bgColor: string;         // tailwind bg color class
}

const VARIANT_MAP: Record<string, TimelineVariant> = {
  PIPELINE_RESULT: 'node',
  TRADE: 'node',
  PRICE_ALERT: 'dot',
  SECTOR: 'dot',
  NEWS: 'card',
  DISCLOSURE: 'card',
  MONITOR_RESULT: 'card',
};

const COLOR_MAP: Record<string, { color: string; bgColor: string }> = {
  NEWS: { color: 'text-blue-400', bgColor: 'bg-blue-500/20' },
  DISCLOSURE: { color: 'text-purple-400', bgColor: 'bg-purple-500/20' },
  SECTOR: { color: 'text-yellow-400', bgColor: 'bg-yellow-500/20' },
  PRICE_ALERT: { color: 'text-orange-400', bgColor: 'bg-orange-500/20' },
  TRADE: { color: 'text-nexus-success', bgColor: 'bg-nexus-success/20' },
  PIPELINE_RESULT: { color: 'text-cyan-400', bgColor: 'bg-cyan-500/20' },
  MONITOR_RESULT: { color: 'text-pink-400', bgColor: 'bg-pink-500/20' },
};

export function formatTimelineEvent(event: TimelineEvent, _eventDate: string): TimelineItem {
  const dayOffset = event.day_offset;
  const label = dayOffset === 0 ? 'D-Day' : `D+${dayOffset}`;
  const colors = COLOR_MAP[event.type] ?? { color: 'text-gray-400', bgColor: 'bg-gray-500/20' };

  return {
    id: event.id,
    label,
    date: event.date,
    type: event.type,
    variant: VARIANT_MAP[event.type] ?? 'card',
    title: event.title,
    content: event.content,
    aiAnalysis: event.ai_analysis,
    color: colors.color,
    bgColor: colors.bgColor,
  };
}

export function formatTimeline(events: TimelineEvent[], eventDate: string): TimelineItem[] {
  return events.map((e) => formatTimelineEvent(e, eventDate));
}
