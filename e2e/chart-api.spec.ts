import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = "http://localhost:8080";
const TEST_SYMBOL = "005930"; // 삼성전자

test.describe("Chart Page — Backend Integration @critical", () => {
  test.beforeEach(async ({ request }) => {
    // Backend health check — skip all tests if backend is not running
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running at :8080");
    } catch {
      test.skip(true, "Backend not reachable at :8080");
    }
  });

  test("API-1: fetches candles when navigating with symbol", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    // Verify the request reached the backend and got a response (not 5xx)
    expect(response.status()).toBeLessThan(500);

    // Verify response structure
    const body = await response.json();
    expect(body).toHaveProperty("data");
    expect(body.data).toHaveProperty("symbol", TEST_SYMBOL);
  });

  test("API-2: sends correct period param on timeframe change", async ({
    chartPage,
  }) => {
    // Navigate and wait for initial 1D candle request
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    // Click "1W" and observe the new API call
    const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("1W");
    const response = await responsePromise;

    expect(response.url()).toContain("period=1W");
    expect(response.status()).toBeLessThan(500);
  });

  test("API-3: chart renders with candle data from backend", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    const body = await response.json();

    if (response.status() === 200 && body.data?.candles?.length > 0) {
      // Backend returned real candle data — chart should render
      await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
      await expect(chartPage.loadingIndicator).not.toBeVisible();
    } else {
      // Backend couldn't fetch from KIS — chart still renders (empty canvas)
      await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    }
  });

  test("API-4: loading state appears during data fetch", async ({
    chartPage,
  }) => {
    // Throttle the candle API to guarantee loading state is visible
    await chartPage.page.route("**/api/v1/candles/**", async (route) => {
      await new Promise((r) => setTimeout(r, 300));
      await route.continue();
    });

    await chartPage.goto(TEST_SYMBOL);
    await expect(chartPage.loadingIndicator).toBeVisible();

    // After data arrives, loading should disappear
    await expect(chartPage.loadingIndicator).not.toBeVisible({
      timeout: 15_000,
    });
  });

  test("API-5: no candle API call without symbol", async ({ chartPage }) => {
    let apiCalled = false;

    chartPage.page.on("request", (req) => {
      if (req.url().includes("/api/v1/candles/")) {
        apiCalled = true;
      }
    });

    await chartPage.goto();
    // Give time for any potential request to fire
    await chartPage.page.waitForTimeout(2_000);

    expect(apiCalled).toBe(false);
    await expect(chartPage.selectStockMessage).toBeVisible();
  });

  test("API-6: each timeframe switch triggers a new API call", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    const timeframes = ["1W", "1M", "1H", "5m"];
    for (const tf of timeframes) {
      const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
      await chartPage.clickTimeframe(tf);
      const response = await responsePromise;

      expect(response.url()).toContain(`period=${tf}`);
      expect(response.status()).toBeLessThan(500);
    }
  });

  test("API-7: candle response has correct data shape", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    expect(response.status()).toBeLessThan(500);
    const body = await response.json();

    // Always has data.symbol
    expect(body.data.symbol).toBe(TEST_SYMBOL);

    // If candles exist, verify their shape
    if (body.data.candles?.length > 0) {
      const candle = body.data.candles[0];
      expect(candle).toHaveProperty("time");
      expect(candle).toHaveProperty("open");
      expect(candle).toHaveProperty("high");
      expect(candle).toHaveProperty("low");
      expect(candle).toHaveProperty("close");
      expect(candle).toHaveProperty("volume");
    }
  });
});
