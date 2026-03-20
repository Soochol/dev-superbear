export type TradeType = 'BUY' | 'SELL';

export interface Trade {
  id: string;
  case_id: string;
  user_id: string;
  type: TradeType;
  price: number;
  quantity: number;
  fee: number;
  date: string;
  note?: string;
  created_at: string;
}
