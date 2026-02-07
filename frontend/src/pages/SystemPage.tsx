import { useState, useCallback, useEffect } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { DashboardGrid, DashboardPanel } from '@/components/layout';
import {
  HealthPanel,
  MetricsPanel,
  MemoryBrowserPanel,
} from '@/components/dashboard';

// Panel IDs for state management
const PANEL_IDS = ['health', 'metrics', 'memory'] as const;

type PanelId = (typeof PANEL_IDS)[number];

// Storage key for persisting panel state
const STORAGE_KEY = 'jax-system-panels';

function loadPanelState(): Record<PanelId, boolean> {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      return JSON.parse(stored);
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

function savePanelState(state: Record<PanelId, boolean>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Ignore storage errors
  }
}

export function SystemPage() {
  useEffect(() => {
    console.log('⚙️ SystemPage MOUNTED');
    return () => console.log('⚙️ SystemPage UNMOUNTED');
  }, []);

  const [panelStates, setPanelStates] =
    useState<Record<PanelId, boolean>>(loadPanelState);

  // Persist panel state
  useEffect(() => {
    savePanelState(panelStates);
  }, [panelStates]);

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
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">
            SYSTEM OVERVIEW
          </p>
          <h1 className="text-2xl font-bold md:text-3xl">System</h1>
          <p className="text-muted-foreground mt-1">
            Monitor system health, events, and memory banks.
          </p>
        </div>

        {/* Expand/Collapse Controls */}
        <div className="flex gap-2">
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

      {/* Dashboard Grid */}
      <DashboardGrid>
        {/* Row 1: Health Panel (wide) */}
        <DashboardPanel colSpan={2}>
          <HealthPanel
            isOpen={panelStates.health}
            onToggle={() => togglePanel('health')}
          />
        </DashboardPanel>

        {/* Row 1: Metrics Panel */}
        <DashboardPanel>
          <MetricsPanel
            isOpen={panelStates.metrics}
            onToggle={() => togglePanel('metrics')}
          />
        </DashboardPanel>

        {/* Row 2: Memory Browser (full width) */}
        <DashboardPanel colSpan={3}>
          <MemoryBrowserPanel
            isOpen={panelStates.memory}
            onToggle={() => togglePanel('memory')}
          />
        </DashboardPanel>
      </DashboardGrid>
    </div>
  );
}
