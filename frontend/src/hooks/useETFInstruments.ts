import { useQuery } from '@tanstack/react-query';
import { systemService } from '@/data/system-service';

export function useETFInstruments() {
  return useQuery({
    queryKey: ['instruments', 'etfs'],
    queryFn: () => systemService.getETFInstruments(),
    retry: false,
    staleTime: 60_000,
  });
}
