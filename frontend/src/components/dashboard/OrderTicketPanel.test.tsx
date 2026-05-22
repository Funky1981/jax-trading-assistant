import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderTicketPanel } from './OrderTicketPanel';
import type { ETFInstrument } from '@/data/types';

const mutate = vi.fn();

vi.mock('@/hooks/useOrders', () => ({
  useCreateOrder: () => ({
    mutate,
    isPending: false,
    error: null,
    data: null,
  }),
}));

vi.mock('@/hooks/useMarketDataStatus', () => ({
  useMarketDataStatus: () => ({
    data: {
      marketDataMode: 'delayed',
      paperTrading: true,
    },
    isError: false,
  }),
}));

vi.mock('@/hooks/useTradingPilotStatus', () => ({
  useTradingPilotStatus: () => ({
    data: {
      readOnly: false,
      reasons: ['Quotes on this screen are non-authoritative during the pilot; confirm in IB/TWS.'],
    },
  }),
}));

let etfInstruments: ETFInstrument[] = [];

vi.mock('@/hooks/useETFInstruments', () => ({
  useETFInstruments: () => ({
    data: {
      instruments: etfInstruments,
    },
  }),
}));

describe('OrderTicketPanel', () => {
  function renderPanel() {
    return render(
      <MemoryRouter>
        <OrderTicketPanel isOpen onToggle={() => undefined} />
      </MemoryRouter>
    );
  }

  beforeEach(() => {
    mutate.mockClear();
    etfInstruments = [];
  });

  it('submits a bracket order when stop loss protection is provided', async () => {
    const user = userEvent.setup();

    renderPanel();

    await user.type(screen.getByLabelText('Symbol'), 'AAPL');
    await user.type(screen.getByLabelText('Quantity'), '10');
    await user.type(screen.getByLabelText('Stop Loss'), '195');

    expect(screen.getByRole('button', { name: 'Submit BUY Bracket' })).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Submit BUY Bracket' }));
    await user.click(screen.getByRole('checkbox'));
    await user.click(screen.getByRole('button', { name: 'Submit Broker Order' }));

    expect(mutate).toHaveBeenCalledWith(
      expect.objectContaining({
        symbol: 'AAPL',
        side: 'buy',
        type: 'market',
        quantity: 10,
        stopLossPrice: 195,
      }),
      expect.any(Object)
    );
  });

  it('blocks manual ETF entries from the order ticket', async () => {
    const user = userEvent.setup();
    etfInstruments = [
      {
        symbol: 'SPY',
        asset_class: 'equity',
        instrument_type: 'etf',
        tradable_modes: ['approval'],
        eligibility_state: 'approval_required',
        effective_date: '2026-01-01',
        change_owner: 'policy',
        exclusions: [],
      },
    ];

    renderPanel();

    await user.type(screen.getByLabelText('Symbol'), 'SPY');
    await user.type(screen.getByLabelText('Quantity'), '10');

    expect(screen.getByText(/Approval required for this ETF/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open approval flow' })).toHaveAttribute('href', '/etf/approvals');
    expect(screen.getByRole('button', { name: 'Submit BUY Order' })).toBeDisabled();
    expect(mutate).not.toHaveBeenCalled();
  });

  it('allows manual ETF entry when policy marks symbol as manual allowed', async () => {
    const user = userEvent.setup();
    etfInstruments = [
      {
        symbol: 'IWM',
        asset_class: 'equity',
        instrument_type: 'etf',
        tradable_modes: ['manual', 'approval'],
        eligibility_state: 'eligible',
        effective_date: '2026-01-01',
        change_owner: 'policy',
        exclusions: [],
      },
    ];

    renderPanel();

    await user.type(screen.getByLabelText('Symbol'), 'IWM');
    await user.type(screen.getByLabelText('Quantity'), '5');

    expect(screen.queryByText(/Approval required for this ETF/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Manual entry is blocked for this ETF/i)).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Submit BUY Order' })).toBeEnabled();
  });

  it('shows blocked policy reason and recovery actions for blocked ETF symbols', async () => {
    const user = userEvent.setup();
    etfInstruments = [
      {
        symbol: 'QQQ',
        asset_class: 'equity',
        instrument_type: 'etf',
        tradable_modes: [],
        eligibility_state: 'blocked',
        effective_date: '2026-01-01',
        change_owner: 'policy',
        exclusions: ['Policy: ETF new entries require managed workflow'],
      },
    ];

    renderPanel();

    await user.type(screen.getByLabelText('Symbol'), 'QQQ');
    await user.type(screen.getByLabelText('Quantity'), '10');

    expect(screen.getByText(/Manual entry is blocked for this ETF/i)).toBeInTheDocument();
    expect(screen.getByText(/Reason:/i)).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open approved ETF workflow' })).toHaveAttribute('href', '/etf/approvals');

    await user.click(screen.getByRole('button', { name: 'Choose another symbol' }));

    expect(screen.getByLabelText('Symbol')).toHaveValue('');
  });
});
