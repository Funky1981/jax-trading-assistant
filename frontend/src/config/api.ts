/**
 * Central API configuration for all backend services.
 */

function devProxyBaseUrl(envUrl: string | undefined, productionFallback: string, devProxyPath = '') {
  return import.meta.env.DEV ? devProxyPath : envUrl || productionFallback;
}

export const API_CONFIG = {
  JAX_API: devProxyBaseUrl(import.meta.env.VITE_JAX_API_URL, 'http://localhost:8081'),
  RESEARCH_SERVICE: devProxyBaseUrl(import.meta.env.VITE_RESEARCH_SERVICE_URL, 'http://localhost:8091'),
  MEMORY_SERVICE: devProxyBaseUrl(import.meta.env.VITE_MEMORY_API_URL, 'http://localhost:8091'),
  IB_BRIDGE: devProxyBaseUrl(import.meta.env.VITE_IB_BRIDGE_URL, 'http://localhost:8092'),
  AGENT0_SERVICE: devProxyBaseUrl(import.meta.env.VITE_AGENT0_SERVICE_URL, 'http://localhost:8093', '/agent0'),
} as const;

export const HEALTH_PROBE_URLS: Record<string, string> = import.meta.env.DEV
  ? {
      'jax-trader': '/health',
      'jax-research': '/research-health',
      'ib-bridge': '/ib-health',
    }
  : {
      'jax-trader': `${import.meta.env.VITE_JAX_API_URL || 'http://localhost:8081'}/health`,
      'jax-research': `${import.meta.env.VITE_RESEARCH_SERVICE_URL || 'http://localhost:8091'}/health`,
      'ib-bridge': `${import.meta.env.VITE_IB_BRIDGE_URL || 'http://localhost:8092'}/health`,
    };

export const ENDPOINTS = {
  HEALTH: '/health',
  POSITIONS: '/api/positions',
  WATCHLIST: '/api/watchlist',
  ORDERS: '/api/orders',
  STRATEGIES: '/api/strategies',
  RISK_METRICS: '/api/risk/metrics',
  METRICS_EVENTS: '/api/metrics/events',
  AI_SUGGEST: '/suggest',
  AI_CONFIG: '/config',
  AI_HEALTH: '/health',
  IB_STATUS: '/status',
  IB_ACCOUNTS: '/accounts',
  IB_POSITIONS: '/positions',
  IB_MARKET_DATA: '/market-data',
} as const;

export function buildUrl(service: keyof typeof API_CONFIG, endpoint: string): string {
  return `${API_CONFIG[service]}${endpoint}`;
}

export const isDevelopment = import.meta.env.DEV;
