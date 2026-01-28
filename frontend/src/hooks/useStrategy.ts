/**
 * React hooks for strategy operations
 */

import { useQuery, useMutation } from '@tanstack/react-query';
import { strategyService } from '../data/strategy-service';

export function useStrategies() {
  return useQuery({
    queryKey: ['strategies', 'list'],
    queryFn: () => strategyService.listStrategies(),
  });
}

export function useStrategySignals(strategyId: string | null, limit = 50) {
  return useQuery({
    queryKey: ['strategies', strategyId, 'signals', limit],
    queryFn: () => strategyId ? strategyService.getSignals(strategyId, limit) : Promise.resolve([]),
    enabled: !!strategyId,
    refetchInterval: 10000, // Refresh every 10s
  });
}

export function useStrategyPerformance(strategyId: string | null) {
  return useQuery({
    queryKey: ['strategies', strategyId, 'performance'],
    queryFn: () => strategyId ? strategyService.getPerformance(strategyId) : Promise.resolve(null),
    enabled: !!strategyId,
  });
}

export function useStrategyAnalyze(strategyId: string) {
  return useMutation({
    mutationFn: ({ symbol, constraints }: { symbol: string; constraints: Record<string, unknown> }) =>
      strategyService.analyze(strategyId, symbol, constraints),
  });
}
