import { apiClient } from './http-client';
import type { MacroCandidate, MacroEventDetail, MacroEventListResponse } from './types';

interface ListMacroEventsParams extends Record<string, string | number | undefined> {
  limit?: number;
  offset?: number;
}

function buildQuery(params: Record<string, string | number | undefined>): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      query.set(key, String(value));
    }
  }
  const suffix = query.toString();
  return suffix ? `?${suffix}` : '';
}

export const macroService = {
  listEvents(params: ListMacroEventsParams = {}): Promise<MacroEventListResponse> {
    return apiClient.get<MacroEventListResponse>(`/api/v1/macro/events${buildQuery(params)}`);
  },

  getEvent(eventId: string): Promise<MacroEventDetail> {
    return apiClient.get<MacroEventDetail>(`/api/v1/macro/events/${eventId}`);
  },

  approveCandidate(candidateId: string, notes = ''): Promise<MacroCandidate> {
    return apiClient.post<MacroCandidate>(`/api/v1/macro/candidates/${candidateId}/approve`, { notes });
  },

  rejectCandidate(candidateId: string, notes = ''): Promise<MacroCandidate> {
    return apiClient.post<MacroCandidate>(`/api/v1/macro/candidates/${candidateId}/reject`, { notes });
  },
};
