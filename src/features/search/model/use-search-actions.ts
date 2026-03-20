import { searchApi } from "../api/search-api";
import { useSearchStore } from "./search.store";
import type { AgentStatus, ValidationState } from "./types";
import type { SearchResult } from "@/entities/search-result";

interface SearchStoreState {
  nlQuery: string;
  dslCode: string;
  agentStatus: AgentStatus;
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
    setState({ agentStatus: "interpreting", agentMessage: "Interpreting query..." });

    try {
      const response = await searchApi.nlSearch(nlQuery);
      setState({
        dslCode: response.dsl,
        agentStatus: "scanning",
        agentMessage: "Scanning stocks...",
      });
      setState({
        results: response.results,
        agentStatus: "done",
        agentMessage: `${response.results.length}개 종목 발견`,
      });
    } catch (err) {
      console.error("runNLSearch failed:", err);
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
      setState({ explanation: "" });
      console.error("explainDSL failed:", err);
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
