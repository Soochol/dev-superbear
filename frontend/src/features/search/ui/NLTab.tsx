"use client";

import { useSearchStore } from "../model/search.store";
import { searchApi } from "../api/search-api";
import { PresetChips } from "./PresetChips";
import { AgentStatusIndicator } from "./AgentStatusIndicator";
import { btnPrimary } from "./styles";

export function NLTab() {
  const nlQuery = useSearchStore((s) => s.nlQuery);
  const setNlQuery = useSearchStore((s) => s.setNlQuery);
  const agentStatus = useSearchStore((s) => s.agentStatus);
  const setAgentStatus = useSearchStore((s) => s.setAgentStatus);
  const setAgentMessage = useSearchStore((s) => s.setAgentMessage);
  const setDslCode = useSearchStore((s) => s.setDslCode);
  const setResults = useSearchStore((s) => s.setResults);
  const isSearching =
    agentStatus !== "idle" && agentStatus !== "done" && agentStatus !== "error";

  async function handleSearch() {
    try {
      setAgentStatus("interpreting");
      setAgentMessage("자연어를 DSL로 변환 중...");
      const response = await searchApi.nlSearch(nlQuery);
      setDslCode(response.dsl);
      setResults(response.results);
      setAgentMessage(response.explanation);
      setAgentStatus("done");
    } catch (err) {
      setAgentStatus("error");
      setAgentMessage(
        err instanceof Error ? err.message : "검색 중 오류가 발생했습니다"
      );
    }
  }

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
          onClick={handleSearch}
        >
          Search
        </button>
      </div>
    </div>
  );
}
