import { render, screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { routes } from '../App';
import { DomainProvider } from '../../domain/store';
import { AuthProvider } from '../../contexts/AuthContext';

const fetchMock = vi.fn();

afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllGlobals();
});

describe('AppRoutes', () => {
  it('renders the blotter page for /blotter', async () => {
    const router = createMemoryRouter(routes, {
      initialEntries: ['/blotter'],
    });
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: false }),
    } as Response);
    vi.stubGlobal('fetch', fetchMock);

    render(
      <DomainProvider>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </DomainProvider>
    );

    expect(await screen.findByText('Review recent orders and their status.')).toBeInTheDocument();
  });
});
