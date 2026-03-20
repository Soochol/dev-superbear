export type TimelineEventType = 'NEWS' | 'DISCLOSURE' | 'SECTOR' | 'PRICE_ALERT' | 'TRADE' | 'PIPELINE_RESULT';

export interface TimelineEvent {
  id: string;
  case_id: string;
  date: string;
  type: TimelineEventType;
  title: string;
  content: string;
  ai_analysis?: string;
  data?: Record<string, unknown>;
  created_at: string;
}
