import { test, expect } from "./fixtures/search.fixture";

test.describe("Search Page Visual Regression", { tag: "@visual" }, () => {
  test("NL tab initial state", async ({ searchPage }) => {
    await searchPage.goto();
    await expect(searchPage.nlTextarea).toBeVisible();
    await expect(searchPage.page).toHaveScreenshot("search-nl-initial.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    });
  });

  test("NL tab after preset chip click", async ({ searchPage }) => {
    await searchPage.goto();
    await searchPage.presetChip2yrMaxVolume.click();
    await expect(searchPage.page).toHaveScreenshot(
      "search-nl-preset-filled.png",
      {
        fullPage: true,
        maxDiffPixelRatio: 0.01,
      }
    );
  });

  test("DSL tab with editor", async ({ searchPage }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();
    await expect(searchPage.dslEditorContainer).toBeVisible();
    await expect(searchPage.page).toHaveScreenshot("search-dsl-tab.png", {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    });
  });
});
