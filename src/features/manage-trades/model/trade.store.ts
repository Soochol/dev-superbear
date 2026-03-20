import { create } from 'zustand';
import { apiGet, apiPost } from '@/shared/api/client';
import type { Trade, PnLSummary, CreateTradeInput } from '@/entities/trade/model/types';

interface TradeState {
  trades: Trade[];
  summary: PnLSummary | null;
  loading: boolean;

  fetchTrades: (caseId: string, currentPrice?: number) => Promise<void>;
  addTrade: (caseId: string, input: CreateTradeInput) => Promise<void>;
}

export const useTradeStore = create<TradeState>()((set) => ({
  trades: [],
  summary: null,
  loading: false,

  fetchTrades: async (caseId, currentPrice = 0) => {
    set({ loading: true });
    try {
      const res = await apiGet<{ data: { trades: Trade[]; summary: PnLSummary } }>(
        `/api/v1/cases/${caseId}/trades?currentPrice=${currentPrice}`
      );
      set({ trades: res.data.trades, summary: res.data.summary, loading: false });
    } catch {
      set({ loading: false });
    }
  },

  addTrade: async (caseId, input) => {
    await apiPost(`/api/v1/cases/${caseId}/trades`, input);
  },
}));
