import { useQuery } from '@tanstack/react-query';

export interface MetricEvent {
  id: string;
  event: string;
  source: string;
  duration: number;
  timestamp: number;
  status: 'success' | 'warning' | 'error';
  metadata?: Record<string, unknown>;
}

// Mock metrics data
const mockMetrics: MetricEvent[] = [
  {
    id: 'm-001',
    event: 'order.submitted',
    source: 'jax-api',
    duration: 45,
    timestamp: Date.now() - 60000,
    status: 'success',
    metadata: { symbol: 'AAPL', side: 'buy' },
  },
  {
    id: 'm-002',
    event: 'market_data.tick',
    source: 'ib-bridge',
    duration: 12,
    timestamp: Date.now() - 120000,
    status: 'success',
    metadata: { symbols: 8 },
  },
  {
    id: 'm-003',
    event: 'strategy.signal',
    source: 'orchestrator',
    duration: 156,
    timestamp: Date.now() - 180000,
    status: 'success',
    metadata: { strategy: 'momentum', action: 'buy' },
  },
  {
    id: 'm-004',
    event: 'memory.query',
    source: 'jax-memory',
    duration: 89,
    timestamp: Date.now() - 240000,
    status: 'success',
    metadata: { bank: 'trades', results: 25 },
  },
  {
    id: 'm-005',
    event: 'risk.check',
    source: 'orchestrator',
    duration: 234,
    timestamp: Date.now() - 300000,
    status: 'warning',
    metadata: { exposure: 0.75, limit: 0.80 },
  },
  {
    id: 'm-006',
    event: 'order.rejected',
    source: 'ib-bridge',
    duration: 32,
    timestamp: Date.now() - 360000,
    status: 'error',
    metadata: { reason: 'insufficient_margin' },
  },
  {
    id: 'm-007',
    event: 'health.check',
    source: 'jax-api',
    duration: 18,
    timestamp: Date.now() - 420000,
    status: 'success',
  },
  {
    id: 'm-008',
    event: 'position.updated',
    source: 'jax-api',
    duration: 67,
    timestamp: Date.now() - 480000,
    status: 'success',
    metadata: { symbol: 'NVDA', pnl: 1250.50 },
  },
];

async function fetchMetrics(): Promise<MetricEvent[]> {
  try {
    const response = await fetch('http://localhost:8080/api/metrics/events');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available
  }
  
  // Return mock data with updated timestamps
  const now = Date.now();
  return mockMetrics.map((metric, index) => ({
    ...metric,
    timestamp: now - (index + 1) * 60000,
  }));
}

export function useMetrics() {
  return useQuery({
    queryKey: ['metrics'],
    queryFn: fetchMetrics,
    refetchInterval: 10000,
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
