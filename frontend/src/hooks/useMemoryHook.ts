import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

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
  const response = await fetch(buildUrl('JAX_API', '/api/memory/banks'));
  if (!response.ok) {
    throw new Error(`Memory banks unavailable (HTTP ${response.status})`);
  }
  return response.json();
}

async function fetchMemoryEntries(bankId: string): Promise<MemoryEntry[]> {
  const response = await fetch(`${buildUrl('JAX_API', '/api/memory/banks')}/${bankId}/entries`);
  if (!response.ok) {
    throw new Error(`Memory entries unavailable (HTTP ${response.status})`);
  }
  return response.json();
}

async function searchMemory(query: string): Promise<MemoryEntry[]> {
  const response = await fetch(`${buildUrl('JAX_API', '/api/memory/search')}?q=${encodeURIComponent(query)}`);
  if (!response.ok) {
    throw new Error(`Memory search unavailable (HTTP ${response.status})`);
  }
  return response.json();
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
