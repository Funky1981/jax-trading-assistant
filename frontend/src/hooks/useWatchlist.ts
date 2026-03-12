import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

export interface WatchlistItem {
  symbol: string;
  price: number;
  change: number | null;
  changePercent: number | null;
  volume: number;
  high: number;
  low: number;
}

const WATCHLIST_STORAGE_KEY = 'jax_watchlist_symbols';
const DEFAULT_SYMBOLS = ['SPY', 'QQQ', 'AAPL', 'TSLA', 'NVDA', 'AMD', 'META', 'AMZN'];

function loadWatchlistSymbols(): string[] {
  try {
    const raw = localStorage.getItem(WATCHLIST_STORAGE_KEY);
    if (!raw) return DEFAULT_SYMBOLS;
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return DEFAULT_SYMBOLS;
    const symbols = parsed
      .map((value) => String(value).trim().toUpperCase())
      .filter((value) => /^[A-Z.-]{1,10}$/.test(value));
    return symbols.length > 0 ? Array.from(new Set(symbols)) : DEFAULT_SYMBOLS;
  } catch {
    return DEFAULT_SYMBOLS;
  }
}

function saveWatchlistSymbols(symbols: string[]) {
  localStorage.setItem(WATCHLIST_STORAGE_KEY, JSON.stringify(symbols));
}

async function fetchWatchlist(): Promise<WatchlistItem[]> {
  const symbols = loadWatchlistSymbols();

  // Fetch quotes from IB Bridge for each symbol in parallel
  const results = await Promise.allSettled(
    symbols.map(async (symbol) => {
      const response = await fetch(buildUrl('IB_BRIDGE', `/quotes/${symbol}`));
      if (!response.ok) throw new Error(`HTTP ${response.status} for ${symbol}`);
      const data = await response.json();
      return {
        symbol: data.symbol ?? symbol,
        price: data.price ?? data.last ?? 0,
        change: data.change ?? null,
        changePercent: data.change_percent ?? null,
        volume: data.volume ?? 0,
        high: data.high ?? 0,
        low: data.low ?? 0,
      } as WatchlistItem;
    })
  );

  const successful = results
    .filter((r): r is PromiseFulfilledResult<WatchlistItem> => r.status === 'fulfilled')
    .map((r) => r.value)
    .filter((item) => item.price > 0); // reject zero-price responses

  if (successful.length === 0) {
    throw new Error('IB Bridge returned no valid quotes — check IB Gateway connection');
  }

  return successful;
}

export function useWatchlist() {
  return useQuery({
    queryKey: ['watchlist'],
    queryFn: fetchWatchlist,
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
    refetchOnWindowFocus: false,
    staleTime: 5000,
  });
}

export function useWatchlistSummary() {
  const { data: watchlist, ...rest } = useWatchlist();
  const validMovers = (watchlist ?? []).filter(
    (item): item is WatchlistItem & { changePercent: number } => item.changePercent !== null
  );
  
  const summary = watchlist && watchlist.length > 0
    ? {
        count: watchlist.length,
        topMover: validMovers.length > 0
          ? validMovers.reduce((best, item) => 
          Math.abs(item.changePercent) > Math.abs(best.changePercent) ? item : best
        )
          : null,
        gainers: validMovers.filter((item) => item.changePercent > 0).length,
        losers: validMovers.filter((item) => item.changePercent < 0).length,
      }
    : null;
  
  return { ...rest, data: summary, watchlist };
}

export function useAddToWatchlist() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (symbol: string) => {
      const normalized = symbol.trim().toUpperCase();
      if (!normalized) {
        throw new Error('Enter a symbol to add.');
      }
      if (!/^[A-Z.-]{1,10}$/.test(normalized)) {
        throw new Error('Enter a valid ticker symbol.');
      }
      const symbols = loadWatchlistSymbols();
      if (symbols.includes(normalized)) {
        throw new Error(`${normalized} is already in the watchlist.`);
      }
      saveWatchlistSymbols([...symbols, normalized]);
      return { success: true, symbol: normalized };
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
      const normalized = symbol.trim().toUpperCase();
      const symbols = loadWatchlistSymbols();
      saveWatchlistSymbols(symbols.filter((item) => item !== normalized));
      return { success: true, symbol: normalized };
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['watchlist'] });
    },
  });
}
