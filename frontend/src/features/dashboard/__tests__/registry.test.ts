import { describe, expect, it } from 'vitest';
import { widgetRegistry } from '../registry';

describe('widget registry', () => {
  it('has unique ids', () => {
    const ids = widgetRegistry.map((widget) => widget.id);
    const unique = new Set(ids);
    expect(unique.size).toBe(ids.length);
  });
});
