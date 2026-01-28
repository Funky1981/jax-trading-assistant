/**
 * API service for strategy operations
 */

import { apiClient } from './http-client';
import type { StrategySignal, StrategyPerformance } from './types';

export const strategyService = {
  /**
   * List available strategies
   */
  async listStrategies(): Promise<Array<{ id: string; name: string; description: string }>> {
    return apiClient.get('/api/v1/strategies');
  },

  /**
   * Get recent signals from a strategy
   */
  async getSignals(strategyId: string, limit = 50): Promise<StrategySignal[]> {
    return apiClient.get<StrategySignal[]>(`/api/v1/strategies/${strategyId}/signals?limit=${limit}`);
  },

  /**
   * Get strategy performance metrics
   */
  async getPerformance(strategyId: string): Promise<StrategyPerformance> {
    return apiClient.get<StrategyPerformance>(`/api/v1/strategies/${strategyId}/performance`);
  },

  /**
   * Analyze a symbol with a specific strategy
   */
  async analyze(strategyId: string, symbol: string, constraints: Record<string, unknown>): Promise<StrategySignal> {
    return apiClient.post<StrategySignal>(`/api/v1/strategies/${strategyId}/analyze`, {
      symbol,
      constraints,
    });
  },
};
