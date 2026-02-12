import { apiClient } from './http-client';
import type {
  Signal,
  SignalListResponse,
  RecommendationListResponse,
} from './types';

interface ListSignalsParams {
  status?: string;
  symbol?: string;
  strategy?: string;
  limit?: number;
  offset?: number;
}

interface ListRecommendationsParams {
  limit?: number;
  offset?: number;
}

function buildQuery(params: Record<string, string | number | undefined>) {
  const entries = Object.entries(params).filter(([, value]) => value !== undefined && value !== '');
  if (entries.length === 0) return '';
  return `?${entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join('&')}`;
}

export const signalsService = {
  list(params: ListSignalsParams = {}) {
    const query = buildQuery({
      status: params.status,
      symbol: params.symbol,
      strategy: params.strategy,
      limit: params.limit,
      offset: params.offset,
    });
    return apiClient.get<SignalListResponse>(`/api/v1/signals${query}`);
  },

  approve(signalId: string, approvedBy: string, modificationNotes?: string) {
    return apiClient.post<Signal>(`/api/v1/signals/${signalId}/approve`, {
      approved_by: approvedBy,
      modification_notes: modificationNotes,
    });
  },

  reject(signalId: string, approvedBy: string, rejectionReason?: string) {
    return apiClient.post<Signal>(`/api/v1/signals/${signalId}/reject`, {
      approved_by: approvedBy,
      rejection_reason: rejectionReason,
    });
  },

  analyze(signalId: string, context?: string) {
    return apiClient.post<{ runId: string; status: string }>(`/api/v1/signals/${signalId}/analyze`, {
      context,
    });
  },

  listRecommendations(params: ListRecommendationsParams = {}) {
    const query = buildQuery({
      limit: params.limit,
      offset: params.offset,
    });
    return apiClient.get<RecommendationListResponse>(`/api/v1/recommendations${query}`);
  },
};
