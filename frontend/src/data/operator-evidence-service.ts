import { apiClient } from './http-client';

export interface OperatorEvidenceOverview {
  runtimeMode: string;
  allowLiveTrading: boolean;
  executionEnabled: boolean;
  executionWorkerEnabled: boolean;
  brokerExecutionAllowed: boolean;
  maximumLeverage: number;
  genuineEvents: number;
  syntheticEvents: number;
  rejectedEvents: number;
  deduplicatedEvents: number;
  candidates: number;
  approvals: number;
  paperTickets: number;
  pendingCheckpoints: number;
  completedCheckpoints: number;
  missingDataCheckpoints: number;
  ambiguousCheckpoints: number;
  checkedAt: string;
}

export interface OutcomeCheckpoint {
  name: string;
  trackingStartedAt: string;
  trackingStartSource: string;
  dueAt: string;
  observationAt?: string;
  entryPrice: number;
  checkpointPrice?: number;
  percentageReturn?: number;
  hypotheticalPnl?: number;
  maximumFavourableExcursion?: number;
  maximumAdverseExcursion?: number;
  targetTouched: boolean;
  stopTouched: boolean;
  firstTargetTouchAt?: string;
  firstStopTouchAt?: string;
  status: string;
  dataQualityStatus: string;
  marketDataSource?: string;
  createdAt: string;
  updatedAt: string;
}

export interface OperatorCandidateEvidence {
  evidenceScore?: number;
  evidenceStatus: string;
  gateStatus: string;
  riskStatus: string;
  approvalId?: string;
  approvalDecision?: string;
  decisionProvenance?: 'human' | 'non_human' | 'none';
  approvedBy?: string;
  approvalReason?: string;
  approvalAt?: string;
  paperTicketId?: string;
  paperTicketStatus?: string;
  entry?: number;
  stop?: number;
  target?: number;
  quantity?: number;
  plannedRisk?: number;
  plannedReward?: number;
  rewardRisk?: number;
  notional?: number;
  accountEquityAssumption?: number;
  leverage?: number;
  checkpoints: OutcomeCheckpoint[];
  selectedExecutionCounts: Record<string, number>;
  historicalExecutionCounts: Record<string, number>;
}

export interface OperatorCandidateSummary {
  candidateId: string;
  symbol: string;
  setupType: string;
  candidateStatus: string;
  humanDecision: string;
  decisionProvenance: 'human' | 'non_human' | 'none';
  paperTicketId?: string;
  paperTicketStatus?: string;
  latestOutcomeStatus?: string;
  completedCheckpoints: number;
  pendingCheckpoints: number;
  missingCheckpoints: number;
  ambiguousCheckpoints: number;
  reason: string;
  blockReason?: string;
  createdAt: string;
  expiresAt?: string;
}

export const operatorEvidenceService = {
  overview: () => apiClient.get<OperatorEvidenceOverview>('/api/v1/operator-evidence/overview'),
  candidates: () =>
    apiClient.get<OperatorCandidateSummary[]>('/api/v1/operator-evidence/candidates'),
  candidate: (id: string) =>
    apiClient.get<OperatorCandidateEvidence>(`/api/v1/operator-evidence/candidates/${id}`),
};
