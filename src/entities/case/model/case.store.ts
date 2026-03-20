import { create } from 'zustand';
import { apiGet, apiPost } from '@/shared/api/client';
import type { Case, TimelineEvent } from './types';

interface CaseState {
  cases: Case[];
  selectedCaseId: string | null;
  selectedCase: Case | null;
  timelineEvents: TimelineEvent[];
  loading: boolean;

  fetchCases: () => Promise<void>;
  selectCase: (id: string) => Promise<void>;
  fetchTimeline: (caseId: string) => Promise<void>;
  closeCase: (id: string, status: 'CLOSED_SUCCESS' | 'CLOSED_FAILURE', reason: string) => Promise<void>;
}

export const useCaseStore = create<CaseState>()((set, get) => ({
  cases: [],
  selectedCaseId: null,
  selectedCase: null,
  timelineEvents: [],
  loading: false,

  fetchCases: async () => {
    set({ loading: true });
    try {
      const res = await apiGet<{ data: Case[]; pagination: unknown }>('/api/v1/cases');
      set({ cases: res.data, loading: false });
    } catch {
      set({ loading: false });
    }
  },

  selectCase: async (id) => {
    set({ selectedCaseId: id });
    try {
      const res = await apiGet<{ data: Case }>(`/api/v1/cases/${id}`);
      set({ selectedCase: res.data });
    } catch {
      set({ selectedCase: null });
    }
  },

  fetchTimeline: async (caseId) => {
    try {
      const res = await apiGet<{ data: TimelineEvent[] }>(`/api/v1/cases/${caseId}/timeline`);
      set({ timelineEvents: res.data });
    } catch {
      set({ timelineEvents: [] });
    }
  },

  closeCase: async (id, status, reason) => {
    await apiPost(`/api/v1/cases/${id}/close`, { status, reason });
    get().fetchCases();
  },
}));
