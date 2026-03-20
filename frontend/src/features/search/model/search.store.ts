import { create } from "zustand";
import type { SearchTab, AgentStatus, ValidationState } from "./types";
import type { SearchResult } from "@/entities/search-result";

interface SearchState {
  activeTab: SearchTab;
  setActiveTab: (tab: SearchTab) => void;
  nlQuery: string;
  setNlQuery: (query: string) => void;
  dslCode: string;
  setDslCode: (code: string) => void;
  agentStatus: AgentStatus;
  setAgentStatus: (status: AgentStatus) => void;
  agentMessage: string;
  setAgentMessage: (msg: string) => void;
  validationState: ValidationState;
  validationMessage: string;
  setValidationState: (state: ValidationState, message?: string) => void;
  results: SearchResult[];
  setResults: (results: SearchResult[]) => void;
  isSearching: boolean;
  setIsSearching: (v: boolean) => void;
  selectedPresetId: string | null;
  setSelectedPresetId: (id: string | null) => void;
}

export const useSearchStore = create<SearchState>()((set) => ({
  activeTab: "nl",
  setActiveTab: (tab) => set({ activeTab: tab }),
  nlQuery: "",
  setNlQuery: (query) => set({ nlQuery: query }),
  dslCode: "",
  setDslCode: (code) => set({ dslCode: code }),
  agentStatus: "idle",
  setAgentStatus: (status) => set({ agentStatus: status }),
  agentMessage: "",
  setAgentMessage: (msg) => set({ agentMessage: msg }),
  validationState: "none",
  validationMessage: "",
  setValidationState: (state, message = "") =>
    set({ validationState: state, validationMessage: message }),
  results: [],
  setResults: (results) => set({ results }),
  isSearching: false,
  setIsSearching: (v) => set({ isSearching: v }),
  selectedPresetId: null,
  setSelectedPresetId: (id) => set({ selectedPresetId: id }),
}));
