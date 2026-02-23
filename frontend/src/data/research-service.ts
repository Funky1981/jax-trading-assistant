import { apiClient } from './http-client';
import type { ResearchProject, ResearchProjectRun } from './types';

interface CreateProjectInput {
  name: string;
  description?: string;
  owner?: string;
  baseInstanceId?: string;
  parameterGrid?: Record<string, unknown>;
  trainFrom?: string;
  trainTo?: string;
  testFrom?: string;
  testTo?: string;
}

interface RunProjectInput {
  from?: string;
  to?: string;
  strategyId?: string;
  symbolsOverride?: string[];
  datasetId?: string;
  seed?: number;
  initialCapital?: number;
  riskPerTrade?: number;
}

interface RunProjectResponse {
  projectId: string;
  status: string;
  parentRunId?: string;
  totalCombos: number;
  failedCombos: number;
  runs: Array<{
    projectRunId: string;
    index: number;
    combo: Record<string, unknown>;
    trainRunId?: string;
    testRunId?: string;
    rankScore?: number;
    metrics?: Record<string, unknown>;
  }>;
}

export const researchService = {
  listProjects(): Promise<ResearchProject[]> {
    return apiClient.get('/api/v1/research/projects');
  },

  createProject(input: CreateProjectInput): Promise<{ id: string; name: string }> {
    return apiClient.post('/api/v1/research/projects', input);
  },

  getProject(id: string): Promise<ResearchProject> {
    return apiClient.get(`/api/v1/research/projects/${id}`);
  },

  runProject(id: string, input: RunProjectInput): Promise<RunProjectResponse> {
    return apiClient.post(`/api/v1/research/projects/${id}/run`, input);
  },

  listProjectRuns(id: string): Promise<ResearchProjectRun[]> {
    return apiClient.get(`/api/v1/research/projects/${id}/runs`);
  },
};
