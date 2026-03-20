"use client";

import type { AgentBlock } from "@/entities/agent-block/model/types";
import BlockCard from "./BlockCard";

interface MonitorCardProps {
  id: string;
  block: AgentBlock;
  cron: string;
  enabled: boolean;
  onEdit?: (block: AgentBlock) => void;
  onDelete?: (id: string) => void;
  onCronChange?: (id: string, cron: string) => void;
  onToggleEnabled?: (id: string) => void;
}

export default function MonitorCard({
  id,
  block,
  cron,
  enabled,
  onEdit,
  onDelete,
  onCronChange,
  onToggleEnabled,
}: MonitorCardProps) {
  return (
    <div className="space-y-2">
      <BlockCard
        block={block}
        onEdit={onEdit}
        onDelete={onDelete ? () => onDelete(id) : undefined}
      />
      <div className="flex items-center gap-2 px-1">
        <input
          type="text"
          value={cron}
          onChange={(e) => onCronChange?.(id, e.target.value)}
          className="bg-nexus-bg border border-nexus-border rounded-md px-2 py-1 text-xs font-mono text-nexus-text-primary w-32 focus:outline-none focus:border-nexus-accent"
          placeholder="cron"
          aria-label="Cron expression"
        />
        <button
          type="button"
          onClick={() => onToggleEnabled?.(id)}
          className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors ${
            enabled ? "bg-nexus-accent" : "bg-nexus-border"
          }`}
          role="switch"
          aria-checked={enabled}
          aria-label="Toggle monitor"
        >
          <span
            className={`pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform ${
              enabled ? "translate-x-4" : "translate-x-0"
            }`}
          />
        </button>
      </div>
    </div>
  );
}
