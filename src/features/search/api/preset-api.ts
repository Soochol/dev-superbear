import { apiGet, apiPost, apiDelete } from "@/shared/api/client";
import type { SearchPreset, CreateSearchPresetInput } from "@/entities/search-preset";

interface ListPresetsResponse {
  data: SearchPreset[];
  pagination: {
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
  };
}

interface CreatePresetResponse {
  data: SearchPreset;
}

export const presetApi = {
  list(page = 1, pageSize = 20): Promise<ListPresetsResponse> {
    return apiGet<ListPresetsResponse>(`/api/v1/search/presets?page=${page}&pageSize=${pageSize}`);
  },

  create(input: CreateSearchPresetInput): Promise<CreatePresetResponse> {
    return apiPost<CreatePresetResponse>("/api/v1/search/presets", input);
  },

  delete(id: string): Promise<void> {
    return apiDelete<void>(`/api/v1/search/presets/${id}`);
  },
};
