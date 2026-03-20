"use client";

import { useCallback, type DragEvent } from "react";
import type { AgentBlock } from "@/entities/agent-block/model/types";
import { usePipelineStore } from "../model/pipeline.store";

export interface DragPayload {
  type: "palette-block";
  block: Omit<AgentBlock, "id" | "userId" | "createdAt" | "updatedAt" | "isPublic" | "isTemplate"> & {
    isPublic?: boolean;
    isTemplate?: boolean;
  };
}

function createBlockFromPayload(
  payload: DragPayload["block"],
): AgentBlock {
  return {
    id: crypto.randomUUID(),
    userId: "",
    name: payload.name,
    objective: payload.objective,
    inputDesc: payload.inputDesc,
    tools: payload.tools,
    outputFormat: payload.outputFormat,
    constraints: payload.constraints ?? null,
    examples: payload.examples ?? null,
    instruction: payload.instruction ?? "",
    systemPrompt: payload.systemPrompt ?? null,
    allowedTools: payload.allowedTools ?? [],
    isPublic: false,
    isTemplate: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
}

export function usePipelineDragDrop() {
  const addBlockToStage = usePipelineStore((s) => s.addBlockToStage);
  const addMonitorBlock = usePipelineStore((s) => s.addMonitorBlock);

  const handleDragStart = useCallback(
    (e: DragEvent, payload: DragPayload) => {
      e.dataTransfer.setData("application/json", JSON.stringify(payload));
      e.dataTransfer.effectAllowed = "copy";
    },
    [],
  );

  const handleDragOver = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
  }, []);

  const handleDropOnStage = useCallback(
    (e: DragEvent, stageOrder: number) => {
      e.preventDefault();
      const raw = e.dataTransfer.getData("application/json");
      if (!raw) return;
      try {
        const payload = JSON.parse(raw) as DragPayload;
        if (payload.type !== "palette-block") return;
        const block = createBlockFromPayload(payload.block);
        addBlockToStage(stageOrder, block);
      } catch {
        // invalid drop data – ignore
      }
    },
    [addBlockToStage],
  );

  const handleDropOnMonitor = useCallback(
    (e: DragEvent, cron = "0 9 * * 1-5") => {
      e.preventDefault();
      const raw = e.dataTransfer.getData("application/json");
      if (!raw) return;
      try {
        const payload = JSON.parse(raw) as DragPayload;
        if (payload.type !== "palette-block") return;
        const block = createBlockFromPayload(payload.block);
        addMonitorBlock(block, cron);
      } catch {
        // invalid drop data – ignore
      }
    },
    [addMonitorBlock],
  );

  return {
    handleDragStart,
    handleDragOver,
    handleDropOnStage,
    handleDropOnMonitor,
  };
}
