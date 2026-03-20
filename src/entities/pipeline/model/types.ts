export interface Pipeline {
  id: string;
  user_id: string;
  name: string;
  description: string;
  analysis_stages: AnalysisStage[];
  monitors: MonitorConfig[];
  success_script: string;
  failure_script: string;
  is_public: boolean;
  created_at: string;
  updated_at: string;
}

export interface AnalysisStage {
  order: number;
  blockIds: string[];
}

export interface MonitorConfig {
  blockId: string;
  cron: string;
  enabled: boolean;
}
