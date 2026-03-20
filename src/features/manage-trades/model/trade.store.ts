import { create } from 'zustand';
import { apiGet, apiPost } from '@/shared/api/client';
import type { Trade, PnLSummary, CreateTradeInput } from '@/entities/trade';

interface TradeState {
  trades: Trade[];
  summary: PnLSummary | null;
  loading: boolean;
  error: string | null;

  fetchTrades: (caseId: string, currentPrice?: number) => Promise<void>;
  addTrade: (caseId: string, input: CreateTradeInput) => Promise<void>;
}

export const useTradeStore = create<TradeState>()((set) => ({
  trades: [],
  summary: null,
  loading: false,
  error: null,

  fetchTrades: async (caseId, currentPrice = 0) => {
    set({ loading: true, error: null });
    try {
      const res = await apiGet<{ data: { trades: Trade[]; summary: PnLSummary } }>(
        `/api/v1/cases/${caseId}/trades?currentPrice=${currentPrice}`
      );
      set({ trades: res.data.trades, summary: res.data.summary, loading: false });
    } catch {
      set({ loading: false, error: 'Failed to load trades' });
    }
  },

  addTrade: async (caseId, input) => {
    try {
      await apiPost(`/api/v1/cases/${caseId}/trades`, input);
    } catch (e) {
      set({ error: 'Failed to add trade' });
      throw e;
    }
  },
}));
