import { describe, expect, it } from 'vitest';
import { calculateTotalExposure, calculateTotalUnrealizedPnl } from '../../domain/calculations';
import type { Position } from '../../domain/models';

function buildPositions(count: number): Position[] {
  const positions: Position[] = [];
  for (let i = 0; i < count; i += 1) {
    positions.push({
      symbol: `SYM${i}`,
      quantity: (i % 10) + 1,
      avgPrice: 100 + (i % 25),
      marketPrice: 100 + (i % 30),
    });
  }
  return positions;
}

describe('domain calculation performance', () => {
  it('computes exposure and pnl within budget', () => {
    const positions = buildPositions(50_000);

    const started = performance.now();
    calculateTotalExposure(positions);
    calculateTotalUnrealizedPnl(positions);
    const elapsed = performance.now() - started;

    expect(elapsed).toBeLessThan(500);
  });
});
