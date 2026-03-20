"use client";

import { useSearchStore } from "../model/search.store";
import type { AgentStatus } from "../model/types";

const STATUS_CONFIG: Record<AgentStatus, { label: string; color: string; animate: boolean }> = {
  idle: { label: "", color: "", animate: false },
  interpreting: { label: "Interpreting query...", color: "text-nexus-warning", animate: true },
  building: { label: "Building DSL...", color: "text-nexus-accent", animate: true },
  scanning: { label: "Scanning stocks...", color: "text-nexus-accent", animate: true },
  done: { label: "Search complete", color: "text-nexus-success", animate: false },
  error: { label: "Error occurred", color: "text-nexus-failure", animate: false },
};

export function AgentStatusIndicator() {
  const agentStatus = useSearchStore((s) => s.agentStatus);
  const agentMessage = useSearchStore((s) => s.agentMessage);
  if (agentStatus === "idle") return null;

  const config = STATUS_CONFIG[agentStatus];

  return (
    <div className={`flex items-center gap-2 text-sm ${config.color}`}>
      {config.animate && (
        <span className="inline-block w-2 h-2 rounded-full bg-current animate-pulse" />
      )}
      <span>{agentMessage || config.label}</span>
    </div>
  );
}
