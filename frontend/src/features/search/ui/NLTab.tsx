"use client";

import { useSearchStore } from "../model/search.store";
import { PresetChips } from "./PresetChips";
import { AgentStatusIndicator } from "./AgentStatusIndicator";

export function NLTab() {
  const { nlQuery, setNlQuery, agentStatus } = useSearchStore();
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
          className="px-6 py-2 rounded-lg text-sm font-medium bg-nexus-accent text-white
                     hover:bg-nexus-accent/90 disabled:opacity-50 disabled:cursor-not-allowed
                     transition-colors"
        >
          Search
        </button>
      </div>
    </div>
  );
}
