/**
 * React hooks for memory operations
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { memoryService } from '../data/memory-service';
import type { MemoryQuery, MemoryItem } from '../data/types';

export function useMemoryRecall(bank: string, query: MemoryQuery) {
  return useQuery({
    queryKey: ['memory', 'recall', bank, query],
    queryFn: () => memoryService.recall(bank, query),
    enabled: !!bank,
  });
}

export function useMemoryRetain(bank: string) {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (item: Omit<MemoryItem, 'id'>) => memoryService.retain(bank, item),
    onSuccess: () => {
      // Invalidate recall queries to refresh
      queryClient.invalidateQueries({ queryKey: ['memory', 'recall', bank] });
    },
  });
}

export function useMemorySearch(queryText: string, bank?: string, limit = 20) {
  return useQuery({
    queryKey: ['memory', 'search', queryText, bank, limit],
    queryFn: () => memoryService.search(queryText, bank, limit),
    enabled: queryText.length > 2, // Only search if query is meaningful
  });
}

export function useMemoryBanks() {
  return useQuery({
    queryKey: ['memory', 'banks'],
    queryFn: () => memoryService.listBanks(),
  });
}

export function useMemory(bank: string, id: string | null) {
  return useQuery({
    queryKey: ['memory', 'item', bank, id],
    queryFn: () => id ? memoryService.getMemory(bank, id) : Promise.resolve(null),
    enabled: !!id && !!bank,
  });
}
