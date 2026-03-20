"use client";

import { useCallback } from "react";
import { apiGet, apiPost } from "@/shared/api/client";
import type { Pipeline, PipelineJob } from "@/entities/pipeline/model/types";
import { usePipelineStore } from "../model/pipeline.store";

interface CreatePipelineBody {
  name: string;
  description: string;
  successScript: string | null;
  failureScript: string | null;
  stages: Array<{
    section: string;
    order: number;
    blocks: Array<{
      name: string;
      objective: string;
      inputDesc: string;
      tools: string[];
      outputFormat: string;
      constraints: string | null;
      examples: string | null;
    }>;
  }>;
  monitors: Array<{
    block: {
      name: string;
      objective: string;
      inputDesc: string;
      tools: string[];
      outputFormat: string;
    };
    cron: string;
    enabled: boolean;
  }>;
  priceAlerts: Array<{
    condition: string;
    label: string;
  }>;
}

async function pollJobStatus(
  jobId: string,
  maxAttempts = 60,
  intervalMs = 2000,
): Promise<PipelineJob> {
  let consecutiveErrors = 0;
  for (let i = 0; i < maxAttempts; i++) {
    try {
      const job = await apiGet<PipelineJob>(`/pipelines/jobs/${jobId}`);
      consecutiveErrors = 0;
      if (job.status === "COMPLETED" || job.status === "FAILED") {
        return job;
      }
    } catch {
      consecutiveErrors++;
      if (consecutiveErrors >= 3) {
        throw new Error("Job polling failed: too many consecutive errors");
      }
    }
    await new Promise((resolve) => setTimeout(resolve, intervalMs));
  }
  throw new Error("Job polling timed out");
}

export function useRegisterAndRun(onComplete?: (job: PipelineJob) => void) {
  const store = usePipelineStore;

  const registerAndRun = useCallback(async () => {
    const state = store.getState();
    const { pipelineName, selectedSymbol, analysisStages, monitorBlocks, successScript, failureScript, priceAlerts } = state;

    if (!pipelineName.trim()) {
      throw new Error("Pipeline name is required");
    }
    if (!selectedSymbol.trim()) {
      throw new Error("Symbol is required");
    }

    state.setIsRunning(true);

    try {
      // 1. Build pipeline body
      const body: CreatePipelineBody = {
        name: pipelineName.trim(),
        description: state.pipelineDescription || pipelineName.trim(),
        successScript: successScript || null,
        failureScript: failureScript || null,
        stages: analysisStages.map((s) => ({
          section: "analysis",
          order: s.order,
          blocks: s.blocks.map((b) => ({
            name: b.name,
            objective: b.objective,
            inputDesc: b.inputDesc,
            tools: b.tools,
            outputFormat: b.outputFormat,
            constraints: b.constraints,
            examples: b.examples,
          })),
        })),
        monitors: monitorBlocks.map((m) => ({
          block: {
            name: m.block.name,
            objective: m.block.objective,
            inputDesc: m.block.inputDesc,
            tools: m.block.tools,
            outputFormat: m.block.outputFormat,
          },
          cron: m.cron,
          enabled: m.enabled,
        })),
        priceAlerts: priceAlerts.map((a) => ({
          condition: a.condition,
          label: a.label,
        })),
      };

      // 2. Create pipeline
      const pipeline = await apiPost<Pipeline>("/pipelines", body);
      state.setPipelineId(pipeline.id);

      // 3. Execute pipeline
      const job = await apiPost<PipelineJob>(
        `/pipelines/${pipeline.id}/execute`,
        { symbol: selectedSymbol.trim() },
      );

      // 4. Poll for completion
      const completedJob = await pollJobStatus(job.id);

      onComplete?.(completedJob);
      return completedJob;
    } finally {
      state.setIsRunning(false);
    }
  }, [store, onComplete]);

  return { registerAndRun };
}
