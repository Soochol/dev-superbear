import { calculateMA } from "../lib/ma";
import { calculateBollingerBands } from "../lib/bollinger";

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
});
