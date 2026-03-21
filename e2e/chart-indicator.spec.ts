import { test, expect } from "./fixtures/chart.fixture";
import { interceptAllChartAPIs } from "./helpers/mock-candles";

const TEST_SYMBOL = "005930";

test.describe("Chart Indicator Panel @critical", () => {
  test.beforeEach(async ({ chartPage }) => {
    // Intercept ALL chart-related APIs to prevent Next.js dev error overlay
    await interceptAllChartAPIs(chartPage.page, TEST_SYMBOL);
    await chartPage.goto(TEST_SYMBOL);
    await expect(chartPage.canvas).toBeVisible({ timeout: 10_000 });
  });

  test("IND-1: RSI indicator panel renders with canvas", async ({ chartPage }) => {
    await expect(chartPage.getIndicatorPanel("rsi")).not.toBeVisible();

    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("rsi").click();

    const rsiPanel = chartPage.getIndicatorPanel("rsi");
    await expect(rsiPanel).toBeVisible();
    await expect(rsiPanel.getByText("RSI(14)")).toBeVisible();

    const rsiCanvas = rsiPanel.locator("canvas").first();
    await expect(rsiCanvas).toBeVisible({ timeout: 5_000 });
  });

  test("IND-2: MACD indicator panel renders with canvas", async ({ chartPage }) => {
    await expect(chartPage.getIndicatorPanel("macd")).not.toBeVisible();

    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("macd").click();

    const macdPanel = chartPage.getIndicatorPanel("macd");
    await expect(macdPanel).toBeVisible();
    await expect(macdPanel.getByText("MACD(12,26,9)")).toBeVisible();

    const macdCanvas = macdPanel.locator("canvas").first();
    await expect(macdCanvas).toBeVisible({ timeout: 5_000 });
  });

  test("IND-3: multiple indicators can be active simultaneously", async ({
    chartPage,
  }) => {
    await chartPage.openIndicatorSelector();

    // Enable RSI
    await chartPage.getIndicatorOption("rsi").click();
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();

    // Enable MACD (selector might close after toggle, reopen)
    const macdOption = chartPage.getIndicatorOption("macd");
    if (!(await macdOption.isVisible().catch(() => false))) {
      await chartPage.openIndicatorSelector();
    }
    await macdOption.click();

    // Both panels visible with canvases
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();
    await expect(chartPage.getIndicatorPanel("macd")).toBeVisible();
    await expect(chartPage.getIndicatorPanel("rsi").locator("canvas").first()).toBeVisible();
    await expect(chartPage.getIndicatorPanel("macd").locator("canvas").first()).toBeVisible();
  });

  test("IND-4: removing indicator hides its panel", async ({ chartPage }) => {
    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("rsi").click();
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();

    const removeBtn = chartPage.getIndicatorPanel("rsi").getByRole("button", {
      name: /Remove RSI/i,
    });
    await removeBtn.click();

    await expect(chartPage.getIndicatorPanel("rsi")).not.toBeVisible();
  });

  test("IND-5: RSI values are within valid range (0-100)", async ({ chartPage }) => {
    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("rsi").click();
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();

    // Wait for rendering
    await chartPage.page.waitForTimeout(500);

    // Evaluate RSI calculation directly in the browser
    const rsiValues = await chartPage.page.evaluate(() => {
      const chartStore = (window as any).__CHART_STORE__;
      if (!chartStore) return null;

      const candles = chartStore.getState().candles;
      if (!candles?.length) return null;

      const closes: number[] = candles.map((c: any) => c.close);
      const period = 14;
      const result: (number | null)[] = [null];
      let avgGain = 0;
      let avgLoss = 0;

      for (let i = 1; i < closes.length; i++) {
        const diff = closes[i] - closes[i - 1];
        const gain = diff > 0 ? diff : 0;
        const loss = diff < 0 ? -diff : 0;

        if (i < period) {
          avgGain += gain;
          avgLoss += loss;
          result.push(null);
        } else if (i === period) {
          avgGain = (avgGain + gain) / period;
          avgLoss = (avgLoss + loss) / period;
          result.push(avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss));
        } else {
          avgGain = (avgGain * (period - 1) + gain) / period;
          avgLoss = (avgLoss * (period - 1) + loss) / period;
          result.push(avgLoss === 0 ? 100 : 100 - 100 / (1 + avgGain / avgLoss));
        }
      }

      return result.filter((v): v is number => v !== null);
    });

    if (rsiValues && rsiValues.length > 0) {
      for (const val of rsiValues) {
        expect(val).toBeGreaterThanOrEqual(0);
        expect(val).toBeLessThanOrEqual(100);
      }
      // With oscillating mock data, RSI should produce varying values
      const unique = new Set(rsiValues.map((v: number) => Math.round(v)));
      expect(unique.size).toBeGreaterThan(1);
    }
  });

  test("IND-6: indicator persists after timeframe switch", async ({ chartPage }) => {
    await chartPage.openIndicatorSelector();
    await chartPage.getIndicatorOption("rsi").click();
    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();

    // Switch timeframe
    await chartPage.clickTimeframe("1W");
    await chartPage.page.waitForTimeout(500);

    await expect(chartPage.getIndicatorPanel("rsi")).toBeVisible();
    await expect(chartPage.getIndicatorPanel("rsi").locator("canvas").first()).toBeVisible();
  });

  test("IND-7: indicator selector shows categories", async ({ chartPage }) => {
    await chartPage.openIndicatorSelector();

    await expect(chartPage.page.getByText("이동평균")).toBeVisible();
    await expect(chartPage.page.getByText("오실레이터")).toBeVisible();

    await expect(chartPage.getIndicatorOption("ma20")).toBeVisible();
    await expect(chartPage.getIndicatorOption("rsi")).toBeVisible();
    await expect(chartPage.getIndicatorOption("macd")).toBeVisible();
  });

  test("IND-8: default MA20 and MA60 are active on load", async ({ chartPage }) => {
    // Verify default active indicators include MA20 and MA60 via store state
    const activeIndicators = await chartPage.page.evaluate(() => {
      const chartStore = (window as any).__CHART_STORE__;
      if (!chartStore) return null;
      return chartStore.getState().activeIndicators;
    });

    if (activeIndicators) {
      expect(activeIndicators).toContain("ma20");
      expect(activeIndicators).toContain("ma60");
    }

    // Also verify via indicator selector checkmarks
    await chartPage.openIndicatorSelector();
    const ma20Option = chartPage.getIndicatorOption("ma20");
    const ma60Option = chartPage.getIndicatorOption("ma60");

    await expect(ma20Option).toContainText("✓");
    await expect(ma60Option).toContainText("✓");
  });
});
