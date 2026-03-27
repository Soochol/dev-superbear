import { apiGet, apiPost, apiDelete } from "@/shared/api/client";
import type { SearchResult } from "@/entities/search-result";

interface WatchlistItem {
  id: number;
  userId: number;
  symbol: string;
  name: string;
  createdAt: string;
}

function toSearchResult(item: WatchlistItem): SearchResult {
  return {
    symbol: item.symbol,
    name: item.name,
    matchedValue: item.symbol,
  };
}

export const watchlistApi = {
  async fetchWatchlist(): Promise<SearchResult[]> {
    const res = await apiGet<{ data: WatchlistItem[] }>("/api/v1/watchlist");
    return (res.data ?? []).map(toSearchResult);
  },

  async addItem(symbol: string, name: string): Promise<void> {
    await apiPost("/api/v1/watchlist", { symbol, name });
  },

  async removeItem(symbol: string): Promise<void> {
    await apiDelete(`/api/v1/watchlist/${symbol}`);
  },
};
