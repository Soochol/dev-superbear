"use client";

import { useStockListStore } from "@/entities/stock";
import type { SearchResult } from "@/entities/search-result";

interface Props {
  item: SearchResult;
  isActive: boolean;
  onClick: () => void;
}

export function StockListItem({ item, isActive, onClick }: Props) {
  const { isInWatchlist, addToWatchlist, removeFromWatchlist } = useStockListStore();
  const inWatchlist = isInWatchlist(item.symbol);

  const handleStarClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (inWatchlist) {
      removeFromWatchlist(item.symbol);
    } else {
      addToWatchlist(item);
    }
  };

  return (
    <div
      data-testid={`stock-item-${item.symbol}`}
      onClick={onClick}
      className={`flex items-center justify-between px-3 py-2 cursor-pointer transition-colors
        ${isActive ? "bg-nexus-accent/10 border-l-2 border-nexus-accent active" : "hover:bg-nexus-border/30"}`}
    >
      <div className="min-w-0">
        <div className="text-sm font-medium truncate">{item.name}</div>
        <div className="text-xs text-nexus-text-muted font-mono">{item.symbol}</div>
      </div>
      <button
        onClick={handleStarClick}
        aria-label={inWatchlist ? "Remove from watchlist" : "Add to watchlist"}
        className={`text-lg transition-colors flex-shrink-0 ml-2 ${
          inWatchlist ? "text-nexus-warning" : "text-nexus-text-muted hover:text-nexus-warning"
        }`}
      >
        {inWatchlist ? "\u2605" : "\u2606"}
      </button>
    </div>
  );
}
