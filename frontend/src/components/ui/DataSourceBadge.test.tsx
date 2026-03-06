import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DataSourceBadge } from './DataSourceBadge';

describe('DataSourceBadge', () => {
  it('renders delayed paper status explicitly', () => {
    render(<DataSourceBadge marketDataMode="delayed" paperTrading />);

    expect(screen.getByText('DELAYED PAPER')).toBeInTheDocument();
  });

  it('renders unavailable when the data source is errored', () => {
    render(<DataSourceBadge marketDataMode="live" paperTrading={false} isError />);

    expect(screen.getByText('UNAVAILABLE')).toBeInTheDocument();
  });
});
