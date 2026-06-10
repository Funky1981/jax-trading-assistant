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
  etfPhase1Enabled: boolean;
  etfPolicyVersion?: string;
  etfPolicyHash?: string;
  etfEntryWorkflow: string;
  etfReadinessReasons: string[];
  reasons: string[];
  checklist: string[];
  checkedAt: string;
}

export interface ETFPolicy {
  phase: string;
  quote_freshness_seconds: number;
  max_spread_bps: number;
  min_bid_size: number;
  min_ask_size: number;
  session_timezone: string;
  regular_session_start: string;
  regular_session_end: string;
  require_stop_loss: boolean;
  require_flatten_by_close: boolean;
  entry_modes: string[];
}

export interface ETFInstrument {
  symbol: string;
  asset_class: string;
  instrument_type: string;
  tradable_modes: string[];
  eligibility_state: string;
  effective_date: string;
  change_owner: string;
  exclusions: string[];
}

export interface ETFInstrumentCatalog {
  version: string;
  hash: string;
  owner: string;
  policy: ETFPolicy;
  instruments: ETFInstrument[];
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
  service?: string;
  status: 'healthy' | 'degraded' | 'unhealthy' | string;
  healthy?: boolean;
  timestamp?: string;
  version?: string;
  uptime?: string;
}

export interface MemoryItem {
  id?: string;
  ts: string;
  type: string;
  symbol?: string;
  summary: string;
  tags?: string[];
  data?: Record<string, unknown>;
  source?: {
    system: string;
    ref?: string;
  };
}

export interface MemoryQuery {
  q?: string;
  symbol?: string;
  tags?: string[];
  type?: string;
  limit?: number;
  since?: string;
}

export interface MemoryRecallResponse {
  items: MemoryItem[];
  total?: number;
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

export type OpportunityConfidenceBand = 'high' | 'medium' | 'low' | 'unknown';

export type OpportunityRoute = 'manual_allowed' | 'approval_required' | 'blocked';

export type OpportunitySourceType = 'signal' | 'candidate' | 'approval';

export type SentimentEvidenceState = 'available' | 'disabled' | 'missing' | 'sparse' | 'low_confidence' | 'degraded' | 'error';

export interface SentimentEvidence {
  score?: number;
  label: 'positive' | 'negative' | 'mixed' | 'unavailable';
  confidence?: number;
  window?: string;
  sourceCount?: number;
  sourceGroups?: Record<string, number>;
  priceAgreement?: 'agreeing' | 'diverging' | 'neutral' | 'unknown';
  topDrivers?: string[];
  limitations?: string[];
  sourceItems?: Array<{
    title: string;
    sourceFamily?: string;
    url?: string;
    publishedAt?: string;
  }>;
  state: SentimentEvidenceState;
  summary?: string;
  snapshotAt?: string;
  intendedUse?: string;
}

export interface OpportunitySummary {
  id: string;
  symbol: string;
  signalType: string;
  confidenceBand: OpportunityConfidenceBand;
  summary: string;
  detectedAt: string;
  expiresAt?: string;
  route: OpportunityRoute;
  routeReason: string;
  sentimentSummary?: string;
  sentiment?: SentimentEvidence;
  status: string;
  sourceType: OpportunitySourceType;
  sourceId: string;
}

export type ScannerSentimentMode = 'filter' | 'rank_boost' | 'required_feature';
export type ScannerSourceTrustMode = 'equal' | 'trust_weighted';

export interface ScannerSentimentSettings {
  enabled: boolean;
  sourceScope: string;
  timeWindow: string;
  minimumThresholdLabel: string;
  minimumSourceCount: number;
  sourceTrustMode: ScannerSourceTrustMode;
  mode: ScannerSentimentMode;
  supported: boolean;
  connected: boolean;
  unsupportedReason?: string;
}

export interface ScannerSettings {
  enabled: boolean;
  assetScope: string;
  symbols: string[];
  universePreset: string;
  intervalSeconds: number;
  minimumConfidence: number;
  connected: boolean;
  sentiment: ScannerSentimentSettings;
}

export interface AIScannerApiSentiment {
  enabled: boolean;
  sourceScope: string;
  window: string;
  threshold: number;
  minimumSourceCount: number;
  sourceTrustWeightingMode: ScannerSourceTrustMode;
  mode: ScannerSentimentMode;
}

export interface AIScannerApiChannels {
  inApp: boolean;
  desktopWeb: boolean;
  mobilePush: boolean;
}

export interface AIScannerApiPolicy {
  manualRouteEnabled: boolean;
  approvalRouteEnabled: boolean;
  blockedReason?: string;
  requiresHumanApproval: boolean;
}

export interface AIScannerApiState {
  enabled: boolean;
  assetScope: string;
  symbols: string[];
  universePreset: string;
  intervalSeconds: number;
  minimumConfidence: number;
  sentiment: AIScannerApiSentiment;
  status: string;
  lastScanCompletedAt?: string;
  nextScanAt?: string;
  channels: AIScannerApiChannels;
  policy: AIScannerApiPolicy;
}

export interface AIOverviewApiResponse {
  checkedAt: string;
  scanner: AIScannerApiState;
  opportunityCounts: {
    signalsPending: number;
    candidates: number;
    approvals: number;
  };
  policySummary: {
    requiresHumanApproval: boolean;
    manualRouteEnabled: boolean;
    approvalRouteEnabled: boolean;
  };
  channelSummary: {
    inApp: boolean;
    desktopWeb: boolean;
    mobilePush: boolean;
  };
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

export interface StrategyTypeRequiredInputs {
  candles: string[];
  needsEarnings: boolean;
  needsNews: boolean;
}

export interface StrategyTypeParameter {
  key: string;
  type: 'int' | 'float' | 'string' | 'bool' | string;
  default: unknown;
  min?: number;
  max?: number;
  description?: string;
}

export interface StrategyTypeMetadata {
  strategyId: string;
  name: string;
  description: string;
  requiredInputs: StrategyTypeRequiredInputs;
  parameters: StrategyTypeParameter[];
}

export interface TradingModeStrategy {
  strategyTypeId: string;
  name: string;
  description: string;
  defaultConfig: Record<string, unknown>;
}

export interface TradingModeRiskDefaults {
  maxTradesPerDay: number;
  maxOpenPositions: number;
  riskPerTradePct: number;
  minConfidence: number;
  flattenBy: string;
  approvalRequired: boolean;
}

export interface TradingMode {
  id: string;
  name: string;
  description: string;
  assetClass: string;
  runtimeMode: string;
  executionPolicy: string;
  universe: string[];
  requiredData: string[];
  riskDefaults: TradingModeRiskDefaults;
  strategies: TradingModeStrategy[];
}

export interface TradingModeCatalog {
  modes: TradingMode[];
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
  sentiment?: SentimentEvidence;
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

export interface PaperReadinessSummary {
  status: string;
  ready?: boolean;
  checkedAt: string;
  requiredGateCount: number;
  passedGateCount: number;
  failedGateCount: number;
  skippedGateCount: number;
  notStartedGateCount: number;
  paperSessionsObserved: number;
  shadowParityRequired: boolean;
  shadowParitySatisfied: boolean;
  gateStatuses: TestingGateStatus[];
  reportUri?: string;
  jsonReportUri?: string;
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

export type NotificationCategory =
  | 'opportunity'
  | 'approval'
  | 'sentiment_triggered'
  | 'sentiment_invalidated'
  | 'analysis'
  | 'settings'
  | 'system';

export interface NotificationInboxEntry {
  id: string;
  category: NotificationCategory;
  eventType: string;
  title: string;
  body: string;
  destinationPath: string;
  createdAt: string;
  stale: boolean;
  channels: string[];
  severity?: string;
  primarySymbol?: string;
  sentimentTriggerType?: string;
  entityType?: string;
  entityId?: string;
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

export interface MacroETFMapping {
  id: string;
  symbol: string;
  theme: string;
  mappingReason: string;
  confidence: number;
  createdAt: string;
}

export interface MacroEvent {
  id: string;
  source: string;
  sourceEventId: string;
  eventType: string;
  region: string;
  eventTimeUtc: string;
  headline: string;
  summary?: string;
  actualValue?: number | null;
  expectedValue?: number | null;
  previousValue?: number | null;
  unit?: string;
  surpriseValue?: number | null;
  surprisePercent?: number | null;
  direction: string;
  confidence: number;
  status: string;
  rawPayload?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  etfMappings?: MacroETFMapping[];
  candidateCount: number;
  evidenceCount: number;
}

export interface MacroReaction {
  id: string;
  symbol: string;
  timeframe: string;
  prePrice: number;
  postPrice: number;
  changeAbs: number;
  changePercent: number;
  highAfter?: number | null;
  lowAfter?: number | null;
  volumeRatio?: number | null;
  atrRatio?: number | null;
  direction: string;
  confirmsEvent: boolean;
  tooExtended: boolean;
  noisy: boolean;
  reason: string;
  rawCandles?: unknown[];
  createdAt: string;
}

export interface MacroScenario {
  id: string;
  scenarioKey: string;
  candidateBias: string;
  primarySymbols: string[];
  secondarySymbols: string[];
  requiredConfirmations: string[];
  expectedReactions: Record<string, unknown>;
  result: string;
  reason: string;
  createdAt: string;
}

export interface MacroPricedInScore {
  id: string;
  symbol: string;
  verdict: string;
  score: number;
  reasons: string[];
  createdAt: string;
}

export interface MacroConfounder {
  id: string;
  confounderType: string;
  headline: string;
  source?: string;
  severity: string;
  reason: string;
  createdAt: string;
}

export interface MacroEvidenceBundle {
  id: string;
  symbol: string;
  status: string;
  verdict: string;
  summary: string;
  evidence: Record<string, unknown>;
  missingEvidence: string[];
  walkawayReasons: string[];
  createdAt: string;
}

export interface MacroCandidate {
  id: string;
  macroEventId: string;
  evidenceBundleId: string;
  symbol: string;
  side: string;
  bias: string;
  entryType: string;
  entryReferencePrice: number;
  stopReferencePrice: number;
  targetReferencePrice: number;
  riskPercent: number;
  timeLimit: string;
  status: string;
  createdReason: string;
  rejectionReason?: string;
  walkawayReasons: string[];
  createdAt: string;
  humanApprovalRequired: boolean;
}

export interface MacroEventListResponse {
  events: MacroEvent[];
  total: number;
  limit: number;
  offset: number;
}

export interface MacroEventDetail {
  event: MacroEvent;
  reactions: MacroReaction[];
  scenarios: MacroScenario[];
  pricedInScores: MacroPricedInScore[];
  confounders: MacroConfounder[];
  evidenceBundles: MacroEvidenceBundle[];
  candidates: MacroCandidate[];
}
