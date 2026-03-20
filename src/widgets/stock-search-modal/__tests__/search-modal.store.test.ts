/** @jest-environment jsdom */
import { useSearchModalStore } from "../model/search-modal.store";

describe("search-modal.store", () => {
  beforeEach(() => {
    useSearchModalStore.setState(useSearchModalStore.getInitialState());
  });

  it("starts closed with search tab", () => {
    const state = useSearchModalStore.getState();
    expect(state.isOpen).toBe(false);
    expect(state.activeTab).toBe("search");
  });

  it("openModal sets isOpen true", () => {
    useSearchModalStore.getState().openModal();
    expect(useSearchModalStore.getState().isOpen).toBe(true);
  });

  it("closeModal sets isOpen false and resets tab", () => {
    useSearchModalStore.getState().openModal();
    useSearchModalStore.getState().setActiveTab("watchlist");
    useSearchModalStore.getState().closeModal();
    const state = useSearchModalStore.getState();
    expect(state.isOpen).toBe(false);
    expect(state.activeTab).toBe("search");
  });

  it("setActiveTab changes tab", () => {
    useSearchModalStore.getState().setActiveTab("recent");
    expect(useSearchModalStore.getState().activeTab).toBe("recent");
  });
});
