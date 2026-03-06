import { apiClient } from './http-client';
import type { MarketDataStatus } from './types';

export const systemService = {
  getMarketDataStatus() {
    return apiClient.get<MarketDataStatus>('/api/v1/system/market-data-status');
  },
};
