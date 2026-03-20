export interface SearchResult {
  symbol: string;
  name: string;
  matchedValue: number | string;
  close?: number;
  volume?: number;
  tradeValue?: number;
  change?: number;
  changePct?: number;
}
