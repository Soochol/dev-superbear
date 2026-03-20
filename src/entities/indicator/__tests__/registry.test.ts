import { INDICATOR_REGISTRY, getIndicator, getOverlayIndicators, getPanelIndicators } from "../model/registry";

describe("indicator registry", () => {
  it("has 8 indicators registered", () => {
    expect(INDICATOR_REGISTRY).toHaveLength(8);
  });

  it("getIndicator returns correct config", () => {
    const rsi = getIndicator("rsi");
    expect(rsi).toBeDefined();
    expect(rsi!.name).toBe("RSI(14)");
    expect(rsi!.type).toBe("panel");
  });

  it("getOverlayIndicators filters correctly", () => {
    const overlays = getOverlayIndicators(["ma20", "rsi", "bb"]);
    expect(overlays).toHaveLength(2); // ma20 and bb
    expect(overlays.every((i) => i.type === "overlay")).toBe(true);
  });

  it("getPanelIndicators filters correctly", () => {
    const panels = getPanelIndicators(["ma20", "rsi", "macd"]);
    expect(panels).toHaveLength(2); // rsi and macd
    expect(panels.every((i) => i.type === "panel")).toBe(true);
  });

  it("returns undefined for unknown id", () => {
    expect(getIndicator("unknown")).toBeUndefined();
  });
});
