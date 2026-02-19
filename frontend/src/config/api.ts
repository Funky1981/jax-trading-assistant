/**
 * Central API configuration for all backend services
 */

// Base URLs for backend services
export const API_CONFIG = {
  // Main JAX API (handles positions, watchlist, orders, risk, etc.)
  // In dev mode, use relative URLs so Vite proxy handles them  
  JAX_API: import.meta.env.VITE_JAX_API_URL || (import.meta.env.DEV ? '' : 'http://localhost:8081'),

  // Research Service (jax-research, port 8091)
  RESEARCH_SERVICE: import.meta.env.VITE_RESEARCH_SERVICE_URL || 'http://localhost:8091',
  
  // Memory Service (Hindsight wrapper)
  MEMORY_SERVICE: import.meta.env.VITE_MEMORY_SERVICE_URL || 'http://localhost:8090',
  
  // IB Bridge (Interactive Brokers connectivity)
  IB_BRIDGE: import.meta.env.VITE_IB_BRIDGE_URL || (import.meta.env.DEV ? '' : 'http://localhost:8092'),
  
  // Agent0 AI Service (trading suggestions)
  AGENT0_SERVICE: import.meta.env.VITE_AGENT0_SERVICE_URL || 'http://localhost:8093',
  
  // Hindsight API (direct access if needed)
  HINDSIGHT: import.meta.env.VITE_HINDSIGHT_URL || 'http://localhost:8888',
} as const;

/**
 * Full URLs for health probing — each entry is the complete URL to GET.
 * In DEV, jax-trader and jax-research are routed through the Vite proxy to
 * avoid CORS. IB Bridge already sets CORS headers so it can be called directly.
 */
export const HEALTH_PROBE_URLS: Record<string, string> = import.meta.env.DEV
  ? {
      'jax-trader':   '/health',
      'jax-research': '/research-health',
      'ib-bridge':    'http://localhost:8092/health',
    }
  : {
      'jax-trader':   (import.meta.env.VITE_JAX_API_URL          || 'http://localhost:8081') + '/health',
      'jax-research': (import.meta.env.VITE_RESEARCH_SERVICE_URL  || 'http://localhost:8091') + '/health',
      'ib-bridge':    (import.meta.env.VITE_IB_BRIDGE_URL         || 'http://localhost:8092') + '/health',
    };

// API Endpoints
export const ENDPOINTS = {
  // Health & Status
  HEALTH: '/health',
  
  // JAX API endpoints
  POSITIONS: '/api/positions',
  WATCHLIST: '/api/watchlist',
  ORDERS: '/api/orders',
  STRATEGIES: '/api/strategies',
  RISK_METRICS: '/api/risk/metrics',
  METRICS_EVENTS: '/api/metrics/events',
  
  // Memory endpoints
  MEMORY_BANKS: '/api/memory/banks',
  MEMORY_SEARCH: '/api/memory/search',
  
  // Agent0 endpoints
  AI_SUGGEST: '/suggest',
  AI_CONFIG: '/config',
  AI_HEALTH: '/health',
  
  // IB Bridge endpoints
  IB_STATUS: '/status',
  IB_ACCOUNTS: '/accounts',
  IB_POSITIONS: '/positions',
  IB_MARKET_DATA: '/market-data',
} as const;

/**
 * Build full URL for an endpoint
 */
export function buildUrl(service: keyof typeof API_CONFIG, endpoint: string): string {
  return `${API_CONFIG[service]}${endpoint}`;
}

/**
 * Helper to check if we're in development mode
 */
export const isDevelopment = import.meta.env.DEV;

/**
 * Helper to check if we should use mock data (when API is unavailable)
 */
export const useMockData = import.meta.env.VITE_USE_MOCK_DATA === 'true';
