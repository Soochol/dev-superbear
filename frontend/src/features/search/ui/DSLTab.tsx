"use client";

import { useSearchStore } from "../model/search.store";
import { DSLEditor } from "./DSLEditor";
import { btnPrimary, btnSecondary } from "./styles";

export function DSLTab() {
  const dslCode = useSearchStore((s) => s.dslCode);
  const validationState = useSearchStore((s) => s.validationState);
  const hasCode = dslCode.trim().length > 0;

  return (
    <div className="flex flex-col gap-4">
      <DSLEditor placeholder="scan where volume > 1000000 sort by trade_value desc limit 50" />

      <div className="flex items-center gap-2">
        <button
          disabled={!hasCode}
          className={btnSecondary}
        >
          Validate
        </button>
        <button
          disabled={!hasCode}
          className={btnSecondary}
        >
          Explain in NL
        </button>
        <div className="flex-1" />
        <button
          disabled={!hasCode || validationState === "invalid"}
          className={btnPrimary}
        >
          Run Search
        </button>
      </div>
    </div>
  );
}
