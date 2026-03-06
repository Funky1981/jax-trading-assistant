import { useQuery } from '@tanstack/react-query';
import { systemService } from '@/data/system-service';

export function useTradingPilotStatus() {
  return useQuery({
    queryKey: ['trading', 'pilot-status'],
    queryFn: () => systemService.getTradingPilotStatus(),
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
    staleTime: 5_000,
  });
}
