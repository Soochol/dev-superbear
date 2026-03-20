/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";
import { ChartPageLayout } from "../ui/ChartPageLayout";

jest.mock("../ui/ChartTopbar", () => ({ ChartTopbar: () => <div data-testid="topbar" /> }));
jest.mock("@/widgets/stock-search-modal", () => ({
  StockSearchModal: () => <div data-testid="search-modal" />,
  useSearchModalStore: jest.fn(),
}));
jest.mock("@/widgets/bottom-info-panel", () => ({ BottomInfoPanel: () => <div data-testid="bottom-panel" /> }));
jest.mock("@/features/chart", () => ({
  MainChart: () => <div data-testid="main-chart" />,
}));

describe("ChartPageLayout", () => {
  it("renders MainChart, topbar, bottom panel, and search modal", () => {
    render(<ChartPageLayout />);
    expect(screen.getByTestId("main-chart")).toBeInTheDocument();
    expect(screen.getByTestId("topbar")).toBeInTheDocument();
    expect(screen.getByTestId("bottom-panel")).toBeInTheDocument();
    expect(screen.getByTestId("search-modal")).toBeInTheDocument();
  });

  it("does not render stock-list-sidebar", () => {
    render(<ChartPageLayout />);
    expect(screen.queryByTestId("sidebar")).not.toBeInTheDocument();
  });
});
