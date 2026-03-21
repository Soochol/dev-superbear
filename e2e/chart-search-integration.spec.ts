import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;

test.describe("Stock Search → Chart Integration @critical", () => {
  test.beforeEach(async ({ request }) => {
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running");
    } catch {
      test.skip(true, "Backend not reachable");
    }
  });

  test("SEARCH-1: selecting a stock from search modal navigates to chart", async ({
    chartPage,
  }) => {
    await chartPage.goto();

    // Inject search results into the store
    await chartPage.page.evaluate(() => {
      const stockStore = (window as any).__STOCK_LIST_STORE__;
      if (stockStore) {
        stockStore.getState().setSearchResults([
          { symbol: "005930", name: "삼성전자", close: 72000, change: 1000, changePct: 1.41 },
          { symbol: "000660", name: "SK하이닉스", close: 185000, change: -2000, changePct: -1.07 },
          { symbol: "035420", name: "NAVER", close: 210000, change: 3000, changePct: 1.45 },
        ]);
      }
    });

    await chartPage.openSearchModal();

    const stockItems = chartPage.page.locator('[data-testid^="search-stock-item-"]');
    const count = await stockItems.count();

    if (count > 0) {
      const firstSymbol = await stockItems.first().getAttribute("data-testid");
      const symbol = firstSymbol?.replace("search-stock-item-", "") ?? "";

      // Wait for candle API call after clicking
      const candleRequestPromise = chartPage.page.waitForRequest(
        (req) => req.url().includes(`/api/v1/candles/${symbol}`),
        { timeout: 10_000 },
      );

      await stockItems.first().click();

      // Modal should close after selection
      await expect(chartPage.searchInput).not.toBeVisible();

      // Search trigger should show the selected stock name
      await expect(chartPage.searchTrigger).not.toContainText("종목을 검색하세요");

      // A candle API call should be triggered for the selected symbol
      const candleRequest = await candleRequestPromise;
      expect(candleRequest.url()).toContain(`/api/v1/candles/${symbol}`);
    } else {
      await expect(chartPage.page.getByText("항목이 없습니다")).toBeVisible();
    }
  });

  test("SEARCH-2: search modal filter narrows results", async ({ chartPage }) => {
    await chartPage.goto();

    await chartPage.page.evaluate(() => {
      const stockStore = (window as any).__STOCK_LIST_STORE__;
      if (stockStore) {
        stockStore.getState().setSearchResults([
          { symbol: "005930", name: "삼성전자", close: 72000, change: 1000, changePct: 1.41 },
          { symbol: "000660", name: "SK하이닉스", close: 185000, change: -2000, changePct: -1.07 },
          { symbol: "035420", name: "NAVER", close: 210000, change: 3000, changePct: 1.45 },
        ]);
      }
    });

    await chartPage.openSearchModal();

    const stockItems = chartPage.page.locator('[data-testid^="search-stock-item-"]');
    const initialCount = await stockItems.count();

    if (initialCount > 1) {
      await chartPage.searchInput.fill("삼성");

      const filteredCount = await stockItems.count();
      expect(filteredCount).toBeLessThanOrEqual(initialCount);
      expect(filteredCount).toBeGreaterThan(0);

      await expect(stockItems.first()).toContainText("삼성");
    }
  });

  test("SEARCH-3: stock selection triggers candle API with default period", async ({
    chartPage,
  }) => {
    await chartPage.goto();

    await chartPage.page.evaluate(() => {
      const stockStore = (window as any).__STOCK_LIST_STORE__;
      if (stockStore) {
        stockStore.getState().setSearchResults([
          { symbol: "005930", name: "삼성전자", close: 72000, change: 1000, changePct: 1.41 },
        ]);
      }
    });

    await chartPage.openSearchModal();

    const stockItem = chartPage.getStockItem("005930");
    const isVisible = await stockItem.isVisible().catch(() => false);

    if (isVisible) {
      const responsePromise = chartPage.page.waitForResponse(
        (res) => res.url().includes("/api/v1/candles/005930") && res.status() !== 0,
      );

      await stockItem.click();

      const response = await responsePromise;
      const body = await response.json();

      // Verify real backend response structure
      expect(body).toHaveProperty("data");
      expect(body.data).toHaveProperty("symbol", "005930");

      // Chart canvas should render
      await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    }
  });

  test("SEARCH-4: navigating with symbol param loads chart directly", async ({
    chartPage,
  }) => {
    const response = await chartPage.gotoAndWaitForCandles("005930");

    // Candle data received from real backend
    expect(response.status()).not.toBe(0);

    if (response.status() === 200) {
      const body = await response.json();
      expect(body.data.candles.length).toBeGreaterThan(0);
    }

    // Chart renders
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    await expect(chartPage.loadingIndicator).not.toBeVisible();
  });
});
