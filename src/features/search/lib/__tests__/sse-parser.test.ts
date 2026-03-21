import { parseSSEBuffer } from "../sse-parser";

describe("parseSSEBuffer", () => {
  it("parses a complete SSE event", () => {
    const buffer = 'event: thinking\ndata: {"message":"분석 중..."}\n\n';
    const { events, remaining } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "thinking", message: "분석 중..." });
    expect(remaining).toBe("");
  });

  it("parses multiple events", () => {
    const buffer =
      'event: thinking\ndata: {"message":"a"}\n\nevent: dsl_ready\ndata: {"dsl":"scan where volume > 100","explanation":"설명"}\n\n';
    const { events, remaining } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(2);
    expect(events[0].type).toBe("thinking");
    expect(events[1].type).toBe("dsl_ready");
  });

  it("keeps incomplete event in remaining", () => {
    const buffer = 'event: thinking\ndata: {"mess';
    const { events, remaining } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(0);
    expect(remaining).toBe(buffer);
  });

  it("handles empty buffer", () => {
    const { events, remaining } = parseSSEBuffer("");
    expect(events).toHaveLength(0);
    expect(remaining).toBe("");
  });

  it("parses done event with results", () => {
    const buffer =
      'event: done\ndata: {"results":[{"symbol":"005930","name":"삼성전자","matchedValue":100}],"count":1}\n\n';
    const { events } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe("done");
    if (events[0].type === "done") {
      expect(events[0].results).toHaveLength(1);
      expect(events[0].count).toBe(1);
    }
  });

  it("parses error event", () => {
    const buffer = 'event: error\ndata: {"message":"timeout"}\n\n';
    const { events } = parseSSEBuffer(buffer);
    expect(events).toHaveLength(1);
    expect(events[0]).toEqual({ type: "error", message: "timeout" });
  });
});
