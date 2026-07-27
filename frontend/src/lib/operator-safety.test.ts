import { describe, expect, it } from 'vitest';
import type { OperatorEvidenceOverview } from '@/data/operator-evidence-service';
import { interpretOperatorSafety, isPaperSafe } from './operator-safety';

const safe = {
  runtimeMode: 'paper',
  allowLiveTrading: false,
  executionEnabled: false,
  executionWorkerEnabled: false,
  brokerExecutionAllowed: false,
  maximumLeverage: 1,
} as OperatorEvidenceOverview;

describe('operator safety interpretation', () => {
  it('uses one paper-safe interpretation for all consumers', () => {
    expect(isPaperSafe(safe)).toBe(true);
    expect(interpretOperatorSafety(safe).state).toBe('safe');
  });

  it.each([
    ['live trading', { allowLiveTrading: true }],
    ['execution', { executionEnabled: true }],
    ['execution worker', { executionWorkerEnabled: true }],
    ['broker execution', { brokerExecutionAllowed: true }],
    ['leverage', { maximumLeverage: 1.5 }],
  ])('warns for unsafe %s', (_name, change) => {
    expect(interpretOperatorSafety({ ...safe, ...change }).state).toBe('unsafe');
  });

  it.each([
    ['broker execution', { brokerExecutionAllowed: null }, 'broker'],
    ['maximum leverage', { maximumLeverage: null }, 'leverage'],
  ] as const)('treats missing %s as unknown, never safe', (_name, change, key) => {
    const result = interpretOperatorSafety({ ...safe, ...change });
    expect(result.state).toBe('unknown');
    expect(result.checks.find((item) => item.key === key)?.value).toBe('Unknown');
    expect(isPaperSafe({ ...safe, ...change })).toBe(false);
  });
});
