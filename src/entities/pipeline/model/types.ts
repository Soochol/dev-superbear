// Frontend types -- mirrors Go backend domain models (API response shape)

import type {
  AgentBlock,
  MonitorBlock,
  PriceAlert,
} from "@/entities/agent-block";

export interface Pipeline {
  id: string;
  userId: string;
  name: string;
  description: string;
  successScript: string | null;
  failureScript: string | null;
  isPublic: boolean;
  createdAt: string;
  updatedAt: string;
  stages?: Stage[];
  monitors?: MonitorBlock[];
  priceAlerts?: PriceAlert[];
}

export interface Stage {
  id: string;
  pipelineId: string;
  section: "analysis" | "monitoring" | "judgment";
  order: number;
  createdAt: string;
  blocks?: AgentBlock[];
}

export interface PipelineJob {
  id: string;
  pipelineId: string;
  symbol: string;
  status: "PENDING" | "RUNNING" | "COMPLETED" | "FAILED";
  result?: Record<string, unknown>;
  error?: string;
  startedAt?: string;
  completedAt?: string;
  createdAt: string;
}

export interface AgentInput {
  instruction: string;
  context: {
    symbol: string;
    symbolName: string;
    eventDate: string;
    eventSnapshot?: EventSnapshot;
    previousResults: PreviousResult[];
  };
}

export interface AgentOutput {
  summary: string;
  data?: Record<string, unknown>;
  confidence?: number;
}

export interface PreviousResult {
  blockName: string;
  summary: string;
  data?: Record<string, unknown>;
}

export interface EventSnapshot {
  high: number;
  low: number;
  close: number;
  volume: number;
  tradeValue: number;
  preMa: Record<number, number>;
}

export interface PipelineExecutionContext {
  symbol: string;
  previousResults: PreviousResult[];
}

// Re-export block types for convenience
export type { AgentBlock, MonitorBlock, PriceAlert };
