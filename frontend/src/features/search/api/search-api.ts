import { apiClient } from "@/shared/api/client";
import type { SearchResult } from "@/entities/search-result";

interface NLSearchResponse {
  dsl: string;
  explanation: string;
  results: SearchResult[];
}

interface DSLSearchResponse {
  results: SearchResult[];
}

interface ValidateResponse {
  valid: boolean;
  error: string | null;
}

interface ExplainResponse {
  explanation: string;
}

export const searchApi = {
  async nlSearch(query: string): Promise<NLSearchResponse> {
    return apiClient<NLSearchResponse>("/api/v1/search/nl-to-dsl", {
      method: "POST",
      body: JSON.stringify({ query }),
    });
  },

  async dslSearch(dsl: string): Promise<DSLSearchResponse> {
    return apiClient<DSLSearchResponse>("/api/v1/search/execute", {
      method: "POST",
      body: JSON.stringify({ dsl }),
    });
  },

  async validate(dsl: string): Promise<ValidateResponse> {
    return apiClient<ValidateResponse>("/api/v1/search/validate", {
      method: "POST",
      body: JSON.stringify({ dsl }),
    });
  },

  async explain(dsl: string): Promise<ExplainResponse> {
    return apiClient<ExplainResponse>("/api/v1/search/explain", {
      method: "POST",
      body: JSON.stringify({ dsl }),
    });
  },
};
