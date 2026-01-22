import { describe, expect, it, vi } from 'vitest';
import { createStreamBuffer } from '../stream-buffer';

describe('stream buffer', () => {
  it('flushes latest items on schedule', () => {
    const onFlush = vi.fn();
    vi.useFakeTimers();

    const buffer = createStreamBuffer({
      flushIntervalMs: 100,
      onFlush,
      getKey: (item: { symbol: string }) => item.symbol,
    });

    buffer.push({ symbol: 'AAPL' });
    buffer.push({ symbol: 'AAPL' });

    vi.advanceTimersByTime(100);

    expect(onFlush).toHaveBeenCalledTimes(1);
    expect(onFlush).toHaveBeenCalledWith([{ symbol: 'AAPL' }]);

    buffer.stop();
    vi.useRealTimers();
  });

  it('flushes immediately when requested', () => {
    const onFlush = vi.fn();
    const buffer = createStreamBuffer({
      flushIntervalMs: 100,
      onFlush,
      getKey: (item: { symbol: string }) => item.symbol,
    });

    buffer.push({ symbol: 'MSFT' });
    buffer.flush();

    expect(onFlush).toHaveBeenCalledWith([{ symbol: 'MSFT' }]);
  });
});
