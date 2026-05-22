import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ScannerSettingsCard } from './ScannerSettingsCard';
import type { ScannerSettings } from '@/data/types';

const baseSettings: ScannerSettings = {
  enabled: true,
  assetScope: 'ETF pilot',
  symbols: ['SPY', 'QQQ', 'IWM'],
  universePreset: 'Phase 1 ETF universe',
  intervalSeconds: 30,
  minimumConfidence: 0.65,
  connected: false,
  sentiment: {
    enabled: true,
    sourceScope: 'Trusted news and filings',
    timeWindow: 'Last 24 hours',
    minimumThresholdLabel: 'Positive or better',
    minimumSourceCount: 2,
    sourceTrustMode: 'trust_weighted',
    mode: 'rank_boost',
    supported: false,
    connected: false,
    unsupportedReason: 'Sentiment routing needs Phase 2 backend support.',
  },
};

describe('ScannerSettingsCard', () => {
  it('renders default scanner and sentiment controls with accessible labels', () => {
    render(<ScannerSettingsCard settings={baseSettings} />);

    expect(screen.getByRole('heading', { name: 'Scanner settings' })).toBeInTheDocument();
    expect(screen.getByText('Watching')).toBeInTheDocument();
    expect(screen.getByLabelText('Asset scope')).toHaveDisplayValue('ETF pilot');
    expect(screen.getByLabelText('Symbols')).toHaveDisplayValue('SPY, QQQ, IWM');
    expect(screen.getByLabelText('Minimum confidence')).toHaveDisplayValue('65%');
    expect(screen.getByLabelText('Sentiment source scope')).toHaveDisplayValue('Trusted news and filings');
    expect(screen.getByLabelText('Sentiment mode')).toHaveDisplayValue('Rank boost');
  });

  it('shows disabled scanner state without hiding the settings', () => {
    render(<ScannerSettingsCard settings={{ ...baseSettings, enabled: false }} />);

    expect(screen.getByText('Paused')).toBeInTheDocument();
    expect(screen.getByLabelText('Scan interval')).toHaveDisplayValue('30 seconds');
  });

  it('explains unsupported sentiment placeholders', () => {
    render(<ScannerSettingsCard settings={baseSettings} />);

    expect(screen.getByText('Unsupported until Phase 2')).toBeInTheDocument();
    expect(screen.getByText('Sentiment routing needs Phase 2 backend support.')).toBeInTheDocument();
    expect(screen.getByText(/Settings changes are not connected until Phase 2/i)).toBeInTheDocument();
  });
});
