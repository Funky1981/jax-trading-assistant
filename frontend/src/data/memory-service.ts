/**
 * API service for memory operations
 */

import { memoryClient } from './http-client';
import type { MemoryItem, MemoryQuery, MemoryRecallResponse } from './types';

export const memoryService = {
  /**
   * Recall memories from a bank
   */
  async recall(bank: string, query: MemoryQuery): Promise<MemoryRecallResponse> {
    const params = new URLSearchParams();
    
    if (query.symbol) params.append('symbol', query.symbol);
    if (query.type) params.append('type', query.type);
    if (query.limit) params.append('limit', query.limit.toString());
    if (query.since) params.append('since', query.since);
    if (query.tags) query.tags.forEach(tag => params.append('tags', tag));

    const queryString = params.toString();
    const path = `/v1/memory/banks/${bank}/items${queryString ? `?${queryString}` : ''}`;
    
    return memoryClient.get<MemoryRecallResponse>(path);
  },

  /**
   * Retain a memory to a bank
   */
  async retain(bank: string, item: Omit<MemoryItem, 'id'>): Promise<{ id: string }> {
    return memoryClient.post<{ id: string }>(`/v1/memory/banks/${bank}/items`, item);
  },

  /**
   * Get a specific memory by ID
   */
  async getMemory(bank: string, id: string): Promise<MemoryItem> {
    return memoryClient.get<MemoryItem>(`/v1/memory/banks/${bank}/items/${id}`);
  },

  /**
   * List all banks
   */
  async listBanks(): Promise<string[]> {
    return memoryClient.get<string[]>('/v1/memory/banks');
  },

  /**
   * Search memories across all banks or in a specific bank
   */
  async search(queryText: string, bank?: string, limit = 20): Promise<MemoryItem[]> {
    const params = new URLSearchParams({
      q: queryText,
      limit: limit.toString(),
    });
    if (bank) params.append('bank', bank);
    
    const response = await memoryClient.get<MemoryRecallResponse>(`/v1/memory/search?${params}`);
    return response.items;
  },
};
