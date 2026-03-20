// Frontend types -- mirrors Go backend domain models

export interface AgentBlock {
  id: string;
  userId: string;
  stageId?: string;
  name: string;
  objective: string;
  inputDesc: string;
  tools: string[];
  outputFormat: string;
  constraints: string | null;
  examples: string | null;
  instruction: string;
  systemPrompt: string | null;
  allowedTools: string[];
  outputSchema?: Record<string, unknown>;
  isPublic: boolean;
  isTemplate: boolean;
  templateId?: string;
  createdAt: string;
  updatedAt: string;
  monitorBlock?: MonitorBlock;
}

export interface MonitorBlock {
  id: string;
  pipelineId: string;
  blockId: string;
  cron: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
  block?: AgentBlock;
}

export interface PriceAlert {
  id: string;
  pipelineId?: string;
  caseId?: string;
  condition: string;
  label: string;
  triggered: boolean;
  triggeredAt?: string;
  createdAt: string;
}

/** Template: reusable block in palette (isTemplate=true). Drag creates a copy. */
export interface AgentBlockTemplate {
  id: string;
  name: string;
  objective: string;
  inputDesc: string;
  tools: string[];
  outputFormat: string;
  constraints: string | null;
  examples: string | null;
  isPublic: boolean;
}
