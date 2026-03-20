"use client";

import { useSearchStore } from "../model/search.store";
import { useSearchActions } from "../model/use-search-actions";
import { DSLEditor } from "./DSLEditor";
import { btnPrimary, btnSecondary } from "./styles";

export function DSLTab() {
  const dslCode = useSearchStore((s) => s.dslCode);
  const validationState = useSearchStore((s) => s.validationState);
  const agentStatus = useSearchStore((s) => s.agentStatus);
  const explanation = useSearchStore((s) => s.explanation);
  const hasCode = dslCode.trim().length > 0;
  const isSearching = agentStatus !== "idle" && agentStatus !== "done" && agentStatus !== "error";
  const { runDSLSearch, validateDSL, explainDSL } = useSearchActions();

  return (
    <div className="flex flex-col gap-4">
      <DSLEditor placeholder="scan where volume > 1000000 sort by trade_value desc limit 50" />

      <div className="flex items-center gap-2">
        <button
          disabled={!hasCode}
          className={btnSecondary}
          onClick={validateDSL}
        >
          Validate
        </button>
        <button
          disabled={!hasCode}
          className={btnSecondary}
          onClick={explainDSL}
        >
          Explain in NL
        </button>
        <div className="flex-1" />
        <button
          disabled={!hasCode || validationState === "invalid" || isSearching}
          className={btnPrimary}
          onClick={runDSLSearch}
        >
          Run Search
        </button>
      </div>

      {explanation && (
        <div className="p-3 bg-nexus-surface border border-nexus-border rounded-lg text-sm text-nexus-text-secondary">
          {explanation}
        </div>
      )}
    </div>
  );
}
