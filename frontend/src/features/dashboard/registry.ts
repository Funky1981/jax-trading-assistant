export type RefreshPolicy = 'stream' | 'snapshot' | 'manual';

export interface WidgetDefinition {
  id: string;
  title: string;
  minSize: { w: number; h: number };
  defaultSize: { w: number; h: number };
  dataNeeds: string[];
  refreshPolicy: RefreshPolicy;
}

export const widgetRegistry: WidgetDefinition[] = [
  {
    id: 'watchlist',
    title: 'Watchlist',
    minSize: { w: 3, h: 2 },
    defaultSize: { w: 4, h: 3 },
    dataNeeds: ['prices', 'quotes'],
    refreshPolicy: 'stream',
  },
  {
    id: 'order-ticket',
    title: 'Order Ticket',
    minSize: { w: 3, h: 3 },
    defaultSize: { w: 4, h: 4 },
    dataNeeds: ['instruments'],
    refreshPolicy: 'manual',
  },
  {
    id: 'positions',
    title: 'Positions',
    minSize: { w: 4, h: 3 },
    defaultSize: { w: 6, h: 4 },
    dataNeeds: ['positions', 'prices'],
    refreshPolicy: 'snapshot',
  },
  {
    id: 'risk-summary',
    title: 'Risk Summary',
    minSize: { w: 3, h: 2 },
    defaultSize: { w: 4, h: 3 },
    dataNeeds: ['risk', 'positions'],
    refreshPolicy: 'snapshot',
  },
  {
    id: 'blotter',
    title: 'Blotter',
    minSize: { w: 5, h: 3 },
    defaultSize: { w: 8, h: 4 },
    dataNeeds: ['orders'],
    refreshPolicy: 'snapshot',
  },
  {
    id: 'system-status',
    title: 'System Status',
    minSize: { w: 3, h: 2 },
    defaultSize: { w: 4, h: 2 },
    dataNeeds: ['latency', 'errors'],
    refreshPolicy: 'snapshot',
  },
];

export function getWidgetById(id: string) {
  return widgetRegistry.find((widget) => widget.id === id) ?? null;
}

export function listWidgets() {
  return [...widgetRegistry];
}
