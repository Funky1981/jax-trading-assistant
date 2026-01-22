import { describe, expect, it } from 'vitest';
import { createPresetLayout, LAYOUT_VERSION } from '../layouts';

describe('dashboard layouts', () => {
  it('builds trader preset with widgets', () => {
    const layout = createPresetLayout('trader');
    expect(layout.version).toBe(LAYOUT_VERSION);
    expect(layout.widgets.length).toBeGreaterThan(0);
    expect(layout.presetId).toBe('trader');
  });
});
