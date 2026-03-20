import type { CandleData } from "@/entities/candle";
import { apiGet } from "@/shared/api/client";
import { logger } from "@/shared/lib/logger";

export const chartApi = {
  async fetchCandles(symbol: string, timeframe: string): Promise<CandleData[]> {
    try {
      const json = await apiGet<{ data?: { candles?: CandleData[] } }>(
        `/api/v1/candles/${symbol}?period=${timeframe}`,
      );
      return json.data?.candles ?? [];
    } catch (err: unknown) {
      logger.error("Failed to fetch candles", {
        symbol,
        message: err instanceof Error ? err.message : String(err),
      });
      return [];
    }
  },

  async fetchCurrentPrice(symbol: string) {
    try {
      const json = await apiGet<{ data?: Record<string, unknown> }>(
        `/api/v1/candles/${symbol}/price`,
      );
      return json.data ?? null;
    } catch (err: unknown) {
      logger.error("Failed to fetch current price", {
        symbol,
        message: err instanceof Error ? err.message : String(err),
      });
      return null;
    }
  },
};
