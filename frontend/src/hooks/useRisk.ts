import { useQuery } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';
import { useIBAccount } from './useIBAccount';
import { usePositionsSummary } from './usePositions';

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

async function fetchRiskMetrics(): Promise<RiskMetrics> {
  // JAX API not currently available - will throw error
  const response = await fetch(buildUrl('JAX_API', '/api/risk/metrics'));
  if (!response.ok) {
    throw new Error('Risk metrics service unavailable');
  }
  return response.json();
}

export function useRiskMetrics() {
  return useQuery({
    queryKey: ['risk', 'metrics'],
    queryFn: fetchRiskMetrics,
    refetchInterval: 10000,
    retry: false, // Don't retry since JAX API is not available
  });
}

// Hook to calculate risk metrics from real IB data
export function useIBRiskMetrics() {
  const { data: account, isError: accountError } = useIBAccount();
  const { data: positionSummary, positions, isError: positionsError } = usePositionsSummary();
  
  const isError = accountError || positionsError;
  const isLoading = !account || !positionSummary;

  const metrics: RiskMetrics | null = account && positionSummary ? {
    exposure: {
      current: positionSummary.totalValue,
      limit: account.net_liquidation,
      utilizationPercent: account.net_liquidation > 0 
        ? (positionSummary.totalValue / account.net_liquidation) * 100 
        : 0,
    },
    dailyPnl: {
      current: positionSummary.totalPnl,
      limit: account.net_liquidation * 0.05, // 5% of account as daily limit
      utilizationPercent: account.net_liquidation > 0
        ? (Math.abs(positionSummary.totalPnl) / (account.net_liquidation * 0.05)) * 100
        : 0,
    },
    drawdown: {
      current: 0, // Requires historical data not available yet
      limit: 10.0,
      utilizationPercent: 0,
    },
    positionCount: {
      current: positionSummary.positionCount,
      limit: 10, // Hard-coded limit, should come from config
    },
    largestPosition: positions && positions.length > 0
      ? (() => {
          const largest = [...positions].sort((a, b) => b.marketValue - a.marketValue)[0];
          return {
            symbol: largest.symbol,
            value: largest.marketValue,
            percentOfPortfolio: account.net_liquidation > 0 
              ? (largest.marketValue / account.net_liquidation) * 100 
              : 0,
          };
        })()
      : {
          symbol: '-',
          value: 0,
          percentOfPortfolio: 0,
        },
    sectorExposure: [], // Requires sector mapping not available yet
  } : null;

  return {
    data: metrics,
    isLoading,
    isError,
  };
}

export function useRiskSummary() {
  // Try to use IB-derived risk metrics (real data)
  const { data: ibMetrics, isLoading: ibLoading, isError: ibError } = useIBRiskMetrics();
  
  // Fall back to JAX API risk metrics only if explicitly needed
  // const { data: jaxMetrics, isLoading: jaxLoading, isError: jaxError } = useRiskMetrics();
  
  const metrics = ibMetrics; // Use IB metrics only
  const isLoading = ibLoading;
  const isError = ibError;
  
  const summary = metrics
    ? {
        exposure: metrics.exposure.current,
        utilizationPercent: metrics.exposure.utilizationPercent,
        dailyPnl: metrics.dailyPnl.current,
        drawdown: metrics.drawdown.current,
        isAtRisk: metrics.exposure.utilizationPercent > 80 || metrics.drawdown.utilizationPercent > 70,
      }
    : null;
  
  return { data: summary, metrics, isLoading, isError };
}
