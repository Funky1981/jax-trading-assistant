import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';

export interface WatchlistItem {
  symbol: string;
  price: number;
  change: number;
  changePercent: number;
  volume: number;
  high: number;
  low: number;
}

// Mock watchlist data
const mockWatchlist: WatchlistItem[] = [
  { symbol: 'SPY', price: 512.45, change: 3.21, changePercent: 0.63, volume: 45_000_000, high: 514.20, low: 509.80 },
  { symbol: 'QQQ', price: 438.72, change: 5.18, changePercent: 1.19, volume: 32_000_000, high: 440.50, low: 433.25 },
  { symbol: 'AAPL', price: 185.92, change: -1.24, changePercent: -0.66, volume: 52_000_000, high: 188.40, low: 184.90 },
  { symbol: 'TSLA', price: 231.45, change: -8.32, changePercent: -3.47, volume: 98_000_000, high: 242.10, low: 229.50 },
  { symbol: 'NVDA', price: 721.33, change: 28.45, changePercent: 4.11, volume: 42_000_000, high: 728.90, low: 695.20 },
  { symbol: 'AMD', price: 156.78, change: 2.34, changePercent: 1.52, volume: 28_000_000, high: 158.20, low: 153.40 },
  { symbol: 'META', price: 485.23, change: 12.67, changePercent: 2.68, volume: 18_000_000, high: 489.50, low: 472.30 },
  { symbol: 'AMZN', price: 178.45, change: -0.89, changePercent: -0.50, volume: 35_000_000, high: 180.20, low: 177.10 },
];

async function fetchWatchlist(): Promise<WatchlistItem[]> {
  try {
    const response = await fetch('http://localhost:8080/api/watchlist');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  // Return mock data with price variations
  return mockWatchlist.map((item) => {
    const priceChange = item.price * (Math.random() * 0.01 - 0.005);
    const newPrice = item.price + priceChange;
    
    return {
      ...item,
      price: Number(newPrice.toFixed(2)),
      change: Number((item.change + priceChange * 0.5).toFixed(2)),
      changePercent: Number(((item.change + priceChange * 0.5) / item.price * 100).toFixed(2)),
    };
  });
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
