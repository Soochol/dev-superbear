import { test, expect } from "./fixtures/chart.fixture";

test.describe("Chart Page", () => {
  test("E2E-1: loads with default state", async ({ chartPage }) => {
    await chartPage.goto();

    // Search trigger shows placeholder
    await expect(chartPage.searchTrigger).toContainText("종목을 검색하세요");

    // Timeframe buttons visible (TradingView groups)
    const timeframes = ["1m", "5m", "15m", "30m", "1H", "4H", "1D", "1W", "1M"];
    for (const tf of timeframes) {
      await expect(chartPage.getTimeframeButton(tf)).toBeVisible();
    }

    // Indicator selector button visible
    await expect(chartPage.indicatorSelectorBtn).toBeVisible();
  });

  test("E2E-2: search modal opens and closes", async ({ chartPage }) => {
    await chartPage.goto();

    // Open modal
    await chartPage.openSearchModal();
    await expect(chartPage.page.getByRole("heading", { name: "종목 검색" })).toBeVisible();

    // Side nav tabs visible
    await expect(chartPage.getModalTab("종목 검색")).toBeVisible();
    await expect(chartPage.getModalTab("관심 종목")).toBeVisible();
    await expect(chartPage.getModalTab("최근 본 종목")).toBeVisible();

    // Close with Esc
    await chartPage.page.keyboard.press("Escape");
    await expect(chartPage.searchInput).not.toBeVisible();
  });

  test("E2E-3: search modal closes on backdrop click", async ({ chartPage }) => {
    await chartPage.goto();
    await chartPage.openSearchModal();
    await chartPage.closeSearchModal();
  });

  test("E2E-4: search modal tab switching", async ({ chartPage }) => {
    await chartPage.goto();
    await chartPage.openSearchModal();

    // Switch to watchlist tab
    await chartPage.getModalTab("관심 종목").click();
    await expect(chartPage.page.getByRole("heading", { name: "관심 종목" })).toBeVisible();

    // Switch to recent tab
    await chartPage.getModalTab("최근 본 종목").click();
    await expect(chartPage.page.getByRole("heading", { name: "최근 본 종목" })).toBeVisible();

    // Back to search
    await chartPage.getModalTab("종목 검색").click();
    await expect(chartPage.page.getByRole("heading", { name: "종목 검색" })).toBeVisible();
  });

  test("E2E-5: timeframe button click changes active state", async ({ chartPage }) => {
    await chartPage.goto();

    // Default active: 1D
    await expect(chartPage.getTimeframeButton("1D")).toHaveClass(/bg-nexus-accent/);

    // Click 1W
    await chartPage.clickTimeframe("1W");
    await expect(chartPage.getTimeframeButton("1W")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("1D")).not.toHaveClass(/bg-nexus-accent/);

    // Click 30m (new timeframe)
    await chartPage.clickTimeframe("30m");
    await expect(chartPage.getTimeframeButton("30m")).toHaveClass(/bg-nexus-accent/);
  });

  test("E2E-6: indicator selector opens and shows categories", async ({ chartPage }) => {
    await chartPage.goto();

    await chartPage.openIndicatorSelector();

    // Category headers visible
    await expect(chartPage.page.getByText("이동평균")).toBeVisible();
    await expect(chartPage.page.getByText("오실레이터")).toBeVisible();
    await expect(chartPage.page.getByText("밴드")).toBeVisible();

    // Some indicator options visible
    await expect(chartPage.getIndicatorOption("ma20")).toBeVisible();
    await expect(chartPage.getIndicatorOption("rsi")).toBeVisible();
    await expect(chartPage.getIndicatorOption("macd")).toBeVisible();
    await expect(chartPage.getIndicatorOption("bb")).toBeVisible();
  });

  test("E2E-7: toggling indicator shows/hides panel", async ({ chartPage }) => {
    await chartPage.goto();

    // RSI panel should not be visible initially
    await expect(chartPage.getIndicatorPanel("rsi")).not.toBeVisible();

    // Toggle RSI on
    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("rsi").click();

    // RSI panel should appear
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();
  });

  test("E2E-8: main chart renders canvas", async ({ chartPage }) => {
    await chartPage.goto("005930");
    const canvas = chartPage.canvas;
    await expect(canvas).toBeVisible({ timeout: 10000 });
  });
});
