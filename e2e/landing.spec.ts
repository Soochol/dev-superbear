import { test, expect } from "@playwright/test";

test.describe("Landing Page", () => {
  test("renders NEXUS title", async ({ page }) => {
    await page.goto("/");
    const heading = page.locator("h1");
    await expect(heading).toHaveText("NEXUS");
  });

  test("shows subtitle", async ({ page }) => {
    await page.goto("/");
    const subtitle = page.locator("p");
    await expect(subtitle).toContainText("AI-Native Investment Intelligence");
  });

  test("shows System Online indicator", async ({ page }) => {
    await page.goto("/");
    const status = page.locator("text=System Online");
    await expect(status).toBeVisible();
  });

  test("has Korean language attribute", async ({ page }) => {
    await page.goto("/");
    const html = page.locator("html");
    await expect(html).toHaveAttribute("lang", "ko");
  });

  test("has dark class on html element", async ({ page }) => {
    await page.goto("/");
    const html = page.locator("html");
    await expect(html).toHaveClass(/dark/);
  });

  test("has correct page title", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/NEXUS/);
  });

  test("has dark background color", async ({ page }) => {
    await page.goto("/");
    const body = page.locator("body");
    const bgColor = await body.evaluate((el) => {
      return window.getComputedStyle(el).backgroundColor;
    });
    // #0a0a0f = rgb(10, 10, 15)
    expect(bgColor).toBe("rgb(10, 10, 15)");
  });

  test("pulse animation is present on status indicator", async ({ page }) => {
    await page.goto("/");
    const dot = page.locator("span.animate-pulse");
    await expect(dot).toBeVisible();
  });
});
