import type { Trade } from '@/entities/trade/model/types';
import type { PriceAlert } from '@/entities/price-alert/model/types';

export type CaseStatus = 'LIVE' | 'CLOSED_SUCCESS' | 'CLOSED_FAILURE' | 'BACKTEST';

export type TimelineEventType =
  | 'NEWS'
  | 'DISCLOSURE'
  | 'SECTOR'
  | 'PRICE_ALERT'
  | 'TRADE'
  | 'PIPELINE_RESULT'
  | 'MONITOR_RESULT';

export interface Case {
  id: string;
  user_id: string;
  pipeline_id: string;
  symbol: string;
  symbol_name: string;
  sector: string | null;
  status: CaseStatus;
  event_date: string;
  event_snapshot: EventSnapshot;
  success_script: string;
  failure_script: string;
  closed_at?: string;
  closed_reason?: string;
  created_at: string;
  updated_at: string;
}

export interface CaseWithRelations extends Case {
  timeline_events?: TimelineEvent[];
  trades?: Trade[];
  price_alerts?: PriceAlert[];
}

export interface TimelineEvent {
  id: string;
  case_id: string;
  date: string;
  day_offset: number;
  type: TimelineEventType;
  title: string;
  content: string;
  ai_analysis: string | null;
  data: Record<string, unknown> | null;
  created_at: string;
}

export interface EventSnapshot {
  high: number;
  low: number;
  close: number;
  volume: number;
  trade_value: number;
  pre_ma: Record<number, number>;
  [key: string]: unknown;
}

export interface CaseSummary {
  id: string;
  symbol: string;
  symbol_name: string;
  sector: string | null;
  status: CaseStatus;
  event_date: string;
  day_offset: number;
  current_return: number;
  peak_return: number;
}

export interface CaseDetail extends Case {
  recent_timeline: TimelineEvent[];
}

export interface CaseFilters {
  status?: CaseStatus;
  symbol?: string;
  sector?: string;
}
