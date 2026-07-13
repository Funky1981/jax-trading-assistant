import { apiClient } from './http-client';
import type { SentimentEvidence } from './types';

export interface CandidateTrade {
  id: string;
  strategyInstanceId: string;
  signalId?: string;
  strategyId?: string;
  artifactId?: string;
  symbol: string;
  signalType: 'BUY' | 'SELL';
  status: string;
  entryPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
  confidence?: number;
  reasoning?: string;
  blockReason?: string;
  blockedReasonCode?: string;
  sessionDate: string;
  expiresAt?: string;
  detectedAt: string;
  qualifiedAt?: string;
  blockedAt?: string;
  submittedAt?: string;
  filledAt?: string;
  dataProvenance: string;
  metadata?: Record<string, unknown>;
  sentiment?: SentimentEvidence;
  latestApproval?: {
    id: string;
    decision: string;
    approvedBy: string;
    decidedAt: string;
  };
  executionInstructionId?: string;
  tradeId?: string;
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

export interface ExecutionSummary {
  id: string;
  approvalId: string;
  candidateId: string;
  tradeId?: string;
  symbol: string;
  signalType: string;
  entryPrice?: number;
  stopLoss?: number;
  takeProfit?: number;
  status: string;
  brokerOrderId?: string;
  fillPrice?: number;
  fillQty?: number;
  errorMessage?: string;
  submittedAt?: string;
  filledAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PaperTicketReview {
  paperTicketId: string;
  candidateId: string;
  createdAt: string;
  updatedAt: string;
  status: string;
  symbol: string;
  direction: string;
  setupType: string;
  catalystSummary: string;
  entryPrice: number;
  stopLossPrice: number;
  targetPrice: number;
  positionSize: number;
  maxNormalLoss: number;
  maxSlippageAdjustedLoss: number;
  rewardRiskRatio: number;
  evidenceStatus: string;
  gateStatus: string;
  riskStatus: string;
  approvalStatus: string;
  paperOnly: true;
  rejectReasons?: string[];
  warningReasons?: string[];
}

export interface CandidateApprovalDetail {
  candidateId: string;
  latestApproval?: CandidateApproval;
  paperTicket?: PaperTicketReview;
  execution?: ExecutionSummary;
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
  metadata?: Record<string, unknown>;
  detectedAt: string;
  expiresAt?: string;
  instanceName: string;
  sentiment?: SentimentEvidence;
}

export interface MobileTelegramApprovalRequest {
  token: string;
  action: 'approved' | 'rejected';
  actor?: string;
  reason?: string;
  guardrailHash?: string;
  runtimeMode?: 'paper' | 'live';
}

export interface MobileTelegramApprovalResponse {
  approvalId: string;
  candidateId: string;
  decision: string;
  runtimeMode: string;
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
    return apiClient.get<CandidateApprovalDetail>(`/api/v1/approvals/${candidateId}`);
  },

  getPaperTicketQueue(limit = 50) {
    return apiClient.get<PaperTicketReview[]>(`/api/v1/paper-tickets?limit=${limit}`);
  },

  markPaperTicketReviewed(paperTicketId: string, note?: string) {
    return apiClient.post<PaperTicketReview>(`/api/v1/paper-tickets/${paperTicketId}/mark-reviewed`, { note });
  },

  cancelPaperTicket(paperTicketId: string, note?: string) {
    return apiClient.post<PaperTicketReview>(`/api/v1/paper-tickets/${paperTicketId}/cancel`, { note });
  },

  addPaperTicketNote(paperTicketId: string, note: string) {
    return apiClient.post<PaperTicketReview>(`/api/v1/paper-tickets/${paperTicketId}/notes`, { note });
  },

  approve(candidateId: string, notes?: string, overrideReason?: ApprovalOverrideReasonInput) {
    return apiClient.post<CandidateApprovalDetail>(`/api/v1/approvals/${candidateId}/approve`, { notes, overrideReason });
  },

  reject(candidateId: string, notes?: string, overrideReason?: ApprovalOverrideReasonInput) {
    return apiClient.post<CandidateApprovalDetail>(`/api/v1/approvals/${candidateId}/reject`, { notes, overrideReason });
  },

  snooze(candidateId: string, snoozeHours = 4, notes?: string, overrideReason?: ApprovalOverrideReasonInput) {
    return apiClient.post<CandidateApprovalDetail>(`/api/v1/approvals/${candidateId}/snooze`, { snoozeHours, notes, overrideReason });
  },

  reanalyze(candidateId: string, notes?: string) {
    return apiClient.post<CandidateApprovalDetail>(`/api/v1/approvals/${candidateId}/reanalyze`, { notes });
  },

  submitTelegramDecision(payload: MobileTelegramApprovalRequest) {
    return apiClient.post<MobileTelegramApprovalResponse>('/api/v1/mobile/telegram/webhook', payload);
  },
};

export interface ApprovalOverrideReasonInput {
  reasonCode: string;
  sentimentEvidenceViewed: boolean;
}
