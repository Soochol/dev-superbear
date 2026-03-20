export type SearchTab = "nl" | "dsl";
export type AgentStatus = "idle" | "interpreting" | "building" | "scanning" | "done" | "error";
export type ValidationState = "none" | "valid" | "invalid";

import type { SearchResult } from "@/entities/search-result";

export type SSEEventType = "thinking" | "tool_call" | "tool_result" | "dsl_ready" | "done" | "error";

export type SSEEvent =
  | { type: "thinking"; message: string }
  | { type: "tool_call"; tool: string; message: string }
  | { type: "tool_result"; tool: string; message: string }
  | { type: "dsl_ready"; dsl: string; explanation: string }
  | { type: "done"; results: SearchResult[]; count: number }
  | { type: "error"; message: string };
