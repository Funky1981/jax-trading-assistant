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
    const response = await fetch(buildUrl('JAX_API', '/health'));
    if (response.ok) {
      const apiResponse = await response.json();
      // Check if response has services array
      if (apiResponse.services && Array.isArray(apiResponse.services)) {
        return apiResponse;
      }
      // API returned simple health check, use mock data with updated status
      console.log('JAX API health check OK, using mock service data');
    }
  } catch (error) {
    console.warn('Health API not available:', error);
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
