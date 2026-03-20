import { calculateMA } from "./ma";

export function calculateBollingerBands(
  closes: number[],
  period: number,
  k: number,
): {
  upper: (number | null)[];
  middle: (number | null)[];
  lower: (number | null)[];
} {
  const middle = calculateMA(closes, period);
  const upper: (number | null)[] = [];
  const lower: (number | null)[] = [];

  for (let i = 0; i < closes.length; i++) {
    if (middle[i] === null) {
      upper.push(null);
      lower.push(null);
    } else {
      const slice = closes.slice(i - period + 1, i + 1);
      const mean = middle[i]!;
      const variance = slice.reduce((sum, val) => sum + (val - mean) ** 2, 0) / period;
      const stddev = Math.sqrt(variance);
      upper.push(mean + k * stddev);
      lower.push(mean - k * stddev);
    }
  }

  return { upper, middle, lower };
}
