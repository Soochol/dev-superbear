import { test, expect } from "./fixtures/search.fixture";

test.describe("NL Search SSE Streaming", { tag: "@critical" }, () => {
  test.beforeEach(async ({ searchPage }) => {
    await searchPage.goto();
  });

  test("NL search shows agent status indicator during SSE stream", async ({
    searchPage,
  }) => {
    await searchPage.fillNlQuery("거래량 100만 이상 종목");
    await searchPage.clickSearch();

    // SSE 스트리밍 시작 시 에이전트 상태가 나타나야 함
    // agentPulse (animate-pulse) 또는 완료 메시지 ("종목 발견" 또는 에러)
    await expect(
      searchPage.agentPulse.or(
        searchPage.page.getByText(/종목 발견|Error|error|unavailable/)
      )
    ).toBeVisible({ timeout: 120_000 });
  });

  test("NL search shows streaming status after clicking Search", async ({
    searchPage,
  }) => {
    await searchPage.fillNlQuery("거래량 많은 종목 찾아줘");
    await searchPage.clickSearch();

    // SSE 시작 후 에이전트 상태 메시지가 표시되어야 함
    await expect(
      searchPage.page.getByText("쿼리 분석 중")
    ).toBeVisible({ timeout: 10_000 });
  });
});

test.describe("DSL Editor Linter", { tag: "@smoke" }, () => {
  test.beforeEach(async ({ searchPage }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();
  });

  test("shows lint error for unknown field name", async ({ searchPage }) => {
    await searchPage.typeDslCode("scan where unknown_field > 100");

    // lint debounce (300ms) + 렌더링 대기
    await expect(searchPage.lintError).toBeVisible({ timeout: 3000 });
  });

  test("no lint error for valid DSL", async ({ searchPage }) => {
    await searchPage.typeDslCode("scan where volume > 1000000");

    // 잠시 대기 후 에러가 없어야 함
    await searchPage.page.waitForTimeout(500);
    await expect(searchPage.lintError).not.toBeVisible();
  });

  test("shows lint error for OR usage", async ({ searchPage }) => {
    await searchPage.typeDslCode(
      "scan where volume > 100 or close > 50000"
    );

    await expect(searchPage.lintError).toBeVisible({ timeout: 3000 });
  });
});

test.describe("DSL Editor Autocompletion", { tag: "@smoke" }, () => {
  test("shows completions when explicitly triggered with Ctrl+Space", async ({
    searchPage,
  }) => {
    await searchPage.goto();
    await searchPage.switchToDsl();

    const cmContent = searchPage.dslEditorContainer.locator(".cm-content");
    await cmContent.click();
    await searchPage.page.keyboard.insertText("scan ");

    // CodeMirror explicit autocomplete trigger
    await searchPage.page.keyboard.press("Control+Space");

    const autocomplete = searchPage.page.locator(".cm-tooltip-autocomplete");
    await expect(autocomplete).toBeVisible({ timeout: 3000 });
    await expect(autocomplete.getByText("where")).toBeVisible();
  });
});
