import { create } from "zustand";
import type { SearchResult } from "@/entities/search-result";
import { watchlistApi } from "@/features/watchlist/api/watchlist-api";
import { logger } from "@/shared/lib/logger";

interface StockListState {
  searchResults: SearchResult[];
  setSearchResults: (results: SearchResult[]) => void;

  selectedSymbol: string | null;
  setSelectedSymbol: (symbol: string | null) => void;

  watchlist: SearchResult[];
  watchlistLoaded: boolean;
  loadWatchlist: () => Promise<void>;
  addToWatchlist: (item: SearchResult) => Promise<void>;
  removeFromWatchlist: (symbol: string) => Promise<void>;
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
  loadWatchlist: async () => {
    if (get().watchlistLoaded) return;
    try {
      const items = await watchlistApi.fetchWatchlist();
      set({ watchlist: items, watchlistLoaded: true });
    } catch (err) {
      logger.error("Failed to load watchlist", { error: err });
    }
  },
  addToWatchlist: async (item) => {
    if (get().watchlist.some((w) => w.symbol === item.symbol)) return;
    try {
      await watchlistApi.addItem(item.symbol, item.name);
      set((state) => ({ watchlist: [...state.watchlist, item] }));
    } catch (err) {
      logger.error("Failed to add to watchlist", { error: err });
    }
  },
  removeFromWatchlist: async (symbol) => {
    try {
      await watchlistApi.removeItem(symbol);
      set((state) => ({
        watchlist: state.watchlist.filter((w) => w.symbol !== symbol),
      }));
    } catch (err) {
      logger.error("Failed to remove from watchlist", { error: err });
    }
  },
  isInWatchlist: (symbol) => get().watchlist.some((w) => w.symbol === symbol),

  recentStocks: [],
  addToRecent: (item) =>
    set((state) => {
      const filtered = state.recentStocks.filter((r) => r.symbol !== item.symbol);
      return { recentStocks: [item, ...filtered].slice(0, 30) };
    }),
}));
