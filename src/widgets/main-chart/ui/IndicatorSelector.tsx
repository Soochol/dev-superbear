"use client";

import { useState, useRef, useEffect } from "react";
import { useChartStore } from "@/features/chart";
import { INDICATOR_REGISTRY, type IndicatorCategory } from "@/entities/indicator";

const CATEGORY_LABELS: Record<IndicatorCategory, string> = {
  "moving-average": "이동평균",
  oscillator: "오실레이터",
  band: "밴드",
};

const CATEGORY_ORDER: IndicatorCategory[] = ["moving-average", "oscillator", "band"];

export function IndicatorSelector() {
  const [isOpen, setIsOpen] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);
  const { activeIndicators, toggleIndicator } = useChartStore();

  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [isOpen]);

  return (
    <div className="relative" ref={popoverRef}>
      <button
        data-testid="indicator-selector-btn"
        onClick={() => setIsOpen(!isOpen)}
        className={`px-2 py-1 text-xs font-medium rounded transition-colors ${
          isOpen
            ? "bg-nexus-accent text-white"
            : "text-nexus-text-secondary hover:text-nexus-text-primary"
        }`}
      >
        지표
      </button>
      {isOpen && (
        <div className="absolute right-0 top-full mt-1 w-56 bg-nexus-surface border border-nexus-border rounded-xl shadow-xl z-50 py-2">
          {CATEGORY_ORDER.map((category) => {
            const items = INDICATOR_REGISTRY.filter((ind) => ind.category === category);
            if (items.length === 0) return null;
            return (
              <div key={category}>
                <div className="px-3 py-1 text-[10px] font-semibold text-nexus-text-muted uppercase tracking-wider">
                  {CATEGORY_LABELS[category]}
                </div>
                {items.map((ind) => {
                  const isActive = activeIndicators.includes(ind.id);
                  return (
                    <button
                      key={ind.id}
                      data-testid={`indicator-${ind.id}`}
                      onClick={() => toggleIndicator(ind.id)}
                      className="w-full flex items-center justify-between px-3 py-1.5 text-xs hover:bg-nexus-border/30 transition-colors"
                    >
                      <span className={isActive ? "text-nexus-text-primary" : "text-nexus-text-secondary"}>
                        {ind.name}
                      </span>
                      <span className={`text-xs ${isActive ? "text-nexus-accent" : "text-nexus-text-muted"}`}>
                        {isActive ? "✓" : ""}
                      </span>
                    </button>
                  );
                })}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
