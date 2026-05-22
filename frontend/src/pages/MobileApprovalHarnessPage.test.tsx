import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { MobileApprovalHarnessPage } from './MobileApprovalHarnessPage';
import { approvalsService } from '@/data/approvals-service';

vi.mock('@/data/approvals-service', () => ({
  approvalsService: {
    submitTelegramDecision: vi.fn(),
  },
}));

describe('MobileApprovalHarnessPage', () => {
  it('submits mobile approval webhook payload and renders response', async () => {
    vi.mocked(approvalsService.submitTelegramDecision).mockResolvedValue({
      approvalId: 'approval-123',
      candidateId: 'candidate-456',
      decision: 'approved',
      runtimeMode: 'paper',
    });

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <MobileApprovalHarnessPage />
        </QueryClientProvider>
      </MemoryRouter>
    );

    fireEvent.change(screen.getByLabelText('One-time Token'), { target: { value: 'token-abc' } });
    fireEvent.click(screen.getByRole('button', { name: 'Submit Mobile Decision' }));

    await waitFor(() => {
      expect(approvalsService.submitTelegramDecision).toHaveBeenCalledWith(
        expect.objectContaining({
          token: 'token-abc',
          action: 'approved',
          runtimeMode: 'paper',
        })
      );
    });

    expect(await screen.findByText(/Decision Accepted/i)).toBeInTheDocument();
    expect(await screen.findByText(/approval-123/i)).toBeInTheDocument();
  });
});
