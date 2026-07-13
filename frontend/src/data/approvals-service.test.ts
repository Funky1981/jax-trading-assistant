import { describe, expect, it, vi } from 'vitest';
import { approvalsService } from './approvals-service';
import { apiClient } from './http-client';

vi.mock('./http-client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('approvalsService paper ticket review APIs', () => {
  it('uses safe paper ticket review queue and action endpoints', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce([]);
    vi.mocked(apiClient.post).mockResolvedValue({
      paperTicketId: 'pt_123',
      candidateId: 'candidate-123',
      status: 'paper_ticket_reviewed',
      paperOnly: true,
    });

    await approvalsService.getPaperTicketQueue(25);
    await approvalsService.markPaperTicketReviewed('pt_123', 'reviewed safely');
    await approvalsService.cancelPaperTicket('pt_123', 'cancel paper review');
    await approvalsService.addPaperTicketNote('pt_123', 'keep paper-only');

    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/paper-tickets?limit=25');
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/paper-tickets/pt_123/mark-reviewed', { note: 'reviewed safely' });
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/paper-tickets/pt_123/cancel', { note: 'cancel paper review' });
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/paper-tickets/pt_123/notes', { note: 'keep paper-only' });
  });
});
