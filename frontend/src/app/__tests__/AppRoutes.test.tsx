import { render, screen } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { routes } from '../App';
import { DomainProvider } from '../../domain/store';

describe('AppRoutes', () => {
  it('renders the blotter page for /blotter', () => {
    const router = createMemoryRouter(routes, {
      initialEntries: ['/blotter'],
    });

    render(
      <DomainProvider>
        <RouterProvider router={router} />
      </DomainProvider>
    );

    expect(screen.getByRole('heading', { name: 'Blotter' })).toBeInTheDocument();
  });
});
