import { useQuery } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

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

async function fetchStrategies(): Promise<Strategy[]> {
  const response = await fetch(buildUrl('JAX_API', '/api/strategies'));
  if (!response.ok) {
    throw new Error('Strategies service unavailable');
  }
  return response.json();
}

export function useStrategies() {
  return useQuery({
    queryKey: ['strategies'],
    queryFn: fetchStrategies,
    refetchInterval: 30000,
    retry: false, // Don't retry since JAX API is not available
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
