/**
 * React hooks for orchestration operations
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { orchestrationService } from '../data/orchestration-service';
import type { OrchestrationRequest } from '../data/types';

export function useOrchestrationRun() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (request: OrchestrationRequest) => orchestrationService.run(request),
    onSuccess: (data: OrchestrationResult) => {
      // Invalidate runs list
      queryClient.invalidateQueries({ queryKey: ['orchestration', 'runs'] });
      
      // If we got a runId, prefetch its status
      if (data.runId) {
        queryClient.setQueryData(['orchestration', 'run', data.runId], data);
      }
    },
  });
}

export function useOrchestrationRunStatus(runId: string | null) {
  return useQuery({
    queryKey: ['orchestration', 'run', runId],
    queryFn: () => runId ? orchestrationService.getRunStatus(runId) : Promise.resolve(null),
    enabled: !!runId,
    refetchInterval: (query) => {
      const data = query?.state?.data;
      // Stop polling if status is terminal
      if (data?.status === 'completed' || data?.status === 'failed') {
        return false;
      }
      return 2000; // Poll every 2s while running
    },
  });
}

export function useOrchestrationRuns(limit = 20) {
  return useQuery({
    queryKey: ['orchestration', 'runs', limit],
    queryFn: () => orchestrationService.listRuns(limit),
    refetchInterval: 10000, // Refresh every 10s
  });
}
