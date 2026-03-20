import { useSearchStore } from "../model/search.store";
import { searchApi } from "../api/search-api";

import { createSearchActions } from "../model/use-search-actions";

// jest.mock is auto-hoisted above imports by Jest
jest.mock("../api/search-api", () => ({
  searchApi: {
    nlSearch: jest.fn(),
    dslSearch: jest.fn(),
    validate: jest.fn(),
    explain: jest.fn(),
  },
}));

const mockedApi = searchApi as jest.Mocked<typeof searchApi>;

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
  jest.clearAllMocks();
});

describe("createSearchActions", () => {
  describe("runNLSearch", () => {
    it("transitions through agent statuses and sets results on success", async () => {
      mockedApi.nlSearch.mockResolvedValue({
        dsl: "scan where volume > 1000000",
        explanation: "거래량 100만 이상",
        results: [{ symbol: "005930", name: "삼성전자", matchedValue: 2840000 }],
      });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ nlQuery: "거래량 많은 종목" });

      await actions.runNLSearch();

      const state = useSearchStore.getState();
      expect(state.agentStatus).toBe("done");
      expect(state.dslCode).toBe("scan where volume > 1000000");
      expect(state.results).toHaveLength(1);
      expect(state.results[0].symbol).toBe("005930");
      expect(mockedApi.nlSearch).toHaveBeenCalledWith("거래량 많은 종목");
    });

    it("sets error status on API failure", async () => {
      mockedApi.nlSearch.mockRejectedValue(new Error("API Error"));

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ nlQuery: "테스트" });

      await actions.runNLSearch();

      const state = useSearchStore.getState();
      expect(state.agentStatus).toBe("error");
      expect(state.agentMessage).toContain("API Error");
    });
  });

  describe("runDSLSearch", () => {
    it("executes DSL search and sets results", async () => {
      mockedApi.dslSearch.mockResolvedValue({
        results: [{ symbol: "000660", name: "SK하이닉스", matchedValue: 5000000 }],
      });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "scan where volume > 5000000" });

      await actions.runDSLSearch();

      const state = useSearchStore.getState();
      expect(state.agentStatus).toBe("done");
      expect(state.results).toHaveLength(1);
      expect(mockedApi.dslSearch).toHaveBeenCalledWith("scan where volume > 5000000");
    });

    it("sets error status on failure", async () => {
      mockedApi.dslSearch.mockRejectedValue(new Error("Execute failed"));

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "invalid dsl" });

      await actions.runDSLSearch();

      expect(useSearchStore.getState().agentStatus).toBe("error");
    });
  });

  describe("validateDSL", () => {
    it("sets valid state on successful validation", async () => {
      mockedApi.validate.mockResolvedValue({ valid: true, error: null });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "scan where volume > 1000000" });

      await actions.validateDSL();

      const state = useSearchStore.getState();
      expect(state.validationState).toBe("valid");
      expect(state.validationMessage).toBe("");
    });

    it("sets invalid state with message on validation failure", async () => {
      mockedApi.validate.mockResolvedValue({ valid: false, error: "syntax error at line 1" });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "bad query" });

      await actions.validateDSL();

      const state = useSearchStore.getState();
      expect(state.validationState).toBe("invalid");
      expect(state.validationMessage).toBe("syntax error at line 1");
    });
  });

  describe("explainDSL", () => {
    it("returns explanation text", async () => {
      mockedApi.explain.mockResolvedValue({
        explanation: "이 쿼리는 거래량이 100만 이상인 종목을 검색합니다",
      });

      const actions = createSearchActions(useSearchStore.getState, useSearchStore.setState);
      useSearchStore.setState({ dslCode: "scan where volume > 1000000" });

      const result = await actions.explainDSL();

      expect(result).toBe("이 쿼리는 거래량이 100만 이상인 종목을 검색합니다");
      expect(mockedApi.explain).toHaveBeenCalledWith("scan where volume > 1000000");
    });
  });
});
