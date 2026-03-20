"use client";

import type { DragEvent } from "react";
import { MonitorCard, type AgentBlock } from "@/entities/agent-block";
import { usePipelineStore } from "../../model/pipeline.store";
import { usePipelineDragDrop } from "../../lib/usePipelineDragDrop";

interface MonitoringSectionProps {
  onEditBlock?: (block: AgentBlock) => void;
}

export default function MonitoringSection({
  onEditBlock,
}: MonitoringSectionProps) {
  const monitorBlocks = usePipelineStore((s) => s.monitorBlocks);
  const removeMonitorBlock = usePipelineStore((s) => s.removeMonitorBlock);
  const updateMonitorCron = usePipelineStore((s) => s.updateMonitorCron);
  const toggleMonitorEnabled = usePipelineStore((s) => s.toggleMonitorEnabled);
  const { handleDragOver, handleDropOnMonitor } = usePipelineDragDrop();

  return (
    <div className="border border-nexus-border rounded-lg p-4">
      <h3 className="text-sm font-semibold text-nexus-text-primary uppercase tracking-wider mb-3">
        Monitoring
      </h3>

      <div
        onDragOver={handleDragOver}
        onDrop={(e: DragEvent<HTMLDivElement>) => handleDropOnMonitor(e)}
        className="bg-nexus-bg/50 border border-dashed border-nexus-border rounded-md p-3 min-h-[72px] transition-colors hover:border-nexus-accent/30"
      >
        {monitorBlocks.length === 0 ? (
          <p className="text-xs text-nexus-text-muted text-center py-3">
            Drop blocks here to create monitors
          </p>
        ) : (
          <div className="flex flex-wrap gap-3">
            {monitorBlocks.map((m) => (
              <MonitorCard
                key={m.id}
                id={m.id}
                block={m.block}
                cron={m.cron}
                enabled={m.enabled}
                onEdit={onEditBlock}
                onDelete={removeMonitorBlock}
                onCronChange={updateMonitorCron}
                onToggleEnabled={toggleMonitorEnabled}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
