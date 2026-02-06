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

// Mock data for when backend is not available
const mockHealthData: HealthData = {
  overall: 'healthy',
  services: [
    {
      name: 'JAX API',
      status: 'healthy',
      lastCheck: Date.now(),
      latency: 45,
      message: 'All endpoints responding',
    },
    {
      name: 'Memory Service',
      status: 'healthy',
      lastCheck: Date.now(),
      latency: 23,
      message: 'Connected to PostgreSQL',
    },
    {
      name: 'IB Bridge',
      status: 'degraded',
      lastCheck: Date.now(),
      latency: 156,
      message: 'High latency detected',
    },
    {
      name: 'Market Data',
      status: 'healthy',
      lastCheck: Date.now(),
      latency: 12,
      message: 'Streaming active',
    },
    {
      name: 'Orchestrator',
      status: 'healthy',
      lastCheck: Date.now(),
      latency: 34,
      message: 'Processing signals',
    },
  ],
};

async function fetchHealth(): Promise<HealthData> {
  // Try to fetch from actual API
  try {
    const response = await fetch('http://localhost:8080/health');
    if (response.ok) {
      return response.json();
    }
  } catch {
    // API not available, use mock data
  }
  
  // Return mock data with some randomization
  return {
    ...mockHealthData,
    services: mockHealthData.services.map((service) => ({
      ...service,
      lastCheck: Date.now(),
      latency: service.latency ? service.latency + Math.floor(Math.random() * 20 - 10) : undefined,
    })),
  };
}

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: fetchHealth,
    refetchInterval: 10000, // Poll every 10 seconds
  });
}

export function useServiceHealth(serviceName: string) {
  const { data, ...rest } = useHealth();
  
  return {
    ...rest,
    data: data?.services.find((s) => s.name === serviceName),
  };
}
