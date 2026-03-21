import { type Locator, type Page, expect } from "@playwright/test";

export class ChartPage {
  // Chart
  readonly canvas: Locator;
  readonly loadingIndicator: Locator;

  // Topbar
  readonly searchTrigger: Locator;
  readonly indicatorSelectorBtn: Locator;

  // Search Modal
  readonly modalBackdrop: Locator;
  readonly searchInput: Locator;

  // Bottom Panel
  readonly bottomPanelEmpty: Locator;
  readonly bottomPanelGrid: Locator;

  constructor(readonly page: Page) {
    this.canvas = page.locator("canvas").first();
    this.loadingIndicator = page.getByText("Loading chart data...");
    this.searchTrigger = page.getByTestId("stock-search-trigger");
    this.indicatorSelectorBtn = page.getByTestId("indicator-selector-btn");
    this.modalBackdrop = page.getByTestId("search-modal-backdrop");
    this.searchInput = page.getByPlaceholder("종목명 또는 코드를 검색하세요...");
    this.bottomPanelEmpty = page.getByTestId("bottom-panel-empty");
    this.bottomPanelGrid = page.getByTestId("bottom-panel-grid");
  }

  async goto(symbol?: string) {
    const url = symbol ? `/chart?symbol=${symbol}` : "/chart";
    await this.page.goto(url);
  }

  async gotoAndWaitForCandles(symbol: string) {
    const responsePromise = this.page.waitForResponse(
      (res) => res.url().includes(`/api/v1/candles/${symbol}`),
    );
    await this.page.goto(`/chart?symbol=${symbol}`);
    return responsePromise;
  }

  // Modal
  async openSearchModal() {
    await this.searchTrigger.click();
    await expect(this.searchInput).toBeVisible();
  }

  async closeSearchModal() {
    // Click backdrop at top-left corner (outside the centered modal)
    await this.modalBackdrop.click({ position: { x: 10, y: 10 } });
    await expect(this.searchInput).not.toBeVisible();
  }

  getModalTab(name: string): Locator {
    return this.page.getByRole("button", { name });
  }

  getStockItem(symbol: string): Locator {
    return this.page.getByTestId(`search-stock-item-${symbol}`);
  }

  // Timeframe
  getTimeframeButton(tf: string): Locator {
    return this.page.getByTestId(`tf-${tf}`);
  }

  async clickTimeframe(tf: string) {
    await this.getTimeframeButton(tf).click();
  }

  async waitForCandleResponse(symbol: string) {
    return this.page.waitForResponse((res) =>
      res.url().includes(`/api/v1/candles/${symbol}`),
    );
  }

  // Indicators
  async openIndicatorSelector() {
    await this.indicatorSelectorBtn.click();
  }

  getIndicatorOption(id: string): Locator {
    return this.page.getByTestId(`indicator-${id}`);
  }

  getIndicatorPanel(id: string): Locator {
    return this.page.getByTestId(`indicator-panel-${id}`);
  }
}
