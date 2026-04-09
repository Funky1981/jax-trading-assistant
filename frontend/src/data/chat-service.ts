import { apiClient } from './http-client';

export interface ChatSession {
  id: string;
  userId?: string;
  title?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AssistantTool {
  name: string;
  description: string;
  argKey: string;
  argLabel: string;
  evidenceLevel?: string;
  freshness?: string;
  allowed?: boolean;
  policyReason?: string;
}

export interface ChatToolsResponse {
  tools: AssistantTool[];
  notice: string;
  mode?: string;
  harnessEnabled?: boolean;
  shadowMode?: boolean;
  sessionRateLimitPerMinute?: number;
}

export interface ChatMessage {
  id: string;
  sessionId: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  traceId?: string;
  toolName?: string;
  toolArgs?: unknown;
  toolResult?: unknown;
  evidenceBundle?: unknown;
  createdAt: string;
}

export interface ChatTraceToolCall {
  name: string;
  args: unknown;
}

export interface ChatTraceToolRun {
  call: ChatTraceToolCall;
  result?: unknown;
  error?: string;
}

export interface ChatTraceValidationAttempt {
  attempt: number;
  answer: string;
  accepted: boolean;
  reasons?: string[];
}

export interface ChatTrace {
  traceId: string;
  sessionId: string;
  question: string;
  toolNames: string[];
  toolRuns?: ChatTraceToolRun[];
  validatorNotes: string[];
  validationAttempts?: ChatTraceValidationAttempt[];
  finalAnswer?: string;
  createdAt: string;
}

export interface ToolCall {
  name: string;
  args: Record<string, unknown>;
}

export interface SendMessageRequest {
  sessionId?: string;
  content: string;
  toolCall?: ToolCall;
}

export interface SendMessageResponse {
  sessionId: string;
  messages: ChatMessage[];
}

export const chatService = {
  listSessions() {
    return apiClient.get<ChatSession[]>('/api/v1/chat/sessions');
  },

  createSession(title?: string) {
    return apiClient.post<ChatSession>('/api/v1/chat/sessions', { title });
  },

  getSession(id: string) {
    return apiClient.get<{ session: ChatSession; messages: ChatMessage[] }>(`/api/v1/chat/sessions/${id}`);
  },

  getHistory(sessionId: string) {
    return apiClient.get<ChatMessage[]>(`/api/v1/chat?session=${sessionId}`);
  },

  sendMessage(req: SendMessageRequest) {
    return apiClient.post<SendMessageResponse>('/api/v1/chat', req);
  },

  getTools() {
    return apiClient.get<ChatToolsResponse>('/api/v1/chat/tools');
  },

  getTrace(traceId: string) {
    return apiClient.get<ChatTrace>(`/api/v1/chat/traces/${traceId}`);
  },
};
