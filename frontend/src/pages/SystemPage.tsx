import { useState, useCallback, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { DashboardGrid, DashboardPanel } from '@/components/layout';
import {
  HealthPanel,
  MetricsPanel,
  MemoryBrowserPanel,
} from '@/components/dashboard';
import { eventsService } from '@/data/events-service';
import { datasetsService } from '@/data/datasets-service';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

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
  const [panelStates, setPanelStates] =
    useState<Record<PanelId, boolean>>(loadPanelState);

  const eventsQuery = useQuery({
    queryKey: ['system-events'],
    queryFn: () => eventsService.list({ limit: 20 }),
  });
  const datasetsQuery = useQuery({
    queryKey: ['system-datasets'],
    queryFn: () => datasetsService.list({ limit: 20 }),
  });

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

        <DashboardPanel colSpan={2}>
          <Card>
            <CardHeader>
              <CardTitle>Recent Events</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Time</TableHead>
                    <TableHead>Kind</TableHead>
                    <TableHead>Symbol</TableHead>
                    <TableHead>Title</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(eventsQuery.data?.events ?? []).map((event) => (
                    <TableRow key={event.id}>
                      <TableCell>{fmtDate(event.eventTime)}</TableCell>
                      <TableCell>{event.kind}</TableCell>
                      <TableCell>{event.primarySymbol || '-'}</TableCell>
                      <TableCell>{event.title}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </DashboardPanel>

        <DashboardPanel>
          <Card>
            <CardHeader>
              <CardTitle>Dataset Snapshots</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Symbol</TableHead>
                    <TableHead>Hash</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(datasetsQuery.data?.datasets ?? []).map((dataset) => (
                    <TableRow key={dataset.datasetId}>
                      <TableCell>{dataset.name || dataset.datasetId}</TableCell>
                      <TableCell>{dataset.symbol || '-'}</TableCell>
                      <TableCell>{dataset.datasetHash?.slice(0, 8) || '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </DashboardPanel>
      </DashboardGrid>
    </div>
  );
}

function fmtDate(raw?: string | null): string {
  if (!raw) {
    return '-';
  }
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return raw;
  }
  return d.toLocaleString();
}
