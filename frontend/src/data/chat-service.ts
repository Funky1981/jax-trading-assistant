import { apiClient } from './http-client';

export interface ChatSession {
  id: string;
  userId?: string;
  title?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ChatMessage {
  id: string;
  sessionId: string;
  role: 'user' | 'assistant' | 'tool';
  content: string;
  toolName?: string;
  toolArgs?: unknown;
  toolResult?: unknown;
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
    return apiClient.get<{ tools: { name: string; description: string }[]; notice: string }>('/api/v1/chat/tools');
  },
};
