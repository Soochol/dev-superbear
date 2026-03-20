import { usePresetStore } from "../model/preset.store";

beforeEach(() => {
  usePresetStore.setState(usePresetStore.getInitialState());
});

describe("Preset Store", () => {
  it("initializes with empty presets", () => {
    const state = usePresetStore.getState();
    expect(state.presets).toEqual([]);
    expect(state.isLoading).toBe(false);
  });

  it("sets presets", () => {
    usePresetStore.getState().setPresets([
      { id: "1", userId: "u1", name: "Test", dsl: "scan where volume > 100", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
    ]);
    expect(usePresetStore.getState().presets).toHaveLength(1);
  });

  it("adds a preset", () => {
    const preset = { id: "2", userId: "u1", name: "New", dsl: "scan where close > 50000", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" };
    usePresetStore.getState().addPreset(preset);
    expect(usePresetStore.getState().presets).toHaveLength(1);
    expect(usePresetStore.getState().presets[0].id).toBe("2");
  });

  it("removes a preset by id", () => {
    usePresetStore.getState().setPresets([
      { id: "1", userId: "u1", name: "A", dsl: "a", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
      { id: "2", userId: "u1", name: "B", dsl: "b", nlQuery: null, isPublic: false, createdAt: "", updatedAt: "" },
    ]);
    usePresetStore.getState().removePreset("1");
    expect(usePresetStore.getState().presets).toHaveLength(1);
    expect(usePresetStore.getState().presets[0].id).toBe("2");
  });
});
