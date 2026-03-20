/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { StockListSidebar } from "../ui/StockListSidebar";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";

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
  useChartStore.setState(useChartStore.getInitialState());
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

  it("click stock item → calls setSelectedSymbol with correct symbol", () => {
    render(<StockListSidebar />);
    const ecoprobmItem = screen.getByTestId("stock-item-247540");
    fireEvent.click(ecoprobmItem);
    expect(useStockListStore.getState().selectedSymbol).toBe("247540");
  });

  it("click stock item → calls addToRecent with correct item", () => {
    render(<StockListSidebar />);
    const ecoprobmItem = screen.getByTestId("stock-item-247540");
    fireEvent.click(ecoprobmItem);
    const recentStocks = useStockListStore.getState().recentStocks;
    expect(recentStocks.some((r) => r.symbol === "247540")).toBe(true);
  });

  it("click stock item → calls setCurrentStock on chartStore with correct data", () => {
    render(<StockListSidebar />);
    const ecoprobmItem = screen.getByTestId("stock-item-247540");
    fireEvent.click(ecoprobmItem);
    const currentStock = useChartStore.getState().currentStock;
    expect(currentStock).toEqual({
      symbol: "247540",
      name: "ecoprobm",
      price: 0,
      change: 0,
      changePct: 0,
    });
  });

  it("star button click → toggles watchlist (adds if not in watchlist)", () => {
    render(<StockListSidebar />);
    // Samsung (005930) is NOT in watchlist initially
    const samsungStar = screen.getByTestId("stock-item-005930").querySelector("button")!;
    expect(samsungStar).toHaveAttribute("aria-label", "Add to watchlist");
    fireEvent.click(samsungStar);
    expect(useStockListStore.getState().watchlist.some((w) => w.symbol === "005930")).toBe(true);
  });

  it("star button click → toggles watchlist (removes if in watchlist)", () => {
    // Switch to watchlist tab where SK Hynix (000660) IS in watchlist
    render(<StockListSidebar />);
    fireEvent.click(screen.getByText(/관심/i));
    const hynixStar = screen.getByTestId("stock-item-000660").querySelector("button")!;
    expect(hynixStar).toHaveAttribute("aria-label", "Remove from watchlist");
    fireEvent.click(hynixStar);
    expect(useStockListStore.getState().watchlist.some((w) => w.symbol === "000660")).toBe(false);
  });

  it("filter input → narrows displayed items", () => {
    render(<StockListSidebar />);
    // Both items are visible initially
    expect(screen.getByText("Samsung")).toBeInTheDocument();
    expect(screen.getByText("ecoprobm")).toBeInTheDocument();

    // Type in filter
    const filterInput = screen.getByPlaceholderText(/종목 검색|search/i);
    fireEvent.change(filterInput, { target: { value: "Sam" } });

    // Only Samsung should be visible
    expect(screen.getByText("Samsung")).toBeInTheDocument();
    expect(screen.queryByText("ecoprobm")).not.toBeInTheDocument();
  });
});
