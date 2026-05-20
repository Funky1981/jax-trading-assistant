/**
 * API service for trading mode operations
 */

import { apiClient } from './http-client';
import type { TradingMode, TradingModeCatalog } from './types';

export const tradingModesService = {
  /**
   * List all available trading modes
   */
  async list(): Promise<TradingMode[]> {
    const catalog = await apiClient.get<TradingModeCatalog>('/api/v1/trading-modes');
    return catalog.modes ?? [];
  },

  /**
   * Get a specific trading mode by ID
   */
  async get(id: string): Promise<TradingMode> {
    return apiClient.get<TradingMode>(`/api/v1/trading-modes/${id}`);
  },
};
