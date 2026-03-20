import { calculateMACD } from "../lib/macd";

describe("calculateMACD", () => {
  const closes = Array.from({ length: 40 }, (_, i) => 100 + Math.sin(i * 0.5) * 10);

  it("returns macd, signal, and histogram arrays", () => {
    const result = calculateMACD(closes, 12, 26, 9);
    expect(result.macd).toHaveLength(40);
    expect(result.signal).toHaveLength(40);
    expect(result.histogram).toHaveLength(40);
  });

  it("MACD line is null for first 25 values (26-period EMA not ready)", () => {
    const result = calculateMACD(closes, 12, 26, 9);
    for (let i = 0; i < 25; i++) {
      expect(result.macd[i]).toBeNull();
    }
    expect(result.macd[25]).not.toBeNull();
  });

  it("histogram = macd - signal", () => {
    const result = calculateMACD(closes, 12, 26, 9);
    for (let i = 33; i < 40; i++) {
      if (result.macd[i] !== null && result.signal[i] !== null) {
        expect(result.histogram[i]).toBeCloseTo(result.macd[i]! - result.signal[i]!, 5);
      }
    }
  });
});
