import { test, expect } from "@playwright/test";

test.describe("Search Page", () => {
  test("navigates to search page and shows tab buttons", async ({ page }) => {
    await page.goto("/search");
    // Should render the NL and DSL tab buttons
    await expect(
      page.getByRole("button", { name: "Natural Language" })
    ).toBeVisible();
    await expect(page.getByRole("button", { name: "DSL" })).toBeVisible();
  });

  test("NL tab is active by default with textarea visible", async ({
    page,
  }) => {
    await page.goto("/search");
    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await expect(textarea).toBeVisible();
  });

  test("preset chips are visible and clickable", async ({ page }) => {
    await page.goto("/search");

    // All 6 preset chips should be visible
    await expect(
      page.getByRole("button", { name: "2yr Max Volume" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Golden Cross" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "RSI Oversold" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "High Trade Value" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "PER < 10" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "52w High" })
    ).toBeVisible();
  });

  test("clicking a preset chip fills the textarea", async ({ page }) => {
    await page.goto("/search");
    const chip = page.getByRole("button", { name: "2yr Max Volume" });
    await chip.click();

    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await expect(textarea).toHaveValue(
      "최근 5년 안에 2년 최대거래량이 발생한 종목"
    );
  });

  test("switches to DSL tab and shows editor", async ({ page }) => {
    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();

    // DSL editor container should appear
    await expect(page.getByTestId("dsl-editor-container")).toBeVisible();

    // Action buttons should be visible
    await expect(
      page.getByRole("button", { name: "Validate" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Explain in NL" })
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Run Search" })
    ).toBeVisible();
  });

  test("DSL tab buttons are disabled when editor is empty", async ({
    page,
  }) => {
    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();

    await expect(
      page.getByRole("button", { name: "Validate" })
    ).toBeDisabled();
    await expect(
      page.getByRole("button", { name: "Explain in NL" })
    ).toBeDisabled();
    await expect(
      page.getByRole("button", { name: "Run Search" })
    ).toBeDisabled();
  });

  test("LIVE DSL panel shows empty state", async ({ page }) => {
    await page.goto("/search");
    await expect(page.getByText("LIVE DSL")).toBeVisible();
    await expect(page.getByText(/DSL이 없습니다/)).toBeVisible();
  });

  test("search results show empty state initially", async ({ page }) => {
    await page.goto("/search");
    await expect(page.getByText("검색 결과가 없습니다")).toBeVisible();
  });

  test("Search button is disabled when textarea is empty", async ({
    page,
  }) => {
    await page.goto("/search");
    const searchBtn = page.getByRole("button", { name: "Search" });
    await expect(searchBtn).toBeDisabled();
  });

  test("Search button enables after typing a query", async ({ page }) => {
    await page.goto("/search");
    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await textarea.fill("2년 최대거래량 종목");

    const searchBtn = page.getByRole("button", { name: "Search" });
    await expect(searchBtn).toBeEnabled();
  });

  test("switching back from DSL to NL tab restores NL view", async ({
    page,
  }) => {
    await page.goto("/search");

    // Switch to DSL
    await page.getByRole("button", { name: "DSL" }).click();
    await expect(page.getByTestId("dsl-editor-container")).toBeVisible();

    // Switch back to NL
    await page.getByRole("button", { name: "Natural Language" }).click();
    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await expect(textarea).toBeVisible();
  });

  test("preset chip selection enables Search button", async ({ page }) => {
    await page.goto("/search");

    // Search button should be disabled initially
    const searchBtn = page.getByRole("button", { name: "Search" });
    await expect(searchBtn).toBeDisabled();

    // Click a preset chip
    await page.getByRole("button", { name: "Golden Cross" }).click();

    // Search button should now be enabled
    await expect(searchBtn).toBeEnabled();

    // Textarea should have the preset query
    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await expect(textarea).toHaveValue(
      "20일 이평선이 60일 이평선을 상향 돌파한 종목"
    );
  });
});

test.describe("Search Flow", () => {
  test("NL search: type query → click Search → see results", async ({ page }) => {
    await page.route("**/api/v1/search/nl-to-dsl", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          dsl: "scan where volume > 1000000",
          explanation: "거래량 100만 이상",
          results: [
            { symbol: "005930", name: "삼성전자", matchedValue: 28400000, close: 71000, changePct: 1.5 },
            { symbol: "000660", name: "SK하이닉스", matchedValue: 15200000, close: 195000, changePct: -0.3 },
          ],
        }),
      });
    });

    await page.goto("/search");
    const textarea = page.getByPlaceholder("자연어로 검색 조건을 입력하세요...");
    await textarea.fill("거래량 많은 종목");
    await page.getByRole("button", { name: "Search" }).click();

    await expect(page.getByText("삼성전자")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("SK하이닉스")).toBeVisible();
    await expect(page.getByText("2개 종목")).toBeVisible();
    await expect(page.getByText(/scan/)).toBeVisible();
  });

  test("DSL search: enter DSL → Run Search → see results", async ({ page }) => {
    await page.route("**/api/v1/search/execute", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          results: [
            { symbol: "035420", name: "NAVER", matchedValue: 5000000, close: 220000, changePct: 2.1 },
          ],
        }),
      });
    });

    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();
    const editor = page.getByTestId("dsl-editor-container");
    await editor.click();
    await page.keyboard.type("scan where volume > 5000000");
    await page.getByRole("button", { name: "Run Search" }).click();

    await expect(page.getByText("NAVER")).toBeVisible({ timeout: 5000 });
    await expect(page.getByText("1개 종목")).toBeVisible();
  });

  test("DSL validate: enter DSL → Validate → see validation badge", async ({ page }) => {
    await page.route("**/api/v1/search/validate", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ valid: true, error: null }),
      });
    });

    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();
    const editor = page.getByTestId("dsl-editor-container");
    await editor.click();
    await page.keyboard.type("scan where volume > 1000000");
    await page.getByRole("button", { name: "Validate" }).click();

    await expect(page.getByText("Validated")).toBeVisible({ timeout: 5000 });
  });

  test("NL search via preset chip: click chip → click Search → see results", async ({ page }) => {
    await page.route("**/api/v1/search/nl-to-dsl", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          dsl: "scan where rsi(14) < 30",
          explanation: "RSI 과매도",
          results: [{ symbol: "003550", name: "LG", matchedValue: 28.5, close: 95000, changePct: -1.2 }],
        }),
      });
    });

    await page.goto("/search");
    await page.getByRole("button", { name: "RSI Oversold" }).click();
    await page.getByRole("button", { name: "Search" }).click();

    await expect(page.getByText("LG")).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Preset Manager", () => {
  test("save and load a preset", async ({ page }) => {
    await page.route("**/api/v1/search/presets", async (route) => {
      if (route.request().method() === "POST") {
        const body = route.request().postDataJSON();
        await route.fulfill({
          status: 201,
          contentType: "application/json",
          body: JSON.stringify({
            data: {
              id: "new-preset-1",
              userId: "u1",
              name: body.name,
              dsl: body.dsl,
              nlQuery: body.nlQuery ?? null,
              isPublic: false,
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            },
          }),
        });
      } else {
        await route.continue();
      }
    });

    await page.route("**/api/v1/search/execute", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ results: [] }),
      });
    });

    await page.goto("/search");
    await page.getByRole("button", { name: "DSL" }).click();
    const editor = page.getByTestId("dsl-editor-container");
    await editor.click();
    await page.keyboard.type("scan where volume > 1000000");

    await page.getByRole("button", { name: /save/i }).click();
    await expect(page.getByText(/Preset/)).toBeVisible({ timeout: 5000 });
  });
});
