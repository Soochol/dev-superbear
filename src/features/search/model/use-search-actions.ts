import { searchApi } from "../api/search-api";
import { useSearchStore } from "./search.store";
import type { AgentStatus, ValidationState } from "./types";
import type { SearchResult } from "@/entities/search-result";

interface SearchStoreState {
  nlQuery: string;
  dslCode: string;
  agentStatus: AgentStatus;
  agentMessage: string;
  explanation: string;
  results: SearchResult[];
}

type GetState = () => SearchStoreState;
type SetState = (partial: Record<string, unknown>) => void;

function extractErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

export function createSearchActions(getState: GetState, setState: SetState) {
  function setError(err: unknown, fallback = "Unknown error"): void {
    setState({
      agentStatus: "error" satisfies AgentStatus,
      agentMessage: extractErrorMessage(err, fallback),
    });
  }

  async function runNLSearch(): Promise<void> {
    const { nlQuery } = getState();
    setState({ agentStatus: "interpreting", agentMessage: "쿼리 분석 중..." });

    try {
      for await (const event of searchApi.nlSearchStream(nlQuery)) {
        switch (event.type) {
          case "thinking":
            setState({ agentStatus: "interpreting", agentMessage: event.message });
            break;
          case "tool_call":
          case "tool_result":
            setState({ agentStatus: "building", agentMessage: event.message });
            break;
          case "dsl_ready":
            setState({
              dslCode: event.dsl,
              explanation: event.explanation,
              agentStatus: "scanning",
              agentMessage: "검색 중...",
            });
            break;
          case "done":
            setState({
              results: event.results,
              agentStatus: "done",
              agentMessage: `${event.count}개 종목 발견`,
            });
            break;
          case "error":
            setState({ agentStatus: "error", agentMessage: event.message });
            break;
        }
      }
    } catch (err) {
      setError(err);
    }
  }

  async function runDSLSearch(): Promise<void> {
    const { dslCode } = getState();
    setState({ agentStatus: "scanning", agentMessage: "Scanning stocks..." });

    try {
      const response = await searchApi.dslSearch(dslCode);
      setState({
        results: response.results,
        agentStatus: "done",
        agentMessage: `${response.results.length}개 종목 발견`,
      });
    } catch (err) {
      console.error("runDSLSearch failed:", err);
      setError(err);
    }
  }

  async function validateDSL(): Promise<void> {
    const { dslCode } = getState();

    try {
      const response = await searchApi.validate(dslCode);
      const state: ValidationState = response.valid ? "valid" : "invalid";
      setState({
        validationState: state,
        validationMessage: response.error ?? "",
      });
    } catch (err) {
      console.error("validateDSL failed:", err);
      setState({
        validationState: "invalid" satisfies ValidationState,
        validationMessage: extractErrorMessage(err, "Validation failed"),
      });
    }
  }

  async function explainDSL(): Promise<void> {
    const { dslCode } = getState();

    try {
      const response = await searchApi.explain(dslCode);
      setState({ explanation: response.explanation });
    } catch (err) {
      console.error("explainDSL failed:", err);
      setState({
        explanation: extractErrorMessage(err, "설명을 가져올 수 없습니다. 다시 시도해 주세요."),
      });
    }
  }

  return { runNLSearch, runDSLSearch, validateDSL, explainDSL };
}

export function useSearchActions() {
  return createSearchActions(
    useSearchStore.getState,
    useSearchStore.setState,
  );
}
