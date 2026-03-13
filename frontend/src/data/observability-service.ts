/**
 * API service for observability metrics and health checks
 */

import { apiClient, memoryClient } from './http-client';
import type { HealthStatus, MetricEvent } from './types';

export const observabilityService = {
  /**
   * Get health status of API service
   */
  async getAPIHealth(): Promise<HealthStatus> {
    return apiClient.get<HealthStatus>('/health');
  },

  /**
   * Get health status of memory service
   */
  async getMemoryHealth(): Promise<HealthStatus> {
    return memoryClient.get<HealthStatus>('/health');
  },

  /**
   * Get recent metrics via the metrics API endpoint
   */
  async getRecentMetrics(limit = 100): Promise<MetricEvent[]> {
    return apiClient.get<MetricEvent[]>(`/api/v1/metrics?limit=${limit}`);
  },

  /**
   * Get metrics for a specific run
   */
  async getRunMetrics(runId: string): Promise<MetricEvent[]> {
    return apiClient.get<MetricEvent[]>(`/api/v1/metrics/runs/${runId}`);
  },
};
