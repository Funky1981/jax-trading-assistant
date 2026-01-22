export interface StreamBufferOptions<T> {
  flushIntervalMs: number;
  onFlush: (items: T[]) => void;
  getKey: (item: T) => string;
}

export function createStreamBuffer<T>(options: StreamBufferOptions<T>) {
  const pending = new Map<string, T>();
  let timer: ReturnType<typeof setTimeout> | null = null;

  const scheduleFlush = () => {
    if (timer) return;
    timer = setTimeout(() => {
      timer = null;
      flush();
    }, options.flushIntervalMs);
  };

  const push = (item: T) => {
    pending.set(options.getKey(item), item);
    scheduleFlush();
  };

  const flush = () => {
    if (pending.size === 0) return;
    const items = Array.from(pending.values());
    pending.clear();
    options.onFlush(items);
  };

  const stop = () => {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
  };

  return { push, flush, stop };
}
