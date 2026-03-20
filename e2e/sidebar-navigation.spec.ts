// e2e/sidebar-navigation.spec.ts
import { test, expect } from "@playwright/test";

test.describe("Sidebar Navigation", () => {
  test("sidebar is visible on all pages", async ({ page }) => {
    await page.goto("/dashboard");
    const sidebar = page.getByTestId("sidebar-nav");
    await expect(sidebar).toBeVisible();
  });

  test("sidebar shows logo", async ({ page }) => {
    await page.goto("/dashboard");
    const logo = page.getByTestId("sidebar-logo");
    await expect(logo).toBeVisible();
  });

  test("clicking nav item navigates to page", async ({ page }) => {
    await page.goto("/dashboard");
    const searchLink = page.getByRole("link", { name: /search/i });
    await searchLink.click();
    await expect(page).toHaveURL(/\/search/);
  });

  test("active nav item is highlighted", async ({ page }) => {
    await page.goto("/dashboard");
    const dashboardLink = page.getByRole("link", { name: /dashboard/i });
    await expect(dashboardLink).toHaveClass(/text-nexus-accent/);
  });

  test("sidebar expands on hover", async ({ page }) => {
    await page.goto("/dashboard");
    const sidebar = page.getByTestId("sidebar-nav");

    const initialWidth = await sidebar.evaluate((el) => el.offsetWidth);
    expect(initialWidth).toBe(64);

    await sidebar.hover();
    await page.waitForTimeout(300);

    const expandedWidth = await sidebar.evaluate((el) => el.offsetWidth);
    expect(expandedWidth).toBe(200);
  });

  test("pin toggle keeps sidebar expanded", async ({ page }) => {
    await page.goto("/dashboard");
    const sidebar = page.getByTestId("sidebar-nav");

    await sidebar.hover();
    await page.waitForTimeout(300);

    const pinBtn = page.getByTestId("pin-toggle");
    await pinBtn.click();

    await page.mouse.move(500, 500);
    await page.waitForTimeout(300);

    const width = await sidebar.evaluate((el) => el.offsetWidth);
    expect(width).toBe(200);
  });

  test("placeholder pages show coming soon", async ({ page }) => {
    await page.goto("/backtest");
    await expect(page.locator("text=Coming soon")).toBeVisible();
  });
});
