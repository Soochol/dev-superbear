/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { StockListSidebar } from "../ui/StockListSidebar";
import { useStockListStore } from "@/entities/stock";

beforeEach(() => {
  useStockListStore.setState({
    ...useStockListStore.getInitialState(),
    searchResults: [
      { symbol: "005930", name: "Samsung", matchedValue: 28400000 },
      { symbol: "247540", name: "ecoprobm", matchedValue: 15200000 },
    ],
    selectedSymbol: "005930",
    watchlist: [
      { symbol: "000660", name: "SK Hynix", matchedValue: 0 },
    ],
    recentStocks: [
      { symbol: "373220", name: "LG Energy", matchedValue: 0 },
    ],
  });
});

describe("StockListSidebar", () => {
  it("renders 3 tabs", () => {
    render(<StockListSidebar />);
    expect(screen.getByText(/검색결과/)).toBeInTheDocument();
    expect(screen.getByText(/관심/i)).toBeInTheDocument();
    expect(screen.getByText(/최근/i)).toBeInTheDocument();
  });

  it("shows search results in first tab", () => {
    render(<StockListSidebar />);
    expect(screen.getByText("Samsung")).toBeInTheDocument();
    expect(screen.getByText("ecoprobm")).toBeInTheDocument();
  });

  it("highlights the selected/active stock", () => {
    render(<StockListSidebar />);
    const samsungItem = screen.getByText("Samsung").closest("[data-testid]");
    expect(samsungItem?.className).toMatch(/active|selected/i);
  });

  it("switches to watchlist tab and shows watchlist items", () => {
    render(<StockListSidebar />);
    fireEvent.click(screen.getByText(/관심/i));
    expect(screen.getByText("SK Hynix")).toBeInTheDocument();
  });

  it("switches to recent tab and shows recent items", () => {
    render(<StockListSidebar />);
    fireEvent.click(screen.getByText(/최근/i));
    expect(screen.getByText("LG Energy")).toBeInTheDocument();
  });

  it("renders search input at top", () => {
    render(<StockListSidebar />);
    expect(screen.getByPlaceholderText(/종목 검색|search/i)).toBeInTheDocument();
  });

  it("renders watchlist toggle (star icon) on each item", () => {
    render(<StockListSidebar />);
    const starButtons = screen.getAllByRole("button", { name: /watchlist|star|관심/i });
    expect(starButtons.length).toBeGreaterThan(0);
  });
});
