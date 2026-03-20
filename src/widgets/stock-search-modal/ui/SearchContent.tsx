"use client";

import { useState } from "react";
import type { SearchResult } from "@/entities/search-result";
import { SearchStockItem } from "./SearchStockItem";

interface Props {
  title: string;
  items: SearchResult[];
  watchlistSymbols: Set<string>;
  onSelect: (item: SearchResult) => void;
  onToggleWatchlist: (item: SearchResult) => void;
}

export function SearchContent({ title, items, watchlistSymbols, onSelect, onToggleWatchlist }: Props) {
  const [filter, setFilter] = useState("");

  const filtered = items.filter(
    (item) =>
      !filter ||
      item.symbol.includes(filter.toUpperCase()) ||
      item.name.toLowerCase().includes(filter.toLowerCase()),
  );

  return (
    <div className="flex-1 flex flex-col min-w-0">
      <div className="flex items-center justify-between px-4 py-3 border-b border-nexus-border">
        <h3 className="text-sm font-semibold text-nexus-text-primary">{title}</h3>
      </div>
      <div className="px-4 py-3 border-b border-nexus-border">
        <input
          type="text"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="종목명 또는 코드를 검색하세요..."
          className="w-full bg-nexus-border/50 rounded-lg px-3 py-2 text-sm text-nexus-text-primary placeholder:text-nexus-text-muted outline-none focus:ring-1 focus:ring-nexus-accent"
          autoFocus
        />
      </div>
      <div className="flex-1 overflow-y-auto px-2 py-1">
        {filtered.length > 0 ? (
          filtered.map((item) => (
            <SearchStockItem
              key={item.symbol}
              item={item}
              isInWatchlist={watchlistSymbols.has(item.symbol)}
              onSelect={onSelect}
              onToggleWatchlist={onToggleWatchlist}
            />
          ))
        ) : (
          <div className="flex items-center justify-center h-full text-nexus-text-muted text-sm">
            항목이 없습니다
          </div>
        )}
      </div>
      <div className="px-4 py-2 border-t border-nexus-border flex justify-between">
        <span className="text-[10px] text-nexus-text-muted">↑↓ 이동 · Enter 선택 · Esc 닫기</span>
        <span className="text-[10px] text-nexus-text-muted">{filtered.length} 종목</span>
      </div>
    </div>
  );
}
