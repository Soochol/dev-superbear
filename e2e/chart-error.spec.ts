import { test, expect } from "./fixtures/chart.fixture";

const TEST_SYMBOL = "005930";

test.describe("Chart Error States @critical", () => {
  test("ERR-1: candle API 500 — chart does not crash, canvas still mounts", async ({
    chartPage,
  }) => {
    await chartPage.page.route(`**/api/v1/candles/${TEST_SYMBOL}*`, (route) => {
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "Internal Server Error" }),
      });
    });

    await chartPage.goto(TEST_SYMBOL);

    // Page should not crash — canvas container still mounts (empty chart)
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Loading indicator should disappear (error handled)
    await expect(chartPage.loadingIndicator).not.toBeVisible({ timeout: 5_000 });
  });

  test("ERR-2: candle API network error — graceful degradation", async ({
    chartPage,
  }) => {
    await chartPage.page.route(`**/api/v1/candles/${TEST_SYMBOL}*`, (route) => {
      route.abort("connectionrefused");
    });

    await chartPage.goto(TEST_SYMBOL);

    // Page should not show unhandled error
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    await expect(chartPage.loadingIndicator).not.toBeVisible({ timeout: 5_000 });

    // Topbar still functional
    await expect(chartPage.searchTrigger).toBeVisible();
    await expect(chartPage.indicatorSelectorBtn).toBeVisible();
  });

  test("ERR-3: slow candle API — loading indicator visible during delay", async ({
    chartPage,
  }) => {
    await chartPage.page.route(`**/api/v1/candles/${TEST_SYMBOL}*`, async (route) => {
      await new Promise((r) => setTimeout(r, 2_000));
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: { symbol: TEST_SYMBOL, candles: [] },
        }),
      });
    });

    await chartPage.goto(TEST_SYMBOL);

    // Loading indicator should be visible during delay
    await expect(chartPage.loadingIndicator).toBeVisible();

    // After response, loading disappears
    await expect(chartPage.loadingIndicator).not.toBeVisible({ timeout: 5_000 });
  });

  test("ERR-4: invalid symbol — no crash, empty chart state", async ({
    chartPage,
  }) => {
    const invalidSymbol = "XXXXXX";

    await chartPage.page.route(`**/api/v1/candles/${invalidSymbol}*`, (route) => {
      route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({ error: "Symbol not found" }),
      });
    });

    await chartPage.goto(invalidSymbol);

    // Page renders without crash
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    await expect(chartPage.loadingIndicator).not.toBeVisible({ timeout: 5_000 });
  });

  test("ERR-5: financials API failure — chart still works", async ({
    chartPage,
  }) => {
    // Fail financials but allow candles through
    await chartPage.page.route(`**/api/v1/financials/**`, (route) => {
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "Internal Server Error" }),
      });
    });

    await chartPage.page.route(`**/api/v1/candles/${TEST_SYMBOL}*`, (route) => {
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            symbol: TEST_SYMBOL,
            candles: Array.from({ length: 30 }, (_, i) => ({
              time: `2025-01-${String(i + 1).padStart(2, "0")}`,
              open: 70000 + i * 100,
              high: 70500 + i * 100,
              low: 69500 + i * 100,
              close: 70200 + i * 100,
              volume: 1000000 + i * 10000,
            })),
          },
        }),
      });
    });

    await chartPage.goto(TEST_SYMBOL);

    // Chart renders fine
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Bottom panel rendered (not crashed by financials error)
    await expect(chartPage.bottomPanelGrid).toBeVisible();

    // Financials shows fallback (dash values)
    await expect(chartPage.page.getByText("Financials")).toBeVisible();
  });

  test("ERR-6: watchlist API failure — modal still functional", async ({
    chartPage,
  }) => {
    await chartPage.page.route("**/api/v1/watchlist", (route) => {
      if (route.request().method() === "GET") {
        route.fulfill({
          status: 500,
          contentType: "application/json",
          body: JSON.stringify({ error: "DB connection failed" }),
        });
      } else {
        route.continue();
      }
    });

    await chartPage.goto();
    await chartPage.openSearchModal();

    // Switch to watchlist tab — should not crash
    await chartPage.getModalTab("관심 종목").click();
    await expect(
      chartPage.page.getByRole("heading", { name: "관심 종목" }),
    ).toBeVisible();

    // Other tabs still work
    await chartPage.getModalTab("종목 검색").click();
    await expect(
      chartPage.page.getByRole("heading", { name: "종목 검색" }),
    ).toBeVisible();
  });
});
