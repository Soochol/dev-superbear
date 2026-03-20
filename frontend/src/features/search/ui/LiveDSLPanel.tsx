"use client";

import { useSearchStore } from "../model/search.store";
import { highlightDSL } from "@/shared/lib/dsl/highlight";

export function LiveDSLPanel() {
  const { dslCode, validationState } = useSearchStore();
  const hasCode = dslCode.trim().length > 0;
  const tokens = hasCode ? highlightDSL(dslCode) : [];

  const handleCopy = async () => {
    await navigator.clipboard.writeText(dslCode);
  };

  return (
    <div className="bg-nexus-surface border border-nexus-border rounded-lg">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-nexus-border">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-nexus-accent uppercase tracking-wider">
            LIVE DSL
          </span>
          {hasCode && (
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${
                validationState === "valid"
                  ? "bg-green-500/20 text-green-400"
                  : validationState === "invalid"
                  ? "bg-red-500/20 text-red-400"
                  : "bg-yellow-500/20 text-yellow-400"
              }`}
            >
              {validationState === "valid"
                ? "Validated"
                : validationState === "invalid"
                ? "Invalid"
                : "Not Validated"}
            </span>
          )}
        </div>

        {hasCode && (
          <div className="flex items-center gap-2">
            <button
              onClick={handleCopy}
              aria-label="Copy"
              className="px-3 py-1 text-xs rounded bg-nexus-border text-nexus-text-secondary
                         hover:text-nexus-text-primary transition-colors"
            >
              Copy
            </button>
            <button
              aria-label="Run Search"
              className="px-3 py-1 text-xs rounded bg-nexus-accent/20 text-nexus-accent
                         hover:bg-nexus-accent/30 transition-colors"
            >
              Run
            </button>
          </div>
        )}
      </div>

      {/* Code display */}
      <div className="p-4 font-mono text-sm min-h-[60px]">
        {hasCode ? (
          <pre className="whitespace-pre-wrap">
            {tokens.map((token, i) => (
              <span key={i} className={token.className}>
                {token.text}
              </span>
            ))}
          </pre>
        ) : (
          <span className="text-nexus-text-secondary/50 italic">
            DSL이 없습니다 — NL 모드에서 검색하거나 DSL 모드에서 직접 작성하세요
          </span>
        )}
      </div>
    </div>
  );
}
