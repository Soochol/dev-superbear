"use client";

import type { SearchModalTab } from "@/shared/model/search-modal.store";

interface Props {
  activeTab: SearchModalTab;
  onTabChange: (tab: SearchModalTab) => void;
}

const NAV_ITEMS: { tab: SearchModalTab; icon: string; label: string }[] = [
  { tab: "search", icon: "🔍", label: "종목 검색" },
  { tab: "watchlist", icon: "⭐", label: "관심 종목" },
  { tab: "recent", icon: "🕐", label: "최근 본 종목" },
];

export function SearchSideNav({ activeTab, onTabChange }: Props) {
  return (
    <nav className="w-[140px] bg-nexus-bg border-r border-nexus-border flex flex-col py-4 flex-shrink-0">
      {NAV_ITEMS.map(({ tab, icon, label }) => (
        <button
          key={tab}
          data-tab={tab}
          data-active={activeTab === tab}
          onClick={() => onTabChange(tab)}
          className={`flex items-center gap-2 px-4 py-2 text-xs transition-colors text-left ${
            activeTab === tab
              ? "text-nexus-accent bg-nexus-accent/10 border-r-2 border-nexus-accent font-semibold"
              : "text-nexus-text-secondary hover:text-nexus-text-primary"
          }`}
        >
          <span>{icon}</span>
          {label}
        </button>
      ))}
    </nav>
  );
}
