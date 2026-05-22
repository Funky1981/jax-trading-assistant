import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { routes } from '../App';
import { AuthProvider } from '../../contexts/AuthContext';

const fetchMock = vi.fn();

afterEach(() => {
  fetchMock.mockReset();
  vi.unstubAllGlobals();
});

describe('AppRoutes', () => {
  it('renders the blotter page for /blotter', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    const router = createMemoryRouter(routes, {
      initialEntries: ['/blotter'],
      future: {
        v7_relativeSplatPath: true,
      },
    });
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: false }),
    } as Response);
    vi.stubGlobal('fetch', fetchMock);

    render(
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <RouterProvider
            router={router}
            future={{
              v7_startTransition: true,
            }}
          />
        </AuthProvider>
      </QueryClientProvider>
    );

    expect(await screen.findByText('Review recent orders and their status.')).toBeInTheDocument();
  });

  it('includes candidate evidence routes for generic and module-scoped paths', () => {
    const root = routes.find((route) => route.path === '/');
    const childPaths = (root?.children ?? []).map((child) => child.path);

    expect(childPaths).toContain('candidates/:candidateId/evidence');
    expect(childPaths).toContain('etf/candidates/:candidateId/evidence');
    expect(childPaths).toContain('equity-alpha/candidates/:candidateId/evidence');
    expect(childPaths).toContain('testing/mobile-approval-harness');
  });
});
