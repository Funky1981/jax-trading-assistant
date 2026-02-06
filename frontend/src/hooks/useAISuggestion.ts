import { useMutation, useQuery } from '@tanstack/react-query';
import { buildUrl } from '@/config/api';

// Types matching the backend models
export type Action = 'BUY' | 'SELL' | 'HOLD' | 'WATCH';

export interface AISuggestion {
  symbol: string;
  action: Action;
  confidence: number;
  reasoning: string;
  risk_assessment: string;
  entry_price?: number;
  target_price?: number;
  stop_loss?: number;
  position_size_pct?: number;
  timeframe?: string;
  timestamp: string;
}

export interface SuggestionRequest {
  symbol: string;
  context?: string;
}

export interface SuggestionResponse {
  suggestion: AISuggestion;
  provider: string;
  model: string;
  tokens_used?: number;
}

export interface AIConfig {
  provider: string;
  model: string;
  memory_service_url: string;
  ib_bridge_url: string;
}

// Fetch AI suggestion for a symbol
async function fetchAISuggestion(request: SuggestionRequest): Promise<SuggestionResponse> {
  const response = await fetch(buildUrl('AGENT0_SERVICE', '/suggest'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ detail: 'Unknown error' }));
    throw new Error(error.detail || `HTTP ${response.status}`);
  }

  return response.json();
}

// Fetch AI service config
async function fetchAIConfig(): Promise<AIConfig> {
  const response = await fetch(buildUrl('AGENT0_SERVICE', '/config'));
  if (!response.ok) {
    throw new Error('Failed to fetch AI config');
  }
  return response.json();
}

// Fetch AI health
async function fetchAIHealth(): Promise<{ status: string; provider: string }> {
  const response = await fetch(buildUrl('AGENT0_SERVICE', '/health'));
  if (!response.ok) {
    throw new Error('AI service unhealthy');
  }
  return response.json();
}

// Hook to get AI suggestion (mutation - on demand)
export function useAISuggestion() {
  return useMutation({
    mutationFn: fetchAISuggestion,
    onError: (error) => {
      console.error('AI suggestion error:', error);
    },
  });
}

// Hook to check AI service health
export function useAIHealth() {
  return useQuery({
    queryKey: ['ai-health'],
    queryFn: fetchAIHealth,
    refetchInterval: 30000, // Check every 30 seconds
    retry: 1,
  });
}

// Hook to get AI configuration
export function useAIConfig() {
  return useQuery({
    queryKey: ['ai-config'],
    queryFn: fetchAIConfig,
    staleTime: 60000, // Cache for 1 minute
  });
}
