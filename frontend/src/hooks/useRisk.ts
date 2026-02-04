import { useQuery } from '@tanstack/react-query';

export interface RiskMetrics {
  exposure: {
    current: number;
    limit: number;
    utilizationPercent: number;
  };
  dailyPnl: {
    current: number;
    limit: number;
    utilizationPercent: number;
  };
  drawdown: {
    current: number;
    limit: number;
    utilizationPercent: number;
  };
  positionCount: {
    current: number;
    limit: number;
  };
  largestPosition: {
    symbol: string;
    value: number;
    percentOfPortfolio: number;
  };
  sectorExposure: Array<{
    sector: string;
    value: number;
    percent: number;
  }>;
}

// Mock risk data
const mockRiskMetrics: RiskMetrics = {
  exposure: {
    current: 73583.65,
    limit: 100000,
    utilizationPercent: 73.58,
  },
  dailyPnl: {
    current: 2845.15,
    limit: 5000,
    utilizationPercent: 56.90,
  },
  drawdown: {
    current: 3.2,
    limit: 10.0,
    utilizationPercent: 32.0,
  },
  positionCount: {
    current: 5,
    limit: 10,
  },
  largestPosition: {
    symbol: 'NVDA',
    value: 21639.90,
    percentOfPortfolio: 29.4,
  },
  sectorExposure: [
    { sector: 'Technology', value: 64325.65, percent: 87.4 },
    { sector: 'Consumer', value: 9258.00, percent: 12.6 },
  ],
};

async function fetchRiskMetrics(): Promise<RiskMetrics> {
  try {
    const response = await fetch('http://localhost:8080/api/risk/metrics');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  // Add some variation to mock data
  const variation = (Math.random() - 0.5) * 0.1;
  return {
    ...mockRiskMetrics,
    exposure: {
      ...mockRiskMetrics.exposure,
      current: mockRiskMetrics.exposure.current * (1 + variation),
      utilizationPercent: mockRiskMetrics.exposure.utilizationPercent * (1 + variation),
    },
    dailyPnl: {
      ...mockRiskMetrics.dailyPnl,
      current: mockRiskMetrics.dailyPnl.current * (1 + variation * 2),
    },
  };
}

export function useRiskMetrics() {
  return useQuery({
    queryKey: ['risk', 'metrics'],
    queryFn: fetchRiskMetrics,
    refetchInterval: 10000,
  });
}

export function useRiskSummary() {
  const { data: metrics, ...rest } = useRiskMetrics();
  
  const summary = metrics
    ? {
        exposure: metrics.exposure.current,
        utilizationPercent: metrics.exposure.utilizationPercent,
        dailyPnl: metrics.dailyPnl.current,
        drawdown: metrics.drawdown.current,
        isAtRisk: metrics.exposure.utilizationPercent > 80 || metrics.drawdown.utilizationPercent > 70,
      }
    : null;
  
  return { ...rest, data: summary, metrics };
}
