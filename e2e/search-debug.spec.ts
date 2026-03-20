import { test, expect } from "@playwright/test";

// Original inline test — verify this still works
test("ORIGINAL NL search inline", async ({ page }) => {
  await page.route("**/api/v1/search/nl-to-dsl", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        dsl: "scan where volume > 1000000",
        explanation: "거래량 100만 이상",
        results: [
          { symbol: "005930", name: "삼성전자", matchedValue: 28400000, close: 71000, changePct: 1.5 },
        ],
      }),
    });
  });
  await page.goto("/search");
  const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
  await textarea.fill("거래량 많은 종목");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.getByText("삼성전자")).toBeVisible({ timeout: 5000 });
});

// POM + mock class test
import { SearchPage } from "./pages/search.page";
import { SearchApiMock } from "./mocks/search-api.mock";

test("POM NL search via class", async ({ page }) => {
  const searchApiMock = new SearchApiMock(page);
  const searchPage = new SearchPage(page);

  await searchApiMock.mockNlSearch({
    dsl: "scan where volume > 1000000",
    explanation: "거래량 100만 이상",
    results: [
      { symbol: "005930", name: "삼성전자", matchedValue: 28400000, close: 71000, changePct: 1.5 },
    ],
  });
  await searchPage.goto();
  await searchPage.fillNlQuery("거래량 많은 종목");
  await searchPage.clickSearch();
  await expect(page.getByText("삼성전자")).toBeVisible({ timeout: 5000 });
});

// POM + fixture test (manual fixture creation to isolate)
test("POM DSL typing test", async ({ page }) => {
  const searchPage = new SearchPage(page);
  await searchPage.goto();
  await searchPage.switchToDsl();
  await searchPage.typeDslCode("scan where volume > 1000000");
  // Check if the button is enabled after typing
  await expect(searchPage.validateButton).toBeEnabled({ timeout: 5000 });
});
