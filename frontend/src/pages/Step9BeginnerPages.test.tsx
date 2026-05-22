import { render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { BeginnerUXProvider } from '@/context/BeginnerUXContext';
import { ETFUniversePage } from './ETFUniversePage';
import { StrategyCardsPage } from './StrategyCardsPage';
import { ResearchTimelinePage } from './ResearchTimelinePage';

function renderWithBeginnerUX(ui: ReactNode) {
  return render(
    <MemoryRouter>
      <BeginnerUXProvider>{ui}</BeginnerUXProvider>
    </MemoryRouter>
  );
}

describe('Step 9 beginner pages', () => {
  it('renders ETF universe page content', async () => {
    renderWithBeginnerUX(<ETFUniversePage />);
    expect(await screen.findByText('ETF Universe')).toBeInTheDocument();
    expect(screen.getByText('All Approved ETFs')).toBeInTheDocument();
  });

  it('renders strategy cards page content', async () => {
    renderWithBeginnerUX(<StrategyCardsPage />);
    expect(await screen.findByText('Trading Strategies')).toBeInTheDocument();
  });

  it('renders research timeline page content', async () => {
    renderWithBeginnerUX(<ResearchTimelinePage />);
    expect(await screen.findByText('Trade Timeline Example')).toBeInTheDocument();
  });
});
