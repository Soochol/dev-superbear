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

export function createSearchActions(getState: GetState, setState: SetState) {
  async function runNLSearch() {
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
      setState({
        agentStatus: "error",
        agentMessage: err instanceof Error ? err.message : "Unknown error",
      });
    }
  }

  async function runDSLSearch() {
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
      setState({
        agentStatus: "error",
        agentMessage: err instanceof Error ? err.message : "Unknown error",
      });
    }
  }

  async function validateDSL() {
    const { dslCode } = getState();

    try {
      const response = await searchApi.validate(dslCode);
      setState({
        validationState: response.valid ? "valid" as ValidationState : "invalid" as ValidationState,
        validationMessage: response.error ?? "",
      });
    } catch (err) {
      setState({
        validationState: "invalid" as ValidationState,
        validationMessage: err instanceof Error ? err.message : "Validation failed",
      });
    }
  }

  async function explainDSL(): Promise<string | null> {
    const { dslCode } = getState();

    try {
      const response = await searchApi.explain(dslCode);
      return response.explanation;
    } catch {
      return null;
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
