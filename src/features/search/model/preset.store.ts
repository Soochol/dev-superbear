import { create } from "zustand";
import type { SearchPreset } from "@/entities/search-preset";

interface PresetState {
  presets: SearchPreset[];
  isLoading: boolean;
  setPresets: (presets: SearchPreset[]) => void;
  addPreset: (preset: SearchPreset) => void;
  removePreset: (id: string) => void;
  setLoading: (loading: boolean) => void;
}

export const usePresetStore = create<PresetState>()((set) => ({
  presets: [],
  isLoading: false,
  setPresets: (presets) => set({ presets }),
  addPreset: (preset) => set((s) => ({ presets: [preset, ...s.presets] })),
  removePreset: (id) => set((s) => ({ presets: s.presets.filter((p) => p.id !== id) })),
  setLoading: (loading) => set({ isLoading: loading }),
}));
