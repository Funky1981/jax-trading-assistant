import { describe, expect, it } from 'vitest';
import { buildUrl, HEALTH_PROBE_URLS } from './api';

describe('dev API config', () => {
  it('uses Vite proxy paths for browser-called service URLs', () => {
    expect(buildUrl('JAX_API', '/api/v1/signals')).toBe('/api/v1/signals');
    expect(buildUrl('MEMORY_SERVICE', '/v1/memory/banks')).toBe('/v1/memory/banks');
    expect(buildUrl('IB_BRIDGE', '/quotes/SPY')).toBe('/quotes/SPY');
    expect(buildUrl('AGENT0_SERVICE', '/suggest')).toBe('/agent0/suggest');
  });

  it('uses proxied health endpoints in dev mode', () => {
    expect(HEALTH_PROBE_URLS).toEqual({
      'jax-trader': '/health',
      'jax-research': '/research-health',
      'ib-bridge': '/ib-health',
    });
  });
});
