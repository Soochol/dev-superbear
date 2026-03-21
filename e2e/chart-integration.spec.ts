import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;
const TEST_SYMBOL = "005930"; // 삼성전자

test.describe("Chart Full Integration @critical", () => {
  test.beforeEach(async ({ request }) => {
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running");
    } catch {
      test.skip(true, "Backend not reachable");
    }
  });

  test("INTEG-1: symbol param loads chart with candles and bottom panel", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    // Backend returned candle data
    expect(response.status()).toBe(200);
    const body = await response.json();
    expect(body.data.symbol).toBe(TEST_SYMBOL);
    expect(body.data.candles.length).toBeGreaterThan(0);

    // Chart canvas renders
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    await expect(chartPage.loadingIndicator).not.toBeVisible();

    // Bottom panel shows stock details (not empty placeholder)
    await expect(chartPage.bottomPanelEmpty).not.toBeVisible();
    await expect(chartPage.bottomPanelGrid).toBeVisible();

    // Sub-panels rendered
    await expect(chartPage.page.getByText("Financials")).toBeVisible();
    await expect(chartPage.page.getByText("AI Fusion")).toBeVisible();
    await expect(chartPage.page.getByText("Sector Compare")).toBeVisible();
  });

  test("INTEG-2: timeframe switch fetches new data and re-renders chart", async ({
    chartPage,
  }) => {
    const initialResponse = await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    // Verify initial data loaded
    const initialBody = await initialResponse.json();
    expect(initialBody.data.candles.length).toBeGreaterThan(0);

    // Switch to 1W
    const weeklyPromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("1W");
    const weeklyResponse = await weeklyPromise;

    expect(weeklyResponse.status()).toBe(200);
    expect(weeklyResponse.url()).toContain("period=1W");

    const weeklyBody = await weeklyResponse.json();
    expect(weeklyBody.data.candles.length).toBeGreaterThan(0);

    // Canvas still visible after re-render
    await expect(chartPage.canvas).toBeVisible();
  });

  test("INTEG-3: indicator + timeframe switch preserves indicator panel", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Enable RSI indicator
    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("rsi").click();
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();

    // Switch timeframe — RSI panel should persist
    const responsePromise = chartPage.waitForCandleResponse(TEST_SYMBOL);
    await chartPage.clickTimeframe("1W");
    await responsePromise;

    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();
    await expect(chartPage.getIndicatorPanel("rsi").locator("canvas").first()).toBeVisible({
      timeout: 5_000,
    });
  });

  test("INTEG-4: no symbol shows empty state without API calls", async ({
    chartPage,
  }) => {
    const requests: string[] = [];
    chartPage.page.on("request", (req) => {
      if (req.url().includes("/api/v1/candles/")) {
        requests.push(req.url());
      }
    });

    await chartPage.goto();

    // Topbar shows placeholder
    await expect(chartPage.searchTrigger).toContainText("종목을 검색하세요");

    // Bottom panel shows empty placeholder
    await expect(chartPage.bottomPanelEmpty).toBeVisible();
    await expect(chartPage.bottomPanelGrid).not.toBeVisible();

    // No candle API calls fired
    await expect(async () => {
      expect(requests).toHaveLength(0);
    }).toPass({ timeout: 2_000 });
  });

  test("INTEG-5: financials panel fetches data for selected symbol", async ({
    chartPage,
  }) => {
    const financialsPromise = chartPage.page.waitForResponse(
      (res) => res.url().includes(`/api/v1/financials/${TEST_SYMBOL}`) && !res.url().includes("/sector"),
    );

    await chartPage.goto(TEST_SYMBOL);

    // Wait for financials API call
    const financialsResponse = await financialsPromise;
    expect(financialsResponse.status()).not.toBe(0);

    // Financials section rendered with metric labels
    await expect(chartPage.page.getByText("Financials")).toBeVisible();
    await expect(chartPage.page.getByText("PER")).toBeVisible();
    await expect(chartPage.page.getByText("ROE")).toBeVisible();
  });

  test("INTEG-6: sector compare panel fetches data for selected symbol", async ({
    chartPage,
  }) => {
    const sectorPromise = chartPage.page.waitForResponse(
      (res) => res.url().includes(`/api/v1/financials/${TEST_SYMBOL}/sector`),
    );

    await chartPage.goto(TEST_SYMBOL);

    const sectorResponse = await sectorPromise;
    expect(sectorResponse.status()).not.toBe(0);

    await expect(chartPage.page.getByText("Sector Compare")).toBeVisible();
  });
});
