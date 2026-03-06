import { apiClient } from './http-client';
import type { MarketDataStatus, TradingPilotStatus } from './types';

export const systemService = {
  getMarketDataStatus() {
    return apiClient.get<MarketDataStatus>('/api/v1/system/market-data-status');
  },
  getTradingPilotStatus() {
    return apiClient.get<TradingPilotStatus>('/api/v1/trading/pilot-status');
  },
};
