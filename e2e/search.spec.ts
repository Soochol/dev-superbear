import { test, expect } from "./fixtures/search.fixture";

test.describe("Search Page", { tag: "@smoke" }, () => {
  test.beforeEach(async ({ searchPage }) => {
    await searchPage.goto();
  });

  test("navigates to search page and shows tab buttons", async ({
    searchPage,
  }) => {
    await expect(searchPage.nlTabButton).toBeVisible();
    await expect(searchPage.dslTabButton).toBeVisible();
  });

  test("NL tab is active by default with textarea visible", async ({
    searchPage,
  }) => {
    await expect(searchPage.nlTextarea).toBeVisible();
  });

  test("preset chips are visible and clickable", async ({ searchPage }) => {
    for (const chip of searchPage.getAllPresetChips()) {
      await expect(chip).toBeVisible();
    }
  });

  test("clicking a preset chip fills the textarea", async ({ searchPage }) => {
    await searchPage.presetChip2yrMaxVolume.click();
    await expect(searchPage.nlTextarea).toHaveValue(
      "최근 5년 안에 2년 최대거래량이 발생한 종목"
    );
  });

  test("switches to DSL tab and shows editor", async ({ searchPage }) => {
    await searchPage.switchToDsl();
    await expect(searchPage.dslEditorContainer).toBeVisible();
    await expect(searchPage.validateButton).toBeVisible();
    await expect(searchPage.explainButton).toBeVisible();
    await expect(searchPage.runSearchButton).toBeVisible();
  });

  test("DSL tab buttons are disabled when editor is empty", async ({
    searchPage,
  }) => {
    await searchPage.switchToDsl();
    await expect(searchPage.validateButton).toBeDisabled();
    await expect(searchPage.explainButton).toBeDisabled();
    await expect(searchPage.runSearchButton).toBeDisabled();
  });

  test("LIVE DSL panel shows empty state", async ({ searchPage }) => {
    await expect(searchPage.liveDslPanel).toBeVisible();
    await expect(searchPage.emptyDslMessage).toBeVisible();
  });

  test("search results show empty state initially", async ({ searchPage }) => {
    await expect(searchPage.emptyResultsMessage).toBeVisible();
  });

  test("Search button is disabled when textarea is empty", async ({
    searchPage,
  }) => {
    await expect(searchPage.searchButton).toBeDisabled();
  });

  test("Search button enables after typing a query", async ({
    searchPage,
  }) => {
    await searchPage.fillNlQuery("2년 최대거래량 종목");
    await expect(searchPage.searchButton).toBeEnabled();
  });

  test("switching back from DSL to NL tab restores NL view", async ({
    searchPage,
  }) => {
    await searchPage.switchToDsl();
    await expect(searchPage.dslEditorContainer).toBeVisible();
    await searchPage.switchToNl();
    await expect(searchPage.nlTextarea).toBeVisible();
  });

  test("preset chip selection enables Search button", async ({
    searchPage,
  }) => {
    await expect(searchPage.searchButton).toBeDisabled();
    await searchPage.presetChipGoldenCross.click();
    await expect(searchPage.searchButton).toBeEnabled();
    await expect(searchPage.nlTextarea).toHaveValue(
      "20일 이평선이 60일 이평선을 상향 돌파한 종목"
    );
  });
});

test.describe("Search Flow", { tag: "@critical" }, () => {
  test("NL search: type query → click Search → see results", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.fillNlQuery("거래량 많은 종목");
    await searchPage.clickSearch();

    // Real backend responds — verify results area updates (no longer empty)
    await expect(searchPage.emptyResultsMessage).not.toBeVisible();
  });

  test("DSL search: enter DSL → Run Search → see results", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 5000000");

    // Buttons should enable after typing DSL code
    await expect(searchPage.runSearchButton).toBeEnabled();
    await searchPage.clickRunSearch();

    // Real backend responds — verify results area updates
    await expect(searchPage.emptyResultsMessage).not.toBeVisible();
  });

  test("DSL validate: enter DSL → Validate → see validation result", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 1000000");

    await expect(searchPage.validateButton).toBeEnabled();
    await searchPage.clickValidate();

    // Backend responds with validation result
    await expect(
      searchPage.page.getByText(/Validated|Invalid/)
    ).toBeVisible();
  });

  test("NL search via preset chip: click chip → click Search → see results", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.clickPresetChip("RSI Oversold");
    await searchPage.clickSearch();

    // Real backend responds
    await expect(searchPage.emptyResultsMessage).not.toBeVisible();
  });
});

test.describe("Preset Manager", { tag: "@critical" }, () => {
  test.fixme(
    "save and load a preset",
    // Preset save API requires authentication — needs auth setup
    async ({ searchPage }) => {
      await searchPage.goto();
      await searchPage.switchToDsl();
      await searchPage.typeDslCode("scan where volume > 1000000");
      await searchPage.clickSave();

      await expect(searchPage.page.getByText(/Preset/)).toBeVisible();
    }
  );
});
