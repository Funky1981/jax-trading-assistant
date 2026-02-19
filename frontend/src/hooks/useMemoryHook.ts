import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { memoryClient } from '@/data/http-client';

export interface MemoryBank {
  id: string;
  name: string;
  description: string;
  entryCount: number;
  lastUpdated: number;
}

export interface MemoryEntry {
  id: string;
  bankId: string;
  content: string;
  metadata: Record<string, unknown>;
  createdAt: number;
  embedding?: number[];
}

// Mock memory banks and entries have been removed.
// All data must come from the real jax-memory / jax-api endpoints.

async function fetchMemoryBanks(): Promise<MemoryBank[]> {
  return memoryClient.get<MemoryBank[]>('/v1/memory/banks');
}

async function fetchMemoryEntries(bankId: string): Promise<MemoryEntry[]> {
  return memoryClient.get<MemoryEntry[]>(`/v1/memory/banks/${bankId}/entries`);
}

async function searchMemory(query: string): Promise<MemoryEntry[]> {
  return memoryClient.get<MemoryEntry[]>(`/v1/memory/search?q=${encodeURIComponent(query)}`);
}

export function useMemoryBanks() {
  return useQuery({
    queryKey: ['memory', 'banks'],
    queryFn: fetchMemoryBanks,
  });
}

export function useMemoryEntries(bankId: string | null) {
  return useQuery({
    queryKey: ['memory', 'entries', bankId],
    queryFn: () => (bankId ? fetchMemoryEntries(bankId) : Promise.resolve([])),
    enabled: !!bankId,
  });
}

export function useMemorySearch(query: string) {
  return useQuery({
    queryKey: ['memory', 'search', query],
    queryFn: () => searchMemory(query),
    enabled: query.length >= 2,
  });
}

export function useCreateMemoryEntry() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (entry: { bankId: string; content: string; metadata?: Record<string, unknown> }) => {
      console.log('Creating memory entry:', entry);
      return { success: true };
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ['memory', 'entries', variables.bankId] });
      queryClient.invalidateQueries({ queryKey: ['memory', 'banks'] });
    },
  });
}
