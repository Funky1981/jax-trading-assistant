import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { ScannerSettingsCard } from './ScannerSettingsCard';
import type { ScannerSettings } from '@/data/types';

const baseSettings: ScannerSettings = {
  enabled: true,
  assetScope: 'etf',
  symbols: ['SPY', 'QQQ', 'IWM'],
  universePreset: 'etf-core',
  intervalSeconds: 300,
  minimumConfidence: 0.7,
  connected: true,
  sentiment: {
    enabled: false,
    sourceScope: 'news',
    timeWindow: '24h',
    minimumThresholdLabel: '60%',
    minimumSourceCount: 3,
    sourceTrustMode: 'equal',
    mode: 'filter',
    supported: true,
    connected: true,
  },
};

describe('ScannerSettingsCard', () => {
  it('renders default scanner and sentiment controls with accessible labels', () => {
    render(<ScannerSettingsCard settings={baseSettings} />);

    expect(screen.getByRole('heading', { name: 'Scanner settings' })).toBeInTheDocument();
    expect(screen.getByText('Watching')).toBeInTheDocument();
    expect(screen.getByText('Connected')).toBeInTheDocument();
    expect(screen.getByLabelText('Asset scope')).toHaveDisplayValue('etf');
    expect(screen.getByLabelText('Symbols')).toHaveDisplayValue('SPY, QQQ, IWM');
    expect(screen.getByLabelText('Minimum confidence')).toHaveDisplayValue('70%');
    expect(screen.getByLabelText('Sentiment source scope')).toHaveDisplayValue('news');
    expect(screen.getByLabelText('Sentiment mode')).toHaveDisplayValue('Filter');
  });

  it('shows disabled scanner state without hiding the settings', () => {
    render(<ScannerSettingsCard settings={{ ...baseSettings, enabled: false }} />);

    expect(screen.getByText('Paused')).toBeInTheDocument();
    expect(screen.getByLabelText('Scan interval')).toHaveDisplayValue('300 seconds');
  });

  it('shows connected scanner help text and toggle button', () => {
    render(<ScannerSettingsCard onToggleScanner={() => undefined} settings={baseSettings} />);

    expect(screen.getByText(/persisted and connected to the AI scanner API/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Pause scanner/i })).toBeInTheDocument();
  });
});
