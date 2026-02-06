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

async function fetchWatchlist(): Promise<WatchlistItem[]> {
  // Default symbols to watch
  const symbols = ['SPY', 'QQQ', 'AAPL', 'TSLA', 'NVDA', 'AMD', 'META', 'AMZN'];
  
  // Fetch quotes from IB Bridge for each symbol
  const quotes = await Promise.allSettled(
    symbols.map(async (symbol) => {
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
      throw new Error(`Failed to fetch ${symbol}`);
    })
  );
  
  // Return successfully fetched quotes
  return quotes
    .filter((result): result is PromiseFulfilledResult<WatchlistItem> => result.status === 'fulfilled')
    .map(result => result.value);
}

export function useWatchlist() {
  return useQuery({
    queryKey: ['watchlist'],
    queryFn: fetchWatchlist,
    refetchInterval: 3000, // Refresh every 3 seconds for live prices
  });
}

export function useWatchlistSummary() {
  const { data: watchlist, ...rest } = useWatchlist();
  
  const summary = watchlist
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
