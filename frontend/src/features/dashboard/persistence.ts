import type { DashboardLayout } from './layouts';
import { LAYOUT_VERSION } from './layouts';

const STORAGE_KEY = 'jax.dashboard.layout.v1';

function resolveStorage(storage?: Storage) {
  if (storage) return storage;
  if (typeof window === 'undefined') return null;
  return window.localStorage;
}

export function saveLayout(layout: DashboardLayout, storage?: Storage) {
  const target = resolveStorage(storage);
  if (!target) return;
  const payload = JSON.stringify(layout);
  target.setItem(STORAGE_KEY, payload);
}

export function loadLayout(storage?: Storage): DashboardLayout | null {
  const target = resolveStorage(storage);
  if (!target) return null;

  const raw = target.getItem(STORAGE_KEY);
  if (!raw) return null;

  try {
    const parsed = JSON.parse(raw) as DashboardLayout;
    if (parsed.version !== LAYOUT_VERSION || !Array.isArray(parsed.widgets)) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function clearLayout(storage?: Storage) {
  const target = resolveStorage(storage);
  if (!target) return;
  target.removeItem(STORAGE_KEY);
}
