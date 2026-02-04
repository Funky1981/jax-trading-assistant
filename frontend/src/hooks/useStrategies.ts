import { useQuery } from '@tanstack/react-query';

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

// Mock strategies data
const mockStrategies: Strategy[] = [
  {
    id: 'strat-momentum',
    name: 'Momentum Alpha',
    description: 'Trend-following strategy using RSI and MACD',
    status: 'active',
    performance: {
      totalPnl: 12450.75,
      winRate: 58.3,
      trades: 127,
      sharpe: 1.85,
    },
    lastSignal: {
      symbol: 'NVDA',
      action: 'buy',
      timestamp: Date.now() - 1800000,
      confidence: 0.78,
    },
  },
  {
    id: 'strat-meanrev',
    name: 'Mean Reversion',
    description: 'Counter-trend strategy on oversold/overbought conditions',
    status: 'active',
    performance: {
      totalPnl: 8320.50,
      winRate: 62.1,
      trades: 89,
      sharpe: 1.42,
    },
    lastSignal: {
      symbol: 'TSLA',
      action: 'sell',
      timestamp: Date.now() - 3600000,
      confidence: 0.65,
    },
  },
  {
    id: 'strat-earnings',
    name: 'Earnings Gap',
    description: 'Trades earnings surprise gaps',
    status: 'paused',
    performance: {
      totalPnl: 3890.25,
      winRate: 54.5,
      trades: 44,
      sharpe: 1.12,
    },
  },
  {
    id: 'strat-volatility',
    name: 'Volatility Spike',
    description: 'Captures volatility expansion moves',
    status: 'active',
    performance: {
      totalPnl: -1250.00,
      winRate: 42.8,
      trades: 35,
      sharpe: 0.68,
    },
    lastSignal: {
      symbol: 'SPY',
      action: 'buy',
      timestamp: Date.now() - 7200000,
      confidence: 0.52,
    },
  },
];

async function fetchStrategies(): Promise<Strategy[]> {
  try {
    const response = await fetch('http://localhost:8080/api/strategies');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  return mockStrategies;
}

export function useStrategies() {
  return useQuery({
    queryKey: ['strategies'],
    queryFn: fetchStrategies,
    refetchInterval: 30000,
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
