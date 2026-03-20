import type { CandleData } from "@/entities/candle";
import { logger } from "@/shared/lib/logger";

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const chartApi = {
  async fetchCandles(symbol: string, timeframe: string): Promise<CandleData[]> {
    try {
      const res = await fetch(`${API_BASE}/api/v1/candles/${symbol}?period=${timeframe}`);
      const json = await res.json();
      return (json as { data?: { candles?: CandleData[] } }).data?.candles ?? [];
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
      const res = await fetch(`${API_BASE}/api/v1/candles/${symbol}/price`);
      const json = await res.json();
      return (json as { data?: Record<string, unknown> }).data ?? null;
    } catch (err: unknown) {
      logger.error("Failed to fetch current price", {
        symbol,
        message: err instanceof Error ? err.message : String(err),
      });
      return null;
    }
  },
};
