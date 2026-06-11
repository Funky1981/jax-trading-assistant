import { useState, useCallback, useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
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
import { HelpHint } from '@/components/ui/help-hint';
import { PilotStatusBanner } from '@/components/ui/PilotStatusBanner';
import { useTradingPilotStatus } from '@/hooks/useTradingPilotStatus';
import { useBeginnerMode } from '@/context/BeginnerUXContextValue';

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
  const { data: pilotStatus } = useTradingPilotStatus();
  const { mode } = useBeginnerMode();
  const location = useLocation();
  const navigate = useNavigate();
  const inETFModule = location.pathname.startsWith('/etf/');
  const isSimple = mode === 'simple';

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
            {isSimple ? 'PAPER TRADING WORKFLOW' : 'TRADING TOOLS'}
          </p>
          <h1 className="flex items-center gap-2 text-2xl font-bold md:text-3xl">
            {isSimple ? 'Manual Paper Trading' : 'Trading'}
            <HelpHint text="Use the panels below to monitor markets, submit and manage broker orders, and review signals." />
          </h1>
          <p className="text-muted-foreground mt-1">
            {isSimple
              ? 'Use this page only after you understand the trade idea. Start with the order ticket, then manage orders and positions.'
              : 'Monitor markets, place protected paper orders, manage working exits, and review signals in one place.'}
          </p>
        </div>

        {/* Expand/Collapse Controls */}
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

      <Card>
        <CardHeader>
          <CardTitle>How to Use This Screen</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm text-muted-foreground md:grid-cols-4">
          <div>
            <p className="font-semibold text-foreground">{isSimple ? '1. Choose the symbol' : '1. Build the entry'}</p>
            <p>{isSimple ? 'Type the ticker you want to paper trade into Order Ticket.' : 'Use Watchlist and Price Chart to pick a symbol, then submit a market or limit entry in Order Ticket.'}</p>
          </div>
          <div>
            <p className="font-semibold text-foreground">{isSimple ? '2. Add protection' : '2. Attach risk'}</p>
            <p>{isSimple ? 'Use a stop loss before you submit. This is the price where the trade idea is wrong.' : 'Add a stop loss and optional take profit in the ticket to send a bracket order from the start.'}</p>
          </div>
          <div>
            <p className="font-semibold text-foreground">{isSimple ? '3. Check pending orders' : '3. Manage working orders'}</p>
            <p>{isSimple ? 'Use Trade Blotter to see orders that have not filled yet.' : 'Use Trade Blotter to cancel pending broker orders before they fill.'}</p>
          </div>
          <div>
            <p className="font-semibold text-foreground">{isSimple ? '4. Check open trades' : '4. Manage live exposure'}</p>
            <p>{isSimple ? 'Use Positions to close or protect anything that is open.' : 'Use Positions to close or re-protect any open position after entry.'}</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Manual Trading vs AI Workflow</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          {inETFModule ? (
            <p>
              You are in the ETF module. New ETF entries are approval-first, so manual Buy/Sell entry is blocked by policy.
            </p>
          ) : (
            <p>
              You are in the Equity Alpha module. Manual Buy/Sell entries are allowed here through Order Ticket.
            </p>
          )}
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={() => navigate('/equity-alpha/order-ticket')}>
              Open Manual Order Ticket
            </Button>
            <Button type="button" variant="outline" onClick={() => navigate('/etf/approvals')}>
              Open ETF Approval Queue
            </Button>
            <Button type="button" variant="outline" onClick={() => navigate('/testing/mobile-approval-harness')}>
              Open Mobile Notification Harness
            </Button>
          </div>
          <p className="text-xs">
            For AI opportunity scanning, use the AI Trading Assistant panel below and enable Auto Scan.
          </p>
        </CardContent>
      </Card>

      {pilotStatus ? (
        <PilotStatusBanner
          title={
            pilotStatus.readOnly
              ? `Trading is in read-only pilot mode for role ${pilotStatus.operatorRole}.`
              : `Pilot trading is enabled for role ${pilotStatus.operatorRole}. Confirm every action in IB/TWS before submitting.`
          }
          readOnly={pilotStatus.readOnly}
          reasons={pilotStatus.reasons}
          checklist={pilotStatus.checklist}
        />
      ) : null}

      {isSimple && (
        <Card>
          <CardHeader>
            <CardTitle>What You Can Do Here</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm md:grid-cols-3">
            <div>
              <p className="font-semibold text-foreground">Paper order</p>
              <p className="text-muted-foreground">Submit a paper order after confirming it in IB/TWS.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">Cancel order</p>
              <p className="text-muted-foreground">Cancel an order that is still waiting to fill.</p>
            </div>
            <div>
              <p className="font-semibold text-foreground">Close or protect</p>
              <p className="text-muted-foreground">Manage a position after it exists.</p>
            </div>
          </CardContent>
        </Card>
      )}

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

        {!isSimple && (
          <DashboardPanel>
            <StrategyMonitorPanel
              isOpen={panelStates.strategy}
              onToggle={() => togglePanel('strategy')}
            />
          </DashboardPanel>
        )}

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
        {!isSimple && (
          <DashboardPanel colSpan={3}>
            <SignalsQueuePanel
              isOpen={panelStates.signalsQueue}
              onToggle={() => togglePanel('signalsQueue')}
            />
          </DashboardPanel>
        )}

        {/* Row 5: AI Assistant (full width) */}
        {!isSimple && (
          <DashboardPanel colSpan={3}>
            <AIAssistantPanel
              isOpen={panelStates.aiAssistant}
              onToggle={() => togglePanel('aiAssistant')}
            />
          </DashboardPanel>
        )}
      </DashboardGrid>
    </div>
  );
}
