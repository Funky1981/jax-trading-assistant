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
  signal: Signal;
  ai_analysis?: OrchestrationRun | null;
}

export interface RecommendationListResponse {
  recommendations: Recommendation[];
  total: number;
  limit: number;
  offset: number;
}
