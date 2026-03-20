import { apiClient } from "@/shared/api/client";
import { API_BASE_URL } from "@/shared/config/constants";
import { parseSSEBuffer } from "../lib/sse-parser";
import type { SSEEvent } from "../model/types";
import type { SearchResult } from "@/entities/search-result";

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
  async *nlSearchStream(query: string): AsyncGenerator<SSEEvent> {
    const response = await fetch(`${API_BASE_URL}/api/v1/search/nl-to-dsl`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ query }),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const reader = response.body!.getReader();
    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const { events, remaining } = parseSSEBuffer(buffer);
      buffer = remaining;

      for (const event of events) {
        yield event;
      }
    }
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
