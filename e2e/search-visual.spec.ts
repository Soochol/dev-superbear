import { test, expect } from "@playwright/test";

test.describe("Search Page Visual Regression", () => {
  test("NL tab initial state", async ({ page }) => {
    await page.goto("/search");
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("search-nl-initial.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    });
  });

  test("NL tab after preset chip click", async ({ page }) => {
    await page.goto("/search");
    await page.getByRole("button", { name: "2yr Max Volume" }).click();
    await expect(page).toHaveScreenshot("search-nl-preset-filled.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    });
  });

  test("DSL tab with editor", async ({ page }) => {
    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();
    await page.waitForLoadState("networkidle");
    await expect(page).toHaveScreenshot("search-dsl-tab.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    });
  });
});
