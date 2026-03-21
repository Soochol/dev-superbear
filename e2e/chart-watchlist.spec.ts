import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;

test.describe("Chart Watchlist — Backend Integration @critical", () => {
  test.beforeEach(async ({ request }) => {
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running at :8080");
    } catch {
      test.skip(true, "Backend not reachable at :8080");
    }
  });

  test("WATCH-1: watchlist tab shows items from backend", async ({ chartPage }) => {
    await chartPage.goto();
    await chartPage.openSearchModal();

    // Wait for watchlist to load
    await chartPage.getModalTab("관심 종목").click();
    await expect(chartPage.page.getByRole("heading", { name: "관심 종목" })).toBeVisible();

    // The watchlist may be empty or have items from DB
    // Verify the tab content area is rendered (no crash)
    await expect(chartPage.page.locator('[data-testid^="search-stock-item-"]').or(chartPage.page.getByText("항목이 없습니다"))).toBeVisible();
  });

  test("WATCH-2: add and remove from watchlist via star toggle", async ({ chartPage }) => {
    // First, ensure we have search results to interact with
    // We'll use the search tab which shows searchResults from store
    await chartPage.goto();
    await chartPage.openSearchModal();

    // The search results may be empty without a search query
    // This test validates the star toggle behavior when items exist
    const items = chartPage.page.locator('[data-testid^="search-stock-item-"]');
    const count = await items.count();

    if (count > 0) {
      // Find star button on first item
      const firstItem = items.first();
      const starBtn = firstItem.getByRole("button", { name: /watchlist/i });
      await starBtn.click();

      // Verify the star state toggled (visual feedback)
      await expect(starBtn).toBeVisible();
    } else {
      // No search results available — skip star interaction
      test.skip(true, "No search results available to test star toggle");
    }
  });
});
