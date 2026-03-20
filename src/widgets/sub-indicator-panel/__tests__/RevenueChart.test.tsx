/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";
import { RevenueChart } from "../ui/RevenueChart";

jest.mock("@/features/chart", () => ({
  useChartStore: jest.fn(() => ({
    candles: [
      { time: "2024-01-01", open: 100, high: 110, low: 90, close: 105, volume: 1000 },
      { time: "2024-01-02", open: 105, high: 115, low: 95, close: 98, volume: 1500 },
    ],
  })),
  CHART_THEME: { layout: { background: { color: "#0a0a0f" } } },
}));

describe("RevenueChart", () => {
  it("renders chart container (not placeholder text)", () => {
    render(<RevenueChart />);
    expect(screen.queryByText(/Revenue chart/)).not.toBeInTheDocument();
  });
});
