import { describe, expect, it } from 'vitest';
import { apiClient, memoryClient } from './http-client';

describe('dev API clients', () => {
  it('resolves JAX API calls against the Vite origin in dev mode', () => {
    expect(apiClient.buildUrl('/api/v1/e2e/results')).toBe(`${window.location.origin}/api/v1/e2e/results`);
  });

  it('resolves memory calls against the Vite origin in dev mode', () => {
    expect(memoryClient.buildUrl('/v1/memory/banks')).toBe(`${window.location.origin}/v1/memory/banks`);
  });
});
