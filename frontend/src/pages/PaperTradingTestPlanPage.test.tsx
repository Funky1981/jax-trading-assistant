import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it } from 'vitest';
import { PaperTradingTestPlanPage } from './PaperTradingTestPlanPage';

const STORAGE_KEY = 'jax.paperTradingTestPlan.v1';

describe('PaperTradingTestPlanPage', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('persists checklist state in localStorage', () => {
    render(
      <MemoryRouter>
        <PaperTradingTestPlanPage />
      </MemoryRouter>
    );

    const targetCheckbox = screen.getByRole('checkbox', {
      name: /Switch to paper mode/i,
    });

    fireEvent.click(targetCheckbox);

    const stored = window.localStorage.getItem(STORAGE_KEY);
    expect(stored).toContain('"paper-1":true');
  });
});
