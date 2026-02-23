import { render } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { axe } from 'vitest-axe';
import { AppShell } from '../../components';
import { AuthProvider } from '../../contexts/AuthContext';

const fetchMock = vi.fn();

afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllGlobals();
});

describe('app shell accessibility', () => {
  it('has no detectable violations', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: false }),
    } as Response);
    vi.stubGlobal('fetch', fetchMock);

    const { container } = render(
      <MemoryRouter initialEntries={['/']}>
        <AuthProvider>
          <Routes>
            <Route element={<AppShell />}>
              <Route index element={<div>Dashboard content</div>} />
            </Route>
          </Routes>
        </AuthProvider>
      </MemoryRouter>
    );

    const results = await axe(container);
    expect(results.violations).toHaveLength(0);
  });
});
