import { test, expect } from "./fixtures/chart.fixture";

const BACKEND_URL = `http://localhost:${process.env.E2E_PORT_API ?? 3300}`;
const TEST_SYMBOL = "005930";

test.describe("Chart Stock Flow @critical", () => {
  test.beforeEach(async ({ request }) => {
    try {
      const res = await request.get(`${BACKEND_URL}/api/v1/health`);
      test.skip(!res.ok(), "Backend not running");
    } catch {
      test.skip(true, "Backend not reachable");
    }
  });

  test("FLOW-1: topbar shows stock info when navigating with symbol", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    // Wait for chart to render
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Topbar should display stock info instead of placeholder
    await expect(chartPage.searchTrigger).not.toContainText("종목을 검색하세요");
    await expect(chartPage.searchTrigger).toContainText(TEST_SYMBOL);
  });

  test("FLOW-2: bottom panel shows empty state when no stock selected", async ({
    chartPage,
  }) => {
    await chartPage.goto();

    await expect(chartPage.bottomPanelEmpty).toBeVisible();
    await expect(chartPage.bottomPanelEmpty).toContainText(
      "Select a stock to view details",
    );

    // Grid should NOT be visible
    await expect(chartPage.bottomPanelGrid).not.toBeVisible();
  });

  test("FLOW-3: bottom panel shows 3-column grid when stock is selected", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);

    // Wait for chart to render (ensures stock is loaded)
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Grid should be visible
    await expect(chartPage.bottomPanelGrid).toBeVisible();

    // Empty state should NOT be visible
    await expect(chartPage.bottomPanelEmpty).not.toBeVisible();
  });

  test("FLOW-4: selected stock appears in recent stocks tab", async ({
    chartPage,
  }) => {
    await chartPage.gotoAndWaitForCandles(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });

    // Open search modal and switch to recent tab
    await chartPage.openSearchModal();
    await chartPage.getModalTab("최근 본 종목").click();
    await expect(
      chartPage.page.getByRole("heading", { name: "최근 본 종목" }),
    ).toBeVisible();

    // The selected symbol should appear in the recent list
    const recentItem = chartPage.getStockItem(TEST_SYMBOL);
    await expect(recentItem).toBeVisible({ timeout: 5_000 });
  });
});
