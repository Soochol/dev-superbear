import { create } from "zustand";
import type { SearchResult } from "@/entities/search-result";

interface StockListState {
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;

  selectedSymbol: string | null;
  setSelectedSymbol: (symbol: string | null) => void;

  watchlist: SearchResult[];
  watchlistLoaded: boolean;
  watchlistError: boolean;
  setWatchlist: (items: SearchResult[]) => void;
  setWatchlistLoaded: (v: boolean) => void;
  setWatchlistError: (v: boolean) => void;
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
  watchlistLoaded: false,
  watchlistError: false,
  setWatchlist: (items) => set({ watchlist: items }),
  setWatchlistLoaded: (v) => set({ watchlistLoaded: v }),
  setWatchlistError: (v) => set({ watchlistError: v }),
  addToWatchlist: (item) =>
    set((state) => {
      if (state.watchlist.some((w) => w.symbol === item.symbol)) return state;
      return { watchlist: [...state.watchlist, item] };
    }),
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
