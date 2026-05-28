import { apiClient } from './http-client';
import type { AIOverviewApiResponse, AIScannerApiState } from './types';

export const aiService = {
  getOverview() {
    return apiClient.get<AIOverviewApiResponse>('/api/v1/ai/overview');
  },
  getScanner() {
    return apiClient.get<AIScannerApiState>('/api/v1/ai/scanner');
  },
  updateScanner(state: AIScannerApiState) {
    return apiClient.put<AIScannerApiState>('/api/v1/ai/scanner', state);
  },
};
