import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "../model/chart.store";

describe("Search -> Chart Navigation", () => {
  beforeEach(() => {
    useStockListStore.setState({
      ...useStockListStore.getInitialState(),
      searchResults: [],
      selectedSymbol: null,
      watchlist: [],
      recentStocks: [],
    });
    useChartStore.setState(useChartStore.getInitialState());
  });

  it("stock list store receives search results when navigating", () => {
    const mockResults = [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      { symbol: "247540", name: "ecoprobm", matchedValue: 15200000 },
    ];

    useStockListStore.getState().setSearchResults(mockResults);
    useStockListStore.getState().setSelectedSymbol("005930");

    const state = useStockListStore.getState();
    expect(state.searchResults).toHaveLength(2);
    expect(state.selectedSymbol).toBe("005930");
  });

  it("chart store updates currentStock from entity store", () => {
    useStockListStore.getState().setSelectedSymbol("005930");

    useChartStore.getState().setCurrentStock({
      symbol: "005930",
      name: "Samsung Electronics",
      price: 0,
      change: 0,
      changePct: 0,
    });

    expect(useChartStore.getState().currentStock?.symbol).toBe("005930");
  });

  it("recent stocks are updated on navigation", () => {
    const item = { symbol: "005930", name: "Samsung", matchedValue: 0 };
    useStockListStore.getState().addToRecent(item);

    expect(useStockListStore.getState().recentStocks).toContainEqual(item);
  });

  it("recent stocks are capped at 30 items", () => {
    for (let i = 0; i < 35; i++) {
      useStockListStore.getState().addToRecent({
        symbol: String(i).padStart(6, "0"),
        name: `Stock ${i}`,
        matchedValue: 0,
      });
    }

    expect(useStockListStore.getState().recentStocks).toHaveLength(30);
  });
});
