/**
 * API service for orchestration operations
 */

import { apiClient } from './http-client';
import type { OrchestrationRequest, OrchestrationResult } from './types';

export const orchestrationService = {
  /**
   * Execute an orchestration run
   */
  async run(request: OrchestrationRequest): Promise<OrchestrationResult> {
    return apiClient.post<OrchestrationResult>('/api/v1/orchestrate', request);
  },

  /**
   * Get status of a specific orchestration run
   */
  async getRunStatus(runId: string): Promise<OrchestrationResult & { status: string }> {
    return apiClient.get(`/api/v1/orchestrate/runs/${runId}`);
  },

  /**
   * List recent orchestration runs
   */
  async listRuns(limit = 20): Promise<Array<{ runId: string; symbol: string; timestamp: string; success: boolean }>> {
    return apiClient.get(`/api/v1/orchestrate/runs?limit=${limit}`);
  },
};
