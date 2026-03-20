import { create } from "zustand";
import type { SearchResult } from "@/entities/search-result";

interface StockListState {
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;
  selectedSymbol: string | null;
  setSelectedSymbol: (symbol: string | null) => void;
  watchlist: SearchResult[];
  addToWatchlist: (item: SearchResult) => void;
  removeFromWatchlist: (symbol: string) => void;
  isInWatchlist: (symbol: string) => boolean;
  recentStocks: SearchResult[];
  addToRecent: (item: SearchResult) => void;
}

export const useStockListStore = create<StockListState>()((set, get) => ({
  searchResults: [],
  setSearchResults: (results) => set({ searchResults: results }),
  selectedSymbol: null,
  setSelectedSymbol: (symbol) => set({ selectedSymbol: symbol }),
  watchlist: [],
  addToWatchlist: (item) =>
    set((state) => ({
      watchlist: state.watchlist.some((w) => w.symbol === item.symbol)
        ? state.watchlist
        : [...state.watchlist, item],
    })),
  removeFromWatchlist: (symbol) =>
    set((state) => ({
      watchlist: state.watchlist.filter((w) => w.symbol !== symbol),
    })),
  isInWatchlist: (symbol) => get().watchlist.some((w) => w.symbol === symbol),
  recentStocks: [],
  addToRecent: (item) =>
    set((state) => {
      const filtered = state.recentStocks.filter((r) => r.symbol !== item.symbol);
      return { recentStocks: [item, ...filtered].slice(0, 30) };
    }),
}));
