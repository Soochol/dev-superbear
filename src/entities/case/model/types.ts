export type CaseStatus = 'LIVE' | 'CLOSED_SUCCESS' | 'CLOSED_FAILURE' | 'BACKTEST';

export interface Case {
  id: string;
  user_id: string;
  pipeline_id: string;
  symbol: string;
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
  pipeline?: Pipeline;
  timeline_events?: TimelineEvent[];
  trades?: Trade[];
  price_alerts?: PriceAlert[];
}

export interface EventSnapshot {
  high: number;
  low: number;
  close: number;
  volume: number;
  trade_value: number;
  pre_ma: Record<number, number>;
}

import type { Pipeline } from '@/entities/pipeline/model/types';
import type { TimelineEvent } from '@/entities/timeline-event/model/types';
import type { Trade } from '@/entities/trade/model/types';
import type { PriceAlert } from '@/entities/price-alert/model/types';
