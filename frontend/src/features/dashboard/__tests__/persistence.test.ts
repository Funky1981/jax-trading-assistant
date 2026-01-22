import { describe, expect, it } from 'vitest';
import { createPresetLayout } from '../layouts';
import { loadLayout, saveLayout } from '../persistence';

class MemoryStorage {
  private store = new Map<string, string>();

  getItem(key: string) {
    return this.store.get(key) ?? null;
  }

  setItem(key: string, value: string) {
    this.store.set(key, value);
  }

  removeItem(key: string) {
    this.store.delete(key);
  }
}

describe('dashboard persistence', () => {
  it('saves and loads layouts', () => {
    const storage = new MemoryStorage() as unknown as Storage;
    const layout = createPresetLayout('risk');

    saveLayout(layout, storage);
    const loaded = loadLayout(storage);

    expect(loaded?.presetId).toBe('risk');
    expect(loaded?.widgets.length).toBe(layout.widgets.length);
  });
});
