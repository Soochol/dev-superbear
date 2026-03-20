import { create } from 'zustand';
import { apiGet, apiPost } from '@/shared/api/client';
import type { Case, TimelineEvent } from './types';

interface CaseState {
  cases: Case[];
  selectedCaseId: string | null;
  selectedCase: Case | null;
  timelineEvents: TimelineEvent[];
  loading: boolean;
  error: string | null;

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
  error: null,

  fetchCases: async () => {
    set({ loading: true, error: null });
    try {
      const res = await apiGet<{ data: Case[]; pagination: unknown }>('/api/v1/cases');
      set({ cases: res.data, loading: false });
    } catch {
      set({ loading: false, error: 'Failed to load cases' });
    }
  },

  selectCase: async (id) => {
    set({ selectedCaseId: id, error: null });
    try {
      const res = await apiGet<{ data: Case }>(`/api/v1/cases/${id}`);
      set({ selectedCase: res.data });
    } catch {
      set({ selectedCase: null, error: 'Failed to load case details' });
    }
  },

  fetchTimeline: async (caseId) => {
    try {
      const res = await apiGet<{ data: TimelineEvent[] }>(`/api/v1/cases/${caseId}/timeline`);
      set({ timelineEvents: res.data });
    } catch {
      set({ timelineEvents: [], error: 'Failed to load timeline' });
    }
  },

  closeCase: async (id, status, reason) => {
    try {
      await apiPost(`/api/v1/cases/${id}/close`, { status, reason });
      await get().fetchCases();
      await get().selectCase(id);
    } catch (e) {
      set({ error: 'Failed to close case' });
      throw e;
    }
  },
}));
