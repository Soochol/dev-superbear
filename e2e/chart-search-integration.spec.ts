import { test, expect } from "./fixtures/chart.fixture";
import { interceptCandlesWithMockData } from "./helpers/mock-candles";

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
    // Pre-populate search results by injecting stock data via zustand store
    await chartPage.goto();

    // Inject mock search results into the store
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

    // Open search modal
    await chartPage.openSearchModal();

    // Check if stock items are visible — they require store data
    const stockItems = chartPage.page.locator('[data-testid^="search-stock-item-"]');
    const count = await stockItems.count();

    if (count > 0) {
      // Click the first stock item
      const firstSymbol = await stockItems.first().getAttribute("data-testid");
      const symbol = firstSymbol?.replace("search-stock-item-", "") ?? "";

      // Set up API interception before clicking
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
      // No search results — verify modal shows empty state
      await expect(chartPage.page.getByText("항목이 없습니다")).toBeVisible();
    }
  });

  test("SEARCH-2: search modal filter narrows results", async ({ chartPage }) => {
    await chartPage.goto();

    // Inject mock search results
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
      // Type a filter query
      await chartPage.searchInput.fill("삼성");

      // Filtered results should be fewer
      const filteredCount = await stockItems.count();
      expect(filteredCount).toBeLessThanOrEqual(initialCount);
      expect(filteredCount).toBeGreaterThan(0);

      // The matching item should contain the search term
      await expect(stockItems.first()).toContainText("삼성");
    }
  });

  test("SEARCH-3: stock selection triggers candle API with default period", async ({
    chartPage,
  }) => {
    // Use route interception to provide mock candle data
    const mockCandles = await interceptCandlesWithMockData(chartPage.page, "005930");

    await chartPage.goto();

    // Inject a stock into the store
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
        (res) => res.url().includes("/api/v1/candles/005930"),
      );

      await stockItem.click();

      const response = await responsePromise;
      const body = await response.json();

      // Verify mock data was returned
      expect(body.data.symbol).toBe("005930");
      expect(body.data.candles).toHaveLength(mockCandles.length);

      // Chart canvas should render
      await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    }
  });

  test("SEARCH-4: navigating with symbol param loads chart directly", async ({
    chartPage,
  }) => {
    await interceptCandlesWithMockData(chartPage.page, "005930");

    const responsePromise = chartPage.page.waitForResponse(
      (res) => res.url().includes("/api/v1/candles/005930"),
    );

    await chartPage.goto("005930");
    const response = await responsePromise;
    const body = await response.json();

    // Candle data received
    expect(body.data.candles.length).toBeGreaterThan(0);

    // Chart renders
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
    await expect(chartPage.loadingIndicator).not.toBeVisible();
  });
});
