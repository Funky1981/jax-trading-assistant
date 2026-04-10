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
    
    if (query.q) params.append('q', query.q);
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
   * Search memories within a required bank
   */
  async search(bank: string, queryText: string, limit = 20): Promise<MemoryItem[]> {
    if (!bank) {
      throw new Error('bank is required');
    }

    const params = new URLSearchParams({
      bank,
      q: queryText,
      limit: limit.toString(),
    });
    
    const response = await memoryClient.get<MemoryRecallResponse>(`/v1/memory/search?${params}`);
    return response.items;
  },
};
