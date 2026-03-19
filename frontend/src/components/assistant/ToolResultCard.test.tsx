import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ToolResultCard } from './ToolResultCard';

describe('ToolResultCard', () => {
  it('renders blocked candidate lists as structured cards', () => {
    render(
      <ToolResultCard
        message={{
          id: 'tool-msg-1',
          sessionId: 'session-1',
          role: 'tool',
          content: 'tool: list_recent_blocked_candidates',
          toolName: 'list_recent_blocked_candidates',
          createdAt: '2026-03-19T15:00:00Z',
          toolResult: {
            ok: true,
            data: [
              {
                id: 'candidate-1',
                symbol: 'AAPL',
                signalType: 'BUY',
                status: 'blocked',
                blockedReasonCode: 'low_confidence',
                blockReason: 'Confidence was below threshold.',
                confidence: 0.41,
                detectedAt: '2026-03-19T14:00:00Z',
                blockedAt: '2026-03-19T14:01:00Z',
              },
            ],
          },
        }}
      />
    );

    expect(screen.getByText('AAPL')).toBeInTheDocument();
    expect(screen.getByText('low_confidence')).toBeInTheDocument();
    expect(screen.getByText('Confidence was below threshold.')).toBeInTheDocument();
    expect(screen.getByText('41%')).toBeInTheDocument();
  });

  it('renders knowledge matches without raw JSON', () => {
    render(
      <ToolResultCard
        message={{
          id: 'tool-msg-2',
          sessionId: 'session-1',
          role: 'tool',
          content: 'tool: query_knowledge',
          toolName: 'query_knowledge',
          createdAt: '2026-03-19T15:00:00Z',
          toolResult: {
            ok: true,
            data: [
              {
                path: 'knowledge/md/paper/finish-plan.md',
                excerpt: 'Paper readiness requires Gate0 through Gate9 to pass before sign-off.',
              },
            ],
          },
        }}
      />
    );

    expect(screen.getByText('finish-plan.md')).toBeInTheDocument();
    expect(screen.getByText('Paper readiness requires Gate0 through Gate9 to pass before sign-off.')).toBeInTheDocument();
    expect(screen.queryByText('{')).not.toBeInTheDocument();
  });
});
