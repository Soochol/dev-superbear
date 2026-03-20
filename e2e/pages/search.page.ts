import { type Locator, type Page } from "@playwright/test";

export class SearchPage {
  // Tab buttons
  readonly nlTabButton: Locator;
  readonly dslTabButton: Locator;

  // NL tab
  readonly nlTextarea: Locator;
  readonly searchButton: Locator;

  // Preset chips
  readonly presetChip2yrMaxVolume: Locator;
  readonly presetChipGoldenCross: Locator;
  readonly presetChipRsiOversold: Locator;
  readonly presetChipHighTradeValue: Locator;
  readonly presetChipPerUnder10: Locator;
  readonly presetChip52wHigh: Locator;

  // DSL tab
  readonly dslEditorContainer: Locator;
  readonly validateButton: Locator;
  readonly explainButton: Locator;
  readonly runSearchButton: Locator;

  // Actions
  readonly saveButton: Locator;

  // Panels
  readonly liveDslPanel: Locator;
  readonly emptyDslMessage: Locator;
  readonly emptyResultsMessage: Locator;

  constructor(readonly page: Page) {
    this.nlTabButton = page.getByRole("button", { name: "Natural Language" });
    this.dslTabButton = page.getByRole("button", { name: "DSL" });

    this.nlTextarea = page.getByPlaceholder(
      "자연어로 검색 조건을 입력하세요..."
    );
    this.searchButton = page.getByRole("button", { name: "Search" });

    this.presetChip2yrMaxVolume = page.getByRole("button", {
      name: "2yr Max Volume",
    });
    this.presetChipGoldenCross = page.getByRole("button", {
      name: "Golden Cross",
    });
    this.presetChipRsiOversold = page.getByRole("button", {
      name: "RSI Oversold",
    });
    this.presetChipHighTradeValue = page.getByRole("button", {
      name: "High Trade Value",
    });
    this.presetChipPerUnder10 = page.getByRole("button", {
      name: "PER < 10",
    });
    this.presetChip52wHigh = page.getByRole("button", { name: "52w High" });

    this.dslEditorContainer = page.getByTestId("dsl-editor-container");
    this.validateButton = page.getByRole("button", { name: "Validate" });
    this.explainButton = page.getByRole("button", { name: "Explain in NL" });
    this.runSearchButton = page.getByRole("button", { name: "Run Search" });

    this.saveButton = page.getByRole("button", { name: /save/i });

    this.liveDslPanel = page.getByText("LIVE DSL");
    this.emptyDslMessage = page.getByText(/DSL이 없습니다/);
    this.emptyResultsMessage = page.getByText("검색 결과가 없습니다");
  }

  async goto() {
    await this.page.goto("/search");
  }

  async fillNlQuery(query: string) {
    await this.nlTextarea.fill(query);
  }

  async clickSearch() {
    await this.searchButton.click();
  }

  async clickPresetChip(name: string) {
    await this.page.getByRole("button", { name }).click();
  }

  async switchToDsl() {
    await this.dslTabButton.click();
  }

  async switchToNl() {
    await this.nlTabButton.click();
  }

  async typeDslCode(code: string) {
    // Focus the CM6 contenteditable element and use insertText
    // which fires an InputEvent that CodeMirror processes correctly.
    const cmContent = this.dslEditorContainer.locator(".cm-content");
    await cmContent.click();
    await this.page.keyboard.insertText(code);
  }

  async clickValidate() {
    await this.validateButton.click();
  }

  async clickExplainInNl() {
    await this.explainButton.click();
  }

  async clickRunSearch() {
    await this.runSearchButton.click();
  }

  async clickSave() {
    await this.saveButton.click();
  }

  getAllPresetChips(): Locator[] {
    return [
      this.presetChip2yrMaxVolume,
      this.presetChipGoldenCross,
      this.presetChipRsiOversold,
      this.presetChipHighTradeValue,
      this.presetChipPerUnder10,
      this.presetChip52wHigh,
    ];
  }
}
