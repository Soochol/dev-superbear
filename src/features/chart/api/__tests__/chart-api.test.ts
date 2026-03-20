import { chartApi } from "../chart-api";
import { apiGet } from "@/shared/api/client";
import { logger } from "@/shared/lib/logger";

jest.mock("@/shared/api/client", () => ({
  apiGet: jest.fn(),
}));

jest.mock("@/shared/lib/logger", () => ({
  logger: {
    error: jest.fn(),
  },
}));

const mockApiGet = apiGet as jest.MockedFunction<typeof apiGet>;
const mockLoggerError = logger.error as jest.MockedFunction<typeof logger.error>;

beforeEach(() => {
  mockApiGet.mockReset();
  mockLoggerError.mockReset();
});

describe("chartApi", () => {
  describe("fetchCandles", () => {
    it("returns candles array on success", async () => {
      const candles = [
        { time: 1, open: 100, high: 110, low: 90, close: 105, volume: 1000 },
        { time: 2, open: 105, high: 115, low: 95, close: 110, volume: 1200 },
      ];
      mockApiGet.mockResolvedValue({ data: { candles } });

      const result = await chartApi.fetchCandles("005930", "1D");
      expect(result).toEqual(candles);
    });

    it("calls apiGet with correct URL", async () => {
      mockApiGet.mockResolvedValue({ data: { candles: [] } });

      await chartApi.fetchCandles("005930", "1W");
      expect(mockApiGet).toHaveBeenCalledWith("/api/v1/candles/005930?period=1W");
    });

    it("returns [] and logs error on failure", async () => {
      mockApiGet.mockRejectedValue(new Error("Network error"));

      const result = await chartApi.fetchCandles("005930", "1D");
      expect(result).toEqual([]);
      expect(mockLoggerError).toHaveBeenCalledWith(
        "Failed to fetch candles",
        expect.objectContaining({ symbol: "005930", message: "Network error" }),
      );
    });

    it("returns [] when data.candles is missing in response", async () => {
      mockApiGet.mockResolvedValue({ data: {} });

      const result = await chartApi.fetchCandles("005930", "1D");
      expect(result).toEqual([]);
    });
  });

  describe("fetchCurrentPrice", () => {
    it("returns price data on success", async () => {
      const priceData = { price: 72500, change: 1500, changePct: 2.11 };
      mockApiGet.mockResolvedValue({ data: priceData });

      const result = await chartApi.fetchCurrentPrice("005930");
      expect(result).toEqual(priceData);
    });

    it("returns null and logs error on failure", async () => {
      mockApiGet.mockRejectedValue(new Error("Server down"));

      const result = await chartApi.fetchCurrentPrice("005930");
      expect(result).toBeNull();
      expect(mockLoggerError).toHaveBeenCalledWith(
        "Failed to fetch current price",
        expect.objectContaining({ symbol: "005930", message: "Server down" }),
      );
    });

    it("returns null when data is missing in response", async () => {
      mockApiGet.mockResolvedValue({});

      const result = await chartApi.fetchCurrentPrice("005930");
      expect(result).toBeNull();
    });
  });
});
