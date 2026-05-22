import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { AuthProvider } from '@/contexts/AuthContext';
import { AppShell } from './AppShell';

function renderShell(path = '/') {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: false }),
    })
  );

  render(
    <AuthProvider>
      <MemoryRouter initialEntries={[path]}>
        <AppShell />
      </MemoryRouter>
    </AuthProvider>
  );
}

describe('AppShell', () => {
  it('uses task-first primary navigation', () => {
    renderShell();

    const primaryNav = screen.getByLabelText('Primary navigation');
    const links = within(primaryNav).getAllByRole('link').map((link) => link.textContent);

    expect(links).toEqual([
      'Home',
      'AI Trading',
      'Manual Trading',
      'Approvals',
      'Research',
      'Analysis',
      'Notifications',
      'Settings',
    ]);
  });

  it('keeps operational and legacy destinations behind advanced navigation', () => {
    renderShell();

    expect(screen.queryByRole('link', { name: 'Trading Modules' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'System' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Admin and QA/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Learn and legacy/i })).toBeInTheDocument();
  });
});
