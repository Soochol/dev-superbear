import { create } from 'zustand';
import { apiGet, apiPost, apiDelete } from '@/shared/api/client';
import type { PriceAlert } from '@/entities/price-alert/model/types';

interface AlertState {
  pendingAlerts: PriceAlert[];
  triggeredAlerts: PriceAlert[];
  loading: boolean;

  fetchAlerts: (caseId: string) => Promise<void>;
  addAlert: (caseId: string, condition: string, label: string) => Promise<void>;
  deleteAlert: (caseId: string, alertId: string) => Promise<void>;
}

export const useAlertStore = create<AlertState>()((set) => ({
  pendingAlerts: [],
  triggeredAlerts: [],
  loading: false,

  fetchAlerts: async (caseId) => {
    set({ loading: true });
    try {
      const res = await apiGet<{ data: { pending: PriceAlert[]; triggered: PriceAlert[] } }>(
        `/api/v1/cases/${caseId}/alerts`
      );
      set({
        pendingAlerts: res.data.pending,
        triggeredAlerts: res.data.triggered,
        loading: false,
      });
    } catch {
      set({ loading: false });
    }
  },

  addAlert: async (caseId, condition, label) => {
    await apiPost(`/api/v1/cases/${caseId}/alerts`, { condition, label });
  },

  deleteAlert: async (caseId, alertId) => {
    await apiDelete(`/api/v1/cases/${caseId}/alerts/${alertId}`);
  },
}));
