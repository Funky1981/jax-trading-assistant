import { describe, expect, it, vi } from 'vitest';
import { emitAnalyticsEvent } from './analytics';

describe('analytics abstraction', () => {
  it('uses configured transport when available', () => {
    const track = vi.fn();
    window.__JAX_ANALYTICS__ = { track };

    emitAnalyticsEvent('page_viewed', { source_surface: 'home' });

    expect(track).toHaveBeenCalledWith('page_viewed', { source_surface: 'home' });
  });

  it('does not throw when no transport is configured', () => {
    delete window.__JAX_ANALYTICS__;

    expect(() => emitAnalyticsEvent('ai_scanner_enabled', { source_surface: 'ai_trading' })).not.toThrow();
  });
});
