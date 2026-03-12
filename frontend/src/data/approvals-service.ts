import { apiClient } from './http-client';

export interface CandidateTrade {
  id: string;
  strategyInstanceId: string;
  symbol: string;
  signalType: 'BUY' | 'SELL';
  status: string;
  entryPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
  confidence?: number;
  reasoning?: string;
  blockReason?: string;
  sessionDate: string;
  expiresAt?: string;
  detectedAt: string;
  qualifiedAt?: string;
  dataProvenance: string;
}

export interface CandidateApproval {
  id: string;
  candidateId: string;
  decision: string;
  approvedBy: string;
  notes?: string;
  expiryAt?: string;
  snoozeUntil?: string;
  reanalysisRequested: boolean;
  decidedAt: string;
}

export interface ApprovalQueueItem {
  id: string;
  symbol: string;
  signalType: string;
  confidence?: number;
  entryPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
  reasoning?: string;
  blockReason?: string;
  detectedAt: string;
  expiresAt?: string;
  instanceName: string;
}

function buildQuery(params: Record<string, string | number | undefined>) {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== '');
  if (!entries.length) return '';
  return `?${entries.map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`).join('&')}`;
}

export const candidatesService = {
  list(params: { status?: string; symbol?: string; limit?: number } = {}) {
    return apiClient.get<CandidateTrade[]>(`/api/v1/candidates${buildQuery(params as Record<string, string | number | undefined>)}`);
  },

  get(id: string) {
    return apiClient.get<CandidateTrade>(`/api/v1/candidates/${id}`);
  },

  refresh(id: string) {
    return apiClient.post<CandidateTrade>(`/api/v1/candidates/${id}/refresh`, {});
  },
};

export const approvalsService = {
  getQueue(limit = 50) {
    return apiClient.get<ApprovalQueueItem[]>(`/api/v1/approvals/queue?limit=${limit}`);
  },

  getByCandidate(candidateId: string) {
    return apiClient.get<CandidateApproval>(`/api/v1/approvals/${candidateId}`);
  },

  approve(candidateId: string, notes?: string) {
    return apiClient.post<CandidateApproval>(`/api/v1/approvals/${candidateId}/approve`, { notes });
  },

  reject(candidateId: string, notes?: string) {
    return apiClient.post<CandidateApproval>(`/api/v1/approvals/${candidateId}/reject`, { notes });
  },

  snooze(candidateId: string, snoozeHours = 4, notes?: string) {
    return apiClient.post<CandidateApproval>(`/api/v1/approvals/${candidateId}/snooze`, { snoozeHours, notes });
  },

  reanalyze(candidateId: string, notes?: string) {
    return apiClient.post<CandidateApproval>(`/api/v1/approvals/${candidateId}/reanalyze`, { notes });
  },
};
