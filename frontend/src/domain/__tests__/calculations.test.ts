import { describe, expect, it } from 'vitest';
import {
  calculatePositionValue,
  calculateTotalExposure,
  calculateTotalUnrealizedPnl,
  calculateUnrealizedPnl,
  isRiskBreached,
} from '../calculations';
import type { Position, RiskLimits } from '../models';

describe('calculations', () => {
  const positions: Position[] = [
    { symbol: 'AAPL', quantity: 10, avgPrice: 100, marketPrice: 110 },
    { symbol: 'MSFT', quantity: -5, avgPrice: 200, marketPrice: 190 },
  ];

  it('computes position value', () => {
    expect(calculatePositionValue(positions[0])).toBe(1100);
  });

  it('computes unrealized pnl', () => {
    expect(calculateUnrealizedPnl(positions[0])).toBe(100);
  });

  it('computes total exposure', () => {
    expect(calculateTotalExposure(positions)).toBe(1100 + 950);
  });

  it('computes total pnl', () => {
    expect(calculateTotalUnrealizedPnl(positions)).toBe(100 + 50);
  });

  it('flags breaches when limits are exceeded', () => {
    const limits: RiskLimits = { maxPositionValue: 1000, maxDailyLoss: 10 };
    expect(isRiskBreached(limits, 1200, -5)).toBe(true);
    expect(isRiskBreached(limits, 900, -15)).toBe(true);
    expect(isRiskBreached(limits, 900, 5)).toBe(false);
  });
});
