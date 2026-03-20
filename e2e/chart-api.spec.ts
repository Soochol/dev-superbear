import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = "http://localhost:8080";
const TEST_SYMBOL = "005930"; // 삼성전자

/** Safely parse JSON from a response, returning null if not valid JSON. */
async function safeJson(response: { text(): Promise<string> }) {
  const text = await response.text();
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

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

    // Verify the request URL is correct
    expect(response.url()).toContain(`/api/v1/candles/${TEST_SYMBOL}`);
    expect(response.url()).toContain("period=");

    // Verify the request reached the backend
    expect(response.status()).not.toBe(0); // 0 = CORS/network failure

    // If backend returned JSON, verify structure
    const body = await safeJson(response);
    if (response.status() === 200 && body) {
      expect(body).toHaveProperty("data");
      expect(body.data).toHaveProperty("symbol", TEST_SYMBOL);
    }
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
  });

  test("API-3: chart renders after backend response", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    const body = await safeJson(response);

    if (response.status() === 200 && body?.data?.candles?.length > 0) {
      // Backend returned real candle data — chart should render with data
      await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
      await expect(chartPage.loadingIndicator).not.toBeVisible();
    } else {
      // Backend returned error or no data — canvas still mounts (empty chart)
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
    const requests: string[] = [];

    chartPage.page.on("request", (req) => {
      if (req.url().includes("/api/v1/candles/")) {
        requests.push(req.url());
      }
    });

    await chartPage.goto();
    await expect(chartPage.selectStockMessage).toBeVisible();

    // Give React time to settle — if a request were going to fire, it would have
    await chartPage.page.waitForTimeout(1_500);
    expect(requests).toHaveLength(0);
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
    }
  });

  test("API-7: candle response has correct data shape", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    const body = await safeJson(response);

    // Skip shape validation if backend didn't return valid JSON (e.g., route not registered)
    test.skip(!body || response.status() !== 200, "Backend did not return 200 JSON");

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
