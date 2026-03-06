export interface MarketTick {
  symbol: string;
  price: number;
  changePct: number;
  timestamp: number;
}

export interface QuoteSnapshot {
  symbol: string;
  bid: number;
  ask: number;
  last: number;
  timestamp: number;
}

export interface MarketDataStatus {
  connected: boolean;
  marketDataMode: string;
  paperTrading: boolean;
  checkedAt: string;
}

export interface TradingPilotStatus {
  pilotMode: boolean;
  authRequired: boolean;
  operatorRole: string;
  allowedRoles: string[];
  operatorAccess: boolean;
  brokerConnected: boolean;
  marketDataMode: string;
  paperTrading: boolean;
  readOnly: boolean;
  canTrade: boolean;
  quoteAuthority: boolean;
  intradayAuthority: boolean;
  executionFromChartBlocked: boolean;
  requiresManualBrokerConfirmation: boolean;
  reviewAgainstBroker: boolean;
  rollbackToReadOnly: boolean;
  reasons: string[];
  checklist: string[];
  checkedAt: string;
}

// Backend API Types for Observability & Intelligence

export interface MetricEvent {
  ts: string;
  timestamp: string;
  level: string;
  event: string;
  name: string;
  source: string;
  run_id?: string;
  task_id?: string;
  symbol?: string;
  provider?: string;
  tool?: string;
  success?: boolean;
  latency_ms?: number;
  strategy?: string;
  type?: string;
  confidence?: number;
  steps?: number;
  stages?: number;
  bank?: string;
  service?: string;
}

export interface HealthStatus {
  status: 'healthy' | 'unhealthy';
  healthy: boolean;
  timestamp: string;
  version?: string;
  uptime?: number;
}

export interface MemoryItem {
  id?: string;
  key: string;
  bank: string;
  ts: string;
  timestamp: string;
  type: string;
  symbol: string;
  summary: string;
  tags: string[];
  data: Record<string, unknown>;
  source?: {
    system: string;
    agent?: string;
  };
}

export interface MemoryQuery {
  symbol?: string;
  tags?: string[];
  type?: string;
  limit?: number;
  since?: string;
}

export interface MemoryRecallResponse {
  items: MemoryItem[];
  total: number;
}

export interface StrategySignal {
  type: 'buy' | 'sell' | 'hold';
  symbol: string;
  entryPrice: number;
  stopLoss?: number;
  takeProfit?: number;
  confidence: number;
  reason: string;
  timestamp: string;
}

export interface StrategyPerformance {
  strategyId: string;
  winRate: number;
  avgReturn: number;
  totalSignals: number;
  successfulSignals: number;
  lastUpdated: string;
}

export interface OrchestrationRequest {
  bank: string;
  symbol: string;
  strategy?: string;
  constraints: Record<string, unknown>;
  userContext: string;
  tags: string[];
  researchQueries?: string[];
}

export interface OrchestrationResult {
  plan: {
    summary: string;
    steps: string[];
    action: string;
    confidence: number;
    reasoningNotes: string;
  };
  tools: Array<{
    name: string;
    success: boolean;
  }>;
  runId?: string;
  duration?: number;
  status?: 'completed' | 'failed' | 'running';
}

export interface Signal {
  id: string;
  symbol: string;
  strategy_id: string;
  signal_type: string;
  confidence: number;
  entry_price?: number | null;
  stop_loss?: number | null;
  take_profit?: number | null;
  reasoning?: string | null;
  generated_at: string;
  expires_at?: string | null;
  status: string;
  orchestration_run_id?: string | null;
  created_at: string;
}

export interface SignalListResponse {
  signals: Signal[];
  total: number;
  limit: number;
  offset: number;
}

export interface OrchestrationRun {
  id: string;
  symbol: string;
  trigger_type: string;
  trigger_id?: string | null;
  agent_suggestion?: string | null;
  confidence?: number | null;
  reasoning?: string | null;
  memories_recalled?: number;
  status: string;
  started_at: string;
  completed_at?: string | null;
  error?: string | null;
}

export interface Recommendation {
  signal?: Signal | null;
  ai_analysis?: OrchestrationRun | null;
}

export interface RecommendationListResponse {
  recommendations: Recommendation[];
  total: number;
  limit: number;
  offset: number;
}

export interface StrategyTypeMetadata {
  id: string;
  name: string;
  version?: string;
  description?: string;
  requiredInputs?: string[];
  tags?: string[];
}

export interface StrategyInstance {
  id: string;
  name: string;
  strategyTypeId: string;
  strategyId?: string;
  enabled: boolean;
  sessionTimezone: string;
  flattenByCloseTime: string;
  configJson: Record<string, unknown>;
  configHash?: string;
  artifactId?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface BacktestRunStats {
  trades?: number;
  totalTrades?: number;
  winRate?: number;
  avgR?: number;
  maxDrawdown?: number;
  sharpe?: number;
  pnl?: number;
  finalCapital?: number;
  totalReturn?: number;
  [key: string]: unknown;
}

export interface BacktestTrade {
  symbol: string;
  side: string;
  entryPrice?: number | null;
  exitPrice?: number | null;
  quantity?: number | null;
  pnl?: number | null;
  pnlPct?: number | null;
  openedAt?: string | null;
  closedAt?: string | null;
  metadata?: Record<string, unknown>;
}

export interface BacktestRunBySymbol {
  symbol: string;
  trades: number;
  winRate: number;
  pnl: number;
}

export interface BacktestRunSummary {
  id: string;
  runId: string;
  instanceId?: string;
  strategyId?: string;
  symbols?: string[];
  from?: string;
  to?: string;
  status: string;
  stats: BacktestRunStats;
  datasetId?: string;
  datasetHash?: string;
  provenance?: ProvenanceInfo;
  startedAt?: string;
  completedAt?: string;
  createdAt?: string;
}

export interface BacktestRunDetail extends BacktestRunSummary {
  parentRunId?: string;
  config?: Record<string, unknown>;
  bySymbol?: BacktestRunBySymbol[];
  trades?: BacktestTrade[];
  error?: string;
}

export interface ResearchProject {
  id: string;
  name: string;
  description?: string;
  owner?: string;
  status?: string;
  baseInstanceId?: string;
  parameterGrid?: Record<string, unknown>;
  trainFrom?: string | null;
  trainTo?: string | null;
  testFrom?: string | null;
  testTo?: string | null;
  createdAt?: string;
  updatedAt?: string;
}

export interface ResearchProjectRun {
  id: string;
  backtestRunId?: string;
  status: string;
  parameters?: Record<string, unknown>;
  metrics?: Record<string, unknown>;
  rankScore?: number;
  lineage?: Record<string, unknown>;
  error?: string;
  startedAt?: string | null;
  completedAt?: string | null;
}

export interface TestingGateStatus {
  gate: string;
  status: string;
  lastRunId?: string;
  details?: Record<string, unknown>;
  lastRunAt?: string | null;
  updatedAt?: string | null;
}

export interface TestRunSummary {
  id: string;
  runId?: string;
  testName: string;
  status: string;
  summary?: Record<string, unknown>;
  artifactUri?: string;
  startedAt?: string | null;
  completedAt?: string | null;
  createdAt?: string | null;
}

export interface TriggerTestResponse {
  gate: string;
  testRunId: string;
  status: string;
  artifactUri?: string;
  summary?: Record<string, unknown>;
}

export interface RunSummary {
  id: string;
  runType: string;
  status: string;
  flowId?: string;
  source?: string;
  instanceId?: string;
  summary?: Record<string, unknown>;
  datasetId?: string;
  datasetHash?: string;
  provenance?: ProvenanceInfo;
  startedAt?: string;
  completedAt?: string | null;
  error?: string;
}

export interface RunTimelineEvent {
  id: string;
  type: string;
  category?: string;
  action?: string;
  outcome?: string;
  message?: string;
  metadata?: Record<string, unknown>;
  ts?: string;
}

export interface ProvenanceInfo {
  dataSourceType?: string;
  sourceProvider?: string;
  isSynthetic?: boolean;
  syntheticReason?: string;
  provenanceVerifiedAt?: string | null;
}

export interface EventSummary {
  id: string;
  kind: string;
  title: string;
  summary?: string;
  severity?: string;
  eventTime?: string;
  sourceId?: string;
  primarySymbol?: string;
  symbols?: string[];
  confidence?: number;
  attributes?: Record<string, unknown>;
  createdAt?: string;
}

export interface EventRaw {
  id: string;
  sourceId: string;
  sourceEventId: string;
  kind: string;
  eventTime: string;
  receivedAt: string;
  symbol?: string;
  payload?: Record<string, unknown>;
  contentHash?: string;
  flowId?: string;
  dataSourceType?: string;
  sourceProvider?: string;
  isSynthetic?: boolean;
  syntheticReason?: string;
  provenanceVerifiedAt?: string | null;
  createdAt?: string;
}

export interface EventDetail extends EventSummary {
  raw?: EventRaw[];
}

export interface EventTimelineEvent {
  type: string;
  ts?: string;
  message?: string;
  rawId?: string;
  flowId?: string;
  payload?: Record<string, unknown>;
  symbol?: string;
  relevance?: number;
  mappingMethod?: string;
  isPrimary?: boolean;
  eventId?: string;
}

export interface EventListResponse {
  events: EventSummary[];
  total: number;
  limit: number;
  offset: number;
}

export interface EventTimelineResponse {
  eventId: string;
  timeline: EventTimelineEvent[];
  totalRows?: number;
}

export interface EventClassification {
  class: string;
  impact: string;
  sentiment: string;
  horizon: string;
  tags?: string[];
  explanation?: string;
}

export interface DatasetSnapshot {
  datasetId: string;
  datasetHash: string;
  name?: string;
  symbol?: string;
  source?: string;
  schemaVer?: string;
  recordCount?: number;
  startDate?: string | null;
  endDate?: string | null;
  filePath?: string;
  metadata?: Record<string, unknown>;
  createdAt?: string;
  updatedAt?: string;
  lastSeenAt?: string;
  linkCount?: number;
}

export interface DatasetSnapshotLink {
  runType: 'run' | 'backtest_run';
  runRefId: string;
  observedHash: string;
  linkedAt: string;
  metadata?: Record<string, unknown>;
}

export interface DatasetListResponse {
  datasets: DatasetSnapshot[];
  limit: number;
  offset: number;
}

export interface DatasetDetail extends DatasetSnapshot {
  links?: DatasetSnapshotLink[];
}
