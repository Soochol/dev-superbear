import { create } from "zustand";

export type SearchModalTab = "search" | "watchlist" | "recent";

interface SearchModalState {
  isOpen: boolean;
  activeTab: SearchModalTab;
  openModal: () => void;
  closeModal: () => void;
  setActiveTab: (tab: SearchModalTab) => void;
}

export const useSearchModalStore = create<SearchModalState>()((set) => ({
  isOpen: false,
  activeTab: "search",
  openModal: () => set({ isOpen: true }),
  closeModal: () => set({ isOpen: false, activeTab: "search" }),
  setActiveTab: (tab) => set({ activeTab: tab }),
}));
