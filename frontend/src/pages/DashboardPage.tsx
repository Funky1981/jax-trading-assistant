import { useState, useCallback, useEffect } from 'react';
import { ChevronDown, ChevronUp, Lock, Unlock, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { WidgetGrid, WidgetPanel, type Layouts } from '@/components/layout';
import {
  HealthPanel,
  WatchlistPanel,
  PositionsPanel,
  RiskSummaryPanel,
  AIAssistantPanel,
  SignalsQueuePanel,
} from '@/components/dashboard';
import { cn } from '@/lib/utils';

// Panel IDs for the overview dashboard
const PANEL_IDS = ['health', 'watchlist', 'positions', 'risk', 'signalsQueue', 'aiAssistant'] as const;

type PanelId = (typeof PANEL_IDS)[number];

// Storage keys
const LAYOUTS_STORAGE_KEY = 'jax-dashboard-widget-layouts';
const PANEL_STATE_STORAGE_KEY = 'jax-dashboard-panel-states';

// Default layout for lg breakpoint (12 columns)
const DEFAULT_LAYOUTS: Layouts = {
  lg: [
    { x: 0, y: 0, w: 4, h: 4, i: 'health' },
    { x: 4, y: 0, w: 4, h: 5, i: 'watchlist' },
    { x: 0, y: 4, w: 8, h: 4, i: 'positions' },
    { x: 8, y: 0, w: 4, h: 5, i: 'risk' },
    { x: 0, y: 8, w: 12, h: 6, i: 'signalsQueue' },
    { x: 0, y: 14, w: 12, h: 6, i: 'aiAssistant' },
  ],
  md: [
    { x: 0, y: 0, w: 5, h: 4, i: 'health' },
    { x: 5, y: 0, w: 5, h: 5, i: 'watchlist' },
    { x: 0, y: 4, w: 6, h: 4, i: 'positions' },
    { x: 6, y: 4, w: 4, h: 5, i: 'risk' },
    { x: 0, y: 9, w: 10, h: 6, i: 'signalsQueue' },
    { x: 0, y: 15, w: 10, h: 6, i: 'aiAssistant' },
  ],
  sm: [
    { x: 0, y: 0, w: 6, h: 4, i: 'health' },
    { x: 0, y: 4, w: 6, h: 5, i: 'watchlist' },
    { x: 0, y: 9, w: 6, h: 4, i: 'positions' },
    { x: 0, y: 13, w: 6, h: 5, i: 'risk' },
    { x: 0, y: 18, w: 6, h: 6, i: 'signalsQueue' },
    { x: 0, y: 24, w: 6, h: 6, i: 'aiAssistant' },
  ],
  xs: [
    { x: 0, y: 0, w: 4, h: 4, i: 'health' },
    { x: 0, y: 4, w: 4, h: 5, i: 'watchlist' },
    { x: 0, y: 9, w: 4, h: 4, i: 'positions' },
    { x: 0, y: 13, w: 4, h: 5, i: 'risk' },
    { x: 0, y: 18, w: 4, h: 6, i: 'signalsQueue' },
    { x: 0, y: 24, w: 4, h: 6, i: 'aiAssistant' },
  ],
};

function loadLayouts(): Layouts {
  try {
    const stored = localStorage.getItem(LAYOUTS_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as Layouts;
      return mergeLayouts(parsed, DEFAULT_LAYOUTS);
    }
  } catch {
    // Ignore storage errors
  }
  return DEFAULT_LAYOUTS;
}

function saveLayouts(layouts: Layouts) {
  try {
    localStorage.setItem(LAYOUTS_STORAGE_KEY, JSON.stringify(layouts));
  } catch {
    // Ignore storage errors
  }
}

function loadPanelStates(): Record<PanelId, boolean> {
  try {
    const stored = localStorage.getItem(PANEL_STATE_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored) as Record<PanelId, boolean>;
      return {
        ...PANEL_IDS.reduce(
          (acc, id) => ({ ...acc, [id]: true }),
          {} as Record<PanelId, boolean>
        ),
        ...parsed,
      };
    }
  } catch {
    // Ignore storage errors
  }
  // Default: all panels expanded
  return PANEL_IDS.reduce(
    (acc, id) => ({ ...acc, [id]: true }),
    {} as Record<PanelId, boolean>
  );
}

function mergeLayouts(current: Layouts, defaults: Layouts): Layouts {
  const result: Layouts = { ...current };
  (Object.keys(defaults) as Array<keyof Layouts>).forEach((breakpoint) => {
    const currentItems = current[breakpoint] ?? [];
    const defaultItems = defaults[breakpoint] ?? [];
    const currentIds = new Set(currentItems.map((item) => item.i));
    const merged = [...currentItems];
    defaultItems.forEach((item) => {
      if (!currentIds.has(item.i)) {
        merged.push(item);
      }
    });
    result[breakpoint] = merged;
  });
  return result;
}

function savePanelStates(state: Record<PanelId, boolean>) {
  try {
    localStorage.setItem(PANEL_STATE_STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Ignore storage errors
  }
}

export function DashboardPage() {
  useEffect(() => {
    console.log('🏠 DashboardPage MOUNTED');
    return () => console.log('🏠 DashboardPage UNMOUNTED');
  }, []);

  const [isEditing, setIsEditing] = useState(false);
  const [layouts, setLayouts] = useState<Layouts>(loadLayouts);
  const [panelStates, setPanelStates] = useState<Record<PanelId, boolean>>(loadPanelStates);

  // Persist layouts when they change
  useEffect(() => {
    saveLayouts(layouts);
  }, [layouts]);

  // Persist panel states when they change
  useEffect(() => {
    savePanelStates(panelStates);
  }, [panelStates]);

  const handleLayoutChange = useCallback((newLayouts: Layouts) => {
    setLayouts(newLayouts);
  }, []);

  const handleResetLayout = useCallback(() => {
    setLayouts(DEFAULT_LAYOUTS);
  }, []);

  const togglePanel = useCallback((panelId: PanelId) => {
    setPanelStates((prev) => ({
      ...prev,
      [panelId]: !prev[panelId],
    }));
  }, []);

  const expandAll = useCallback(() => {
    setPanelStates(
      PANEL_IDS.reduce(
        (acc, id) => ({ ...acc, [id]: true }),
        {} as Record<PanelId, boolean>
      )
    );
  }, []);

  const collapseAll = useCallback(() => {
    setPanelStates(
      PANEL_IDS.reduce(
        (acc, id) => ({ ...acc, [id]: false }),
        {} as Record<PanelId, boolean>
      )
    );
  }, []);

  const allExpanded = PANEL_IDS.every((id) => panelStates[id]);
  const allCollapsed = PANEL_IDS.every((id) => !panelStates[id]);

  return (
    <div
      className={cn(
        'space-y-6',
        isEditing && 'ring-2 ring-primary/20 rounded-lg p-4'
      )}
    >
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">
            OVERVIEW
          </p>
          <h1 className="text-2xl font-bold md:text-3xl">Dashboard</h1>
          <p className="text-muted-foreground mt-1">
            Quick overview of your trading system. Customize the layout by
            clicking the unlock button.
          </p>
        </div>

        {/* Controls */}
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setIsEditing(!isEditing)}
          >
            {isEditing ? (
              <>
                <Lock className="h-4 w-4 mr-1" />
                Lock Layout
              </>
            ) : (
              <>
                <Unlock className="h-4 w-4 mr-1" />
                Edit Layout
              </>
            )}
          </Button>
          {isEditing && (
            <Button variant="outline" size="sm" onClick={handleResetLayout}>
              <RotateCcw className="h-4 w-4 mr-1" />
              Reset Layout
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={expandAll}
            disabled={allExpanded}
          >
            <ChevronDown className="h-4 w-4 mr-1" />
            Expand All
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={collapseAll}
            disabled={allCollapsed}
          >
            <ChevronUp className="h-4 w-4 mr-1" />
            Collapse All
          </Button>
        </div>
      </div>

      {/* Widget Grid */}
      <WidgetGrid
        layouts={layouts}
        onLayoutChange={handleLayoutChange}
        isEditable={isEditing}
      >
        <div key="health">
          <WidgetPanel id="health" title="System Health" isEditable={isEditing}>
            <HealthPanel
              isOpen={panelStates.health}
              onToggle={() => togglePanel('health')}
            />
          </WidgetPanel>
        </div>

        <div key="watchlist">
          <WidgetPanel id="watchlist" title="Watchlist" isEditable={isEditing}>
            <WatchlistPanel
              isOpen={panelStates.watchlist}
              onToggle={() => togglePanel('watchlist')}
            />
          </WidgetPanel>
        </div>

        <div key="positions">
          <WidgetPanel id="positions" title="Positions" isEditable={isEditing}>
            <PositionsPanel
              isOpen={panelStates.positions}
              onToggle={() => togglePanel('positions')}
            />
          </WidgetPanel>
        </div>

        <div key="risk">
          <WidgetPanel id="risk" title="Risk Summary" isEditable={isEditing}>
            <RiskSummaryPanel
              isOpen={panelStates.risk}
              onToggle={() => togglePanel('risk')}
            />
          </WidgetPanel>
        </div>

        <div key="signalsQueue">
          <WidgetPanel id="signalsQueue" title="Trading Approvals" isEditable={isEditing}>
            <SignalsQueuePanel
              isOpen={panelStates.signalsQueue}
              onToggle={() => togglePanel('signalsQueue')}
            />
          </WidgetPanel>
        </div>

        <div key="aiAssistant">
          <WidgetPanel id="aiAssistant" title="AI Trading Assistant" isEditable={isEditing}>
            <AIAssistantPanel
              isOpen={panelStates.aiAssistant}
              onToggle={() => togglePanel('aiAssistant')}
            />
          </WidgetPanel>
        </div>
      </WidgetGrid>
    </div>
  );
}
