import { apiClient } from './http-client';
import type { TestRunSummary, TestingGateStatus, TriggerTestResponse } from './types';

export const testingService = {
  getStatus(): Promise<TestingGateStatus[]> {
    return apiClient.get('/api/v1/testing/status');
  },

  getGates(): Promise<TestingGateStatus[]> {
    return apiClient.get('/api/v1/gates');
  },

  getTestRuns(limit = 50): Promise<TestRunSummary[]> {
    return apiClient.get(`/api/v1/test-runs?limit=${limit}`);
  },

  triggerDataRecon(): Promise<TriggerTestResponse> {
    return apiClient.post('/api/v1/testing/recon/data');
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
};

