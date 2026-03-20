import { test, expect } from "@playwright/test";

test.describe("Chart Page", () => {
  test("E2E-1: loads with default state", async ({ page }) => {
    await page.goto("/chart");

    // Verify "Select a stock" text visible (no stock selected)
    await expect(page.getByText("Select a stock", { exact: true })).toBeVisible();

    // Verify timeframe buttons visible
    const timeframes = ["1m", "5m", "15m", "1H", "1D", "1W", "1M"];
    for (const tf of timeframes) {
      await expect(
        page.getByRole("button", { name: tf, exact: true })
      ).toBeVisible();
    }

    // Verify sidebar tabs rendered
    await expect(page.getByText("검색결과")).toBeVisible();
    await expect(page.getByText("관심종목")).toBeVisible();
    await expect(page.getByText("최근")).toBeVisible();
  });

  test("E2E-2: URL param sets current stock", async ({ page }) => {
    await page.goto("/chart?symbol=005930");

    // Verify topbar shows the symbol
    await expect(page.getByText("005930").first()).toBeVisible();
  });

  test("E2E-3: sidebar tab switching", async ({ page }) => {
    await page.goto("/chart");

    // Click "관심종목" tab and verify it becomes active
    const watchlistTab = page.getByText("관심종목");
    await watchlistTab.click();
    await expect(watchlistTab).toHaveClass(/text-nexus-accent/);
    await expect(watchlistTab).toHaveClass(/border-nexus-accent/);

    // Click "최근" tab and verify it becomes active
    const recentTab = page.getByText("최근");
    await recentTab.click();
    await expect(recentTab).toHaveClass(/text-nexus-accent/);
    await expect(recentTab).toHaveClass(/border-nexus-accent/);

    // Click "검색결과" tab and verify it's back to active
    const resultsTab = page.getByText("검색결과");
    await resultsTab.click();
    await expect(resultsTab).toHaveClass(/text-nexus-accent/);
    await expect(resultsTab).toHaveClass(/border-nexus-accent/);
  });

  test("E2E-4: timeframe button click changes active state", async ({ page }) => {
    await page.goto("/chart");

    // Default active timeframe is "1D"
    const dayButton = page.getByRole("button", { name: "1D", exact: true });
    await expect(dayButton).toHaveClass(/bg-nexus-accent/);

    // Click "1W" and verify it gets active styling
    const weekButton = page.getByRole("button", { name: "1W", exact: true });
    await weekButton.click();
    await expect(weekButton).toHaveClass(/bg-nexus-accent/);
    await expect(dayButton).not.toHaveClass(/bg-nexus-accent/);

    // Click "1m" and verify it gets active styling
    const minButton = page.getByRole("button", { name: "1m", exact: true });
    await minButton.click();
    await expect(minButton).toHaveClass(/bg-nexus-accent/);
    await expect(weekButton).not.toHaveClass(/bg-nexus-accent/);
  });

  test("E2E-5: empty state messages", async ({ page }) => {
    await page.goto("/chart");

    // Verify bottom panel shows empty state when no stock selected
    await expect(page.getByText("Select a stock to view details")).toBeVisible();

    // Verify sidebar shows empty state text for search results
    await expect(page.getByText("검색 결과가 없습니다")).toBeVisible();
  });

  test("E2E-6: main chart renders canvas (not placeholder)", async ({ page }) => {
    await page.goto("/chart?symbol=005930");

    // placeholder 텍스트가 사라졌는지 확인
    await expect(page.getByText("Main Chart Area")).not.toBeVisible();

    // lightweight-charts는 canvas 요소를 생성
    const canvas = page.locator("canvas").first();
    await expect(canvas).toBeVisible({ timeout: 10000 });
  });

});
