"use client";

import { useSearchStore } from "../model/search.store";
import { NLTab } from "./NLTab";
import { DSLTab } from "./DSLTab";
import { LiveDSLPanel } from "./LiveDSLPanel";
import { PresetManager } from "./PresetManager";
import { SearchResults } from "./SearchResults";

export function SearchPageLayout() {
  const activeTab = useSearchStore((s) => s.activeTab);
  const setActiveTab = useSearchStore((s) => s.setActiveTab);

  return (
    <div className="flex flex-col h-full gap-4 p-6">
      <div className="flex gap-1 bg-nexus-surface rounded-lg p-1 w-fit">
        <button
          onClick={() => setActiveTab("nl")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            activeTab === "nl"
              ? "bg-nexus-accent text-white"
              : "text-nexus-text-secondary hover:text-nexus-text-primary"
          }`}
        >
          Natural Language
        </button>
        <button
          onClick={() => setActiveTab("dsl")}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            activeTab === "dsl"
              ? "bg-nexus-accent text-white"
              : "text-nexus-text-secondary hover:text-nexus-text-primary"
          }`}
        >
          DSL
        </button>
      </div>

      <div className="bg-nexus-surface border border-nexus-border rounded-lg p-4">
        {activeTab === "nl" ? <NLTab /> : <DSLTab />}
      </div>

      <LiveDSLPanel />
      <PresetManager />
      <SearchResults />
    </div>
  );
}
