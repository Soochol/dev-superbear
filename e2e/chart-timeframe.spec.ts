import { test, expect } from "./fixtures/chart.fixture";
import { interceptCandlesWithMockData, generateMockCandles } from "./helpers/mock-candles";

const TEST_SYMBOL = "005930";

test.describe("Chart Timeframe Switching @critical", () => {
  test("TF-1: minute candles — 1m, 5m, 15m, 30m trigger correct API params", async ({
    chartPage,
  }) => {
    await interceptCandlesWithMockData(chartPage.page, TEST_SYMBOL);
    await chartPage.goto(TEST_SYMBOL);

    // Wait for initial load
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    const minuteFrames = ["1m", "5m", "15m", "30m"];
    for (const tf of minuteFrames) {
      const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
      await chartPage.clickTimeframe(tf);
      const response = await responsePromise;

      expect(response.url()).toContain(`period=${tf}`);

      // Active button should have accent color
      await expect(chartPage.getTimeframeButton(tf)).toHaveClass(/bg-nexus-accent/);
    }
  });

  test("TF-2: daily candles — 1W, 1M trigger correct API params", async ({
    chartPage,
  }) => {
    await interceptCandlesWithMockData(chartPage.page, TEST_SYMBOL);
    await chartPage.goto(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // 1D is the default active timeframe, so start from 1W
    const dailyFrames = ["1W", "1M"];
    for (const tf of dailyFrames) {
      const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
      await chartPage.clickTimeframe(tf);
      const response = await responsePromise;

      expect(response.url()).toContain(`period=${tf}`);
      await expect(chartPage.getTimeframeButton(tf)).toHaveClass(/bg-nexus-accent/);
    }
  });

  test("TF-3: switching timeframe changes active button and deactivates previous", async ({
    chartPage,
  }) => {
    await interceptCandlesWithMockData(chartPage.page, TEST_SYMBOL);
    await chartPage.goto(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Default: 1D active
    await expect(chartPage.getTimeframeButton("1D")).toHaveClass(/bg-nexus-accent/);

    // Switch to 1W
    await chartPage.clickTimeframe("1W");
    await expect(chartPage.getTimeframeButton("1W")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("1D")).not.toHaveClass(/bg-nexus-accent/);

    // Switch to 5m
    await chartPage.clickTimeframe("5m");
    await expect(chartPage.getTimeframeButton("5m")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("1W")).not.toHaveClass(/bg-nexus-accent/);

    // Switch to 1M
    await chartPage.clickTimeframe("1M");
    await expect(chartPage.getTimeframeButton("1M")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("5m")).not.toHaveClass(/bg-nexus-accent/);
  });

  test("TF-4: chart re-renders after timeframe switch with new data", async ({
    chartPage,
  }) => {
    // Use different candle counts per period to verify data changes
    let requestCount = 0;

    await chartPage.page.route(`**/api/v1/candles/${TEST_SYMBOL}*`, async (route) => {
      requestCount++;
      const url = new URL(route.request().url());
      const period = url.searchParams.get("period") ?? "1D";

      // Return different data lengths per timeframe to differentiate
      const count = period.includes("m") || period.includes("H") ? 30 : 60;
      const candles = generateMockCandles(count);

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: { symbol: TEST_SYMBOL, candles } }),
      });
    });

    await chartPage.goto(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    const initialRequestCount = requestCount;

    // Switch timeframe — new request should fire
    await chartPage.clickTimeframe("1W");
    await chartPage.page.waitForTimeout(500);

    expect(requestCount).toBeGreaterThan(initialRequestCount);

    // Canvas should still be visible (chart re-rendered)
    await expect(chartPage.canvas).toBeVisible();
    await expect(chartPage.loadingIndicator).not.toBeVisible();
  });

  test("TF-5: hour candles — 1H, 4H trigger correct API params", async ({
    chartPage,
  }) => {
    await interceptCandlesWithMockData(chartPage.page, TEST_SYMBOL);
    await chartPage.goto(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    const hourFrames = ["1H", "4H"];
    for (const tf of hourFrames) {
      const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
      await chartPage.clickTimeframe(tf);
      const response = await responsePromise;

      expect(response.url()).toContain(`period=${tf}`);
      await expect(chartPage.getTimeframeButton(tf)).toHaveClass(/bg-nexus-accent/);
    }
  });
});
