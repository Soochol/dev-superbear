/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";
import { BottomInfoPanel } from "../ui/BottomInfoPanel";
import { useChartStore } from "@/features/chart";

beforeAll(() => {
  global.fetch = jest.fn(() =>
    Promise.resolve({ json: () => Promise.resolve({ data: null }) })
  ) as jest.Mock;
});

beforeEach(() => {
  useChartStore.setState({
    ...useChartStore.getInitialState(),
    currentStock: {
      symbol: "005930",
      name: "Samsung Electronics",
      price: 78400,
      change: 1600,
      changePct: 2.08,
    },
  });
});

describe("BottomInfoPanel", () => {
  it("renders all 3 column headers", () => {
    render(<BottomInfoPanel />);
    expect(screen.getByText(/financials/i)).toBeInTheDocument();
    expect(screen.getByText(/ai fusion/i)).toBeInTheDocument();
    expect(screen.getByText(/sector compare/i)).toBeInTheDocument();
  });

  it("shows financial metrics labels", () => {
    render(<BottomInfoPanel />);
    expect(screen.getByText(/revenue/i)).toBeInTheDocument();
    expect(screen.getByText(/PER/)).toBeInTheDocument();
    expect(screen.getByText(/ROE/)).toBeInTheDocument();
  });

  it("shows empty state when no stock selected", () => {
    useChartStore.setState({ currentStock: null });
    render(<BottomInfoPanel />);
    expect(screen.getByText(/종목을 선택|select a stock/i)).toBeInTheDocument();
  });
});
