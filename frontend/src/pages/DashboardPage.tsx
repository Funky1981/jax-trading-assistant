import { useState, useCallback, useEffect } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { DashboardGrid, DashboardPanel } from '@/components/layout';
import {
  HealthPanel,
  WatchlistPanel,
  PositionsPanel,
  RiskSummaryPanel,
  AIAssistantPanel,
  SignalsQueuePanel,
} from '@/components/dashboard';
import { HelpHint } from '@/components/ui/help-hint';

const PANEL_IDS = ['health', 'watchlist', 'positions', 'risk', 'signalsQueue', 'aiAssistant'] as const;
type PanelId = (typeof PANEL_IDS)[number];

const STORAGE_KEY = 'jax-dashboard-panels';

function loadPanelState(): Record<PanelId, boolean> {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      return JSON.parse(stored);
    }
  } catch {
    // Ignore storage errors
  }
  return PANEL_IDS.reduce((acc, id) => ({ ...acc, [id]: true }), {} as Record<PanelId, boolean>);
}

function savePanelState(state: Record<PanelId, boolean>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Ignore storage errors
  }
}

export function DashboardPage() {
  const [panelStates, setPanelStates] = useState<Record<PanelId, boolean>>(loadPanelState);

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
    setPanelStates(PANEL_IDS.reduce((acc, id) => ({ ...acc, [id]: true }), {} as Record<PanelId, boolean>));
  }, []);

  const collapseAll = useCallback(() => {
    setPanelStates(PANEL_IDS.reduce((acc, id) => ({ ...acc, [id]: false }), {} as Record<PanelId, boolean>));
  }, []);

  const allExpanded = PANEL_IDS.every((id) => panelStates[id]);
  const allCollapsed = PANEL_IDS.every((id) => !panelStates[id]);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">
            OVERVIEW
          </p>
          <h1 className="flex items-center gap-2 text-2xl font-bold md:text-3xl">
            Dashboard
            <HelpHint text="Customize this layout to monitor health, risk, signals, and AI context." />
          </h1>
          <p className="text-muted-foreground mt-1">
            Start here for a quick overview. Use the panel toggles for compact or expanded views.
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={expandAll}
            disabled={allExpanded}
            className="w-full sm:w-auto"
          >
            <ChevronDown className="h-4 w-4 mr-1" />
            Expand All
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={collapseAll}
            disabled={allCollapsed}
            className="w-full sm:w-auto"
          >
            <ChevronUp className="h-4 w-4 mr-1" />
            Collapse All
          </Button>
        </div>
      </div>

      <DashboardGrid>
        <DashboardPanel>
          <HealthPanel
            isOpen={panelStates.health}
            onToggle={() => togglePanel('health')}
          />
        </DashboardPanel>

        <DashboardPanel>
          <WatchlistPanel
            isOpen={panelStates.watchlist}
            onToggle={() => togglePanel('watchlist')}
          />
        </DashboardPanel>

        <DashboardPanel>
          <RiskSummaryPanel
            isOpen={panelStates.risk}
            onToggle={() => togglePanel('risk')}
          />
        </DashboardPanel>

        <DashboardPanel colSpan={3}>
          <PositionsPanel
            isOpen={panelStates.positions}
            onToggle={() => togglePanel('positions')}
          />
        </DashboardPanel>

        <DashboardPanel colSpan={3}>
          <SignalsQueuePanel
            isOpen={panelStates.signalsQueue}
            onToggle={() => togglePanel('signalsQueue')}
          />
        </DashboardPanel>

        <DashboardPanel colSpan={3}>
          <AIAssistantPanel
            isOpen={panelStates.aiAssistant}
            onToggle={() => togglePanel('aiAssistant')}
          />
        </DashboardPanel>
      </DashboardGrid>
    </div>
  );
}
