import { apiClient } from './http-client';
import type { RobustPerformance } from './types';

export const robustService = {
  getPerformance(): Promise<RobustPerformance> {
    return apiClient.get<RobustPerformance>('/api/v1/robust/performance');
  },
};
