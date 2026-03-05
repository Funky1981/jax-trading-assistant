import { apiClient } from './http-client';
import type {
  EventClassification,
  EventDetail,
  EventListResponse,
  EventTimelineResponse,
} from './types';

interface ListEventsParams {
  kind?: string;
  symbol?: string;
  sourceId?: string;
  search?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

interface ClassifyEventInput {
  kind?: string;
  title?: string;
  summary?: string;
  severity?: string;
  symbols?: string[];
  attributes?: Record<string, unknown>;
}

export const eventsService = {
  async list(params: ListEventsParams = {}): Promise<EventListResponse> {
    const query = new URLSearchParams();
    if (params.kind) query.set('kind', params.kind);
    if (params.symbol) query.set('symbol', params.symbol);
    if (params.sourceId) query.set('sourceId', params.sourceId);
    if (params.search) query.set('search', params.search);
    if (params.from) query.set('from', params.from);
    if (params.to) query.set('to', params.to);
    if (params.limit) query.set('limit', String(params.limit));
    if (params.offset) query.set('offset', String(params.offset));
    const suffix = query.toString();
    return apiClient.get<EventListResponse>(suffix ? `/api/v1/events?${suffix}` : '/api/v1/events');
  },

  get(eventId: string): Promise<EventDetail> {
    return apiClient.get<EventDetail>(`/api/v1/events/${eventId}`);
  },

  async timeline(eventId: string): Promise<EventTimelineResponse> {
    return apiClient.get<EventTimelineResponse>(`/api/v1/events/${eventId}/timeline`);
  },

  async classify(input: ClassifyEventInput): Promise<EventClassification> {
    const out = await apiClient.post<{ classification: EventClassification }>('/api/v1/events/classify', input);
    return out.classification;
  },

  async classifyById(eventId: string): Promise<EventClassification> {
    const out = await apiClient.get<{ eventId: string; classification: EventClassification }>(
      `/api/v1/events/${eventId}/classify`
    );
    return out.classification;
  },
};
