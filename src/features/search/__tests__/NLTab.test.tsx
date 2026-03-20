/** @jest-environment jsdom */
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { NLTab } from "../ui/NLTab";
import { useSearchStore } from "../model/search.store";

import { searchApi } from "../api/search-api";

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
});

describe("NLTab", () => {
  it("renders textarea for NL query input", () => {
    render(<NLTab />);
    expect(screen.getByPlaceholderText(/자연어로 검색 조건/i)).toBeInTheDocument();
  });

  it("renders preset chips", () => {
    render(<NLTab />);
    expect(screen.getByText("2yr Max Volume")).toBeInTheDocument();
    expect(screen.getByText("Golden Cross")).toBeInTheDocument();
    expect(screen.getByText("RSI Oversold")).toBeInTheDocument();
  });

  it("clicking a preset chip fills the NL query", () => {
    render(<NLTab />);
    fireEvent.click(screen.getByText("2yr Max Volume"));
    const textarea = screen.getByPlaceholderText(/자연어로 검색 조건/i) as HTMLTextAreaElement;
    expect(textarea.value).toContain("2년 최대거래량");
  });

  it("shows agent status when searching", () => {
    useSearchStore.setState({ agentStatus: "interpreting" });
    render(<NLTab />);
    expect(screen.getByText(/Interpreting/i)).toBeInTheDocument();
  });

  it("has a search button", () => {
    render(<NLTab />);
    expect(screen.getByRole("button", { name: /검색|search/i })).toBeInTheDocument();
  });

  it("calls NL search API when Search button is clicked", async () => {
    mockedApi.nlSearch.mockResolvedValue({
      dsl: "scan where volume > 1000000",
      explanation: "test",
      results: [{ symbol: "005930", name: "삼성전자", matchedValue: 100 }],
    });

    useSearchStore.setState({ nlQuery: "거래량 많은 종목" });
    render(<NLTab />);

    const searchBtn = screen.getByRole("button", { name: /search/i });
    fireEvent.click(searchBtn);

    await waitFor(() => {
      expect(mockedApi.nlSearch).toHaveBeenCalledWith("거래량 많은 종목");
    });
  });
});
