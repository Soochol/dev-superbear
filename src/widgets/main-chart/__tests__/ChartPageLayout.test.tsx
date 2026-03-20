/** @jest-environment jsdom */
import { render, screen } from "@testing-library/react";
import { ChartPageLayout } from "../ui/ChartPageLayout";

jest.mock("../ui/ChartTopbar", () => ({ ChartTopbar: () => <div data-testid="topbar" /> }));
jest.mock("@/widgets/stock-list-sidebar", () => ({ StockListSidebar: () => <div data-testid="sidebar" /> }));
jest.mock("@/widgets/bottom-info-panel", () => ({ BottomInfoPanel: () => <div data-testid="bottom-panel" /> }));
jest.mock("@/features/chart", () => ({
  MainChart: () => <div data-testid="main-chart" />,
}));

describe("ChartPageLayout", () => {
  it("renders MainChart component instead of placeholder", () => {
    render(<ChartPageLayout />);
    expect(screen.getByTestId("main-chart")).toBeInTheDocument();
    expect(screen.queryByText("Main Chart Area")).not.toBeInTheDocument();
  });
});
