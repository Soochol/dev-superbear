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
    searchApiMock,
  }) => {
    await searchApiMock.mockNlSearch({
      dsl: "scan where volume > 1000000",
      explanation: "거래량 100만 이상",
      results: [
        {
          symbol: "005930",
          name: "삼성전자",
          matchedValue: 28400000,
          close: 71000,
          changePct: 1.5,
        },
        {
          symbol: "000660",
          name: "SK하이닉스",
          matchedValue: 15200000,
          close: 195000,
          changePct: -0.3,
        },
      ],
    });

    await searchPage.goto();
    await searchPage.fillNlQuery("거래량 많은 종목");
    await searchPage.clickSearch();

    await expect(searchPage.page.getByText("삼성전자")).toBeVisible();
    await expect(searchPage.page.getByText("SK하이닉스")).toBeVisible();
    await expect(searchPage.page.getByText("2개 종목")).toBeVisible();
    await expect(searchPage.page.getByText(/scan/)).toBeVisible();
  });

  test("DSL search: enter DSL → Run Search → see results", async ({
    searchPage,
    searchApiMock,
  }) => {
    await searchApiMock.mockDslExecute({
      results: [
        {
          symbol: "035420",
          name: "NAVER",
          matchedValue: 5000000,
          close: 220000,
          changePct: 2.1,
        },
      ],
    });

    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 5000000");
    await searchPage.clickRunSearch();

    await expect(searchPage.page.getByText("NAVER")).toBeVisible();
    await expect(searchPage.page.getByText("1개 종목")).toBeVisible();
  });

  test("DSL validate: enter DSL → Validate → see validation badge", async ({
    searchPage,
    searchApiMock,
  }) => {
    await searchApiMock.mockValidate({ valid: true, error: null });

    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 1000000");
    await searchPage.clickValidate();

    await expect(searchPage.page.getByText("Validated")).toBeVisible();
  });

  test("NL search via preset chip: click chip → click Search → see results", async ({
    searchPage,
    searchApiMock,
  }) => {
    await searchApiMock.mockNlSearch({
      dsl: "scan where rsi(14) < 30",
      explanation: "RSI 과매도",
      results: [
        {
          symbol: "003550",
          name: "LG",
          matchedValue: 28.5,
          close: 95000,
          changePct: -1.2,
        },
      ],
    });

    await searchPage.goto();
    await searchPage.clickPresetChip("RSI Oversold");
    await searchPage.clickSearch();

    await expect(searchPage.page.getByText("LG")).toBeVisible();
  });
});

test.describe("Preset Manager", { tag: "@critical" }, () => {
  test("save and load a preset", async ({ searchPage, searchApiMock }) => {
    await searchApiMock.mockPresetSave();
    await searchApiMock.mockDslExecute({ results: [] });

    await searchPage.goto();
    await searchPage.switchToDsl();
    await searchPage.typeDslCode("scan where volume > 1000000");
    await searchPage.clickSave();

    await expect(searchPage.page.getByText(/Preset/)).toBeVisible();
  });
});
