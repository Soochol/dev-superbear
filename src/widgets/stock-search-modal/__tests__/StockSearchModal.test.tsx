/** @jest-environment jsdom */
import { render, screen, fireEvent } from "@testing-library/react";
import { StockSearchModal } from "../ui/StockSearchModal";
import { useSearchModalStore } from "@/shared/model/search-modal.store";
import { useStockListStore } from "@/entities/stock";
import { useChartStore } from "@/features/chart";

jest.mock("@/features/watchlist", () => ({
  watchlistApi: {
    fetchWatchlist: jest.fn().mockResolvedValue([]),
    addItem: jest.fn().mockResolvedValue(undefined),
    removeItem: jest.fn().mockResolvedValue(undefined),
  },
}));

jest.mock("@/shared/lib/logger", () => ({
  logger: { error: jest.fn() },
}));

beforeEach(() => {
  useSearchModalStore.setState({ isOpen: true, activeTab: "search" });
  useStockListStore.setState({
    ...useStockListStore.getInitialState(),
    searchResults: [
      { symbol: "005930", name: "삼성전자", matchedValue: "005930", close: 71200, changePct: 2.3 },
    ],
    watchlist: [{ symbol: "000660", name: "SK하이닉스", matchedValue: "000660" }],
    recentStocks: [{ symbol: "035420", name: "NAVER", matchedValue: "035420" }],
    watchlistLoaded: true,
  });
  useChartStore.setState(useChartStore.getInitialState());
});

describe("StockSearchModal", () => {
  it("renders when isOpen is true", () => {
    render(<StockSearchModal />);
    expect(screen.getByRole("heading", { name: "종목 검색" })).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    useSearchModalStore.setState({ isOpen: false });
    render(<StockSearchModal />);
    expect(screen.queryByRole("heading", { name: "종목 검색" })).not.toBeInTheDocument();
  });

  it("closes on Esc key", () => {
    render(<StockSearchModal />);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(useSearchModalStore.getState().isOpen).toBe(false);
  });

  it("closes on backdrop click", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByTestId("search-modal-backdrop"));
    expect(useSearchModalStore.getState().isOpen).toBe(false);
  });

  it("selects stock → updates chart store → closes modal", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByTestId("search-stock-item-005930"));
    expect(useChartStore.getState().currentStock?.symbol).toBe("005930");
    expect(useSearchModalStore.getState().isOpen).toBe(false);
  });

  it("switches to watchlist tab and shows watchlist items", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByText("관심 종목"));
    expect(screen.getByText("SK하이닉스")).toBeInTheDocument();
  });

  it("switches to recent tab and shows recent items", () => {
    render(<StockSearchModal />);
    fireEvent.click(screen.getByText("최근 본 종목"));
    expect(screen.getByText("NAVER")).toBeInTheDocument();
  });
});
