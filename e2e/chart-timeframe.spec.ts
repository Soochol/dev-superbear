import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;
const TEST_SYMBOL = "005930";

test.describe("Chart Timeframe Switching @critical", () => {
  test.beforeEach(async ({ request }) => {
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running");
    } catch {
      test.skip(true, "Backend not reachable");
    }
  });

  test("TF-1: minute candles — 1m, 5m, 15m, 30m trigger correct API params", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
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
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

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
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Default: 1D active
    await expect(chartPage.getTimeframeButton("1D")).toHaveClass(/bg-nexus-accent/);

    // Switch to 1W
    const res1W = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("1W");
    await res1W;
    await expect(chartPage.getTimeframeButton("1W")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("1D")).not.toHaveClass(/bg-nexus-accent/);

    // Switch to 5m
    const res5m = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("5m");
    await res5m;
    await expect(chartPage.getTimeframeButton("5m")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("1W")).not.toHaveClass(/bg-nexus-accent/);

    // Switch to 1M
    const res1M = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("1M");
    await res1M;
    await expect(chartPage.getTimeframeButton("1M")).toHaveClass(/bg-nexus-accent/);
    await expect(chartPage.getTimeframeButton("5m")).not.toHaveClass(/bg-nexus-accent/);
  });

  test("TF-4: chart re-renders after timeframe switch with new data", async ({
    chartPage,
  }) => {
    const requests: string[] = [];

    chartPage.page.on("request", (req) => {
      if (req.url().includes(`/api/v1/candles/${TEST_SYMBOL}`)) {
        requests.push(req.url());
      }
    });

    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    const initialCount = requests.length;

    // Switch timeframe — new request should fire
    const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("1W");
    await responsePromise;

    expect(requests.length).toBeGreaterThan(initialCount);

    // Canvas should still be visible (chart re-rendered)
    await expect(chartPage.canvas).toBeVisible();
    await expect(chartPage.loadingIndicator).not.toBeVisible();
  });

  test("TF-5: hour candles — 1H, 4H trigger correct API params", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
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
