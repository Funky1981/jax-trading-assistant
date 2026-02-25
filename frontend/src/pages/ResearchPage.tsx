import { ChangeEvent, useEffect, useMemo, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Play, Upload, Download, Save, Plus, RefreshCw } from 'lucide-react';
import { instancesService } from '@/data/instances-service';
import { backtestService } from '@/data/backtest-service';
import { researchService } from '@/data/research-service';
import { datasetsService } from '@/data/datasets-service';
import type { DatasetSnapshot, ResearchProject, StrategyInstance } from '@/data/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

type InstanceEditorState = {
  id?: string;
  name: string;
  strategyTypeId: string;
  strategyId: string;
  enabled: boolean;
  sessionTimezone: string;
  flattenByCloseTime: string;
  artifactId: string;
  configText: string;
};

const defaultEditorState: InstanceEditorState = {
  name: '',
  strategyTypeId: '',
  strategyId: '',
  enabled: false,
  sessionTimezone: 'America/New_York',
  flattenByCloseTime: '15:55',
  artifactId: '',
  configText: '{\n  "universe": ["SPY"]\n}',
};

function toEditorState(instance: StrategyInstance): InstanceEditorState {
  return {
    id: instance.id,
    name: instance.name,
    strategyTypeId: instance.strategyTypeId,
    strategyId: instance.strategyId ?? '',
    enabled: instance.enabled,
    sessionTimezone: instance.sessionTimezone,
    flattenByCloseTime: instance.flattenByCloseTime,
    artifactId: instance.artifactId ?? '',
    configText: JSON.stringify(instance.configJson ?? {}, null, 2),
  };
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

function getInitialDateRange() {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  const toStr = to.toISOString().slice(0, 10);
  const fromStr = from.toISOString().slice(0, 10);
  return { fromStr, toStr };
}

export function ResearchPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const importInputRef = useRef<HTMLInputElement>(null);
  const [activeTab, setActiveTab] = useState<'instances' | 'projects' | 'runs'>('instances');
  const [selectedInstanceId, setSelectedInstanceId] = useState('');
  const [editor, setEditor] = useState<InstanceEditorState>(defaultEditorState);
  const [editorError, setEditorError] = useState<string>('');
  const [symbolsOverride, setSymbolsOverride] = useState('SPY');
  const [runRange, setRunRange] = useState(getInitialDateRange());
  const [datasetId, setDatasetId] = useState('');
  const [runsInstanceFilter, setRunsInstanceFilter] = useState('all');
  const [runsLimit, setRunsLimit] = useState(100);
  const [projectGridText, setProjectGridText] = useState('{\n  "riskPerTrade": [0.005, 0.01]\n}');
  const [projectForm, setProjectForm] = useState({
    name: '',
    description: '',
    owner: '',
    baseInstanceId: '',
    trainFrom: runRange.fromStr,
    trainTo: runRange.toStr,
    testFrom: runRange.fromStr,
    testTo: runRange.toStr,
  });
  const [selectedProjectId, setSelectedProjectId] = useState('');
  const [projectError, setProjectError] = useState('');

  const instancesQuery = useQuery({
    queryKey: ['instances'],
    queryFn: () => instancesService.list(),
  });
  const strategyTypesQuery = useQuery({
    queryKey: ['strategy-types'],
    queryFn: () => instancesService.listStrategyTypes(),
  });
  const projectsQuery = useQuery({
    queryKey: ['research-projects'],
    queryFn: () => researchService.listProjects(),
  });
  const runsQuery = useQuery({
    queryKey: ['backtest-runs', runsInstanceFilter, runsLimit],
    queryFn: () =>
      backtestService.list({
        instanceId: runsInstanceFilter === 'all' ? undefined : runsInstanceFilter,
        limit: runsLimit,
      }),
  });
  const datasetsQuery = useQuery({
    queryKey: ['datasets'],
    queryFn: () => datasetsService.list({ limit: 200 }),
  });
  const projectRunsQuery = useQuery({
    queryKey: ['research-project-runs', selectedProjectId],
    queryFn: () => researchService.listProjectRuns(selectedProjectId),
    enabled: selectedProjectId.length > 0,
  });

  const instanceById = useMemo(() => {
    const map = new Map<string, StrategyInstance>();
    for (const instance of instancesQuery.data ?? []) {
      map.set(instance.id, instance);
    }
    return map;
  }, [instancesQuery.data]);

  useEffect(() => {
    if (!instancesQuery.data || instancesQuery.data.length === 0) {
      return;
    }
    if (selectedInstanceId && instanceById.has(selectedInstanceId)) {
      return;
    }
    const first = instancesQuery.data[0];
    setSelectedInstanceId(first.id);
    setEditor(toEditorState(first));
  }, [instancesQuery.data, instanceById, selectedInstanceId]);

  useEffect(() => {
    if (!projectsQuery.data || projectsQuery.data.length === 0) {
      return;
    }
    if (selectedProjectId) {
      return;
    }
    setSelectedProjectId(projectsQuery.data[0].id);
  }, [projectsQuery.data, selectedProjectId]);

  const saveInstanceMutation = useMutation({
    mutationFn: async () => {
      let parsed: Record<string, unknown> = {};
      try {
        parsed = JSON.parse(editor.configText) as Record<string, unknown>;
      } catch (err) {
        throw new Error(`Config JSON is invalid: ${(err as Error).message}`);
      }

      const payload = {
        name: editor.name.trim(),
        strategyTypeId: editor.strategyTypeId.trim(),
        strategyId: editor.strategyId.trim(),
        enabled: editor.enabled,
        sessionTimezone: editor.sessionTimezone.trim(),
        flattenByCloseTime: editor.flattenByCloseTime.trim(),
        artifactId: editor.artifactId.trim(),
        configJson: parsed,
      };

      if (!payload.name || !payload.strategyTypeId) {
        throw new Error('Name and strategy type are required.');
      }

      if (editor.id) {
        await instancesService.update(editor.id, payload);
        return editor.id;
      }
      const created = await instancesService.create(payload);
      return created.id;
    },
    onSuccess: async (id) => {
      await queryClient.invalidateQueries({ queryKey: ['instances'] });
      setSelectedInstanceId(id);
      setEditorError('');
    },
    onError: (err) => {
      setEditorError((err as Error).message);
    },
  });

  const enableMutation = useMutation({
    mutationFn: async (instance: StrategyInstance) => {
      if (instance.enabled) {
        return instancesService.disable(instance.id);
      }
      return instancesService.enable(instance.id);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['instances'] });
    },
  });

  const runBacktestMutation = useMutation({
    mutationFn: async (instanceId: string) => {
      const symbols = symbolsOverride
        .split(',')
        .map((symbol) => symbol.trim().toUpperCase())
        .filter(Boolean);
      return backtestService.run({
        instanceId,
        from: runRange.fromStr,
        to: runRange.toStr,
        symbolsOverride: symbols,
        datasetId: datasetId || undefined,
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['backtest-runs'] });
      setActiveTab('runs');
    },
  });

  const createProjectMutation = useMutation({
    mutationFn: async () => {
      if (!projectForm.name.trim()) {
        throw new Error('Project name is required.');
      }
      let grid: Record<string, unknown> = {};
      try {
        grid = JSON.parse(projectGridText) as Record<string, unknown>;
      } catch (err) {
        throw new Error(`Parameter grid JSON is invalid: ${(err as Error).message}`);
      }
      return researchService.createProject({
        name: projectForm.name.trim(),
        description: projectForm.description.trim(),
        owner: projectForm.owner.trim(),
        baseInstanceId: projectForm.baseInstanceId || undefined,
        parameterGrid: grid,
        trainFrom: projectForm.trainFrom,
        trainTo: projectForm.trainTo,
        testFrom: projectForm.testFrom,
        testTo: projectForm.testTo,
      });
    },
    onSuccess: async () => {
      setProjectError('');
      await queryClient.invalidateQueries({ queryKey: ['research-projects'] });
    },
    onError: (err) => {
      setProjectError((err as Error).message);
    },
  });

  const runProjectMutation = useMutation({
    mutationFn: async (project: ResearchProject) => {
      return researchService.runProject(project.id, {
        from: runRange.fromStr,
        to: runRange.toStr,
        datasetId: datasetId || undefined,
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['research-project-runs'] });
      await queryClient.invalidateQueries({ queryKey: ['backtest-runs'] });
      setActiveTab('runs');
    },
  });

  const onSelectInstance = (instance: StrategyInstance) => {
    setSelectedInstanceId(instance.id);
    setEditor(toEditorState(instance));
    setEditorError('');
  };

  const onNewInstance = () => {
    setSelectedInstanceId('');
    setEditor(defaultEditorState);
    setEditorError('');
  };

  const selectedDataset = useMemo(() => {
    return (datasetsQuery.data?.datasets ?? []).find((ds) => ds.datasetId === datasetId);
  }, [datasetsQuery.data, datasetId]);

  const onImportConfig = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const parsed = JSON.parse(String(reader.result ?? '{}')) as Record<string, unknown>;
        const imported = parsed as Partial<StrategyInstance>;
        if (typeof imported.name === 'string' && typeof imported.strategyTypeId === 'string') {
          setEditor({
            id: imported.id,
            name: imported.name,
            strategyTypeId: imported.strategyTypeId,
            strategyId: imported.strategyId ?? '',
            enabled: Boolean(imported.enabled),
            sessionTimezone: imported.sessionTimezone ?? 'America/New_York',
            flattenByCloseTime: imported.flattenByCloseTime ?? '15:55',
            artifactId: imported.artifactId ?? '',
            configText: JSON.stringify(imported.configJson ?? {}, null, 2),
          });
          return;
        }
        setEditor((prev) => ({
          ...prev,
          configText: JSON.stringify(parsed, null, 2),
        }));
      } catch (err) {
        setEditorError(`Import failed: ${(err as Error).message}`);
      }
    };
    reader.readAsText(file);
  };

  const onExportConfig = () => {
    try {
      const payload = {
        id: editor.id,
        name: editor.name,
        strategyTypeId: editor.strategyTypeId,
        strategyId: editor.strategyId,
        enabled: editor.enabled,
        sessionTimezone: editor.sessionTimezone,
        flattenByCloseTime: editor.flattenByCloseTime,
        artifactId: editor.artifactId,
        configJson: JSON.parse(editor.configText),
      };
      const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
      const href = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = href;
      link.download = `${editor.name || 'strategy-instance'}.json`;
      link.click();
      URL.revokeObjectURL(href);
    } catch (err) {
      setEditorError(`Export failed: ${(err as Error).message}`);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-primary mb-1">RESEARCH LAB</p>
        <h1 className="text-2xl font-bold md:text-3xl">Research</h1>
        <p className="text-muted-foreground mt-1">
          Manage strategy instances, run projects, and launch deterministic backtests.
        </p>
      </div>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as 'instances' | 'projects' | 'runs')}>
        <TabsList>
          <TabsTrigger value="instances">Instances</TabsTrigger>
          <TabsTrigger value="projects">Projects</TabsTrigger>
          <TabsTrigger value="runs">Runs</TabsTrigger>
        </TabsList>

        <TabsContent value="instances" className="space-y-4">
          <div className="grid gap-4 lg:grid-cols-5">
            <Card className="lg:col-span-3">
              <CardHeader className="flex-row items-center justify-between space-y-0">
                <div>
                  <CardTitle>Strategy Instances</CardTitle>
                  <CardDescription>Database-backed configs for execution and research.</CardDescription>
                </div>
                <Button variant="outline" size="sm" onClick={onNewInstance}>
                  <Plus className="mr-1 h-4 w-4" />
                  New
                </Button>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Instance</TableHead>
                      <TableHead>Strategy</TableHead>
                      <TableHead>Enabled</TableHead>
                      <TableHead>Universe</TableHead>
                      <TableHead>Updated</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(instancesQuery.data ?? []).map((instance) => {
                      const universe = Array.isArray((instance.configJson as { universe?: unknown }).universe)
                        ? ((instance.configJson as { universe?: unknown[] }).universe?.length ?? 0)
                        : 0;
                      return (
                        <TableRow key={instance.id} className={instance.id === selectedInstanceId ? 'bg-muted/40' : ''}>
                          <TableCell>{instance.name}</TableCell>
                          <TableCell>{instance.strategyId || instance.strategyTypeId}</TableCell>
                          <TableCell>{instance.enabled ? 'Yes' : 'No'}</TableCell>
                          <TableCell>{universe}</TableCell>
                          <TableCell>{fmtDate(instance.updatedAt)}</TableCell>
                          <TableCell className="text-right space-x-2">
                            <Button size="sm" variant="outline" onClick={() => onSelectInstance(instance)}>
                              View/Edit
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={enableMutation.isPending}
                              onClick={() => enableMutation.mutate(instance)}
                            >
                              {instance.enabled ? 'Disable' : 'Enable'}
                            </Button>
                            <Button
                              size="sm"
                              onClick={() => runBacktestMutation.mutate(instance.id)}
                              disabled={runBacktestMutation.isPending}
                            >
                              <Play className="mr-1 h-4 w-4" />
                              Run
                            </Button>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>

            <Card className="lg:col-span-2">
              <CardHeader>
                <CardTitle>Instance Editor</CardTitle>
                <CardDescription>Create or update instance JSON config.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <Input
                  placeholder="Name"
                  value={editor.name}
                  onChange={(event) => setEditor((prev) => ({ ...prev, name: event.target.value }))}
                />
                <Select
                  value={editor.strategyTypeId || 'none'}
                  onValueChange={(value) => setEditor((prev) => ({ ...prev, strategyTypeId: value === 'none' ? '' : value }))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Strategy Type" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Select strategy type</SelectItem>
                    {(strategyTypesQuery.data ?? []).map((strategyType) => (
                      <SelectItem key={strategyType.id} value={strategyType.id}>
                        {strategyType.id}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  placeholder="Strategy ID (optional)"
                  value={editor.strategyId}
                  onChange={(event) => setEditor((prev) => ({ ...prev, strategyId: event.target.value }))}
                />
                <Input
                  placeholder="Session Timezone"
                  value={editor.sessionTimezone}
                  onChange={(event) => setEditor((prev) => ({ ...prev, sessionTimezone: event.target.value }))}
                />
                <Input
                  placeholder="Flatten By Close (HH:mm)"
                  value={editor.flattenByCloseTime}
                  onChange={(event) => setEditor((prev) => ({ ...prev, flattenByCloseTime: event.target.value }))}
                />
                <Input
                  placeholder="Artifact ID (optional)"
                  value={editor.artifactId}
                  onChange={(event) => setEditor((prev) => ({ ...prev, artifactId: event.target.value }))}
                />
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={editor.enabled}
                    onChange={(event) => setEditor((prev) => ({ ...prev, enabled: event.target.checked }))}
                  />
                  Enabled
                </label>
                <textarea
                  className="min-h-48 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
                  value={editor.configText}
                  onChange={(event) => setEditor((prev) => ({ ...prev, configText: event.target.value }))}
                />
                {editorError && <p className="text-sm text-destructive">{editorError}</p>}
                <div className="grid grid-cols-2 gap-2">
                  <Button onClick={() => saveInstanceMutation.mutate()} disabled={saveInstanceMutation.isPending}>
                    <Save className="mr-1 h-4 w-4" />
                    Save
                  </Button>
                  <Button type="button" variant="outline" onClick={onExportConfig}>
                    <Download className="mr-1 h-4 w-4" />
                    Export
                  </Button>
                  <Button type="button" variant="outline" onClick={() => importInputRef.current?.click()}>
                    <Upload className="mr-1 h-4 w-4" />
                    Import
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setEditor(defaultEditorState)}
                  >
                    <RefreshCw className="mr-1 h-4 w-4" />
                    Reset
                  </Button>
                </div>
                <input ref={importInputRef} type="file" className="hidden" accept=".json,application/json" onChange={onImportConfig} />
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    type="date"
                    value={runRange.fromStr}
                    onChange={(event) => setRunRange((prev) => ({ ...prev, fromStr: event.target.value }))}
                  />
                  <Input
                    type="date"
                    value={runRange.toStr}
                    onChange={(event) => setRunRange((prev) => ({ ...prev, toStr: event.target.value }))}
                  />
                  <Input
                    className="col-span-2"
                    value={symbolsOverride}
                    onChange={(event) => setSymbolsOverride(event.target.value)}
                    placeholder="Symbols override (comma-separated)"
                  />
                  <Select value={datasetId || 'none'} onValueChange={(value) => setDatasetId(value === 'none' ? '' : value)}>
                    <SelectTrigger className="col-span-2">
                      <SelectValue placeholder="Dataset snapshot (optional)" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">Use runtime default</SelectItem>
                      {(datasetsQuery.data?.datasets ?? []).map((dataset) => (
                        <SelectItem key={dataset.datasetId} value={dataset.datasetId}>
                          {dataset.name || dataset.datasetId} {dataset.symbol ? `(${dataset.symbol})` : ''}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  {selectedDataset && (
                    <div className="col-span-2 rounded-md border border-border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
                      <div>Dataset: {selectedDataset.name || selectedDataset.datasetId}</div>
                      <div>Hash: {selectedDataset.datasetHash}</div>
                      <div>
                        Range: {fmtDate(selectedDataset.startDate)} → {fmtDate(selectedDataset.endDate)}
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="projects" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Research Projects</CardTitle>
              <CardDescription>Create parameter sweeps or walk-forward projects.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 md:grid-cols-3">
                <Input
                  placeholder="Project name"
                  value={projectForm.name}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, name: event.target.value }))}
                />
                <Input
                  placeholder="Description"
                  value={projectForm.description}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, description: event.target.value }))}
                />
                <Input
                  placeholder="Owner"
                  value={projectForm.owner}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, owner: event.target.value }))}
                />
                <Select
                  value={projectForm.baseInstanceId || 'none'}
                  onValueChange={(value) => setProjectForm((prev) => ({ ...prev, baseInstanceId: value === 'none' ? '' : value }))}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Base Instance" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">No base instance</SelectItem>
                    {(instancesQuery.data ?? []).map((instance) => (
                      <SelectItem key={instance.id} value={instance.id}>
                        {instance.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  type="date"
                  value={projectForm.trainFrom}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, trainFrom: event.target.value }))}
                />
                <Input
                  type="date"
                  value={projectForm.trainTo}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, trainTo: event.target.value }))}
                />
                <Input
                  type="date"
                  value={projectForm.testFrom}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, testFrom: event.target.value }))}
                />
                <Input
                  type="date"
                  value={projectForm.testTo}
                  onChange={(event) => setProjectForm((prev) => ({ ...prev, testTo: event.target.value }))}
                />
              </div>
              <textarea
                className="min-h-40 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
                value={projectGridText}
                onChange={(event) => setProjectGridText(event.target.value)}
              />
              {projectError && <p className="text-sm text-destructive">{projectError}</p>}
              <Button onClick={() => createProjectMutation.mutate()} disabled={createProjectMutation.isPending}>
                Create Project
              </Button>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Project List</CardTitle>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Base Instance</TableHead>
                    <TableHead>Updated</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(projectsQuery.data ?? []).map((project) => (
                    <TableRow
                      key={project.id}
                      className={selectedProjectId === project.id ? 'bg-muted/40' : ''}
                      onClick={() => setSelectedProjectId(project.id)}
                    >
                      <TableCell>{project.name}</TableCell>
                      <TableCell>{project.status ?? '-'}</TableCell>
                      <TableCell>{project.baseInstanceId || '-'}</TableCell>
                      <TableCell>{fmtDate(project.updatedAt)}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          size="sm"
                          onClick={(event) => {
                            event.stopPropagation();
                            runProjectMutation.mutate(project);
                          }}
                          disabled={runProjectMutation.isPending}
                        >
                          <Play className="mr-1 h-4 w-4" />
                          Run
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {selectedProjectId && (
            <Card>
              <CardHeader>
                <CardTitle>Project Runs</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Run ID</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Rank</TableHead>
                      <TableHead>Started</TableHead>
                      <TableHead>Completed</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(projectRunsQuery.data ?? []).map((run) => (
                      <TableRow key={run.id}>
                        <TableCell>{run.backtestRunId || run.id}</TableCell>
                        <TableCell>{run.status}</TableCell>
                        <TableCell>{run.rankScore ?? '-'}</TableCell>
                        <TableCell>{fmtDate(run.startedAt)}</TableCell>
                        <TableCell>{fmtDate(run.completedAt)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="runs" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Backtest Runs</CardTitle>
              <CardDescription>Filter by instance and open run details in Analysis.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="grid gap-3 md:grid-cols-3">
                <Select value={runsInstanceFilter} onValueChange={setRunsInstanceFilter}>
                  <SelectTrigger>
                    <SelectValue placeholder="Instance filter" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All instances</SelectItem>
                    {(instancesQuery.data ?? []).map((instance) => (
                      <SelectItem key={instance.id} value={instance.id}>
                        {instance.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  type="number"
                  min={1}
                  max={500}
                  value={runsLimit}
                  onChange={(event) => setRunsLimit(Math.max(1, Number(event.target.value || 100)))}
                />
              </div>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Run ID</TableHead>
                    <TableHead>Instance</TableHead>
                    <TableHead>Dataset</TableHead>
                    <TableHead>From</TableHead>
                    <TableHead>To</TableHead>
                    <TableHead>Win Rate</TableHead>
                    <TableHead>Drawdown</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(runsQuery.data ?? []).map((run) => (
                    <TableRow
                      key={run.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/analysis?runId=${encodeURIComponent(run.runId)}`)}
                    >
                      <TableCell>{run.runId}</TableCell>
                      <TableCell>{run.instanceId || '-'}</TableCell>
                      <TableCell>{run.datasetId ? run.datasetId.slice(0, 8) : '-'}</TableCell>
                      <TableCell>{fmtDate(run.from)}</TableCell>
                      <TableCell>{fmtDate(run.to)}</TableCell>
                      <TableCell>{num(run.stats.winRate)}</TableCell>
                      <TableCell>{num(run.stats.maxDrawdown)}</TableCell>
                      <TableCell>{run.status}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function num(value: unknown): string {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }
  return value.toFixed(4);
}
