import { fireEvent, render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '@/contexts/AuthContext';
import { AppShell } from './AppShell';

function renderShell(path = '/') {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({ ok: true, json: async () => ({ enabled: false }) }),
  );
  render(
    <AuthProvider>
      <MemoryRouter initialEntries={[path]}>
        <AppShell />
      </MemoryRouter>
    </AuthProvider>,
  );
}

describe('AppShell', () => {
  it('shows exactly six beginner-first primary destinations', () => {
    renderShell();
    const links = within(screen.getByLabelText('Primary navigation')).getAllByRole('link');
    expect(links.map((link) => link.textContent)).toEqual([
      'Home',
      'Guide',
      'Evidence Inbox',
      'Candidates',
      'Outcomes',
      'System Safety',
    ]);
    expect(links.map((link) => link.getAttribute('href'))).toEqual([
      '/',
      '/guide',
      '/monitor/inbox',
      '/etf/approvals',
      '/outcomes',
      '/system',
    ]);
  });

  it('collapses Review by default and keeps old pages reachable', () => {
    renderShell();
    const review = screen.getByRole('button', { name: 'Review' });
    expect(review).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByRole('link', { name: 'Manual Trading' })).not.toBeInTheDocument();
    fireEvent.click(review);
    expect(review).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('link', { name: 'Manual Trading' })).toHaveAttribute(
      'href',
      '/manual-trading',
    );
    expect(screen.getByRole('link', { name: 'Testing' })).toHaveAttribute('href', '/testing');
  });

  it('expands Review and shows its notice for an active Review route', () => {
    renderShell('/research');
    expect(screen.getByRole('button', { name: 'Review' })).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByRole('link', { name: 'Research' })).toHaveAttribute('aria-current', 'page');
    expect(screen.getByRole('note')).toHaveTextContent('this area has not yet been redesigned');
  });
});
