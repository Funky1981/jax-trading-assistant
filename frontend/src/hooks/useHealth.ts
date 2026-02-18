import { useQuery } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

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

async function fetchHealth(): Promise<HealthData> {
  const response = await fetch(buildUrl('JAX_API', '/health'));
  if (!response.ok) {
    throw new Error(`Health endpoint returned HTTP ${response.status}`);
  }
  const apiResponse = await response.json();
  if (apiResponse.services && Array.isArray(apiResponse.services)) {
    return apiResponse as HealthData;
  }
  // API is reachable but doesn't expose per-service breakdown yet
  throw new Error('Health endpoint does not return per-service data');
}

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: fetchHealth,
    refetchInterval: 30000, // Poll every 30 seconds instead of 10
    retry: false, // Don't retry
    staleTime: 15000, // Consider fresh for 15 seconds
  });
}

export function useServiceHealth(serviceName: string) {
  const { data, ...rest } = useHealth();
  
  return {
    ...rest,
    data: data?.services.find((s) => s.name === serviceName),
  };
}
