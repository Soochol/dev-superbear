import type { SSEEvent } from "../model/types";

interface ParseResult {
  events: SSEEvent[];
  remaining: string;
}

export function parseSSEBuffer(buffer: string): ParseResult {
  const events: SSEEvent[] = [];
  const blocks = buffer.split("\n\n");

  // Last block may be incomplete
  const remaining = blocks[blocks.length - 1];

  for (let i = 0; i < blocks.length - 1; i++) {
    const block = blocks[i].trim();
    if (!block) continue;

    let eventType = "";
    let data = "";

    for (const line of block.split("\n")) {
      if (line.startsWith("event: ")) {
        eventType = line.slice(7);
      } else if (line.startsWith("data: ")) {
        data = line.slice(6);
      }
    }

    if (eventType && data) {
      try {
        const parsed = JSON.parse(data);
        events.push({ type: eventType, ...parsed } as SSEEvent);
      } catch {
        // skip malformed events
      }
    }
  }

  return { events, remaining };
}
