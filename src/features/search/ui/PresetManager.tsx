"use client";

import { useState, useEffect } from "react";
import { usePresetStore } from "../model/preset.store";
import { useSearchStore } from "../model/search.store";
import { presetApi } from "../api/preset-api";
import { btnSecondary } from "./styles";

export function PresetManager() {
  const presets = usePresetStore((s) => s.presets);
  const removePreset = usePresetStore((s) => s.removePreset);
  const addPreset = usePresetStore((s) => s.addPreset);
  const setPresets = usePresetStore((s) => s.setPresets);
  const dslCode = useSearchStore((s) => s.dslCode);
  const setDslCode = useSearchStore((s) => s.setDslCode);
  const nlQuery = useSearchStore((s) => s.nlQuery);
  const setActiveTab = useSearchStore((s) => s.setActiveTab);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    presetApi.list().then((res) => setPresets(res.data)).catch(() => {});
  }, [setPresets]);

  async function handleSave(): Promise<void> {
    if (!dslCode.trim()) return;
    setSaving(true);
    setError(null);
    try {
      const name = `Preset ${new Date().toLocaleString("ko-KR", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}`;
      const response = await presetApi.create({
        name,
        dsl: dslCode,
        nlQuery: nlQuery || undefined,
      });
      addPreset(response.data);
    } catch (err) {
      console.error("preset save failed:", err);
      setError("저장에 실패했습니다");
    } finally {
      setSaving(false);
    }
  }

  function handleLoad(dsl: string): void {
    setDslCode(dsl);
    setActiveTab("dsl");
  }

  async function handleDelete(id: string): Promise<void> {
    setError(null);
    setDeletingId(id);
    try {
      await presetApi.delete(id);
      removePreset(id);
    } catch (err) {
      console.error("preset delete failed:", err);
      setError("삭제에 실패했습니다");
    } finally {
      setDeletingId(null);
    }
  }

  return (
    <div className="flex flex-col gap-2">
      {error && (
        <div className="text-xs text-nexus-failure bg-red-500/10 px-3 py-1 rounded">
          {error}
        </div>
      )}
      <div className="flex items-center justify-between">
        <span className="text-xs font-semibold text-nexus-text-secondary uppercase tracking-wider">
          Saved Presets
        </span>
        <button
          onClick={handleSave}
          disabled={!dslCode.trim() || saving}
          className={btnSecondary}
        >
          {saving ? "Saving..." : "Save Preset"}
        </button>
      </div>

      {presets.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {presets.map((preset) => (
            <div
              key={preset.id}
              className="flex items-center gap-1 bg-nexus-surface border border-nexus-border rounded-lg px-3 py-1"
            >
              <button
                onClick={() => handleLoad(preset.dsl)}
                className="text-sm text-nexus-text-primary hover:text-nexus-accent transition-colors"
              >
                {preset.name}
              </button>
              <button
                onClick={() => handleDelete(preset.id)}
                disabled={deletingId === preset.id}
                aria-label={`Delete preset ${preset.name}`}
                className="text-xs text-nexus-text-secondary hover:text-nexus-failure ml-1"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
