import { apiClient } from './http-client';
import type { TestRunSummary, TestingGateStatus, TriggerTestResponse } from './types';

function normalizeArtifactUri(uri?: string): string | undefined {
  if (!uri) return undefined;
  if (uri.startsWith('http://') || uri.startsWith('https://')) {
    return uri;
  }
  const path = uri.startsWith('/') ? uri : `/${uri}`;
  return apiClient.buildUrl(path);
}

function normalizeDetails(details?: Record<string, unknown>): Record<string, unknown> | undefined {
  if (!details) return details;
  const artifactRaw = (details as Record<string, unknown>).artifactUri ?? (details as Record<string, unknown>).artifactURI;
  const artifactUri = normalizeArtifactUri(typeof artifactRaw === 'string' ? artifactRaw : undefined);
  if (!artifactUri) return details;
  return { ...details, artifactUri, artifactURI: artifactUri };
}

export const testingService = {
  async getStatus(): Promise<TestingGateStatus[]> {
    const data = await apiClient.get<TestingGateStatus[]>('/api/v1/testing/status');
    return data.map((gate) => ({ ...gate, details: normalizeDetails(gate.details) }));
  },

  async getGates(): Promise<TestingGateStatus[]> {
    const data = await apiClient.get<TestingGateStatus[]>('/api/v1/gates');
    return data.map((gate) => ({ ...gate, details: normalizeDetails(gate.details) }));
  },

  async getTestRuns(limit = 50): Promise<TestRunSummary[]> {
    const data = await apiClient.get<TestRunSummary[]>(`/api/v1/test-runs?limit=${limit}`);
    return data.map((run) => ({ ...run, artifactUri: normalizeArtifactUri(run.artifactUri) }));
  },

  triggerDataRecon(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/recon/data');
  },

  triggerConfigIntegrity(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/config-integrity');
  },

  triggerDeterministicReplay(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/replay');
  },

  triggerArtifactPromotion(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/artifact-promotion');
  },

  triggerExecutionIntegration(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/execution-path');
  },

  triggerPnlRecon(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/recon/pnl');
  },

  triggerFailureSuite(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/failure-tests/run');
  },

  triggerFlattenProof(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/flatten-proof');
  },

  triggerAIAudit(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/ai-audit');
  },

  triggerProvenanceIntegrity(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/provenance');
  },

  triggerShadowParity(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/shadow-parity');
  },

  triggerAllGates(): Promise<{ status: string; runs: TriggerTestResponse[] }> {
    return apiClient.post('/api/v1/testing/run-all');
  },
};
