import { useEffect, useMemo, useState } from 'react';
import { createPresetLayout, type DashboardLayout } from './layouts';
import { loadLayout, saveLayout } from './persistence';

export const presetOptions = [
  { label: 'Trader', value: 'trader' },
  { label: 'Risk', value: 'risk' },
  { label: 'Ops', value: 'ops' },
];

export function useDashboardLayout() {
  const initial = useMemo(() => loadLayout() ?? createPresetLayout('trader'), []);
  const [layout, setLayout] = useState<DashboardLayout>(initial);

  useEffect(() => {
    saveLayout(layout);
  }, [layout]);

  const applyPreset = (presetId: string) => {
    setLayout(createPresetLayout(presetId));
  };

  return { layout, setLayout, applyPreset };
}
