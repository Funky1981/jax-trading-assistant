/**
 * React hooks for observability metrics
 */

import { useQuery } from '@tanstack/react-query';
import { observabilityService } from '../data/observability-service';

export function useAPIHealth() {
  return useQuery({
    queryKey: ['health', 'api'],
    queryFn: () => observabilityService.getAPIHealth(),
    refetchInterval: 30000, // Refresh every 30s
    retry: 3,
  });
}

export function useMemoryHealth() {
  return useQuery({
    queryKey: ['health', 'memory'],
    queryFn: () => observabilityService.getMemoryHealth(),
    refetchInterval: 30000,
    retry: 3,
  });
}

export function useRecentMetrics(limit = 100) {
  return useQuery({
    queryKey: ['metrics', 'recent', limit],
    queryFn: () => observabilityService.getRecentMetrics(limit),
    refetchInterval: 5000, // Refresh every 5s for near real-time
  });
}

export function useRunMetrics(runId: string | null) {
  return useQuery({
    queryKey: ['metrics', 'run', runId],
    queryFn: () => runId ? observabilityService.getRunMetrics(runId) : Promise.resolve([]),
    enabled: !!runId,
  });
}
