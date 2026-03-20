"use client";

import type { SearchResult } from "@/entities/search-result";

interface Props {
  item: SearchResult;
  isInWatchlist: boolean;
  onSelect: (item: SearchResult) => void;
  onToggleWatchlist: (item: SearchResult) => void;
}

export function SearchStockItem({ item, isInWatchlist, onSelect, onToggleWatchlist }: Props) {
  return (
    <div
      data-testid={`search-stock-item-${item.symbol}`}
      onClick={() => onSelect(item)}
      className="flex items-center justify-between px-3 py-2.5 rounded-lg cursor-pointer transition-colors hover:bg-nexus-border/30"
    >
      <div className="flex items-center gap-3 min-w-0">
        <span className="text-sm font-medium text-nexus-text-primary truncate">{item.name}</span>
        <span className="text-xs text-nexus-text-muted font-mono flex-shrink-0">{item.symbol}</span>
      </div>
      <div className="flex items-center gap-3 flex-shrink-0">
        {item.close != null && (
          <div className="text-right">
            <span className="text-sm text-nexus-text-primary font-mono">{item.close.toLocaleString()}</span>
            {item.changePct != null && (
              <span
                className={`text-xs font-mono ml-2 ${
                  item.changePct >= 0 ? "text-nexus-success" : "text-nexus-failure"
                }`}
              >
                {item.changePct >= 0 ? "+" : ""}{item.changePct.toFixed(2)}%
              </span>
            )}
          </div>
        )}
        <button
          onClick={(e) => {
            e.stopPropagation();
            onToggleWatchlist(item);
          }}
          aria-label={isInWatchlist ? "Remove from watchlist" : "Add to watchlist"}
          className={`text-base transition-colors ${
            isInWatchlist ? "text-nexus-warning" : "text-nexus-text-muted hover:text-nexus-warning"
          }`}
        >
          {isInWatchlist ? "\u2605" : "\u2606"}
        </button>
      </div>
    </div>
  );
}
