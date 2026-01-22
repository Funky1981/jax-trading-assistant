import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { PnLIndicator } from '../PnLIndicator';

describe('PnLIndicator', () => {
  it('renders positive values with plus sign', () => {
    render(<PnLIndicator value={12.34} />);
    expect(screen.getByText('+12.34')).toBeInTheDocument();
  });

  it('renders negative values without plus sign', () => {
    render(<PnLIndicator value={-8.5} />);
    expect(screen.getByText('-8.50')).toBeInTheDocument();
  });
});
