import { useQuery } from '@tanstack/react-query';
import { operatorEvidenceService } from '@/data/operator-evidence-service';

export function useOperatorEvidenceOverview() {
  return useQuery({
    queryKey: ['operator-evidence-overview'],
    queryFn: operatorEvidenceService.overview,
    refetchInterval: 30_000,
  });
}
