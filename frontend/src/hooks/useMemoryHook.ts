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

// Mock memory banks
const mockBanks: MemoryBank[] = [
  {
    id: 'bank-trades',
    name: 'Trade History',
    description: 'Historical trade records and outcomes',
    entryCount: 1247,
    lastUpdated: Date.now() - 300000,
  },
  {
    id: 'bank-signals',
    name: 'Strategy Signals',
    description: 'Generated trading signals and their results',
    entryCount: 892,
    lastUpdated: Date.now() - 600000,
  },
  {
    id: 'bank-market',
    name: 'Market Context',
    description: 'Market conditions and sentiment analysis',
    entryCount: 2341,
    lastUpdated: Date.now() - 120000,
  },
  {
    id: 'bank-risk',
    name: 'Risk Events',
    description: 'Risk alerts and portfolio adjustments',
    entryCount: 156,
    lastUpdated: Date.now() - 3600000,
  },
];

// Mock memory entries
const mockEntries: MemoryEntry[] = [
  {
    id: 'entry-001',
    bankId: 'bank-trades',
    content: 'Bought 100 AAPL at $184.50. Momentum indicator triggered.',
    metadata: { symbol: 'AAPL', action: 'buy', price: 184.50 },
    createdAt: Date.now() - 3600000,
  },
  {
    id: 'entry-002',
    bankId: 'bank-signals',
    content: 'Bearish divergence detected on TSLA daily chart.',
    metadata: { symbol: 'TSLA', signal: 'bearish', timeframe: '1d' },
    createdAt: Date.now() - 7200000,
  },
  {
    id: 'entry-003',
    bankId: 'bank-market',
    content: 'VIX elevated at 18.5. Market showing increased volatility.',
    metadata: { indicator: 'VIX', value: 18.5, sentiment: 'cautious' },
    createdAt: Date.now() - 1800000,
  },
];

async function fetchMemoryBanks(): Promise<MemoryBank[]> {
  try {
    const response = await fetch(buildUrl('JAX_API', '/api/memory/banks'));
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  return mockBanks;
}

async function fetchMemoryEntries(bankId: string): Promise<MemoryEntry[]> {
  try {
    const response = await fetch(`${buildUrl('JAX_API', '/api/memory/banks')}/${bankId}/entries`);
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  return mockEntries.filter((e) => e.bankId === bankId);
}

async function searchMemory(query: string): Promise<MemoryEntry[]> {
  try {
    const response = await fetch(`${buildUrl('JAX_API', '/api/memory/search')}?q=${encodeURIComponent(query)}`);
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  // Simple mock search
  const lowerQuery = query.toLowerCase();
  return mockEntries.filter(
    (e) =>
      e.content.toLowerCase().includes(lowerQuery) ||
      JSON.stringify(e.metadata).toLowerCase().includes(lowerQuery)
  );
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
