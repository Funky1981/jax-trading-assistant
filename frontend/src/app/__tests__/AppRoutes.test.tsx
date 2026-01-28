import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AppRoutes } from '../App';
import { DomainProvider } from '../../domain/store';

describe('AppRoutes', () => {
  it('renders the blotter page for /blotter', () => {
    render(
      <DomainProvider>
        <MemoryRouter initialEntries={['/blotter']}>
          <AppRoutes />
        </MemoryRouter>
      </DomainProvider>
    );

    expect(screen.getByRole('heading', { name: 'Blotter' })).toBeInTheDocument();
  });
});
