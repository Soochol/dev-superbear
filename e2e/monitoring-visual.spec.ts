import { test, expect } from "@playwright/test";

const CASE_WITH_MONITORS = "00000000-0000-0000-0000-000000000100";

test.describe("Monitoring Panel Visual Regression", () => {
  test("모니터 패널 초기 상태", async ({ page }) => {
    await page.goto(`/cases/${CASE_WITH_MONITORS}`);
    await page.waitForLoadState("networkidle");

    const panel = page.getByTestId("monitor-panel");
    await expect(panel).toBeVisible();
    await expect(panel).toHaveScreenshot("monitor-panel-initial.png", {
      maxDiffPixelRatio: 0.01,
    });
  });

  test("모니터 블록이 없는 케이스", async ({ page }) => {
    await page.goto("/cases/00000000-0000-0000-0000-000000000999");
    await page.waitForLoadState("networkidle");

    await expect(page).toHaveScreenshot("monitor-panel-empty.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    });
  });

  test("모니터 블록 토글 OFF", async ({ page }) => {
    await page.goto(`/cases/${CASE_WITH_MONITORS}`);
    await page.waitForLoadState("networkidle");

    const panel = page.getByTestId("monitor-panel");
    await expect(panel).toBeVisible();

    const firstToggle = panel.locator("button").first();
    await firstToggle.click();

    await expect(panel).toHaveScreenshot("monitor-panel-toggled.png", {
      maxDiffPixelRatio: 0.01,
    });
  });
});
