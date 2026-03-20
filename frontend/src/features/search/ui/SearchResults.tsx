"use client";

import { useSearchStore } from "../model/search.store";
import { ResultsTable } from "./ResultsTable";

export function SearchResults() {
  const results = useSearchStore((s) => s.results);
  const agentStatus = useSearchStore((s) => s.agentStatus);
  const isSearching = agentStatus !== "idle" && agentStatus !== "done" && agentStatus !== "error";

  if (isSearching) {
    return (
      <div className="flex items-center justify-center py-12 text-nexus-text-secondary">
        <span className="inline-block w-4 h-4 border-2 border-nexus-accent border-t-transparent rounded-full animate-spin mr-3" />
        검색 중...
      </div>
    );
  }

  if (results.length === 0) {
    return (
      <div className="flex items-center justify-center py-12 text-nexus-text-secondary">
        검색 결과가 없습니다
      </div>
    );
  }

  return (
    <div className="bg-nexus-surface border border-nexus-border rounded-lg overflow-hidden">
      <div className="flex items-center justify-between px-4 py-2 border-b border-nexus-border">
        <span className="text-sm text-nexus-text-secondary">
          {results.length}개 종목
        </span>
      </div>
      <ResultsTable />
    </div>
  );
}
