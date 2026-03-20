"use client";

import { useSearchStore } from "../model/search.store";
import { PresetChips } from "./PresetChips";
import { AgentStatusIndicator } from "./AgentStatusIndicator";
import { btnPrimary } from "./styles";

export function NLTab() {
  const nlQuery = useSearchStore((s) => s.nlQuery);
  const setNlQuery = useSearchStore((s) => s.setNlQuery);
  const agentStatus = useSearchStore((s) => s.agentStatus);
  const isSearching = agentStatus !== "idle" && agentStatus !== "done" && agentStatus !== "error";

  return (
    <div className="flex flex-col gap-4">
      <textarea
        value={nlQuery}
        onChange={(e) => setNlQuery(e.target.value)}
        placeholder="자연어로 검색 조건을 입력하세요..."
        className="w-full h-24 bg-transparent border border-nexus-border rounded-lg p-3
                   text-nexus-text-primary placeholder:text-nexus-text-secondary/50
                   focus:outline-none focus:border-nexus-accent resize-none"
      />

      <PresetChips />

      <div className="flex items-center justify-between">
        <AgentStatusIndicator />
        <button
          disabled={isSearching || !nlQuery.trim()}
          className={btnPrimary}
        >
          Search
        </button>
      </div>
    </div>
  );
}
