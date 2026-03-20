import { create } from "zustand";

interface SidebarState {
  isPinned: boolean;
  isExpanded: boolean;
  togglePin: () => void;
  setExpanded: (expanded: boolean) => void;
}

function readPinned(): boolean {
  if (typeof window === "undefined") return false;
  return localStorage.getItem("sidebar-pinned") === "true";
}

export const useSidebarStore = create<SidebarState>()((set, get) => ({
  isPinned: readPinned(),
  isExpanded: readPinned(),
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
