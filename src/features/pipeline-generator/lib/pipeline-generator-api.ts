import { apiPost } from "@/shared/api/client";

export interface GenerateResponse {
  data: {
    name: string;
    description: string;
    stages: Array<{
      section: string;
      order: number;
      blocks: Array<{
        name: string;
        objective: string;
        inputDesc: string;
        tools: string[];
        outputFormat: string;
      }>;
    }>;
    monitors: Array<{
      block: { name: string; objective: string };
      cron: string;
      enabled: boolean;
    }>;
    successScript: string | null;
    failureScript: string | null;
    priceAlerts: Array<{ condition: string; label: string }>;
  };
}

export function generatePipeline(description: string) {
  return apiPost<GenerateResponse>("/pipelines/generate", { description });
}
