import { calculateMA } from "../lib/ma";
import { calculateBollingerBands } from "../lib/bollinger";
import { calculateRSI } from "../lib/rsi";

const sampleData = [
  { close: 100 }, { close: 102 }, { close: 98 }, { close: 104 }, { close: 106 },
  { close: 103 }, { close: 107 }, { close: 110 }, { close: 108 }, { close: 112 },
];

describe("Technical Indicators", () => {
  describe("calculateMA", () => {
    it("calculates 5-day MA correctly", () => {
      const ma = calculateMA(sampleData.map((d) => d.close), 5);
      expect(ma).toHaveLength(10);
      expect(ma[0]).toBeNull();
      expect(ma[3]).toBeNull();
      expect(ma[4]).toBeCloseTo((100 + 102 + 98 + 104 + 106) / 5);
    });

    it("returns all null for period > data length", () => {
      const ma = calculateMA([100, 200], 5);
      expect(ma.every((v) => v === null)).toBe(true);
    });
  });

  describe("calculateBollingerBands", () => {
    it("returns upper, middle, lower bands", () => {
      const bb = calculateBollingerBands(sampleData.map((d) => d.close), 5, 2);
      expect(bb.upper).toHaveLength(10);
      expect(bb.middle).toHaveLength(10);
      expect(bb.lower).toHaveLength(10);
    });

    it("middle band equals MA", () => {
      const closes = sampleData.map((d) => d.close);
      const ma = calculateMA(closes, 5);
      const bb = calculateBollingerBands(closes, 5, 2);
      for (let i = 0; i < 10; i++) {
        expect(bb.middle[i]).toEqual(ma[i]);
      }
    });

    it("upper > middle > lower when valid", () => {
      const bb = calculateBollingerBands(sampleData.map((d) => d.close), 5, 2);
      for (let i = 4; i < 10; i++) {
        expect(bb.upper[i]!).toBeGreaterThan(bb.middle[i]!);
        expect(bb.middle[i]!).toBeGreaterThan(bb.lower[i]!);
      }
    });
  });

  describe("calculateRSI", () => {
    const closes = [
      44, 44.34, 44.09, 43.61, 44.33, 44.83, 45.10, 45.42, 45.84,
      46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.00, 46.03, 46.41,
      46.22, 45.64,
    ];

    it("returns null for first `period` values", () => {
      const rsi = calculateRSI(closes, 14);
      expect(rsi[0]).toBeNull();
      expect(rsi[13]).toBeNull();
      expect(rsi[14]).not.toBeNull();
    });

    it("RSI is between 0 and 100", () => {
      const rsi = calculateRSI(closes, 14);
      for (const val of rsi) {
        if (val !== null) {
          expect(val).toBeGreaterThanOrEqual(0);
          expect(val).toBeLessThanOrEqual(100);
        }
      }
    });

    it("returns array of same length as input", () => {
      const rsi = calculateRSI(closes, 14);
      expect(rsi).toHaveLength(closes.length);
    });

    it("handles all-up prices (RSI near 100)", () => {
      const allUp = [100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114];
      const rsi = calculateRSI(allUp, 14);
      const lastRSI = rsi[rsi.length - 1];
      expect(lastRSI).toBe(100);
    });
  });
});
