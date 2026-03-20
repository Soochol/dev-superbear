import { useStockListStore } from "../model/stock-list.store";

describe("StockListStore", () => {
  beforeEach(() => {
    useStockListStore.setState(useStockListStore.getInitialState());
  });

  it("manages search results", () => {
    const results = [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
    ];
    useStockListStore.getState().setSearchResults(results);
    expect(useStockListStore.getState().searchResults).toEqual(results);
  });

  it("manages selected symbol", () => {
    useStockListStore.getState().setSelectedSymbol("005930");
    expect(useStockListStore.getState().selectedSymbol).toBe("005930");
  });

  it("adds and removes from watchlist", () => {
    const item = { symbol: "005930", name: "Samsung", matchedValue: 0 };
    useStockListStore.getState().addToWatchlist(item);
    expect(useStockListStore.getState().isInWatchlist("005930")).toBe(true);
    useStockListStore.getState().removeFromWatchlist("005930");
    expect(useStockListStore.getState().isInWatchlist("005930")).toBe(false);
  });

  it("prevents duplicate watchlist entries", () => {
    const item = { symbol: "005930", name: "Samsung", matchedValue: 0 };
    useStockListStore.getState().addToWatchlist(item);
    useStockListStore.getState().addToWatchlist(item);
    expect(useStockListStore.getState().watchlist).toHaveLength(1);
  });

  it("manages recent stocks with cap at 30", () => {
    for (let i = 0; i < 35; i++) {
      useStockListStore.getState().addToRecent({
        symbol: String(i).padStart(6, "0"),
        name: `Stock ${i}`,
        matchedValue: 0,
      });
    }
    expect(useStockListStore.getState().recentStocks).toHaveLength(30);
  });

  it("deduplicates recent stocks (most recent first)", () => {
    const item = { symbol: "005930", name: "Samsung", matchedValue: 0 };
    const item2 = { symbol: "000660", name: "SK Hynix", matchedValue: 0 };
    useStockListStore.getState().addToRecent(item);
    useStockListStore.getState().addToRecent(item2);
    useStockListStore.getState().addToRecent(item);
    const recent = useStockListStore.getState().recentStocks;
    expect(recent).toHaveLength(2);
    expect(recent[0].symbol).toBe("005930");
  });
});
