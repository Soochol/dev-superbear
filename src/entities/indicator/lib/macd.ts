function ema(data: number[], period: number): (number | null)[] {
  const result: (number | null)[] = [];
  const k = 2 / (period + 1);

  for (let i = 0; i < data.length; i++) {
    if (i < period - 1) {
      result.push(null);
    } else if (i === period - 1) {
      const sum = data.slice(0, period).reduce((a, b) => a + b, 0);
      result.push(sum / period);
    } else {
      result.push(data[i] * k + (result[i - 1] as number) * (1 - k));
    }
  }

  return result;
}

export interface MACDResult {
  macd: (number | null)[];
  signal: (number | null)[];
  histogram: (number | null)[];
}

export function calculateMACD(
  closes: number[],
  shortPeriod: number = 12,
  longPeriod: number = 26,
  signalPeriod: number = 9,
): MACDResult {
  const shortEma = ema(closes, shortPeriod);
  const longEma = ema(closes, longPeriod);

  const macdLine: (number | null)[] = closes.map((_, i) => {
    if (shortEma[i] === null || longEma[i] === null) return null;
    return shortEma[i]! - longEma[i]!;
  });

  const validMacd = macdLine.filter((v): v is number => v !== null);
  const signalEma = ema(validMacd, signalPeriod);

  const signal: (number | null)[] = [];
  let validIdx = 0;
  for (let i = 0; i < closes.length; i++) {
    if (macdLine[i] === null) {
      signal.push(null);
    } else {
      signal.push(signalEma[validIdx] ?? null);
      validIdx++;
    }
  }

  const histogram: (number | null)[] = closes.map((_, i) => {
    if (macdLine[i] === null || signal[i] === null) return null;
    return macdLine[i]! - signal[i]!;
  });

  return { macd: macdLine, signal, histogram };
}
