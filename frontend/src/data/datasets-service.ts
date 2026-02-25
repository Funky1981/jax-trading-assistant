import { apiClient } from './http-client';
import type { DatasetDetail, DatasetListResponse } from './types';

interface ListDatasetsParams {
  symbol?: string;
  limit?: number;
  offset?: number;
}

export const datasetsService = {
  async list(params: ListDatasetsParams = {}): Promise<DatasetListResponse> {
    const query = new URLSearchParams();
    if (params.symbol) query.set('symbol', params.symbol);
    if (params.limit) query.set('limit', String(params.limit));
    if (params.offset) query.set('offset', String(params.offset));
    const suffix = query.toString();
    return apiClient.get<DatasetListResponse>(suffix ? `/api/v1/datasets?${suffix}` : '/api/v1/datasets');
  },

  get(datasetId: string): Promise<DatasetDetail> {
    return apiClient.get<DatasetDetail>(`/api/v1/datasets/${datasetId}`);
  },
};
