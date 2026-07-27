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

export interface WorldMonitorInboxItem {
  id: string;
  source: string;
  sourceEventId: string;
  worldMonitorEventId: string;
  status: string;
  rejectionReason?: string;
  eventType: string;
  headline: string;
  summary?: string;
  sourceUrls: string[];
  sourceCount: number;
  eventTime: string;
  receivedAt: string;
  collectedAt?: string;
  rawEventId?: string;
  provenanceAvailable?: boolean;
  isSynthetic?: boolean;
  syntheticReason?: string;
  discoveryMethod?: string;
  analysisIdentity?: string;
  aiProvider?: string;
  aiModel?: string;
  region?: string;
  possibleAffectedEtfs: string[];
  assetThemes: string[];
  severity: string;
  sourceTier: string;
  confidence: number;
  confidenceReasons: string[];
  mappingReason: string;
  normalizedEventId?: string;
  normalizedAt?: string;
  candidateId?: string;
  candidateSymbol?: string;
  candidateStatus?: string;
  candidateCreatedAt?: string;
  approvalId?: string;
  approvalDecision?: string;
  approvalAt?: string;
  paperTicketId?: string;
  paperTicketCreatedAt?: string;
  outcomeCount: number;
  latestOutcomeAt?: string;
  operatorDecision?: string;
  operatorReason?: string;
  decision?: WorldMonitorEventDecision;
  decisionHistory?: WorldMonitorEventDecision[];
  rawPayload: Record<string, unknown>;
}

export interface WorldMonitorEventDecision {
  decisionId: string;
  decision: 'NO_TRADE' | 'WATCH' | 'CANDIDATE';
  decisionVersion: number;
  rulesetVersion: string;
  processorIdentity: string;
  processingMode: 'deterministic';
  decisionAt: string;
  evidenceScore: number;
  evidenceScoreSource: string;
  affectedAssets: string[];
  unknownAssets: boolean;
  assetMappingProvenance: Record<string, unknown>;
  reasons: string[];
  blockingReasons: string[];
  missingEvidence: string[];
  trustGateState: string;
  riskReviewState: string;
  candidateId?: string;
  replayIdentity: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorldMonitorInboxResponse {
  items: WorldMonitorInboxItem[];
  total: number;
  counts: {
    genuine: number;
    syntheticTests: number;
    rejected: number;
    duplicates: number;
    candidatesCreated: number;
    noTrade?: number;
    watch?: number;
    candidate?: number;
    awaitingProcessing?: number;
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
  getWorldMonitorInbox(params: { status?: string; limit?: number } = {}) {
    const query = new URLSearchParams();
    if (params.status && params.status !== 'all') {
      query.set('status', params.status);
    }
    if (params.limit) {
      query.set('limit', String(params.limit));
    }
    const qs = query.toString();
    return apiClient.get<WorldMonitorInboxResponse>(
      `/api/v1/research/events/world-monitor/inbox${qs ? `?${qs}` : ''}`,
    );
  },
};
