import { useQuery } from '@tanstack/react-query';
import { HEALTH_PROBE_URLS } from '@/config/api';

export interface ServiceHealth {
  name: string;
  status: 'healthy' | 'degraded' | 'unhealthy';
  lastCheck: number;
  latency?: number;
  message?: string;
}

export interface HealthData {
  services: ServiceHealth[];
  overall: 'healthy' | 'degraded' | 'unhealthy';
}

const SERVICES = Object.entries(HEALTH_PROBE_URLS).map(([name, url]) => ({ name, url }));

async function pingService(name: string, url: string): Promise<ServiceHealth> {
  const start = Date.now();
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(4000) });
    const latency = Date.now() - start;
    if (!res.ok) {
      return { name, status: 'unhealthy', lastCheck: Date.now(), latency, message: `HTTP ${res.status}` };
    }
    const body = await res.json().catch(() => ({}));
    const rawStatus = (body.status ?? 'healthy') as string;
    const status: ServiceHealth['status'] =
      rawStatus === 'healthy' ? 'healthy'
      : rawStatus === 'degraded' ? 'degraded'
      : 'unhealthy';
    return { name, status, lastCheck: Date.now(), latency, message: body.uptime };
  } catch {
    return { name, status: 'unhealthy', lastCheck: Date.now(), message: 'Unreachable' };
  }
}

async function fetchHealth(): Promise<HealthData> {
  const services = await Promise.all(
    SERVICES.map((s) => pingService(s.name, s.url))
  );
  const overall: HealthData['overall'] =
    services.every((s) => s.status === 'healthy') ? 'healthy'
    : services.some((s) => s.status === 'healthy') ? 'degraded'
    : 'unhealthy';
  return { services, overall };
}

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: fetchHealth,
    refetchInterval: 30_000,
    retry: false,
    staleTime: 15_000,
  });
}

export function useServiceHealth(serviceName: string) {
  const { data, ...rest } = useHealth();
  return {
    ...rest,
    data: data?.services.find((s) => s.name === serviceName),
  };
}

