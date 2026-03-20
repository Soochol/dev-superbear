import { create } from "zustand";

interface SidebarState {
  isPinned: boolean;
  isExpanded: boolean;
  _hydrated: boolean;
  togglePin: () => void;
  setExpanded: (expanded: boolean) => void;
  hydrate: () => void;
}

export const useSidebarStore = create<SidebarState>()((set, get) => ({
  isPinned: false,
  isExpanded: false,
  _hydrated: false,
  hydrate: () => {
    if (get()._hydrated) return;
    set({ _hydrated: true });
    const pinned = localStorage.getItem("sidebar-pinned") === "true";
    if (pinned) set({ isPinned: true, isExpanded: true });
  },
  togglePin: () => {
    const next = !get().isPinned;
    set({ isPinned: next, isExpanded: next });
    localStorage.setItem("sidebar-pinned", String(next));
  },
  setExpanded: (expanded) => {
    if (!get().isPinned && get().isExpanded !== expanded) {
      set({ isExpanded: expanded });
    }
  },
}));
