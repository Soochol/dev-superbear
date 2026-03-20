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

export interface PnLSummary {
  total_buy_quantity: number;
  total_sell_quantity: number;
  remaining_quantity: number;
  average_buy_price: number;
  realized_pnl: number;
  realized_return: number;
  unrealized_pnl: number;
  unrealized_return: number;
  total_fees: number;
}

export interface CreateTradeInput {
  type: TradeType;
  price: number;
  quantity: number;
  fee?: number;
  date: string;
  note?: string;
}

export interface TradesResponse {
  trades: Trade[];
  summary: PnLSummary;
}
