import { type Locator, type Page } from "@playwright/test";

export class ChartPage {
  readonly canvas: Locator;
  readonly loadingIndicator: Locator;
  readonly selectStockMessage: Locator;

  constructor(readonly page: Page) {
    this.canvas = page.locator("canvas").first();
    this.loadingIndicator = page.getByText("Loading chart data...");
    this.selectStockMessage = page.getByText("Select a stock", { exact: true });
  }

  async goto(symbol?: string) {
    const url = symbol ? `/chart?symbol=${symbol}` : "/chart";
    await this.page.goto(url);
  }

  /** Navigate to chart with symbol and wait for the candle API response. */
  async gotoAndWaitForCandles(symbol: string) {
    const responsePromise = this.page.waitForResponse(
      (res) => res.url().includes(`/api/v1/candles/${symbol}`),
    );
    await this.page.goto(`/chart?symbol=${symbol}`);
    return responsePromise;
  }

  getTimeframeButton(tf: string): Locator {
    return this.page.getByRole("button", { name: tf, exact: true });
  }

  async clickTimeframe(tf: string) {
    await this.getTimeframeButton(tf).click();
  }

  async waitForCandleResponse(symbol: string) {
    return this.page.waitForResponse((res) =>
      res.url().includes(`/api/v1/candles/${symbol}`),
    );
  }
}
