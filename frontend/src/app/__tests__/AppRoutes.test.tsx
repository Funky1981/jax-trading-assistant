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
  it('renders Home as the first route experience', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    const router = createMemoryRouter(routes, {
      initialEntries: ['/'],
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

    expect(await screen.findByRole('heading', { name: 'Start Here' })).toBeInTheDocument();
    expect(screen.getByText('Jax is a paper-trading assistant. It helps you find ideas, check evidence, and keep every action behind safety steps.')).toBeInTheDocument();
    expect(screen.getByText('How The App Fits Together')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Find AI opportunities/i })).toHaveAttribute('href', '/ai-trading');
    expect(screen.getByRole('link', { name: /Place a manual trade/i })).toHaveAttribute('href', '/manual-trading');
    expect(screen.getByRole('link', { name: /Test a strategy/i })).toHaveAttribute('href', '/research');
  });

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

  it('renders Notification Centre for /notifications', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    const router = createMemoryRouter(routes, {
      initialEntries: ['/notifications'],
      future: {
        v7_relativeSplatPath: true,
      },
    });
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);

      if (url.includes('/api/v1/events')) {
        return {
          ok: true,
          json: async () => ({ events: [], total: 0, limit: 100, offset: 0 }),
        } as Response;
      }

      return {
        ok: true,
        json: async () => ({ enabled: false }),
      } as Response;
    });
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

    expect(await screen.findByRole('heading', { name: 'Notification Centre' })).toBeInTheDocument();
  });

  it('includes candidate evidence routes for generic and module-scoped paths', () => {
    const root = routes.find((route) => route.path === '/');
    const childPaths = (root?.children ?? []).map((child) => child.path);

    expect(childPaths).toContain('ai-trading');
    expect(childPaths).toContain('manual-trading');
    expect(childPaths).toContain('swing-trading');
    expect(childPaths).toContain('notifications');
    expect(childPaths).toContain('macro/events');
    expect(childPaths).toContain('monitor/inbox');
    expect(childPaths).toContain('candidates/:candidateId/evidence');
    expect(childPaths).toContain('etf/candidates/:candidateId/evidence');
    expect(childPaths).toContain('equity-alpha/candidates/:candidateId/evidence');
    expect(childPaths).toContain('testing/mobile-approval-harness');
  });
});
