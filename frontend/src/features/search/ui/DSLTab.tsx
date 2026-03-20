"use client";

import { useState } from "react";
import { useSearchStore } from "../model/search.store";
import { searchApi } from "../api/search-api";
import { DSLEditor } from "./DSLEditor";
import { btnPrimary, btnSecondary } from "./styles";

export function DSLTab() {
  const dslCode = useSearchStore((s) => s.dslCode);
  const validationState = useSearchStore((s) => s.validationState);
  const setValidationState = useSearchStore((s) => s.setValidationState);
  const setResults = useSearchStore((s) => s.setResults);
  const hasCode = dslCode.trim().length > 0;
  const [isLoading, setIsLoading] = useState(false);

  async function handleValidate() {
    try {
      setIsLoading(true);
      const response = await searchApi.validate(dslCode);
      setValidationState(
        response.valid ? "valid" : "invalid",
        response.error ?? ""
      );
    } catch (err) {
      setValidationState(
        "invalid",
        err instanceof Error ? err.message : "검증 중 오류 발생"
      );
    } finally {
      setIsLoading(false);
    }
  }

  async function handleExplain() {
    try {
      setIsLoading(true);
      await searchApi.explain(dslCode);
    } catch {
      // silently handle
    } finally {
      setIsLoading(false);
    }
  }

  async function handleRunSearch() {
    try {
      setIsLoading(true);
      const response = await searchApi.dslSearch(dslCode);
      setResults(response.results);
    } catch {
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <DSLEditor placeholder="scan where volume > 1000000 sort by trade_value desc limit 50" />

      <div className="flex items-center gap-2">
        <button
          disabled={!hasCode || isLoading}
          className={btnSecondary}
          onClick={handleValidate}
        >
          Validate
        </button>
        <button
          disabled={!hasCode || isLoading}
          className={btnSecondary}
          onClick={handleExplain}
        >
          Explain in NL
        </button>
        <div className="flex-1" />
        <button
          disabled={!hasCode || validationState === "invalid" || isLoading}
          className={btnPrimary}
          onClick={handleRunSearch}
        >
          Run Search
        </button>
      </div>
    </div>
  );
}
