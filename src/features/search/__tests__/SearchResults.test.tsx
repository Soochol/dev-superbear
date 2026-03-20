/**
 * @jest-environment jsdom
 */
import { render, screen } from "@testing-library/react";
import { SearchResults } from "../ui/SearchResults";
import { useSearchStore } from "../model/search.store";

// Mock next/navigation
jest.mock("next/navigation", () => ({
  useRouter: () => ({
    push: jest.fn(),
  }),
}));

beforeEach(() => {
  useSearchStore.setState(useSearchStore.getInitialState());
});

describe("SearchResults", () => {
  it("shows empty state when no results", () => {
    render(<SearchResults />);
    expect(screen.getByText(/검색 결과가 없습니다/i)).toBeInTheDocument();
  });

  it("renders results table with stock data", () => {
    useSearchStore.setState({
      results: [
        { symbol: "005930", name: "Samsung Electronics", matchedValue: 28400000 },
        { symbol: "247540", name: "ecoprobm", matchedValue: 15200000 },
      ],
    });
    render(<SearchResults />);
    expect(screen.getByText("Samsung Electronics")).toBeInTheDocument();
    expect(screen.getByText("ecoprobm")).toBeInTheDocument();
    expect(screen.getByText("005930")).toBeInTheDocument();
  });

  it("shows result count", () => {
    useSearchStore.setState({
      results: [
        { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      ],
    });
    render(<SearchResults />);
    expect(screen.getByText(/1/)).toBeInTheDocument();
  });

  it("renders Chart button for each row", () => {
    useSearchStore.setState({
      results: [
        { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      ],
    });
    render(<SearchResults />);
    expect(screen.getByRole("button", { name: /chart/i })).toBeInTheDocument();
  });

  it("shows loading state while searching", () => {
    useSearchStore.setState({ agentStatus: "interpreting" });
    render(<SearchResults />);
    expect(screen.getByText(/검색 중/i)).toBeInTheDocument();
  });
});
