/**
 * API service for trading mode operations
 */

import { apiClient } from './http-client';
import type { TradingMode, TradingModeCatalog } from './types';

const SWING_COPY =
  'Swing Trading researches multi-day ETF setups. It creates approval-gated paper candidates only after evidence, chart history, and daily revalidation checks pass.';

function withDisplayMetadata(mode: TradingMode): TradingMode {
  if (mode.id !== 'etf_swing_research' && mode.id !== 'etf_swing_paper') {
    return mode;
  }
  return {
    ...mode,
    displayCopy: mode.displayCopy ?? SWING_COPY,
    horizonLabel: mode.horizonLabel ?? 'Swing',
  };
}

export const tradingModesService = {
  /**
   * List all available trading modes
   */
  async list(): Promise<TradingMode[]> {
    const catalog = await apiClient.get<TradingModeCatalog>('/api/v1/trading-modes');
    return (catalog.modes ?? []).map(withDisplayMetadata);
  },

  /**
   * Get a specific trading mode by ID
   */
  async get(id: string): Promise<TradingMode> {
    const mode = await apiClient.get<TradingMode>(`/api/v1/trading-modes/${id}`);
    return withDisplayMetadata(mode);
  },
};
