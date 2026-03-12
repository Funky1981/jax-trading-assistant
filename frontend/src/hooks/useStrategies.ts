import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/data/http-client';

export interface Strategy {
  id: string;
  name: string;
  description: string;
  status: 'active' | 'paused' | 'disabled';
  performance: {
    totalPnl: number;
    winRate: number;
    trades: number;
    sharpe: number;
  };
  lastSignal?: {
    symbol: string;
    action: 'buy' | 'sell';
    timestamp: number;
    confidence: number;
  };
}

function toNumber(value: unknown, fallback = 0): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback;
}

function toStatus(value: unknown): Strategy['status'] {
  return value === 'active' || value === 'paused' || value === 'disabled'
    ? value
    : 'disabled';
}

function mapStrategy(raw: unknown): Strategy {
  const source = raw && typeof raw === 'object' ? raw as Record<string, unknown> : {};
  const performanceSource =
    source.performance && typeof source.performance === 'object'
      ? source.performance as Record<string, unknown>
      : {};
  const lastSignalSource =
    source.lastSignal && typeof source.lastSignal === 'object'
      ? source.lastSignal as Record<string, unknown>
      : source.last_signal && typeof source.last_signal === 'object'
        ? source.last_signal as Record<string, unknown>
        : null;

  return {
    id: typeof source.id === 'string' ? source.id : 'unknown-strategy',
    name: typeof source.name === 'string' ? source.name : 'Unnamed strategy',
    description: typeof source.description === 'string' ? source.description : '',
    status: toStatus(source.status),
    performance: {
      totalPnl: toNumber(performanceSource.totalPnl ?? performanceSource.total_pnl),
      winRate: toNumber(performanceSource.winRate ?? performanceSource.win_rate),
      trades: toNumber(performanceSource.trades),
      sharpe: toNumber(performanceSource.sharpe),
    },
    lastSignal: lastSignalSource
      ? {
          symbol: typeof lastSignalSource.symbol === 'string' ? lastSignalSource.symbol : '',
          action: lastSignalSource.action === 'sell' ? 'sell' : 'buy',
          timestamp: toNumber(lastSignalSource.timestamp ?? lastSignalSource.generated_at ?? lastSignalSource.created_at, Date.now()),
          confidence: toNumber(lastSignalSource.confidence),
        }
      : undefined,
  };
}

async function fetchStrategies(): Promise<Strategy[]> {
  const data = await apiClient.get<{ strategies?: Strategy[] } | Strategy[]>('/api/v1/strategies');
  // API returns { strategies: [...] } envelope
  const raw = Array.isArray(data) ? data : (data.strategies ?? []);
  return raw.map(mapStrategy);
}

export function useStrategies() {
  return useQuery({
    queryKey: ['strategies'],
    queryFn: fetchStrategies,
    refetchInterval: (query) => (query.state.error ? false : 30_000),
    retry: false,
  });
}

export function useStrategiesSummary() {
  const { data: strategies, ...rest } = useStrategies();
  
  const summary = strategies
    ? {
        total: strategies.length,
        active: strategies.filter((s) => s.status === 'active').length,
        totalPnl: strategies.reduce((sum, s) => sum + s.performance.totalPnl, 0),
        recentSignal: strategies
          .filter((s) => s.lastSignal)
          .sort((a, b) => (b.lastSignal?.timestamp ?? 0) - (a.lastSignal?.timestamp ?? 0))[0]?.lastSignal,
      }
    : null;
  
  return { ...rest, data: summary, strategies };
}
