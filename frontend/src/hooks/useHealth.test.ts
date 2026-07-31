import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { pingService } from './useHealth';

describe('research health diagnostics', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetAllMocks();
  });

  it('reports a healthy response', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ status: 'healthy', uptime: '1m' }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }),
    );

    await expect(pingService('jax-research', '/diagnostics/research/health')).resolves.toMatchObject({
      name: 'jax-research',
      status: 'healthy',
      message: '1m',
    });
  });

  it.each([
    ['unavailable service', () => Promise.reject(new TypeError('network unavailable')), 'Unreachable'],
    [
      'non-2xx response',
      () => Promise.resolve(new Response('', { status: 503 })),
      'HTTP 503',
    ],
    [
      'invalid response body',
      () =>
        Promise.resolve(
          new Response(JSON.stringify({ ready: true }), {
            status: 200,
            headers: { 'content-type': 'application/json' },
          }),
        ),
      'Invalid response',
    ],
  ])('reports an honest unavailable state for %s without rejecting', async (_caseName, response, message) => {
    fetchMock.mockImplementation(response);

    await expect(pingService('jax-research', '/diagnostics/research/health')).resolves.toMatchObject({
      name: 'jax-research',
      status: 'unhealthy',
      message,
    });
  });
});
