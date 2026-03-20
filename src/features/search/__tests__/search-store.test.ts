import { useSearchStore } from "../model/search.store";

describe("Search Store", () => {
  beforeEach(() => {
    useSearchStore.setState(useSearchStore.getInitialState());
  });

  it("initializes with NL mode active", () => {
    const state = useSearchStore.getState();
    expect(state.activeTab).toBe("nl");
    expect(state.dslCode).toBe("");
    expect(state.nlQuery).toBe("");
    expect(state.results).toEqual([]);
  });

  it("switches tabs", () => {
    useSearchStore.getState().setActiveTab("dsl");
    expect(useSearchStore.getState().activeTab).toBe("dsl");
  });

  it("updates NL query", () => {
    useSearchStore.getState().setNlQuery("2년 최대거래량 종목");
    expect(useSearchStore.getState().nlQuery).toBe("2년 최대거래량 종목");
  });

  it("updates DSL code", () => {
    useSearchStore.getState().setDslCode("scan where volume > 1000000");
    expect(useSearchStore.getState().dslCode).toBe("scan where volume > 1000000");
  });

  it("tracks agent status transitions", () => {
    const { setAgentStatus } = useSearchStore.getState();
    setAgentStatus("interpreting");
    expect(useSearchStore.getState().agentStatus).toBe("interpreting");
    setAgentStatus("building");
    expect(useSearchStore.getState().agentStatus).toBe("building");
    setAgentStatus("scanning");
    expect(useSearchStore.getState().agentStatus).toBe("scanning");
  });

  it("stores search results", () => {
    const results = [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
    ];
    useSearchStore.getState().setResults(results);
    expect(useSearchStore.getState().results).toEqual(results);
  });

  it("tracks validation state", () => {
    useSearchStore.getState().setValidationState("valid");
    expect(useSearchStore.getState().validationState).toBe("valid");
  });
});
