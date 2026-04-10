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
    expect(url.pathname).toBe('/v1/memory/search');
    expect(url.searchParams.get('bank')).toBe('trades');
    expect(url.searchParams.get('q')).toBe('AAPL');
    expect(url.searchParams.get('limit')).toBe('5');
  });
});
