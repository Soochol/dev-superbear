import { calculateRSI } from "../lib/rsi";

describe("calculateRSI edge cases", () => {
  it("flat prices (all same value) → RSI should be 50", () => {
    // 16 identical values with period=14: first value is null, next 13 are null, then RSI=50
    const flat = Array(16).fill(100);
    const rsi = calculateRSI(flat, 14);
    const computed = rsi.filter((v) => v !== null);
    expect(computed.length).toBeGreaterThan(0);
    for (const val of computed) {
      expect(val).toBe(50);
    }
  });

  it("all-down prices (monotonically decreasing) → RSI should be 0", () => {
    // 16 values each decreasing by 1
    const allDown = Array.from({ length: 16 }, (_, i) => 200 - i);
    const rsi = calculateRSI(allDown, 14);
    const computed = rsi.filter((v) => v !== null);
    expect(computed.length).toBeGreaterThan(0);
    for (const val of computed) {
      expect(val).toBe(0);
    }
  });

  it("empty input → returns empty array", () => {
    const rsi = calculateRSI([]);
    expect(rsi).toEqual([]);
  });

  it("single value → returns [null]", () => {
    const rsi = calculateRSI([100]);
    expect(rsi).toEqual([null]);
  });

  it("all-up prices (monotonically increasing) → RSI should be 100", () => {
    const allUp = Array.from({ length: 16 }, (_, i) => 100 + i);
    const rsi = calculateRSI(allUp, 14);
    const computed = rsi.filter((v) => v !== null);
    expect(computed.length).toBeGreaterThan(0);
    for (const val of computed) {
      expect(val).toBe(100);
    }
  });

  it("period larger than data length → all nulls", () => {
    const rsi = calculateRSI([100, 101, 102, 103, 104], 14);
    for (const val of rsi) {
      expect(val).toBeNull();
    }
  });
});
