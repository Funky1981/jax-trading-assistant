import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { AppRoutes } from '../App';

describe('AppRoutes', () => {
  it('renders the blotter page for /blotter', () => {
    render(
      <MemoryRouter initialEntries={['/blotter']}>
        <AppRoutes />
      </MemoryRouter>
    );

    expect(screen.getByRole('heading', { name: 'Blotter' })).toBeInTheDocument();
  });
});
