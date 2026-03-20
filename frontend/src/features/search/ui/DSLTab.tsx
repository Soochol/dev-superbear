"use client";

import { useSearchStore } from "../model/search.store";
import { DSLEditor } from "./DSLEditor";

export function DSLTab() {
  const { dslCode, validationState } = useSearchStore();
  const hasCode = dslCode.trim().length > 0;

  return (
    <div className="flex flex-col gap-4">
      <DSLEditor placeholder="scan where volume > 1000000 sort by trade_value desc limit 50" />

      <div className="flex items-center gap-2">
        <button
          disabled={!hasCode}
          className="px-4 py-2 rounded-lg text-sm font-medium
                     bg-nexus-border text-nexus-text-primary
                     hover:bg-nexus-border/80 disabled:opacity-50 disabled:cursor-not-allowed
                     transition-colors"
        >
          Validate
        </button>
        <button
          disabled={!hasCode}
          className="px-4 py-2 rounded-lg text-sm font-medium
                     bg-nexus-border text-nexus-text-primary
                     hover:bg-nexus-border/80 disabled:opacity-50 disabled:cursor-not-allowed
                     transition-colors"
        >
          Explain in NL
        </button>
        <div className="flex-1" />
        <button
          disabled={!hasCode || validationState === "invalid"}
          className="px-6 py-2 rounded-lg text-sm font-medium bg-nexus-accent text-white
                     hover:bg-nexus-accent/90 disabled:opacity-50 disabled:cursor-not-allowed
                     transition-colors"
        >
          Run Search
        </button>
      </div>
    </div>
  );
}
