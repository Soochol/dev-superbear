"use client";

import { useEffect, useCallback, useMemo } from "react";
import { useSearchModalStore } from "../model/search-modal.store";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";
import { SearchSideNav } from "./SearchSideNav";
import { SearchContent } from "./SearchContent";
import type { SearchResult } from "@/entities/search-result";

const TAB_TITLES = {
  search: "종목 검색",
  watchlist: "관심 종목",
  recent: "최근 본 종목",
} as const;

export function StockSearchModal() {
  const { isOpen, activeTab, closeModal, setActiveTab } = useSearchModalStore();
  const { searchResults, watchlist, recentStocks, addToRecent, isInWatchlist, addToWatchlist, removeFromWatchlist } =
    useStockListStore();
  const setCurrentStock = useChartStore((s) => s.setCurrentStock);

  const handleEscape = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") closeModal();
    },
    [closeModal],
  );

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("keydown", handleEscape);
      return () => document.removeEventListener("keydown", handleEscape);
    }
  }, [isOpen, handleEscape]);

  const handleSelect = useCallback(
    (item: SearchResult) => {
      setCurrentStock({
        symbol: item.symbol,
        name: item.name,
        price: item.close ?? 0,
        change: item.change ?? 0,
        changePct: item.changePct ?? 0,
      });
      addToRecent(item);
      closeModal();
    },
    [setCurrentStock, addToRecent, closeModal],
  );

  const handleToggleWatchlist = useCallback(
    (item: SearchResult) => {
      if (isInWatchlist(item.symbol)) {
        removeFromWatchlist(item.symbol);
      } else {
        addToWatchlist(item);
      }
    },
    [isInWatchlist, addToWatchlist, removeFromWatchlist],
  );

  const watchlistSymbols = useMemo(
    () => new Set(watchlist.map((w) => w.symbol)),
    [watchlist],
  );

  const tabItems = { search: searchResults, watchlist, recent: recentStocks };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        data-testid="search-modal-backdrop"
        className="absolute inset-0 bg-black/60"
        onClick={closeModal}
      />
      <div className="relative w-[560px] max-h-[420px] bg-nexus-surface border border-nexus-border rounded-2xl shadow-2xl flex overflow-hidden">
        <SearchSideNav activeTab={activeTab} onTabChange={setActiveTab} />
        <SearchContent
          title={TAB_TITLES[activeTab]}
          items={tabItems[activeTab]}
          watchlistSymbols={watchlistSymbols}
          onSelect={handleSelect}
          onToggleWatchlist={handleToggleWatchlist}
        />
      </div>
    </div>
  );
}
