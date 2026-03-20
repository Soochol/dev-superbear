/**
 * @jest-environment jsdom
 */
import { useSidebarStore } from "@/shared/model/sidebar.store";

describe("sidebarStore", () => {
  beforeEach(() => {
    localStorage.clear();
    useSidebarStore.setState(useSidebarStore.getInitialState());
  });

  it("starts collapsed and unpinned", () => {
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(false);
    expect(state.isExpanded).toBe(false);
  });

  it("setExpanded changes isExpanded when not pinned", () => {
    useSidebarStore.getState().setExpanded(true);
    expect(useSidebarStore.getState().isExpanded).toBe(true);

    useSidebarStore.getState().setExpanded(false);
    expect(useSidebarStore.getState().isExpanded).toBe(false);
  });

  it("setExpanded is ignored when pinned", () => {
    useSidebarStore.setState({ isPinned: true, isExpanded: true });

    useSidebarStore.getState().setExpanded(false);
    expect(useSidebarStore.getState().isExpanded).toBe(true);
  });

  it("togglePin pins and expands", () => {
    useSidebarStore.getState().togglePin();
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(true);
    expect(state.isExpanded).toBe(true);
  });

  it("togglePin unpins and collapses", () => {
    useSidebarStore.setState({ isPinned: true, isExpanded: true });

    useSidebarStore.getState().togglePin();
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(false);
    expect(state.isExpanded).toBe(false);
  });

  it("persists isPinned to localStorage", () => {
    useSidebarStore.getState().togglePin();
    expect(localStorage.getItem("sidebar-pinned")).toBe("true");

    useSidebarStore.getState().togglePin();
    expect(localStorage.getItem("sidebar-pinned")).toBe("false");
  });

  it("hydrate restores pinned state from localStorage", () => {
    localStorage.setItem("sidebar-pinned", "true");
    useSidebarStore.getState().hydrate();
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(true);
    expect(state.isExpanded).toBe(true);
  });

  it("hydrate does nothing when localStorage has no pinned value", () => {
    useSidebarStore.getState().hydrate();
    const state = useSidebarStore.getState();
    expect(state.isPinned).toBe(false);
    expect(state.isExpanded).toBe(false);
  });
});
