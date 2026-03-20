import { useChartStore } from "../model/chart.store";

describe("Chart Store", () => {
  beforeEach(() => {
    useChartStore.setState(useChartStore.getInitialState());
  });

  it("initializes with default timeframe 1D", () => {
    expect(useChartStore.getState().timeframe).toBe("1D");
  });

  it("sets current stock info", () => {
    useChartStore.getState().setCurrentStock({
      symbol: "005930",
      name: "Samsung Electronics",
      price: 78400,
      change: 1600,
      changePct: 2.08,
    });
    const state = useChartStore.getState();
    expect(state.currentStock?.symbol).toBe("005930");
    expect(state.currentStock?.price).toBe(78400);
  });

  it("switches timeframe", () => {
    useChartStore.getState().setTimeframe("1W");
    expect(useChartStore.getState().timeframe).toBe("1W");
  });

  it("toggles indicator overlays", () => {
    useChartStore.getState().toggleIndicator("ma20");
    expect(useChartStore.getState().activeIndicators).not.toContain("ma20");
    useChartStore.getState().toggleIndicator("ma20");
    expect(useChartStore.getState().activeIndicators).toContain("ma20");
  });

  it("manages sub-indicator panels", () => {
    useChartStore.getState().toggleSubIndicator("rsi");
    expect(useChartStore.getState().activeSubIndicators).toContain("rsi");
  });

  it("tracks candle data loading state", () => {
    useChartStore.getState().setIsLoading(true);
    expect(useChartStore.getState().isLoading).toBe(true);
  });
});
