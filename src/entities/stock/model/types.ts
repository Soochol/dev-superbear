export interface StockInfo {
  symbol: string;
  name: string;
  price: number;
  change: number;
  changePct: number;
}

export interface StockListItem {
  symbol: string;
  name: string;
  matchedValue?: number | string;
}
