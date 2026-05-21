import { afterEach, describe, expect, it, vi } from 'vitest';
import { getBeginnerMode, setBeginnerMode } from './beginner-helpers';

describe('beginner-helpers storage safety', () => {
  afterEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it('returns simple mode when localStorage throws', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('blocked');
    });

    expect(getBeginnerMode()).toBe('simple');
  });

  it('returns persisted mode when localStorage value is valid', () => {
    window.localStorage.setItem('beginner-mode', 'technical');

    expect(getBeginnerMode()).toBe('technical');
  });

  it('does not throw when setting mode if localStorage throws', () => {
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new Error('blocked');
    });

    expect(() => setBeginnerMode('detailed')).not.toThrow();
  });
});
