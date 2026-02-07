import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

export interface WatchlistItem {
  symbol: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
  high: number;
  low: number;
}

// Mock data for when IB Bridge is not available
const mockWatchlistData: WatchlistItem[] = [
  { symbol: 'SPY', price: 487.32, change: 2.15, changePercent: 0.44, volume: 75234000, high: 488.12, low: 485.20 },
  { symbol: 'QQQ', price: 412.85, change: -1.23, changePercent: -0.30, volume: 42156000, high: 414.50, low: 411.90 },
  { symbol: 'AAPL', price: 185.42, change: 1.85, changePercent: 1.01, volume: 58943000, high: 186.10, low: 184.20 },
  { symbol: 'TSLA', price: 242.18, change: -3.42, changePercent: -1.39, volume: 125467000, high: 245.80, low: 241.50 },
  { symbol: 'NVDA', price: 875.28, change: 12.45, changePercent: 1.44, volume: 48923000, high: 878.90, low: 868.40 },
  { symbol: 'AMD', price: 142.67, change: -0.85, changePercent: -0.59, volume: 38215000, high: 144.20, low: 142.10 },
  { symbol: 'META', price: 478.32, change: 5.67, changePercent: 1.20, volume: 25678000, high: 480.15, low: 475.80 },
  { symbol: 'AMZN', price: 182.45, change: -1.12, changePercent: -0.61, volume: 42387000, high: 184.20, low: 181.90 },
];

async function fetchWatchlist(): Promise<WatchlistItem[]> {
  // Default symbols to watch
  const symbols = ['SPY', 'QQQ', 'AAPL', 'TSLA', 'NVDA', 'AMD', 'META', 'AMZN'];
  
  // Fetch quotes from IB Bridge for each symbol (suppress console errors)
  const quotes = await Promise.allSettled(
    symbols.map(async (symbol) => {
      try {
        const response = await fetch(buildUrl('IB_BRIDGE', `/quotes/${symbol}`));
        if (response.ok) {
          const data = await response.json();
          return {
            symbol: data.symbol,
            price: data.last || data.close || 0,
            change: data.change || 0,
            changePercent: data.change_percent || 0,
            volume: data.volume || 0,
            high: data.high || 0,
            low: data.low || 0,
          };
        }
      } catch (error) {
        // Silently fail - IB Bridge not connected
      }
      throw new Error(`Failed to fetch ${symbol}`);
    })
  );
  
  // Return successfully fetched quotes
  const successfulQuotes = quotes
    .filter((result): result is PromiseFulfilledResult<WatchlistItem> => result.status === 'fulfilled')
    .map(result => result.value);
  
  // If no quotes were successful (IB Bridge not connected), return mock data
  if (successfulQuotes.length === 0) {
    console.warn('IB Bridge not available, using mock watchlist data');
    return mockWatchlistData;
  }
  
  return successfulQuotes;
}

export function useWatchlist() {
  return useQuery({
    queryKey: ['watchlist'],
    queryFn: fetchWatchlist,
    refetchInterval: 10000, // Slow down to 10 seconds
    retry: false, // Don't retry failed requests
    staleTime: 5000, // Consider data fresh for 5 seconds
  });
}

export function useWatchlistSummary() {
  const { data: watchlist, ...rest } = useWatchlist();
  
  const summary = watchlist && watchlist.length > 0
    ? {
        count: watchlist.length,
        topMover: watchlist.reduce((best, item) => 
          Math.abs(item.changePercent) > Math.abs(best.changePercent) ? item : best
        ),
        gainers: watchlist.filter((item) => item.changePercent > 0).length,
        losers: watchlist.filter((item) => item.changePercent < 0).length,
      }
    : null;
  
  return { ...rest, data: summary, watchlist };
}

export function useAddToWatchlist() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (symbol: string) => {
      console.log('Adding to watchlist:', symbol);
      return { success: true, symbol };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] });
    },
  });
}

export function useRemoveFromWatchlist() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (symbol: string) => {
      console.log('Removing from watchlist:', symbol);
      return { success: true, symbol };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] });
    },
  });
}
