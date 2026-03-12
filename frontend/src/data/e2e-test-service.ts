import { apiClient } from './http-client';

export interface PlaywrightRunResult {
  status: 'idle' | 'running' | 'passed' | 'failed';
  startedAt?: string;
  completedAt?: string;
  durationMs?: number;
  exitCode?: number;
  spec?: string;
  output?: string;
  message?: string;
}

export const e2eTestService = {
  /** Trigger a Playwright run. Pass an optional spec name (e.g. "auth") to scope it. */
  run(spec?: string): Promise<PlaywrightRunResult> {
    const params = spec ? `?spec=${encodeURIComponent(spec)}` : '';
    return apiClient.post<PlaywrightRunResult>(`/api/v1/e2e/run${params}`);
  },

  /** Poll the result of the in-progress or last completed run. */
  getResults(): Promise<PlaywrightRunResult> {
    return apiClient.get<PlaywrightRunResult>('/api/v1/e2e/results');
  },
};
