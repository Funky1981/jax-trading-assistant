import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { TestingPage } from './TestingPage';
import { testingService } from '@/data/testing-service';

vi.mock('@/data/testing-service', () => ({
  testingService: {
    getStatus: vi.fn(),
    getReadiness: vi.fn(),
    getTestRuns: vi.fn(),
    triggerAllGates: vi.fn(),
    triggerConfigIntegrity: vi.fn(),
    triggerDeterministicReplay: vi.fn(),
    triggerArtifactPromotion: vi.fn(),
    triggerExecutionIntegration: vi.fn(),
    triggerDataRecon: vi.fn(),
    triggerPnlRecon: vi.fn(),
    triggerFailureSuite: vi.fn(),
    triggerFlattenProof: vi.fn(),
    triggerAIAudit: vi.fn(),
    triggerProvenanceIntegrity: vi.fn(),
    triggerShadowParity: vi.fn(),
  },
}));

vi.mock('@/hooks/useHealth', () => ({
  useHealth: () => ({
    data: {
      services: [
        {
          name: 'trader',
          status: 'healthy',
          lastCheck: Date.parse('2026-03-19T14:04:02Z'),
          latency: 12,
        },
      ],
      overall: 'healthy',
    },
  }),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <TestingPage />
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe('TestingPage', () => {
  it('renders readiness metrics, report link, and gate artifacts', async () => {
    vi.mocked(testingService.getReadiness).mockResolvedValue({
      status: 'ready',
      ready: true,
      checkedAt: '2026-03-19T14:04:02Z',
      requiredGateCount: 10,
      passedGateCount: 10,
      failedGateCount: 0,
      skippedGateCount: 0,
      notStartedGateCount: 0,
      paperSessionsObserved: 3,
      shadowParityRequired: true,
      shadowParitySatisfied: true,
      reportUri: 'http://localhost:8181/reports/paper-readiness/latest.md',
      jsonReportUri: 'http://localhost:8181/reports/paper-readiness/latest.json',
      gateStatuses: [
        {
          gate: 'Gate4',
          status: 'passed',
          lastRunId: 'run-4',
          lastRunAt: '2026-03-19T14:03:00Z',
          updatedAt: '2026-03-19T14:03:00Z',
          details: {
            artifactUri: 'http://localhost:8181/reports/gate4/summary.md',
          },
        },
      ],
    });
    vi.mocked(testingService.getStatus).mockResolvedValue([
      {
        gate: 'Gate4',
        status: 'passed',
        lastRunId: 'run-4',
        lastRunAt: '2026-03-19T14:03:00Z',
        updatedAt: '2026-03-19T14:03:00Z',
        details: {
          artifactUri: 'http://localhost:8181/reports/gate4/summary.md',
        },
      },
    ]);
    vi.mocked(testingService.getTestRuns).mockResolvedValue([
      {
        id: 'test-run-1',
        testName: 'execution_integration',
        status: 'passed',
        artifactUri: 'http://localhost:8181/reports/gate4/summary.md',
        startedAt: '2026-03-19T14:03:00Z',
        completedAt: '2026-03-19T14:04:00Z',
      },
    ]);

    renderPage();

    expect(await screen.findByText('Paper Readiness')).toBeInTheDocument();
    expect(await screen.findByText('ready')).toBeInTheDocument();
    expect(await screen.findByText('10/10')).toBeInTheDocument();
    expect(await screen.findByText('3')).toBeInTheDocument();
    expect(await screen.findByRole('link', { name: 'Open paper-readiness report' })).toHaveAttribute(
      'href',
      'http://localhost:8181/reports/paper-readiness/latest.md'
    );
    expect(await screen.findAllByRole('link', { name: 'Artifact' })).not.toHaveLength(0);
    expect(await screen.findByText('execution_integration')).toBeInTheDocument();
  });
});
