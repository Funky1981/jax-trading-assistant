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
   * Get recent metrics (if exposed via API)
   * Note: In production, metrics would likely go to a time-series DB
   * This is a placeholder for potential metrics endpoint
   */
  async getRecentMetrics(limit = 100): Promise<MetricEvent[]> {
    // Placeholder - would need actual metrics endpoint
    return apiClient.get<MetricEvent[]>(`/api/v1/metrics?limit=${limit}`);
  },

  /**
   * Get metrics for a specific run
   */
  async getRunMetrics(runId: string): Promise<MetricEvent[]> {
    return apiClient.get<MetricEvent[]>(`/api/v1/metrics/runs/${runId}`);
  },
};
