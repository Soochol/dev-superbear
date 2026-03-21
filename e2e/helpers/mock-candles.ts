import type { Page, Route } from "@playwright/test";

/** OHLCV candle matching the backend response shape. */
export interface MockCandle {
  time: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

/**
 * Generate deterministic mock candle data.
 * Prices oscillate so RSI and MACD produce non-trivial values.
 */
export function generateMockCandles(count: number = 60): MockCandle[] {
  const candles: MockCandle[] = [];
  const baseDate = new Date("2026-01-02");
  let price = 60000;

  for (let i = 0; i < count; i++) {
    const date = new Date(baseDate);
    date.setDate(baseDate.getDate() + i);

    // Oscillating pattern for non-trivial indicator values
    const direction = Math.sin(i * 0.3) * 1000 + (i % 7 === 0 ? -500 : 300);
    const open = price;
    const close = Math.round(price + direction);
    const high = Math.max(open, close) + Math.round(Math.abs(direction) * 0.3);
    const low = Math.min(open, close) - Math.round(Math.abs(direction) * 0.2);
    const volume = 1_000_000 + Math.round(Math.abs(direction) * 500);

    candles.push({
      time: date.toISOString().split("T")[0],
      open,
      high,
      low,
      close: Math.max(close, 100),
      volume,
    });

    price = Math.max(close, 100);
  }

  return candles;
}

/** Intercept candle API and respond with mock data for a given symbol. */
export async function interceptCandlesWithMockData(
  page: Page,
  symbol: string,
  candles?: MockCandle[],
) {
  const mockCandles = candles ?? generateMockCandles();

  await page.route(`**/api/v1/candles/${symbol}*`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: {
          symbol,
          candles: mockCandles,
        },
      }),
    });
  });

  return mockCandles;
}

/**
 * Intercept all chart-related APIs for a symbol (candles, financials, sector, price).
 * This prevents Next.js dev error overlays from 404/502 responses.
 */
export async function interceptAllChartAPIs(
  page: Page,
  symbol: string,
  candles?: MockCandle[],
) {
  const mockCandles = candles ?? generateMockCandles();

  // Candle data (trailing * to match ?period=xxx query params)
  await page.route(`**/api/v1/candles/${symbol}?*`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: { symbol, candles: mockCandles },
      }),
    });
  });

  // Current price
  await page.route(`**/api/v1/candles/${symbol}/price`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        data: { symbol, price: 72000, change: 1000, changePct: 1.41 },
      }),
    });
  });

  // Financials
  await page.route(`**/api/v1/financials/${symbol}`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: {} }),
    });
  });

  // Sector data (expects array)
  await page.route(`**/api/v1/financials/${symbol}/sector`, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: [] }),
    });
  });

  return mockCandles;
}
