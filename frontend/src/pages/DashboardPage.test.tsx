import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { OperatorEvidenceOverview } from '@/data/operator-evidence-service';
import { OperatorSummary } from './DashboardPage';

const safeOverview: OperatorEvidenceOverview = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
  genuineEvents: 6,
  syntheticEvents: 2,
  rejectedEvents: 1,
  deduplicatedEvents: 6,
  candidates: 1,
  approvals: 1,
  paperTickets: 1,
  pendingCheckpoints: 1,
  completedCheckpoints: 2,
  missingDataCheckpoints: 0,
  ambiguousCheckpoints: 0,
  checkedAt: '2026-07-21T10:00:00Z',
};

describe('OperatorSummary', () => {
  it('shows the paper-safe banner only when every safety condition passes', () => {
    render(<OperatorSummary data={safeOverview} loading={false} failed={false} />);
    expect(screen.getByText('PAPER-SAFE RUNTIME')).toBeInTheDocument();
    expect(screen.getAllByText('6')).toHaveLength(2);
  });

  it('warns when any runtime safety condition differs', () => {
    render(<OperatorSummary data={{ ...safeOverview, executionEnabled: true }} loading={false} failed={false} />);
    expect(screen.getByText('RUNTIME SAFETY WARNING')).toBeInTheDocument();
  });

  it('does not assume safety when the read model fails', () => {
    render(<OperatorSummary loading={false} failed />);
    expect(screen.getByText(/Runtime safety must not be assumed/i)).toBeInTheDocument();
  });
});
