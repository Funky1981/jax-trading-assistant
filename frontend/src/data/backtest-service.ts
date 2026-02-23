import { apiClient } from './http-client';
import type {
  BacktestRunDetail,
  BacktestRunStats,
  BacktestRunSummary,
  BacktestTrade,
  RunTimelineEvent,
  RunSummary,
} from './types';

interface BacktestRunSummaryDTO {
  id: string;
  runId: string;
  instanceId?: string;
  strategyId?: string;
  symbols?: string[];
  from?: string;
  to?: string;
  status: string;
  stats?: BacktestRunStats;
  startedAt?: string;
  completedAt?: string;
  createdAt?: string;
}

interface BacktestRunDetailDTO extends BacktestRunSummaryDTO {
  parentRunId?: string;
  config?: Record<string, unknown>;
  bySymbol?: Array<{ symbol: string; trades: number; winRate: number; pnl: number }>;
  trades?: BacktestTrade[];
  error?: string;
}

function toSummary(dto: BacktestRunSummaryDTO): BacktestRunSummary {
  return {
    id: dto.id,
    runId: dto.runId,
    instanceId: dto.instanceId,
    strategyId: dto.strategyId,
    symbols: dto.symbols ?? [],
    from: dto.from,
    to: dto.to,
    status: dto.status,
    stats: dto.stats ?? {},
    startedAt: dto.startedAt,
    completedAt: dto.completedAt,
    createdAt: dto.createdAt,
  };
}

function toDetail(dto: BacktestRunDetailDTO): BacktestRunDetail {
  return {
    ...toSummary(dto),
    parentRunId: dto.parentRunId,
    config: dto.config ?? {},
    bySymbol: dto.bySymbol ?? [],
    trades: dto.trades ?? [],
    error: dto.error,
  };
}

interface RunBacktestInput {
  instanceId: string;
  strategyId?: string;
  strategyConfigId?: string;
  from: string;
  to: string;
  symbolsOverride?: string[];
  datasetId?: string;
  seed?: number;
  initialCapital?: number;
  riskPerTrade?: number;
}

export const backtestService = {
  run(input: RunBacktestInput): Promise<{ runId: string; status: string; parentRunId?: string }> {
    return apiClient.post('/api/v1/backtests/run', input);
  },

  async list(params: { instanceId?: string; limit?: number } = {}): Promise<BacktestRunSummary[]> {
    const query = new URLSearchParams();
    if (params.instanceId) {
      query.set('instanceId', params.instanceId);
    }
    if (params.limit) {
      query.set('limit', String(params.limit));
    }
    const suffix = query.toString();
    const rows = await apiClient.get<BacktestRunSummaryDTO[]>(
      suffix ? `/api/v1/backtests/runs?${suffix}` : '/api/v1/backtests/runs'
    );
    return rows.map(toSummary);
  },

  async get(runId: string): Promise<BacktestRunDetail> {
    const dto = await apiClient.get<BacktestRunDetailDTO>(`/api/v1/backtests/runs/${runId}`);
    return toDetail(dto);
  },

  listRuns(limit = 100): Promise<RunSummary[]> {
    return apiClient.get<RunSummary[]>(`/api/v1/runs?limit=${limit}`);
  },

  async getRunTimeline(runId: string): Promise<RunTimelineEvent[]> {
    const out = await apiClient.get<{ runId: string; timeline: RunTimelineEvent[] }>(`/api/v1/runs/${runId}/timeline`);
    return out.timeline ?? [];
  },
};

