import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { memoryService } from './memory-service';

describe('memoryService', () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.resetAllMocks();
  });

  it('requires a bank for search', async () => {
    await expect(memoryService.search('', 'AAPL')).rejects.toThrow('bank is required');
  });

  it('calls the bank-scoped memory search endpoint', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }),
    );

    await memoryService.search('trades', 'AAPL', 5);

    const url = new URL(fetchMock.mock.calls[0][0] as string);
    expect(url.pathname).toBe('/diagnostics/research/v1/memory/search');
    expect(url.searchParams.get('bank')).toBe('trades');
    expect(url.searchParams.get('q')).toBe('AAPL');
    expect(url.searchParams.get('limit')).toBe('5');
  });

  it('returns a valid memory-bank diagnostic response from the same origin', async () => {
    fetchMock.mockResolvedValue(
      new Response(JSON.stringify(['research', 'trades']), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }),
    );

    await expect(memoryService.listBanks()).resolves.toEqual(['research', 'trades']);
    expect(new URL(fetchMock.mock.calls[0][0] as string).pathname).toBe(
      '/diagnostics/research/v1/memory/banks',
    );
  });

  it.each([
    ['network unavailable', () => Promise.reject(new TypeError('network unavailable'))],
    [
      'non-2xx response',
      () =>
        Promise.resolve(
          new Response(JSON.stringify({ error: 'unavailable' }), {
            status: 503,
            headers: { 'content-type': 'application/json' },
          }),
        ),
    ],
    [
      'invalid response body',
      () =>
        Promise.resolve(
          new Response(JSON.stringify({ banks: ['research'] }), {
            status: 200,
            headers: { 'content-type': 'application/json' },
          }),
        ),
    ],
  ])('rejects an honest unavailable state for %s', async (_caseName, response) => {
    fetchMock.mockImplementation(response);
    await expect(memoryService.listBanks()).rejects.toThrow();
  });
});
