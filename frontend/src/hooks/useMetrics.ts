import { useQuery } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

export interface MetricEvent {
  id: string;
  event: string;
  source: string;
  duration: number;
  timestamp: number;
  status: 'success' | 'warning' | 'error';
  metadata?: Record<string, unknown>;
}

async function fetchMetrics(): Promise<MetricEvent[]> {
  const response = await fetch(buildUrl('JAX_API', '/api/v1/metrics'));
  if (!response.ok) {
    throw new Error('Metrics service unavailable');
  }
  return response.json();
}

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: fetchMetrics,
    refetchInterval: 10000,
    retry: false, // Don't retry since JAX API is not available
  });
}

export function useMetricsSummary() {
  const { data: metrics, ...rest } = useMetrics();
  
  const summary = metrics
    ? {
        total: metrics.length,
        lastEvent: metrics[0],
        successCount: metrics.filter((m) => m.status === 'success').length,
        warningCount: metrics.filter((m) => m.status === 'warning').length,
        errorCount: metrics.filter((m) => m.status === 'error').length,
        avgDuration: metrics.reduce((sum, m) => sum + m.duration, 0) / metrics.length,
      }
    : null;
  
  return { ...rest, data: summary, metrics };
}
