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
import { useQuery } from '@tanstack/react-query';
import { operatorEvidenceService } from '@/data/operator-evidence-service';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

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
  const evidence = useQuery({ queryKey: ['operator-evidence-overview'], queryFn: operatorEvidenceService.overview, refetchInterval: 30_000 });
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

      <OperatorSummary data={evidence.data} loading={evidence.isPending} failed={evidence.isError} />

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

function OperatorSummary({ data, loading, failed }: { data?: Awaited<ReturnType<typeof operatorEvidenceService.overview>>; loading: boolean; failed: boolean }) {
  if (loading) return <Card><CardContent className="p-4 text-sm text-muted-foreground">Loading persisted operator evidence…</CardContent></Card>;
  if (failed || !data) return <Card><CardContent className="p-4 text-sm text-destructive">Operator evidence is unavailable. Runtime safety must not be assumed.</CardContent></Card>;
  const safe = data.runtimeMode === 'paper' && !data.allowLiveTrading && !data.executionEnabled && !data.executionWorkerEnabled && !data.brokerExecutionAllowed && data.maximumLeverage <= 1;
  const values = [
    ['Runtime mode', data.runtimeMode], ['Live trading allowed', yesNo(data.allowLiveTrading)], ['Execution enabled', yesNo(data.executionEnabled)],
    ['Execution worker enabled', yesNo(data.executionWorkerEnabled)], ['Broker execution allowed', yesNo(data.brokerExecutionAllowed)], ['Maximum leverage', `${data.maximumLeverage}x`],
    ['Genuine events', data.genuineEvents], ['Synthetic events', data.syntheticEvents], ['Rejected events', data.rejectedEvents], ['Deduplicated events', data.deduplicatedEvents],
    ['Candidates', data.candidates], ['Approvals', data.approvals], ['Paper tickets', data.paperTickets], ['Pending checkpoints', data.pendingCheckpoints],
    ['Completed checkpoints', data.completedCheckpoints], ['Missing-data checkpoints', data.missingDataCheckpoints], ['Ambiguous checkpoints', data.ambiguousCheckpoints],
  ];
  return <Card className={safe ? 'border-success' : 'border-destructive'}>
    <CardHeader><CardTitle>{safe ? 'PAPER-SAFE RUNTIME' : 'RUNTIME SAFETY WARNING'}</CardTitle></CardHeader>
    <CardContent><p className="mb-4 text-sm text-muted-foreground">{safe ? 'Paper mode is active and every execution path reported here is disabled.' : 'One or more paper-safety conditions differ. Review before continuing.'}</p>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">{values.map(([label, value]) => <div key={label} className="rounded border p-3"><div className="text-xs text-muted-foreground">{label}</div><div className="font-semibold">{value}</div></div>)}</div>
    </CardContent></Card>;
}

function yesNo(value: boolean) { return value ? 'Yes' : 'No'; }
