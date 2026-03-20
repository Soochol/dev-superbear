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
  test("NL search: type query → click Search → LIVE DSL updates", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.fillNlQuery("거래량 많은 종목");
    await searchPage.clickSearch();

    // Backend converts NL to DSL — LIVE DSL panel should show the generated DSL
    await expect(searchPage.emptyDslMessage).not.toBeVisible();
    await expect(searchPage.page.getByText(/scan/)).toBeVisible();
  });

  test("DSL tab: typing code enables action buttons", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 5000000");

    await expect(searchPage.validateButton).toBeEnabled();
    await expect(searchPage.explainButton).toBeEnabled();
    await expect(searchPage.runSearchButton).toBeEnabled();
  });

  test("DSL validate: enter DSL → Validate → see validation badge", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 1000000");

    await expect(searchPage.validateButton).toBeEnabled();
    await searchPage.clickValidate();

    // Backend validates DSL — LIVE DSL panel shows "Validated" badge
    await expect(searchPage.page.getByText("Validated")).toBeVisible();
  });

  test("NL search via preset chip: click chip → click Search → LIVE DSL updates", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.clickPresetChip("RSI Oversold");
    await searchPage.clickSearch();

    // Backend converts NL to DSL
    await expect(searchPage.emptyDslMessage).not.toBeVisible();
  });
});
