import { apiClient } from './http-client';
import type { AIOverviewApiResponse, AIScannerApiState } from './types';

export interface AISuggestionPromotionRequest {
  symbol: string;
  action: 'BUY' | 'SELL';
  confidence: number;
  reasoning: string;
  risk: string;
  source: string;
}

export interface AISuggestionPromotionResponse {
  candidateId: string;
  signalId: string;
  route: 'approval_required' | 'manual_allowed';
  status: string;
}

export interface WorldMonitorStatus {
  connected: boolean;
  lastReceivedAt?: string;
  lastSourceEventId?: string;
  lastStatus?: string;
  lastHeadline?: string;
  lastSymbols?: string[];
  lastCandidateId?: string;
  counts: {
    total: number;
    pending: number;
    candidatesCreated: number;
    rejected: number;
    ignored: number;
  };
  checkedAt: string;
}

export const aiService = {
  getOverview() {
    return apiClient.get<AIOverviewApiResponse>('/api/v1/ai/overview');
  },
  getScanner() {
    return apiClient.get<AIScannerApiState>('/api/v1/ai/scanner');
  },
  updateScanner(state: AIScannerApiState) {
    return apiClient.put<AIScannerApiState>('/api/v1/ai/scanner', state);
  },
  promoteSuggestion(request: AISuggestionPromotionRequest) {
    return apiClient.post<AISuggestionPromotionResponse>('/api/v1/ai/suggestions/promote', request);
  },
  getWorldMonitorStatus() {
    return apiClient.get<WorldMonitorStatus>('/api/v1/research/events/world-monitor/status');
  },
};
