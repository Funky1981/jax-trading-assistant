import { apiClient } from './http-client';
import type { StrategyInstance, StrategyTypeMetadata } from './types';

interface InstanceApiDTO {
  id: string;
  name: string;
  strategyTypeId: string;
  strategyId?: string;
  enabled: boolean;
  sessionTimezone: string;
  flattenByCloseTime: string;
  configJson?: Record<string, unknown>;
  configHash?: string;
  artifactId?: string;
  createdAt?: string;
  updatedAt?: string;
}

interface SaveInstanceInput {
  id?: string;
  name: string;
  strategyTypeId: string;
  strategyId?: string;
  enabled: boolean;
  sessionTimezone?: string;
  flattenByCloseTime?: string;
  configJson: Record<string, unknown>;
  artifactId?: string;
}

function normalizeConfigJson(configJson?: Record<string, unknown>): Record<string, unknown> {
  const normalized: Record<string, unknown> = { ...(configJson ?? {}) };
  const rawSymbols = Array.isArray(normalized.symbols)
    ? normalized.symbols
    : Array.isArray(normalized.universe)
      ? normalized.universe
      : [];
  const symbols = rawSymbols
    .map((value) => (typeof value === 'string' ? value.trim().toUpperCase() : ''))
    .filter(Boolean);
  normalized.symbols = symbols;
  delete normalized.universe;
  return normalized;
}

function toInstance(dto: InstanceApiDTO): StrategyInstance {
  return {
    id: dto.id,
    name: dto.name,
    strategyTypeId: dto.strategyTypeId,
    strategyId: dto.strategyId ?? '',
    enabled: dto.enabled,
    sessionTimezone: dto.sessionTimezone,
    flattenByCloseTime: dto.flattenByCloseTime,
    configJson: normalizeConfigJson(dto.configJson),
    configHash: dto.configHash,
    artifactId: dto.artifactId,
    createdAt: dto.createdAt,
    updatedAt: dto.updatedAt,
  };
}

export const instancesService = {
  async list(): Promise<StrategyInstance[]> {
    const rows = await apiClient.get<InstanceApiDTO[]>('/api/v1/instances');
    return rows.map(toInstance);
  },

  async create(input: SaveInstanceInput): Promise<StrategyInstance> {
    const dto = await apiClient.post<InstanceApiDTO>('/api/v1/instances', {
      ...input,
      configJson: normalizeConfigJson(input.configJson),
    });
    return toInstance(dto);
  },

  async update(id: string, input: Partial<SaveInstanceInput>): Promise<{ ok: boolean; instance?: StrategyInstance }> {
    const payload = {
      ...input,
      id,
      configJson: input.configJson ? normalizeConfigJson(input.configJson) : undefined,
    };
    const out = await apiClient.put<{ ok: boolean; instance?: InstanceApiDTO }>(`/api/v1/instances/${id}`, payload);
    return {
      ok: out.ok,
      instance: out.instance ? toInstance(out.instance) : undefined,
    };
  },

  async enable(id: string): Promise<{ id: string; enabled: boolean }> {
    return apiClient.post<{ id: string; enabled: boolean }>(`/api/v1/instances/${id}/enable`);
  },

  async disable(id: string): Promise<{ id: string; enabled: boolean }> {
    return apiClient.post<{ id: string; enabled: boolean }>(`/api/v1/instances/${id}/disable`);
  },

  async listStrategyTypes(): Promise<StrategyTypeMetadata[]> {
    const rows = await apiClient.get<StrategyTypeMetadata[]>('/api/v1/strategy-types');
    return rows;
  },
};
