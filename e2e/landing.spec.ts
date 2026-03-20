// e2e/landing.spec.ts
import { test, expect } from "@playwright/test";

test.describe("Landing Page Redirect", () => {
  test("redirects / to /dashboard", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveURL(/\/dashboard/);
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
});
