"use client";

import { useState } from "react";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";
import { StockListItem } from "./StockListItem";
import { SidebarSearchInput } from "./SidebarSearchInput";
import type { SearchResult } from "@/entities/search-result";

type SidebarTab = "results" | "watchlist" | "recent";

export function StockListSidebar() {
  const [activeTab, setActiveTab] = useState<SidebarTab>("results");
  const [filter, setFilter] = useState("");
  const { searchResults, watchlist, recentStocks, selectedSymbol, setSelectedSymbol, addToRecent } =
    useStockListStore();
  const { setCurrentStock } = useChartStore();

  const tabConfig: Record<SidebarTab, { label: string; items: SearchResult[] }> = {
    results: { label: "검색결과", items: searchResults },
    watchlist: { label: "관심종목", items: watchlist },
    recent: { label: "최근", items: recentStocks },
  };

  const items = tabConfig[activeTab].items.filter(
    (item) =>
      !filter ||
      item.symbol.includes(filter.toUpperCase()) ||
      item.name.toLowerCase().includes(filter.toLowerCase()),
  );

  const handleStockClick = (item: SearchResult) => {
    setSelectedSymbol(item.symbol);
    addToRecent(item);
    setCurrentStock({
      symbol: item.symbol,
      name: item.name,
      price: 0,
      change: 0,
      changePct: 0,
    });
  };

  return (
    <div className="flex flex-col h-full bg-nexus-surface">
      <div className="flex border-b border-nexus-border">
        {(Object.keys(tabConfig) as SidebarTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`flex-1 py-2 text-xs font-medium transition-colors ${
              activeTab === tab
                ? "text-nexus-accent border-b-2 border-nexus-accent"
                : "text-nexus-text-secondary hover:text-nexus-text-primary"
            }`}
          >
            {tabConfig[tab].label}
          </button>
        ))}
      </div>
      <SidebarSearchInput value={filter} onChange={setFilter} />
      <div className="flex-1 overflow-y-auto">
        {items.map((item) => (
          <StockListItem
            key={item.symbol}
            item={item}
            isActive={item.symbol === selectedSymbol}
            onClick={() => handleStockClick(item)}
          />
        ))}
        {items.length === 0 && (
          <div className="p-4 text-center text-nexus-text-muted text-sm">
            {activeTab === "results" ? "검색 결과가 없습니다" : "항목이 없습니다"}
          </div>
        )}
      </div>
    </div>
  );
}
