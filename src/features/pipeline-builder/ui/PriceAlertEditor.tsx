"use client";

import { useState } from "react";
import { usePipelineStore } from "../model/pipeline.store";

export default function PriceAlertEditor() {
  const priceAlerts = usePipelineStore((s) => s.priceAlerts);
  const addPriceAlert = usePipelineStore((s) => s.addPriceAlert);
  const removePriceAlert = usePipelineStore((s) => s.removePriceAlert);

  const [newCondition, setNewCondition] = useState("");
  const [newLabel, setNewLabel] = useState("");

  const handleAdd = () => {
    if (!newCondition.trim() || !newLabel.trim()) return;
    addPriceAlert(newCondition.trim(), newLabel.trim());
    setNewCondition("");
    setNewLabel("");
  };

  return (
    <div>
      <label className="block text-xs font-medium text-nexus-warning mb-1.5">
        Price Alerts
      </label>

      {priceAlerts.length > 0 && (
        <div className="space-y-2 mb-3">
          {priceAlerts.map((alert) => (
            <div
              key={alert.id}
              className="flex items-center gap-2 bg-nexus-bg border border-nexus-border rounded-md px-3 py-2"
            >
              <code className="text-xs font-mono text-nexus-text-primary flex-1 truncate">
                {alert.condition}
              </code>
              <span className="text-xs text-nexus-text-muted shrink-0">
                {alert.label}
              </span>
              <button
                type="button"
                onClick={() => removePriceAlert(alert.id)}
                className="p-0.5 text-nexus-text-muted hover:text-nexus-failure transition-colors shrink-0"
                aria-label="Remove alert"
              >
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M18 6 6 18" />
                  <path d="m6 6 12 12" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="flex gap-2">
        <input
          type="text"
          value={newCondition}
          onChange={(e) => setNewCondition(e.target.value)}
          placeholder="price > 50000"
          className="flex-1 bg-nexus-bg border border-nexus-border rounded-md px-3 py-1.5 text-xs font-mono text-nexus-text-primary placeholder:text-nexus-text-muted/50 focus:outline-none focus:border-nexus-accent"
        />
        <input
          type="text"
          value={newLabel}
          onChange={(e) => setNewLabel(e.target.value)}
          placeholder="Label"
          className="w-28 bg-nexus-bg border border-nexus-border rounded-md px-3 py-1.5 text-xs text-nexus-text-primary placeholder:text-nexus-text-muted/50 focus:outline-none focus:border-nexus-accent"
        />
        <button
          type="button"
          onClick={handleAdd}
          disabled={!newCondition.trim() || !newLabel.trim()}
          className="px-3 py-1.5 text-xs bg-nexus-accent hover:bg-nexus-accent/80 text-white rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          + Add
        </button>
      </div>
    </div>
  );
}
