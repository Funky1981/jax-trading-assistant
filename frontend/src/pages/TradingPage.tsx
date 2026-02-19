import { useState, useCallback, useEffect } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { DashboardGrid, DashboardPanel } from '@/components/layout';
import {
  WatchlistPanel,
  OrderTicketPanel,
  PositionsPanel,
  RiskSummaryPanel,
  TradeBlotterPanel,
  PriceChartPanel,
  StrategyMonitorPanel,
  AIAssistantPanel,
  SignalsQueuePanel,
} from '@/components/dashboard';

// Panel IDs for state management
const PANEL_IDS = [
  'watchlist',
  'orderTicket',
  'positions',
  'risk',
  'blotter',
  'chart',
  'strategy',
  'signalsQueue',
  'aiAssistant',
] as const;

type PanelId = typeof PANEL_IDS[number];

// Storage key for persisting panel state
const STORAGE_KEY = 'jax-trading-panels';

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
  return PANEL_IDS.reduce((acc, id) => ({ ...acc, [id]: true }), {} as Record<PanelId, boolean>);
}

function savePanelState(state: Record<PanelId, boolean>) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Ignore storage errors
  }
}

export function TradingPage() {
  const [panelStates, setPanelStates] = useState<Record<PanelId, boolean>>(loadPanelState);

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
    setPanelStates(PANEL_IDS.reduce((acc, id) => ({ ...acc, [id]: true }), {} as Record<PanelId, boolean>));
  }, []);

  const collapseAll = useCallback(() => {
    setPanelStates(PANEL_IDS.reduce((acc, id) => ({ ...acc, [id]: false }), {} as Record<PanelId, boolean>));
  }, []);

  const allExpanded = PANEL_IDS.every((id) => panelStates[id]);
  const allCollapsed = PANEL_IDS.every((id) => !panelStates[id]);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">
            TRADING TOOLS
          </p>
          <h1 className="text-2xl font-bold md:text-3xl">Trading</h1>
          <p className="text-muted-foreground mt-1">
            Execute trades, monitor positions, and analyze market data.
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
        {/* Row 1: Watchlist, Order Ticket, Risk Summary */}
        <DashboardPanel>
          <WatchlistPanel
            isOpen={panelStates.watchlist}
            onToggle={() => togglePanel('watchlist')}
          />
        </DashboardPanel>

        <DashboardPanel>
          <OrderTicketPanel
            isOpen={panelStates.orderTicket}
            onToggle={() => togglePanel('orderTicket')}
          />
        </DashboardPanel>

        <DashboardPanel>
          <RiskSummaryPanel
            isOpen={panelStates.risk}
            onToggle={() => togglePanel('risk')}
          />
        </DashboardPanel>

        {/* Row 2: Positions (wide), Strategy Monitor */}
        <DashboardPanel colSpan={2}>
          <PositionsPanel
            isOpen={panelStates.positions}
            onToggle={() => togglePanel('positions')}
          />
        </DashboardPanel>

        <DashboardPanel>
          <StrategyMonitorPanel
            isOpen={panelStates.strategy}
            onToggle={() => togglePanel('strategy')}
          />
        </DashboardPanel>

        {/* Row 3: Price Chart (wide), Trade Blotter */}
        <DashboardPanel colSpan={2}>
          <PriceChartPanel
            isOpen={panelStates.chart}
            onToggle={() => togglePanel('chart')}
          />
        </DashboardPanel>

        <DashboardPanel>
          <TradeBlotterPanel
            isOpen={panelStates.blotter}
            onToggle={() => togglePanel('blotter')}
          />
        </DashboardPanel>

        {/* Row 4: AI Assistant (full width) */}
        <DashboardPanel colSpan={3}>
          <SignalsQueuePanel
            isOpen={panelStates.signalsQueue}
            onToggle={() => togglePanel('signalsQueue')}
          />
        </DashboardPanel>

        {/* Row 5: AI Assistant (full width) */}
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
