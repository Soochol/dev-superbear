import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;
const TEST_SYMBOL = "005930";
const TEST_STOCK_NAME = "삼성전자";

test.describe("Watchlist CRUD Cycle @critical", () => {
  test.beforeEach(async ({ request }) => {
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running");
    } catch {
      test.skip(true, "Backend not reachable");
    }
  });

  test("CRUD-1: add item to watchlist via API, verify in modal", async ({
    chartPage,
  }) => {
    // Clean up first — remove if exists (ignore errors)
    await chartPage.page.request
      .delete(`${BACKEND_URL}/api/v1/watchlist/${TEST_SYMBOL}`)
      .catch(() => {});

    // Add via API
    const addRes = await chartPage.page.request.post(
      `${BACKEND_URL}/api/v1/watchlist`,
      { data: { symbol: TEST_SYMBOL, name: TEST_STOCK_NAME } },
    );
    expect(addRes.ok()).toBe(true);

    // Open chart and check watchlist tab
    await chartPage.goto();
    await chartPage.openSearchModal();
    await chartPage.getModalTab("관심 종목").click();
    await expect(
      chartPage.page.getByRole("heading", { name: "관심 종목" }),
    ).toBeVisible();

    // The added stock should appear
    await expect(
      chartPage.getStockItem(TEST_SYMBOL),
    ).toBeVisible({ timeout: 5_000 });
  });

  test("CRUD-2: remove item from watchlist via API, verify removal in modal", async ({
    chartPage,
  }) => {
    // Ensure item exists first
    await chartPage.page.request
      .post(`${BACKEND_URL}/api/v1/watchlist`, {
        data: { symbol: TEST_SYMBOL, name: TEST_STOCK_NAME },
      })
      .catch(() => {});

    // Remove via API
    const removeRes = await chartPage.page.request.delete(
      `${BACKEND_URL}/api/v1/watchlist/${TEST_SYMBOL}`,
    );
    expect(removeRes.ok()).toBe(true);

    // Open chart and check watchlist tab
    await chartPage.goto();
    await chartPage.openSearchModal();
    await chartPage.getModalTab("관심 종목").click();

    // The removed stock should not appear
    await expect(
      chartPage.getStockItem(TEST_SYMBOL),
    ).not.toBeVisible({ timeout: 3_000 });
  });

  test("CRUD-3: full add-list-remove-list cycle via API", async ({
    chartPage,
  }) => {
    // Clean slate
    await chartPage.page.request
      .delete(`${BACKEND_URL}/api/v1/watchlist/${TEST_SYMBOL}`)
      .catch(() => {});

    // 1. List — should NOT contain test symbol
    const listBefore = await chartPage.page.request.get(
      `${BACKEND_URL}/api/v1/watchlist`,
    );
    expect(listBefore.ok()).toBe(true);
    const beforeBody = await listBefore.json();
    const beforeSymbols = (beforeBody.data ?? []).map(
      (w: { symbol: string }) => w.symbol,
    );
    expect(beforeSymbols).not.toContain(TEST_SYMBOL);

    // 2. Add
    const addRes = await chartPage.page.request.post(
      `${BACKEND_URL}/api/v1/watchlist`,
      { data: { symbol: TEST_SYMBOL, name: TEST_STOCK_NAME } },
    );
    expect(addRes.ok()).toBe(true);

    // 3. List — should contain test symbol
    const listAfterAdd = await chartPage.page.request.get(
      `${BACKEND_URL}/api/v1/watchlist`,
    );
    const afterAddBody = await listAfterAdd.json();
    const afterAddSymbols = (afterAddBody.data ?? []).map(
      (w: { symbol: string }) => w.symbol,
    );
    expect(afterAddSymbols).toContain(TEST_SYMBOL);

    // 4. Remove
    const removeRes = await chartPage.page.request.delete(
      `${BACKEND_URL}/api/v1/watchlist/${TEST_SYMBOL}`,
    );
    expect(removeRes.ok()).toBe(true);

    // 5. List — should NOT contain test symbol again
    const listAfterRemove = await chartPage.page.request.get(
      `${BACKEND_URL}/api/v1/watchlist`,
    );
    const afterRemoveBody = await listAfterRemove.json();
    const afterRemoveSymbols = (afterRemoveBody.data ?? []).map(
      (w: { symbol: string }) => w.symbol,
    );
    expect(afterRemoveSymbols).not.toContain(TEST_SYMBOL);
  });

  test("CRUD-4: watchlist fetch happens on modal open", async ({
    chartPage,
  }) => {
    // Ensure one item in watchlist
    await chartPage.page.request
      .post(`${BACKEND_URL}/api/v1/watchlist`, {
        data: { symbol: TEST_SYMBOL, name: TEST_STOCK_NAME },
      })
      .catch(() => {});

    await chartPage.goto();

    // Intercept the watchlist API call
    const watchlistPromise = chartPage.page.waitForResponse(
      (res) => res.url().includes("/api/v1/watchlist") && res.request().method() === "GET",
    );

    await chartPage.openSearchModal();
    await chartPage.getModalTab("관심 종목").click();

    const watchlistResponse = await watchlistPromise;
    expect(watchlistResponse.status()).toBe(200);

    const body = await watchlistResponse.json();
    expect(body.data).toBeDefined();
    expect(body.data.length).toBeGreaterThan(0);

    // Clean up
    await chartPage.page.request
      .delete(`${BACKEND_URL}/api/v1/watchlist/${TEST_SYMBOL}`)
      .catch(() => {});
  });
});
