import { create } from "zustand";

interface SidebarState {
  isPinned: boolean;
  isExpanded: boolean;
  togglePin: () => void;
  setExpanded: (expanded: boolean) => void;
  hydrate: () => void;
}

export const useSidebarStore = create<SidebarState>()((set, get) => ({
  isPinned: false,
  isExpanded: false,
  hydrate: () => {
    if (typeof window !== "undefined") {
      const pinned = localStorage.getItem("sidebar-pinned") === "true";
      if (pinned) set({ isPinned: true, isExpanded: true });
    }
  },
  togglePin: () => {
    const next = !get().isPinned;
    set({ isPinned: next, isExpanded: next });
    if (typeof window !== "undefined") {
      localStorage.setItem("sidebar-pinned", String(next));
    }
  },
  setExpanded: (expanded) => {
    if (!get().isPinned) {
      set({ isExpanded: expanded });
    }
  },
}));
