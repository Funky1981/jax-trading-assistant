import { useQuery } from '@tanstack/react-query';
import { systemService } from '@/data/system-service';

export function useMarketDataStatus() {
  return useQuery({
    queryKey: ['system', 'market-data-status'],
    queryFn: () => systemService.getMarketDataStatus(),
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
    staleTime: 5_000,
  });
}
