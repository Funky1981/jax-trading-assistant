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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mapApiMetric(raw: any): MetricEvent {
  const startTs = raw.started_at ? new Date(raw.started_at).getTime() : (raw.timestamp ?? Date.now());
  const endTs = raw.completed_at ? new Date(raw.completed_at).getTime() : startTs;
  const statusMap: Record<string, MetricEvent['status']> = {
    completed: 'success',
    failed: 'error',
    running: 'warning',
  };
  return {
    id: raw.id ?? crypto.randomUUID(),
    event: raw.event ?? (raw.symbol ? `analysis:${raw.symbol}` : 'metric'),
    source: raw.source ?? raw.symbol ?? 'system',
    duration: raw.duration ?? Math.max(0, endTs - startTs),
    timestamp: raw.timestamp ?? startTs,
    status: raw.status in statusMap ? statusMap[raw.status] : (raw.status ?? 'success'),
    metadata: raw.metadata ?? { confidence: raw.confidence, symbol: raw.symbol },
  };
}

async function fetchMetrics(): Promise<MetricEvent[]> {
  const response = await fetch(buildUrl('JAX_API', '/api/v1/metrics'));
  if (!response.ok) {
    throw new Error('Metrics service unavailable');
  }
  const data = await response.json();
  // API returns { metrics: [...] } envelope with snake_case / different field names
  const raw = Array.isArray(data) ? data : (data.metrics ?? []);
  return raw.map(mapApiMetric);
}

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: fetchMetrics,
    refetchInterval: (query) => (query.state.error ? false : 10_000),
    retry: false,
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
