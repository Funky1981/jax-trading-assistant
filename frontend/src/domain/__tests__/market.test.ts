import { describe, expect, it } from 'vitest';
import { formatPrice } from '../market';

describe('formatPrice', () => {
  it('formats to two decimals', () => {
    expect(formatPrice(249.4)).toBe('249.40');
  });
});
