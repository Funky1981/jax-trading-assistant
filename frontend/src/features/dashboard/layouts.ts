import { getWidgetById } from './registry';

export const LAYOUT_VERSION = 1;

export interface WidgetLayout {
  id: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface DashboardLayout {
  version: number;
  presetId: string;
  updatedAt: number;
  widgets: WidgetLayout[];
}

function withDefaults(id: string, x: number, y: number): WidgetLayout {
  const definition = getWidgetById(id);
  const size = definition?.defaultSize ?? { w: 4, h: 3 };
  return { id, x, y, w: size.w, h: size.h };
}

export function createPresetLayout(presetId: string): DashboardLayout {
  const widgets: WidgetLayout[] = [];

  switch (presetId) {
    case 'risk':
      widgets.push(
        withDefaults('risk-summary', 0, 0),
        withDefaults('positions', 4, 0),
        withDefaults('system-status', 0, 3)
      );
      break;
    case 'ops':
      widgets.push(
        withDefaults('system-status', 0, 0),
        withDefaults('blotter', 4, 0),
        withDefaults('watchlist', 0, 3)
      );
      break;
    case 'trader':
    default:
      widgets.push(
        withDefaults('watchlist', 0, 0),
        withDefaults('order-ticket', 4, 0),
        withDefaults('positions', 0, 3),
        withDefaults('risk-summary', 6, 3)
      );
  }

  return {
    version: LAYOUT_VERSION,
    presetId,
    updatedAt: Date.now(),
    widgets,
  };
}
